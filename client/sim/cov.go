package sim

import (
	"net/netip"
	"time"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// COVSubscription is one subscription a simulated device holds.
type COVSubscription struct {
	Subscriber netip.AddrPort
	ProcessID  uint32
	Object     objects.ObjectIdentifier
	Confirmed  bool
	Lifetime   time.Duration
	// Requests counts the SubscribeCOV requests taken for it: the one that
	// made it and each renewal.
	Requests int
	// Sent counts the notifications sent, and Acked the confirmed ones the
	// subscriber acknowledged.
	Sent, Acked int
}

type covSub struct {
	COVSubscription
	dev     *Device
	expires time.Time
}

// COVSubscriptions lists the subscriptions the device with the instance
// holds that have not lapsed.
func (n *Network) COVSubscriptions(instance uint32) []COVSubscription {
	var list []COVSubscription
	for _, st := range n.stations {
		st.mu.Lock()
		for _, s := range st.liveCOV() {
			if s.dev.Instance == instance {
				list = append(list, s.COVSubscription)
			}
		}
		st.mu.Unlock()
	}
	return list
}

// NotifyCOV has the device with the instance tell the subscribers of one of
// its objects what the object's value is now, as it would after a change. It
// returns how many notifications it sent.
func (n *Network) NotifyCOV(instance uint32, object objects.ObjectIdentifier) int {
	sent := 0
	for _, st := range n.stations {
		st.mu.Lock()
		var subs []*covSub
		for _, s := range st.liveCOV() {
			if s.dev.Instance == instance && s.Object == object {
				subs = append(subs, s)
			}
		}
		st.mu.Unlock()
		for _, s := range subs {
			st.notify(s)
			sent++
		}
	}
	return sent
}

// liveCOV drops the subscriptions that have lapsed and returns the rest. The
// caller holds st.mu.
func (st *station) liveCOV() []*covSub {
	now := time.Now()
	live := st.cov[:0]
	for _, s := range st.cov {
		if s.expires.After(now) {
			live = append(live, s)
		}
	}
	st.cov = live
	return live
}

// subscribe answers a SubscribeCOV: the process identifier [0], the object
// [1], and to subscribe, whether notifications are confirmed [2] and the
// lifetime [3]. Without those two it is a cancellation.
func (st *station) subscribe(src netip.AddrPort, npdu *plumbing.NPDU, req *plumbing.APDU) {
	dev := st.target(npdu)
	if dev == nil {
		if npdu.Control&0x20 != 0 && len(st.nets()) > 0 {
			st.rejectNetwork(src, npdu.DNET)
		}
		return
	}
	var tags []*objects.Object
	for _, p := range req.Objects {
		if t, ok := p.(*objects.Object); ok {
			tags = append(tags, t)
		}
	}
	fault := func(class, code uint8) {
		e := plumbing.NewAPDU(plumbing.Error, req.Service, services.ErrorObjects(class, code))
		e.InvokeID = req.InvokeID
		st.send(src, dev, e)
	}
	if len(tags) < 2 || !context(tags[0], 0) || !context(tags[1], 1) || len(tags[1].Data) != 4 {
		fault(objects.ErrorClassServices, 9) // inconsistent parameters
		return
	}
	process := uintOf(tags[0])
	id, _ := objects.DecObjectIdentifier(tags[1])
	sub := &covSub{dev: dev, COVSubscription: COVSubscription{Subscriber: src, ProcessID: process, Object: id}}
	cancel := true
	for _, t := range tags[2:] {
		switch {
		case context(t, 2):
			sub.Confirmed = len(t.Data) == 1 && t.Data[0] != 0
			cancel = false
		case context(t, 3):
			sub.Lifetime = time.Duration(uintOf(t)) * time.Second
			cancel = false
		}
	}
	if !cancel {
		o := dev.find(id)
		if o == nil {
			fault(objects.ErrorClassObject, objects.ErrorCodeUnknownObject)
			return
		}
		if o.find(objects.PropertyIdPresentValue) == nil {
			fault(objects.ErrorClassServices, 45) // optional functionality not supported
			return
		}
	}

	// A subscription is known by its subscriber, process and object: a
	// second request for one renews it, or cancels it.
	st.mu.Lock()
	var held *covSub
	kept := st.liveCOV()[:0]
	for _, s := range st.cov {
		if s.dev == dev && s.Subscriber == src && s.ProcessID == process && s.Object == id {
			held = s
			if cancel {
				continue
			}
		}
		kept = append(kept, s)
	}
	if !cancel {
		if held == nil {
			held = sub
			kept = append(kept, held)
		}
		held.Confirmed, held.Lifetime = sub.Confirmed, sub.Lifetime
		held.expires = time.Now().Add(sub.Lifetime)
		held.Requests++
	}
	st.cov = kept
	st.mu.Unlock()

	ack := &plumbing.APDU{Type: plumbing.SimpleAck, InvokeID: req.InvokeID, Service: req.Service}
	st.send(src, dev, ack)
	if !cancel {
		// A subscriber is told the value as it is when it subscribes.
		st.notify(held)
	}
}

// notify sends one notification of a subscription: the object's present
// value and status flags.
func (st *station) notify(s *covSub) {
	o := s.dev.find(s.Object)
	if o == nil {
		return
	}
	st.mu.Lock()
	s.Sent++
	confirmed := s.Confirmed
	remaining := max(0, time.Until(s.expires)/time.Second)
	st.mu.Unlock()
	payload := []objects.APDUPayload{
		objects.ContextTag(0, objects.EncUnsignedInteger(uint(s.ProcessID))),
		objects.EncObjectIdentifier(true, 1, objects.ObjectTypeDevice, s.dev.Instance),
		objects.EncObjectIdentifier(true, 2, s.Object.ObjectType, s.Object.InstanceNumber),
		objects.ContextTag(3, objects.EncUnsignedInteger(uint(remaining))),
		objects.EncOpeningTag(4),
	}
	for _, id := range []uint16{objects.PropertyIdPresentValue, objects.PropertyIdStatusFlags} {
		p := o.find(id)
		if p == nil {
			continue
		}
		payload = append(payload, objects.ContextTag(0, objects.EncUnsignedInteger(uint(id))), objects.EncOpeningTag(2))
		for _, t := range p.get() {
			payload = append(payload, t)
		}
		payload = append(payload, objects.EncClosingTag(2))
	}
	payload = append(payload, objects.EncClosingTag(4))

	note := plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedCOVNotification, payload)
	if confirmed {
		note = plumbing.NewAPDU(plumbing.ConfirmedReq, services.ServiceConfirmedCOVNotification, payload)
		note.MaxSize = 5
		st.mu.Lock()
		st.covInvoke++
		note.InvokeID = st.covInvoke
		st.unacked[txKey{peer: s.Subscriber, invoke: note.InvokeID}] = s
		st.mu.Unlock()
	}
	st.send(s.Subscriber, s.dev, note)
}

// covAcked counts the SimpleACK of a confirmed notification.
func (st *station) covAcked(from netip.AddrPort, ack *plumbing.APDU) {
	if ack.Service != services.ServiceConfirmedCOVNotification {
		return
	}
	key := txKey{peer: from, invoke: ack.InvokeID}
	st.mu.Lock()
	if s := st.unacked[key]; s != nil {
		s.Acked++
		delete(st.unacked, key)
	}
	st.mu.Unlock()
}
