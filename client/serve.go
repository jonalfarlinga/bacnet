package client

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Request is a confirmed request another device sent the program.
type Request struct {
	// From is the device that asks, and where the reply goes.
	From     Address
	Service  uint8
	InvokeID uint8
	// Device is the instance of the client's identity: the device that is
	// asked.
	Device uint32
	// Tags is the parameters of the service, as they came.
	Tags []*objects.Object
	// MaxReply is the longest reply, in bytes of APDU, that the requester
	// takes and the client sends. A longer one is not sent: the requester
	// gets an Abort, since the client does not segment its replies.
	MaxReply int
}

// Handler answers the confirmed requests of one service. It returns the
// parameters of the ComplexACK, or none for a service that is answered with a
// SimpleACK. An error is the answer too: a *BACnetError goes as an Error, a
// *RejectError as a Reject and an *AbortError as an Abort; any other error
// goes as the Error device: operational-problem, and is logged.
//
// It is called on a goroutine of its own, so it may take its time and may use
// the client. ctx ends when the client closes.
type Handler func(ctx context.Context, req Request) ([]*objects.Object, error)

// MaxServing is how many requests the client has with the program's handlers
// at once. One that comes when they are all busy is aborted.
const MaxServing = 64

// ErrNoIdentity is a handler for a client that is no device.
var ErrNoIdentity = errors.New("the client has no identity")

// Error class device and code operational-problem: what a handler that
// failed is reported as.
const (
	errorClassDevice            = 0
	errorCodeOperationalProblem = 25
)

// Sizes of the max-APDU codes of a confirmed request.
var maxAPDUSizes = [...]int{50, 128, 206, 480, 1024, 1476}

// Handle gives the confirmed requests of a service to a handler, or with a
// nil handler, to none. A client with an identity then answers them: the
// client matches the reply to the request and sends it back the way the
// request came, and what the reply says is the handler's. A request for a
// service with no handler is rejected as an unrecognized service. Replies are
// not segmented.
//
// It fails with ErrNoIdentity without Config.Identity, and with
// ErrServiceNotAllowed with Config.ReadOnly: such a client answers nothing.
// The confirmed COV notifications of the client's own subscriptions are
// acknowledged by the client and never reach a handler.
func (c *Client) Handle(service uint8, h Handler) error {
	if c.cfg.Identity == nil {
		return ErrNoIdentity
	}
	if c.cfg.ReadOnly {
		return fmt.Errorf("answering service %d on a read-only client: %w", service, ErrServiceNotAllowed)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if h == nil {
		delete(c.handlers, service)
		return nil
	}
	c.handlers[service] = h
	return nil
}

// request takes a confirmed request: it goes to the handler of its service,
// or is turned down.
func (c *Client) request(src netip.AddrPort, bvlc *plumbing.BVLC, npdu *plumbing.NPDU, a *plumbing.APDU) {
	if c.cfg.Identity == nil || c.cfg.ReadOnly {
		c.log.Debug("ignored a confirmed request", "from", sourceOf(src, npdu), "service", a.Service)
		return
	}
	// A confirmed request is for one device: a broadcast is not one, and one
	// that still names a network is for a device there.
	if bvlc.Function != plumbing.BVLCFuncUnicast || npdu.Control&0x20 != 0 {
		return
	}
	c.noteSource(src, npdu)
	from := sourceOf(src, npdu)
	key := keyOf(from, a.InvokeID)

	c.mu.Lock()
	if _, again := c.serving[key]; again {
		c.mu.Unlock()
		return // sent again while the handler has it
	}
	h := c.handlers[a.Service]
	busy := len(c.serving) >= MaxServing
	c.serving[key] = struct{}{}
	c.mu.Unlock()

	req := Request{From: from, Service: a.Service, InvokeID: a.InvokeID, Device: c.cfg.Identity.Instance, MaxReply: identityMaxAPDU}
	if int(a.MaxSize) < len(maxAPDUSizes) {
		req.MaxReply = min(req.MaxReply, maxAPDUSizes[a.MaxSize])
	}
	var err error
	switch {
	case a.IsSegmented():
		err = &AbortError{Reason: objects.AbortReasonSegmentationNotSupported}
	case h == nil:
		err = &RejectError{Reason: objects.RejectReasonUnrecognizedService}
	case busy:
		err = &AbortError{Reason: objects.AbortReasonOutOfResources}
	default:
		if req.Tags, err = toTags(a.Objects); err != nil {
			err = &RejectError{Reason: objects.RejectReasonInvalidTag}
		}
	}
	if err != nil {
		c.log.Debug("turned down a confirmed request", "from", from, "service", a.Service, "error", err)
		c.answer(key, src, req, nil, err)
		return
	}
	go func() {
		tags, err := h(c.ctx, req)
		c.answer(key, src, req, tags, err)
	}()
}

// unparsed takes a datagram the parser has no type for: a confirmed request
// for a service the library does not know is still a request, which a
// handler may be there for, and which is rejected otherwise.
func (c *Client) unparsed(src netip.AddrPort, raw []byte) bool {
	var bvlc plumbing.BVLC
	if bvlc.UnmarshalBinary(raw) != nil {
		return false
	}
	offset := bvlc.MarshalLen()
	var npdu plumbing.NPDU
	if offset > len(raw) || npdu.UnmarshalBinary(raw[offset:]) != nil || npdu.Control&plumbing.NetworkLayerMessage != 0 {
		return false
	}
	offset += npdu.MarshalLen()
	var apdu plumbing.APDU
	if offset >= len(raw) || apdu.UnmarshalBinary(raw[offset:]) != nil || apdu.Type != plumbing.ConfirmedReq {
		return false
	}
	c.request(src, &bvlc, &npdu, &apdu)
	return true
}

// answer sends the reply to a request, and with it the request is done.
func (c *Client) answer(key txKey, src netip.AddrPort, req Request, tags []*objects.Object, err error) {
	defer func() {
		c.mu.Lock()
		delete(c.serving, key)
		c.mu.Unlock()
	}()
	reply := c.replyTo(req, tags, err)
	if reply.MarshalLen() > req.MaxReply {
		c.log.Warn("a reply is longer than the requester takes", "to", req.From, "service", req.Service,
			"bytes", reply.MarshalLen(), "max", req.MaxReply)
		reply = &plumbing.APDU{
			Type: plumbing.Abort, Flags: 0x1, InvokeID: req.InvokeID,
			Reason: objects.AbortReasonSegmentationNotSupported,
		}
	}
	// The reply goes back the way the request came.
	if err := c.sendAPDU(plumbing.BVLCFuncUnicast, src, npduTo(req.From, false), reply); err != nil {
		c.log.Debug("a reply was not sent", "to", req.From, "service", req.Service, "error", err)
	}
}

// replyTo builds the PDU that says what a handler returned.
func (c *Client) replyTo(req Request, tags []*objects.Object, err error) *plumbing.APDU {
	var (
		be *BACnetError
		re *RejectError
		ae *AbortError
	)
	switch {
	case err == nil && len(tags) == 0:
		return &plumbing.APDU{Type: plumbing.SimpleAck, InvokeID: req.InvokeID, Service: req.Service}
	case err == nil:
		ack := plumbing.NewAPDU(plumbing.ComplexAck, req.Service, nil)
		ack.InvokeID = req.InvokeID
		for _, t := range tags {
			ack.Objects = append(ack.Objects, t)
		}
		return ack
	case errors.As(err, &re):
		return &plumbing.APDU{Type: plumbing.Reject, InvokeID: req.InvokeID, Reason: re.Reason}
	case errors.As(err, &ae):
		return &plumbing.APDU{Type: plumbing.Abort, Flags: 0x1, InvokeID: req.InvokeID, Reason: ae.Reason}
	case !errors.As(err, &be):
		c.log.Warn("a handler failed", "from", req.From, "service", req.Service, "error", err)
		be = &BACnetError{Class: errorClassDevice, Code: errorCodeOperationalProblem}
	}
	fault := plumbing.NewAPDU(plumbing.Error, req.Service,
		[]objects.APDUPayload{objects.EncEnumerated(be.Class), objects.EncEnumerated(be.Code)})
	fault.InvokeID = req.InvokeID
	return fault
}

// replyAllowed reports whether an APDU is one that answers a confirmed
// request: an acknowledgement, an Error, a Reject, or a server's Abort. The
// client sends one only to a device with a request open, and never in
// segments.
func replyAllowed(a *plumbing.APDU) bool {
	switch a.Type {
	case plumbing.SimpleAck, plumbing.Error, plumbing.Reject:
		return true
	case plumbing.ComplexAck:
		return !a.IsSegmented()
	case plumbing.Abort:
		return a.Flags&0x1 != 0
	}
	return false
}

// ---- ReadProperty and ReadPropertyMultiple ---------------------------------

// PropertyRef names a property of an object, or one element of it.
type PropertyRef struct {
	ID uint32
	// Index is the element of an array property that is asked for, 0 for
	// the number of elements; nil for the whole property.
	Index *uint32
}

// thisDevice gives the Device object of the identity for the one with the
// wildcard instance, which is how a requester that does not know the
// instance names it.
func (r Request) thisDevice(object ObjectID) ObjectID {
	if object.Type == objects.ObjectTypeDevice && object.Instance == WildcardInstance {
		object.Instance = r.Device
	}
	return object
}

// PropertyReader gives the value of one property of one of the program's
// objects, as the tags it is encoded in. A Device object asked for by the
// wildcard instance is passed, and answered, as the identity's own. An error that is a *BACnetError is
// what the requester is told: object: unknown-object or property:
// unknown-property, for the most part. The properties all (8), required
// (105) and optional (80) of a ReadPropertyMultiple are passed as they are.
type PropertyReader func(ctx context.Context, object ObjectID, property PropertyRef) ([]*objects.Object, error)

// ServeReadProperty is a Handler for ReadProperty that takes the request
// apart, asks read for the value and puts the reply together.
func ServeReadProperty(read PropertyReader) Handler {
	return func(ctx context.Context, req Request) ([]*objects.Object, error) {
		// Object identifier [0], property [1] and perhaps an array index [2].
		t := req.Tags
		if len(t) < 2 || len(t) > 3 || !isContext(t[0], 0) || len(t[0].Data) != 4 ||
			!isContext(t[1], 1) || len(t[1].Data) == 0 || len(t[1].Data) > 4 {
			return nil, &RejectError{Reason: objects.RejectReasonMissingRequiredParameter}
		}
		object := req.thisDevice(objectIDOf(t[0].Data))
		property := PropertyRef{ID: uint32(beUint(t[1].Data))}
		if len(t) == 3 {
			if !isContext(t[2], 2) || len(t[2].Data) == 0 || len(t[2].Data) > 4 {
				return nil, &RejectError{Reason: objects.RejectReasonInvalidTag}
			}
			index := uint32(beUint(t[2].Data))
			property.Index = &index
		}
		value, err := read(ctx, object, property)
		if err != nil {
			return nil, err
		}

		reply := []*objects.Object{
			objects.EncObjectIdentifier(true, 0, object.Type, object.Instance),
			objects.ContextTag(1, objects.EncUnsignedInteger(uint(property.ID))),
		}
		if property.Index != nil {
			reply = append(reply, objects.ContextTag(2, objects.EncUnsignedInteger(uint(*property.Index))))
		}
		reply = append(reply, objects.EncOpeningTag(3))
		reply = append(reply, value...)
		return append(reply, objects.EncClosingTag(3)), nil
	}
}

// ServeReadPropertyMultiple is a Handler for ReadPropertyMultiple that takes
// the request apart, asks read for each value and puts the reply together.
// An error read gives for one property is the answer for that property alone.
func ServeReadPropertyMultiple(read PropertyReader) Handler {
	return func(ctx context.Context, req Request) ([]*objects.Object, error) {
		// For each object its identifier [0] and, between tags 1, properties
		// [0] with perhaps an array index [1].
		var reply []*objects.Object
		t := req.Tags
		if len(t) == 0 {
			return nil, &RejectError{Reason: objects.RejectReasonMissingRequiredParameter}
		}
		for len(t) > 0 {
			if len(t) < 3 || !isContext(t[0], 0) || len(t[0].Data) != 4 || !isOpening(t[1], 1) {
				return nil, &RejectError{Reason: objects.RejectReasonInvalidTag}
			}
			object := req.thisDevice(objectIDOf(t[0].Data))
			t = t[2:]
			reply = append(reply, objects.EncObjectIdentifier(true, 0, object.Type, object.Instance), objects.EncOpeningTag(1))
			for {
				if len(t) == 0 {
					return nil, &RejectError{Reason: objects.RejectReasonMissingRequiredParameter}
				}
				if t[0].Kind == objects.TagClosing && t[0].TagNumber == 1 {
					t = t[1:]
					reply = append(reply, objects.EncClosingTag(1))
					break
				}
				if !isContext(t[0], 0) || len(t[0].Data) == 0 || len(t[0].Data) > 4 {
					return nil, &RejectError{Reason: objects.RejectReasonInvalidTag}
				}
				property := PropertyRef{ID: uint32(beUint(t[0].Data))}
				t = t[1:]
				if len(t) > 0 && isContext(t[0], 1) && len(t[0].Data) >= 1 && len(t[0].Data) <= 4 {
					index := uint32(beUint(t[0].Data))
					property.Index = &index
					t = t[1:]
				}

				reply = append(reply, objects.ContextTag(2, objects.EncUnsignedInteger(uint(property.ID))))
				if property.Index != nil {
					reply = append(reply, objects.ContextTag(3, objects.EncUnsignedInteger(uint(*property.Index))))
				}
				value, err := read(ctx, object, property)
				if err == nil {
					reply = append(reply, objects.EncOpeningTag(4))
					reply = append(reply, value...)
					reply = append(reply, objects.EncClosingTag(4))
					continue
				}
				var be *BACnetError
				if !errors.As(err, &be) {
					be = &BACnetError{Class: errorClassDevice, Code: errorCodeOperationalProblem}
				}
				reply = append(reply, objects.EncOpeningTag(5),
					objects.EncEnumerated(be.Class), objects.EncEnumerated(be.Code), objects.EncClosingTag(5))
			}
		}
		return reply, nil
	}
}

// The services ServeReadProperty and ServeReadPropertyMultiple are handlers
// for, as Handle takes them.
const (
	ServiceReadProperty         = services.ServiceConfirmedReadProperty
	ServiceReadPropertyMultiple = services.ServiceConfirmedReadPropMultiple
)
