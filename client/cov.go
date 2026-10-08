package client

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Defaults for COVOptions.
const (
	// DefaultCOVLifetime is how long a device keeps a subscription that is
	// not renewed.
	DefaultCOVLifetime = 5 * time.Minute
	// DefaultCOVProcessID is the subscriber process identifier the client
	// subscribes with.
	DefaultCOVProcessID = 1
)

// Backoff between renewals of a subscription after one fails. It never waits
// longer than a renewal that succeeded would.
const (
	covBackoffMin = 5 * time.Second
	covBackoffMax = 5 * time.Minute
)

// ErrSubscribed is a subscription the client already holds: the same object
// of the same device, with the same process identifier.
var ErrSubscribed = errors.New("already subscribed")

// COVOptions says how an object is subscribed to.
type COVOptions struct {
	// Confirmed asks for confirmed notifications, which the device sends
	// again until the client acknowledges them. Otherwise a notification
	// that is lost is lost.
	Confirmed bool
	// Lifetime is how long the device keeps the subscription without a
	// renewal; the client renews it at half of that. DefaultCOVLifetime if
	// not set. It is sent in whole seconds. There is no subscription without
	// an end: one the client does not live to cancel lapses on its own.
	Lifetime time.Duration
	// ProcessID is the subscriber process identifier, which with the
	// client's address and the object is what the device knows a
	// subscription by. DefaultCOVProcessID if 0.
	ProcessID uint32
}

// COVNotification is a device telling a subscriber what an object's
// properties are now.
type COVNotification struct {
	// From is where the notification came from.
	From Address
	// Device is the instance of the device that sent it.
	Device    uint32
	ProcessID uint32
	Object    ObjectID
	// Remaining is how long the subscription has left on the device.
	Remaining time.Duration
	// Confirmed says the device asked for an acknowledgement, which the
	// client has sent.
	Confirmed bool
	// Props is the properties reported: for most objects the present value
	// and the status flags. The priority a value may come with is not kept.
	Props []PropertyResult
}

// COVStatus is where a subscription stands.
type COVStatus struct {
	// Renewed is when the device last accepted the subscription, and Expires
	// when it drops it unless it is renewed again.
	Renewed time.Time
	Expires time.Time
	// Err is why the last subscription or renewal failed, nil when it did
	// not. The client keeps trying.
	Err error
	// Notifications counts the notifications delivered, and Last is when the
	// latest one came.
	Notifications int
	Last          time.Time
}

// covKey is what a notification says about the subscription it is for.
type covKey struct {
	peer    txKey
	process uint32
	object  ObjectID
}

// Subscription is one COV subscription the client keeps alive.
type Subscription struct {
	c       *Client
	key     covKey
	to      Address
	object  ObjectID
	opts    COVOptions
	seconds uint
	notify  func(COVNotification)

	// ctx ends when the subscription is cancelled.
	ctx  context.Context
	stop context.CancelFunc
	// sending is held while a renewal or the cancellation is with the
	// device, so that a renewal cannot follow the cancellation.
	sending   sync.Mutex
	cancelled bool

	mu     sync.Mutex
	status COVStatus
}

// SubscribeCOV asks a device to tell the client when an object changes, and
// renews the subscription until it is cancelled or the client closes. notify
// is called from the read loop for every notification of the subscription,
// the first of them as soon as the device accepts it, with the values as they
// are then. It must not block, nor wait for the client: a Cancel or a read
// made from it gets no answer. A confirmed notification the device sends
// again, for want of the acknowledgement, is delivered again.
//
// Close does not cancel: Cancel each subscription first, or leave it to lapse
// on the device at the end of its lifetime.
func (c *Client) SubscribeCOV(ctx context.Context, to Address, object ObjectID, opts COVOptions, notify func(COVNotification)) (*Subscription, error) {
	if notify == nil {
		return nil, errors.New("a subscription with nowhere to deliver its notifications")
	}
	if opts.Lifetime <= 0 {
		opts.Lifetime = DefaultCOVLifetime
	}
	if opts.ProcessID == 0 {
		opts.ProcessID = DefaultCOVProcessID
	}
	s := &Subscription{
		c:       c,
		key:     covKey{peer: keyOf(to, 0), process: opts.ProcessID, object: object},
		to:      to,
		object:  object,
		opts:    opts,
		seconds: uint((opts.Lifetime + time.Second - 1) / time.Second),
		notify:  notify,
	}
	s.ctx, s.stop = context.WithCancel(context.Background())

	// The device may send the first notification ahead of its answer.
	c.mu.Lock()
	if c.subs[s.key] != nil {
		c.mu.Unlock()
		s.stop()
		return nil, fmt.Errorf("%w: object %d,%d of %v, process %d", ErrSubscribed, object.Type, object.Instance, to, opts.ProcessID)
	}
	c.subs[s.key] = s
	c.mu.Unlock()

	if err := s.subscribe(ctx); err != nil {
		s.forget()
		return nil, err
	}
	go s.renewLoop()
	return s, nil
}

// CancelCOV tells a device to drop a subscription the client does not hold,
// such as one an earlier run left behind. A subscription the client holds is
// ended with its Cancel.
func (c *Client) CancelCOV(ctx context.Context, to Address, object ObjectID, processID uint32) error {
	if processID == 0 {
		processID = DefaultCOVProcessID
	}
	_, err := c.confirmed(ctx, to, services.ServiceConfirmedSubscribeCOV,
		services.CancelCOVOBjects(uint(processID), object.Type, object.Instance))
	return err
}

// Cancel ends the subscription: the client stops renewing it and tells the
// device to drop it. Whatever the device answers, no notification is
// delivered once Cancel has returned; when it returns an error the device may
// still hold the subscription, and Cancel can be called again.
func (s *Subscription) Cancel(ctx context.Context) error {
	s.stop()
	defer s.forget()
	s.sending.Lock()
	defer s.sending.Unlock()
	if s.cancelled {
		return nil
	}
	err := s.c.CancelCOV(ctx, s.to, s.object, s.opts.ProcessID)
	s.cancelled = err == nil
	return err
}

// Status reports where the subscription stands.
func (s *Subscription) Status() COVStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// forget stops the delivery of the subscription's notifications.
func (s *Subscription) forget() {
	s.stop()
	s.c.mu.Lock()
	if s.c.subs[s.key] == s {
		delete(s.c.subs, s.key)
	}
	s.c.mu.Unlock()
}

// subscribe sends one SubscribeCOV, which makes the subscription or renews
// it, and notes what came of it.
func (s *Subscription) subscribe(ctx context.Context) error {
	_, err := s.c.confirmed(ctx, s.to, services.ServiceConfirmedSubscribeCOV,
		services.COVObjects(uint(s.opts.ProcessID), s.object.Type, s.object.Instance, s.opts.Confirmed, s.seconds))
	s.mu.Lock()
	s.status.Err = err
	if err == nil {
		s.status.Renewed = time.Now()
		s.status.Expires = s.status.Renewed.Add(time.Duration(s.seconds) * time.Second)
	}
	s.mu.Unlock()
	return err
}

// renewLoop renews the subscription halfway through each lifetime, until it
// is cancelled or the client closes.
func (s *Subscription) renewLoop() {
	every := time.Duration(s.seconds) * time.Second / 2
	wait, backoff := every, covBackoffMin
	for {
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-s.c.closed:
			timer.Stop()
			return
		}

		s.sending.Lock()
		if s.ctx.Err() != nil {
			s.sending.Unlock()
			return
		}
		err := s.subscribe(s.ctx)
		s.sending.Unlock()
		switch {
		case err == nil:
			wait, backoff = every, covBackoffMin
		case s.ctx.Err() != nil || errors.Is(err, ErrClosed):
			return
		default:
			wait = min(backoff, every)
			backoff = min(2*backoff, covBackoffMax)
			s.c.log.Warn("COV subscription not renewed", "device", s.to, "type", s.object.Type,
				"instance", s.object.Instance, "error", err, "retry", wait)
		}
	}
}

// covNotification delivers a notification to the subscription it is for, and
// acknowledges it if the device asked. One that is for no subscription of
// the client's gets no answer.
func (c *Client) covNotification(src netip.AddrPort, npdu *plumbing.NPDU, a *plumbing.APDU) {
	c.noteSource(src, npdu)
	from := sourceOf(src, npdu)
	tags, err := toTags(a.Objects)
	if err != nil {
		c.log.Debug("dropped a COV notification that could not be decoded", "from", from, "error", err)
		return
	}
	n, err := decodeCOV(tags)
	if err != nil {
		c.log.Debug("dropped a COV notification that could not be decoded", "from", from, "error", err)
		return
	}
	if c.dropDenied(from, n.Device) {
		return
	}
	n.From = from
	n.Confirmed = a.Type == plumbing.ConfirmedReq

	c.mu.Lock()
	s := c.subs[covKey{peer: keyOf(from, 0), process: n.ProcessID, object: n.Object}]
	c.mu.Unlock()
	if s == nil {
		c.log.Debug("dropped a COV notification for no subscription", "from", from, "process", n.ProcessID,
			"type", n.Object.Type, "instance", n.Object.Instance)
		return
	}
	if n.Confirmed {
		// The acknowledgement goes back the way the notification came.
		c.sendAPDU(plumbing.BVLCFuncUnicast, src, npduTo(from, false), &plumbing.APDU{
			Type: plumbing.SimpleAck, InvokeID: a.InvokeID, Service: services.ServiceConfirmedCOVNotification,
		})
	}
	s.mu.Lock()
	s.status.Notifications++
	s.status.Last = time.Now()
	s.mu.Unlock()
	s.notify(n)
}

// decodeCOV decodes a COV notification: the process identifier [0], the
// device [1], the object [2], the time remaining [3] and, between tags 4, a
// run of property [0], perhaps array index [1], a value between tags 2 and
// perhaps a priority [3].
func decodeCOV(tags []*objects.Object) (COVNotification, error) {
	var n COVNotification
	if len(tags) < 5 || !isContext(tags[0], 0) || len(tags[0].Data) > 4 ||
		!isContext(tags[1], 1) || len(tags[1].Data) != 4 ||
		!isContext(tags[2], 2) || len(tags[2].Data) != 4 ||
		!isContext(tags[3], 3) || len(tags[3].Data) > 4 || !isOpening(tags[4], 4) {
		return n, errors.New("expected a process, a device, an object, a time remaining and values")
	}
	n.ProcessID = uint32(beUint(tags[0].Data))
	n.Device = objectIDOf(tags[1].Data).Instance
	n.Object = objectIDOf(tags[2].Data)
	n.Remaining = time.Duration(beUint(tags[3].Data)) * time.Second
	tags = tags[5:]

	for {
		if len(tags) == 0 {
			return n, errors.New("values are not closed")
		}
		if tags[0].Kind == objects.TagClosing && tags[0].TagNumber == 4 {
			return n, nil
		}
		if !isContext(tags[0], 0) || len(tags[0].Data) == 0 || len(tags[0].Data) > 4 {
			return n, errors.New("expected a property identifier")
		}
		prop := PropertyResult{ID: uint32(beUint(tags[0].Data))}
		tags = tags[1:]
		if len(tags) > 0 && isContext(tags[0], 1) && len(tags[0].Data) >= 1 && len(tags[0].Data) <= 4 {
			index := uint32(beUint(tags[0].Data))
			prop.Index = &index
			tags = tags[1:]
		}
		if len(tags) == 0 || !isOpening(tags[0], 2) {
			return n, errors.New("expected a property value")
		}
		values, rest, err := decodeUntilClose(tags[1:])
		if err != nil {
			return n, err
		}
		if len(rest) == 0 || rest[0].TagNumber != 2 {
			return n, errors.New("a property value is not closed")
		}
		prop.Values = values
		tags = rest[1:]
		if len(tags) > 0 && isContext(tags[0], 3) {
			tags = tags[1:]
		}
		n.Props = append(n.Props, prop)
	}
}
