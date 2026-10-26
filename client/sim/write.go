package sim

import (
	"net/netip"
	"sync"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Write is one WriteProperty a device took, and what it answered: Class
// and Code are 0 when it acknowledged the write.
type Write struct {
	From     netip.AddrPort
	Device   uint32
	Object   objects.ObjectIdentifier
	Property uint32
	Index    *uint32
	Value    []*objects.Object
	// Priority is what the request named, 0 for none.
	Priority    uint8
	Class, Code uint8
}

// Service is one confirmed request of a service the devices only record:
// the ones a program built on the client sends only when told to, such as
// a LifeSafetyOperation. The device acknowledges it with a SimpleACK and
// keeps the tags, so that a test can see it arrived.
type Service struct {
	From    netip.AddrPort
	Device  uint32
	Service uint8
	Tags    []*objects.Object
}

// recordedServices is the confirmed services the devices record and
// acknowledge without acting on them.
var recordedServices = map[uint8]bool{
	services.ServiceConfirmedAcknowledgeAlarm:           true,
	services.ServiceConfirmedGetAlarmSummary:            true,
	services.ServiceConfirmedGetEnrollmentSummary:       true,
	services.ServiceConfirmedAtomicReadFile:             true,
	services.ServiceConfirmedAtomicWriteFile:            true,
	services.ServiceConfirmedAddListElement:             true,
	services.ServiceConfirmedRemoveListElement:          true,
	services.ServiceConfirmedCreateObject:               true,
	services.ServiceConfirmedDeleteObject:               true,
	services.ServiceConfirmedWritePropMultiple:          true,
	services.ServiceConfirmedDeviceCommunicationControl: true,
	services.ServiceConfirmedTextMessage:                true,
	services.ServiceConfirmedReinitializeDevice:         true,
	27: true, // LifeSafetyOperation
	29: true, // GetEventInformation
}

// Writes lists every WriteProperty the devices took, station by station in
// the order taken, the refused ones included.
func (n *Network) Writes() []Write {
	var list []Write
	for _, st := range n.stations {
		st.mu.Lock()
		list = append(list, st.writes...)
		st.mu.Unlock()
	}
	return list
}

// Services lists every request of a recorded service the devices took.
func (n *Network) Services() []Service {
	var list []Service
	for _, st := range n.stations {
		st.mu.Lock()
		list = append(list, st.recorded...)
		st.mu.Unlock()
	}
	return list
}

// tagsOf gives a request's payload as tags.
func tagsOf(a *plumbing.APDU) []*objects.Object {
	tags := make([]*objects.Object, 0, len(a.Objects))
	for _, p := range a.Objects {
		if t, ok := p.(*objects.Object); ok {
			tags = append(tags, t)
		}
	}
	return tags
}

// write answers a WriteProperty: object [0], property [1], perhaps an array
// index [2], the value between tags 3, and perhaps a priority [4]. A
// well-formed write is recorded whatever the answer.
func (st *station) write(to netip.AddrPort, dev *Device, req *plumbing.APDU) {
	tags := tagsOf(req)
	reject := func(reason uint8) {
		st.send(to, dev, &plumbing.APDU{Type: plumbing.Reject, InvokeID: req.InvokeID, Reason: reason})
	}
	if len(tags) < 2 || !context(tags[0], 0) || len(tags[0].Data) != 4 || !context(tags[1], 1) {
		reject(objects.RejectReasonMissingRequiredParameter)
		return
	}
	id, _ := objects.DecObjectIdentifier(tags[0])
	w := Write{From: to, Device: dev.Instance, Object: id, Property: uintOf(tags[1])}
	rest := tags[2:]
	if len(rest) > 0 && context(rest[0], 2) {
		i := uintOf(rest[0])
		w.Index = &i
		rest = rest[1:]
	}
	if len(rest) == 0 || rest[0].Kind != objects.TagOpening || rest[0].TagNumber != 3 {
		reject(objects.RejectReasonMissingRequiredParameter)
		return
	}
	// The value runs to the closing tag 3 at its own depth.
	end, depth := 0, 0
scan:
	for i := 1; i < len(rest); i++ {
		switch rest[i].Kind {
		case objects.TagOpening:
			depth++
		case objects.TagClosing:
			if depth > 0 {
				depth--
				continue
			}
			end = i
			break scan
		}
	}
	if end == 0 || rest[end].TagNumber != 3 {
		reject(objects.RejectReasonInvalidTag)
		return
	}
	w.Value = rest[1:end]
	rest = rest[end+1:]
	if len(rest) > 0 && context(rest[0], 4) {
		p := uintOf(rest[0])
		if p == 0 || p > 16 {
			reject(objects.RejectReasonParameterOutOfRange)
			return
		}
		w.Priority = uint8(p)
	}

	w.Class, w.Code = dev.write(w)
	st.mu.Lock()
	st.writes = append(st.writes, w)
	st.mu.Unlock()
	if w.Class == 0 && w.Code == 0 {
		st.send(to, dev, &plumbing.APDU{Type: plumbing.SimpleAck, InvokeID: req.InvokeID, Service: req.Service})
		return
	}
	e := plumbing.NewAPDU(plumbing.Error, req.Service, services.ErrorObjects(w.Class, w.Code))
	e.InvokeID = req.InvokeID
	st.send(to, dev, e)
}

// write applies a write to one of the device's objects and says what to
// answer: 0 and 0 to acknowledge it.
func (d *Device) write(w Write) (class, code uint8) {
	o := d.find(w.Object)
	if o == nil {
		return objects.ErrorClassObject, objects.ErrorCodeUnknownObject
	}
	var p *property
	if w.Property <= 0xFFFF {
		p = o.find(uint16(w.Property))
	}
	if p == nil {
		return objects.ErrorClassProperty, 32 // unknown property
	}
	if p.set == nil || w.Index != nil {
		return objects.ErrorClassProperty, 40 // write access denied
	}
	return p.set(w.Value, w.Priority)
}

// unhandled looks at a frame the parser has no type for: a confirmed
// request of a recorded service, addressed to one of the station's devices,
// is recorded and acknowledged. Anything else is dropped.
func (st *station) unhandled(src netip.AddrPort, raw []byte) {
	var bvlc plumbing.BVLC
	if bvlc.UnmarshalBinary(raw) != nil {
		return
	}
	offset := bvlc.MarshalLen()
	var npdu plumbing.NPDU
	if npdu.UnmarshalBinary(raw[offset:]) != nil || npdu.Control&plumbing.NetworkLayerMessage != 0 {
		return
	}
	offset += npdu.MarshalLen()
	var apdu plumbing.APDU
	if apdu.UnmarshalBinary(raw[offset:]) != nil || apdu.Type != plumbing.ConfirmedReq || !recordedServices[apdu.Service] {
		return
	}
	dev := st.target(&npdu)
	if dev == nil {
		return
	}
	st.mu.Lock()
	st.recorded = append(st.recorded, Service{From: src, Device: dev.Instance, Service: apdu.Service, Tags: tagsOf(&apdu)})
	st.mu.Unlock()
	st.send(src, dev, &plumbing.APDU{Type: plumbing.SimpleAck, InvokeID: apdu.InvokeID, Service: apdu.Service})
}

// ---- writable properties ---------------------------------------------------

// commandable is the priority array of an object whose present value can be
// written: sixteen slots, the highest of which that is not Null is the
// present value, and below them the relinquish default.
type commandable struct {
	mu    sync.Mutex
	slots [16]*objects.Object
	// kind is the application tag a value must have.
	kind uint8
	// relinquish gives the relinquish default.
	relinquish func() []*objects.Object
}

// present is the present value: the highest slot set, or the relinquish
// default.
func (c *commandable) present() []*objects.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.slots {
		if s != nil {
			return []*objects.Object{s}
		}
	}
	return c.relinquish()
}

// array is the priority array, a Null in each slot not set.
func (c *commandable) array() []*objects.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*objects.Object, len(c.slots))
	for i, s := range c.slots {
		if s == nil {
			out[i] = objects.EncNull()
		} else {
			out[i] = s
		}
	}
	return out
}

// set writes a value into the slot of the priority, 16 when none was
// named, or clears the slot for a Null.
func (c *commandable) set(value []*objects.Object, priority uint8) (class, code uint8) {
	if priority == 0 {
		priority = 16
	}
	if len(value) != 1 || value[0].TagClass || value[0].Kind != objects.TagPrimitive {
		return objects.ErrorClassProperty, 9 // invalid data type
	}
	v := value[0]
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case isNull(v):
		c.slots[priority-1] = nil
	case v.TagNumber == c.kind:
		c.slots[priority-1] = v
	default:
		return objects.ErrorClassProperty, 9
	}
	return 0, 0
}

func isNull(t *objects.Object) bool {
	return !t.TagClass && t.Kind == objects.TagPrimitive && t.TagNumber == objects.TagNull && len(t.Data) == 0
}

// cell is a writable scalar property of one application tag.
type cell struct {
	mu   sync.Mutex
	v    *objects.Object
	kind uint8
}

func (c *cell) get() []*objects.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	return []*objects.Object{c.v}
}

func (c *cell) set(value []*objects.Object, _ uint8) (class, code uint8) {
	if len(value) != 1 || value[0].TagClass || value[0].Kind != objects.TagPrimitive || value[0].TagNumber != c.kind {
		return objects.ErrorClassProperty, 9 // invalid data type
	}
	c.mu.Lock()
	c.v = value[0]
	c.mu.Unlock()
	return 0, 0
}

// writable is a property that takes a write of one application tag.
func writable(id uint16, v *objects.Object, kind uint8) property {
	c := &cell{v: v, kind: kind}
	return property{id: id, get: c.get, set: c.set}
}
