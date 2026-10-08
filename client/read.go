package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// WildcardInstance in a Device object identifier means "this device".
const WildcardInstance = 4194303

// ReadSpec asks for some properties of one object.
type ReadSpec struct {
	Object ObjectID
	Props  []uint16
}

// PropertyResult is one property of a ReadPropertyMultiple reply or of a COV
// notification: its value, or the error the device gave for it.
type PropertyResult struct {
	ID     uint32
	Index  *uint32
	Values []Value
	Err    error
}

// ObjectResult is the properties read from one object.
type ObjectResult struct {
	Object ObjectID
	Props  []PropertyResult
}

// ReadProperty reads one property. A property that is an array or a list
// comes back as several values.
func (c *Client) ReadProperty(ctx context.Context, to Address, obj ObjectID, prop uint16) ([]Value, error) {
	return c.readProperty(ctx, to, services.ConfirmedReadPropertyObjects(obj.Type, obj.Instance, prop))
}

// ReadPropertyIndex reads one element of an array property. Element 0 is the
// number of elements.
func (c *Client) ReadPropertyIndex(ctx context.Context, to Address, obj ObjectID, prop uint16, index uint32) ([]Value, error) {
	return c.readProperty(ctx, to, services.ConfirmedReadPropertyIndexedObjects(obj.Type, obj.Instance, prop, index))
}

func (c *Client) readProperty(ctx context.Context, to Address, payload []objects.APDUPayload) ([]Value, error) {
	tags, err := c.confirmed(ctx, to, services.ServiceConfirmedReadProperty, payload)
	if err != nil {
		return nil, err
	}
	return propertyValue(tags)
}

// propertyValue takes the value out of a ReadProperty ack.
func propertyValue(tags []*objects.Object) ([]Value, error) {
	// Object identifier [0], property [1] and perhaps an array index [2],
	// then the value between opening and closing tag 3.
	for i, t := range tags {
		if t.Kind != objects.TagOpening {
			continue
		}
		if t.TagNumber != 3 {
			break
		}
		values, rest, err := decodeUntilClose(tags[i+1:])
		if err != nil {
			return nil, fmt.Errorf("decoding the reply: %w", err)
		}
		if len(rest) != 1 || rest[0].TagNumber != 3 {
			break
		}
		return values, nil
	}
	return nil, errors.New("decoding the reply: no property value in it")
}

// ReadPropertyMultiple reads properties of several objects in one request.
// The reply has to fit in what the device can send, so the caller sizes its
// batches to the device.
func (c *Client) ReadPropertyMultiple(ctx context.Context, to Address, specs []ReadSpec) ([]ObjectResult, error) {
	var payload []objects.APDUPayload
	for _, s := range specs {
		payload = append(payload,
			services.ConfirmedReadPropertyMultipleObjects(s.Object.Type, s.Object.Instance, s.Props)...)
	}
	tags, err := c.confirmed(ctx, to, services.ServiceConfirmedReadPropMultiple, payload)
	if err != nil {
		return nil, err
	}
	results, err := decodeRPM(tags)
	if err != nil {
		return nil, fmt.Errorf("decoding the reply: %w", err)
	}
	return results, nil
}

// decodeRPM decodes a ReadPropertyMultiple ack: for each object its
// identifier [0] and, between tags 1, a run of property [2], perhaps array
// index [3], and either a value between tags 4 or an error between tags 5.
func decodeRPM(tags []*objects.Object) ([]ObjectResult, error) {
	var results []ObjectResult
	for len(tags) > 0 {
		if len(tags) < 2 || !isContext(tags[0], 0) || len(tags[0].Data) != 4 || !isOpening(tags[1], 1) {
			return nil, errors.New("expected an object identifier and its results")
		}
		res := ObjectResult{Object: objectIDOf(tags[0].Data)}
		tags = tags[2:]

		for {
			if len(tags) == 0 {
				return nil, errors.New("results are not closed")
			}
			if tags[0].Kind == objects.TagClosing && tags[0].TagNumber == 1 {
				tags = tags[1:]
				break
			}
			if !isContext(tags[0], 2) || len(tags[0].Data) == 0 || len(tags[0].Data) > 4 {
				return nil, errors.New("expected a property identifier")
			}
			prop := PropertyResult{ID: uint32(beUint(tags[0].Data))}
			tags = tags[1:]
			if len(tags) > 0 && isContext(tags[0], 3) && len(tags[0].Data) >= 1 && len(tags[0].Data) <= 4 {
				index := uint32(beUint(tags[0].Data))
				prop.Index = &index
				tags = tags[1:]
			}
			if len(tags) == 0 || tags[0].Kind != objects.TagOpening || (tags[0].TagNumber != 4 && tags[0].TagNumber != 5) {
				return nil, errors.New("expected a property value or error")
			}
			opened := tags[0].TagNumber
			values, rest, err := decodeUntilClose(tags[1:])
			if err != nil {
				return nil, err
			}
			if len(rest) == 0 || rest[0].TagNumber != opened {
				return nil, errors.New("a property value is not closed")
			}
			tags = rest[1:]

			if opened == 4 {
				prop.Values = values
			} else {
				e := &BACnetError{}
				if len(values) == 2 {
					e.Class, e.Code = uint32(values[0].Uint), uint32(values[1].Uint)
				}
				prop.Err = e
			}
			res.Props = append(res.Props, prop)
		}
		results = append(results, res)
	}
	return results, nil
}

func isContext(t *objects.Object, number uint8) bool {
	return t.TagClass && t.Kind == objects.TagPrimitive && t.TagNumber == number
}

func isOpening(t *objects.Object, number uint8) bool {
	return t.Kind == objects.TagOpening && t.TagNumber == number
}

// ReadObjectList reads the identifiers of every object in a device. It asks
// for the whole array first and, where the device declines that, as one that
// cannot segment does for a long list, reads it element by element. progress,
// if not nil, is told how many of how many it has in that case.
func (c *Client) ReadObjectList(ctx context.Context, dev Device, progress func(done, total int)) ([]ObjectID, error) {
	device := ObjectID{Type: objects.ObjectTypeDevice, Instance: dev.Instance}
	values, err := c.ReadProperty(ctx, dev.Addr, device, objects.PropertyIdObjectList)
	if err == nil {
		return objectIDs(values), nil
	}
	if !Declined(err) {
		return nil, err
	}

	values, err = c.ReadPropertyIndex(ctx, dev.Addr, device, objects.PropertyIdObjectList, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 || values[0].Kind != KindUnsigned {
		return nil, errors.New("the object list's length is not a number")
	}
	total := int(values[0].Uint)
	list := make([]ObjectID, 0, total)
	for i := 1; i <= total; i++ {
		values, err := c.ReadPropertyIndex(ctx, dev.Addr, device, objects.PropertyIdObjectList, uint32(i))
		if err != nil {
			return nil, fmt.Errorf("object list element %d: %w", i, err)
		}
		list = append(list, objectIDs(values)...)
		if progress != nil {
			progress(i, total)
		}
	}
	return list, nil
}

func objectIDs(values []Value) []ObjectID {
	ids := make([]ObjectID, 0, len(values))
	for _, v := range values {
		if v.Kind == KindObjectID {
			ids = append(ids, v.Object)
		}
	}
	return ids
}

var errorClassNames = []string{
	"device", "object", "property", "resources", "security", "services", "vt", "communication",
}

// ErrorClassName names a BACnetErrorClass.
func ErrorClassName(v uint32) string {
	if int(v) < len(errorClassNames) {
		return errorClassNames[v]
	}
	return fmt.Sprintf("class %d", v)
}

// Error codes worth telling apart from the rest.
const (
	ErrorCodeUnknownObject   = 31
	ErrorCodeUnknownProperty = 32
)

var errorCodeNames = map[uint32]string{
	0: "other", 2: "configuration in progress", 3: "device busy", 9: "inconsistent parameters",
	25: "operational problem", 27: "read access denied", 29: "service request denied", 30: "timeout",
	31: "unknown object", 32: "unknown property", 37: "value out of range", 40: "write access denied",
	42: "invalid array index", 44: "not a COV property", 45: "optional functionality not supported",
	50: "property is not an array",
}

// ErrorCodeName names a BACnetErrorCode.
func ErrorCodeName(v uint32) string {
	if name, ok := errorCodeNames[v]; ok {
		return name
	}
	return fmt.Sprintf("code %d", v)
}

// ---- ReadMany --------------------------------------------------------------

// errNotInReply is the error given for a property of an object the device
// left out of a ReadPropertyMultiple reply.
var errNotInReply = errors.New("not in the reply")

// ReadMany reads properties of many objects of one device, with
// ReadPropertyMultiple in requests sized to what the device can answer, and
// with ReadProperty one property at a time where the device has no
// ReadPropertyMultiple, or declines one for a single object. Once a device
// has said it has no ReadPropertyMultiple, the client does not ask it again.
//
// The results come in the order of specs, one for each, with the error the
// device gave for a property in its PropertyResult. ReadMany itself returns
// an error only when the device stops answering, or ctx ends.
func (c *Client) ReadMany(ctx context.Context, dev Device, specs []ReadSpec) ([]ObjectResult, error) {
	results := make([]ObjectResult, len(specs))
	for i, s := range specs {
		results[i] = ObjectResult{Object: s.Object}
	}
	batch := readManyBatch(dev, specs)
	for start := 0; start < len(specs); start += batch {
		end := min(start+batch, len(specs))
		if err := c.readMany(ctx, dev, specs[start:end], results[start:end]); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// readManyBatch is how many objects to ask about in one request. A device
// that sends segmented replies can answer for many at once; one that cannot
// has to fit the reply in one APDU, so the batch is sized by an estimate of
// what each object and property comes back as.
func readManyBatch(dev Device, specs []ReadSpec) int {
	const (
		most           = 32
		perObject      = 8
		perProperty    = 24
		replyOverhead  = 16
		smallestAPDU   = 50
		segmentsBoth   = 0
		segmentsToSend = 1
	)
	if dev.Segmentation == segmentsBoth || dev.Segmentation == segmentsToSend {
		return most
	}
	props := 0
	for _, s := range specs {
		props += len(s.Props)
	}
	if len(specs) > 0 {
		props = max(1, props/len(specs))
	}
	budget := int(dev.MaxAPDU)
	if budget < smallestAPDU {
		budget = smallestAPDU
	}
	return max(1, min(most, (budget-replyOverhead)/(perObject+perProperty*props)))
}

// readMany fills out, one result for each spec, from one request where it
// can, and from less at a time where the device declines.
func (c *Client) readMany(ctx context.Context, dev Device, specs []ReadSpec, out []ObjectResult) error {
	if c.lacksRPM(dev.Addr) {
		for i := range specs {
			if err := c.readSingly(ctx, dev.Addr, specs[i], &out[i]); err != nil {
				return err
			}
		}
		return nil
	}

	got, err := c.ReadPropertyMultiple(ctx, dev.Addr, specs)
	switch {
	case err == nil:
		byObject := make(map[ObjectID]ObjectResult, len(got))
		for _, r := range got {
			byObject[r.Object] = r
		}
		for i, s := range specs {
			if r, ok := byObject[s.Object]; ok {
				out[i].Props = r.Props
				continue
			}
			for _, p := range s.Props {
				out[i].Props = append(out[i].Props, PropertyResult{ID: uint32(p), Err: errNotInReply})
			}
		}
		return nil
	case gone(err):
		return err
	case noRPM(err):
		c.mu.Lock()
		c.noRPM[keyOf(dev.Addr, 0)] = true
		c.mu.Unlock()
		c.log.Info("device has no ReadPropertyMultiple", "device", dev.Instance, "at", dev.Addr)
		return c.readMany(ctx, dev, specs, out)
	case len(specs) > 1:
		// The device declined, or sent something unreadable; most often the
		// reply would not fit. Ask for less at a time.
		half := len(specs) / 2
		if err := c.readMany(ctx, dev, specs[:half], out[:half]); err != nil {
			return err
		}
		return c.readMany(ctx, dev, specs[half:], out[half:])
	}
	// One object the device declined, for whatever reason: property by
	// property, so that its answer for each is kept.
	return c.readSingly(ctx, dev.Addr, specs[0], &out[0])
}

// readSingly reads an object with one ReadProperty for each property.
func (c *Client) readSingly(ctx context.Context, to Address, spec ReadSpec, out *ObjectResult) error {
	for _, prop := range spec.Props {
		values, err := c.ReadProperty(ctx, to, spec.Object, prop)
		if gone(err) {
			return err
		}
		out.Props = append(out.Props, PropertyResult{ID: uint32(prop), Values: values, Err: err})
	}
	return nil
}

func (c *Client) lacksRPM(to Address) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.noRPM[keyOf(to, 0)]
}

// noRPM reports whether an error is the device saying it has no
// ReadPropertyMultiple, rather than declining one particular request.
func noRPM(err error) bool {
	var re *RejectError
	if errors.As(err, &re) {
		return re.Reason == objects.RejectReasonUnrecognizedService
	}
	var be *BACnetError
	if errors.As(err, &be) {
		return be.Class == errorClassServices &&
			(be.Code == errorCodeServiceRequestDenied || be.Code == errorCodeOptionalFunctionalityNotSupported)
	}
	return false
}

const (
	errorClassServices                         = 5
	errorCodeServiceRequestDenied              = 29
	errorCodeOptionalFunctionalityNotSupported = 45
)

// gone reports whether err means the device is not answering, or that
// nobody is waiting for the answer any more. Any other error is the device's
// answer to one request, and the next request is worth making.
func gone(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, ErrClosed) || errors.Is(err, ErrSourceDenied) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
