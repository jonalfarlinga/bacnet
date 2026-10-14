package client_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// The simulated devices take writes: the present value of a commandable
// object through its priority array, out-of-service and object-name; they
// refuse the rest; and they record what they took, as they record and
// acknowledge the services they only record.
func TestWritePropertyToSim(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer network.Close()
	c, heard := openOn(t, network, func(cfg *client.Config) {
		cfg.Writable = []client.SourceRule{{IP: netip.MustParsePrefix("127.0.0.1/32")}}
		cfg.MaxRequestsPerSec = -1
	})
	ahu := awaitDevices(t, c, heard, 1001)[1001]
	ctx := context.Background()

	setpoint := client.ObjectID{Type: objects.ObjectTypeAnalogValue, Instance: 0} // AHU1.SAT.SP, steady at 55
	fan := client.ObjectID{Type: objects.ObjectTypeBinaryInput, Instance: 0}
	start := client.ObjectID{Type: objects.ObjectTypeBinaryOutput, Instance: 0}
	mode := client.ObjectID{Type: objects.ObjectTypeMultiStateValue, Instance: 0}
	real := func(v float32) []*objects.Object { return []*objects.Object{objects.EncReal(v)} }
	null := []*objects.Object{objects.EncNull()}
	write := func(obj client.ObjectID, prop uint16, value []*objects.Object, priority uint8) error {
		return c.WriteProperty(ctx, ahu.Addr, obj, prop, value, priority)
	}
	read := func(obj client.ObjectID, prop uint16) client.Value {
		t.Helper()
		values, err := c.ReadProperty(ctx, ahu.Addr, obj, prop)
		if err != nil || len(values) != 1 {
			t.Fatalf("reading %v property %d: %+v, %v", obj, prop, values, err)
		}
		return values[0]
	}
	present := func(obj client.ObjectID) float64 {
		t.Helper()
		return read(obj, objects.PropertyIdPresentValue).Float
	}
	slot := func(i uint32) float64 {
		t.Helper()
		values, err := c.ReadPropertyIndex(ctx, ahu.Addr, setpoint, objects.PropertyIdPriorityArray, i)
		if err != nil || len(values) != 1 {
			t.Fatalf("reading slot %d: %+v, %v", i, values, err)
		}
		return values[0].Float
	}

	// A write at 13 shows in the present value and in slot 13; a lower
	// priority waits behind it; relinquishing uncovers the next slot, then
	// the relinquish default. No priority is 16.
	if err := write(setpoint, objects.PropertyIdPresentValue, real(58), 13); err != nil {
		t.Fatalf("a write at 13: %v", err)
	}
	if got := present(setpoint); got != 58 {
		t.Errorf("after a write at 13 the present value is %v, want 58", got)
	}
	if got := slot(13); got != 58 {
		t.Errorf("slot 13 is %v, want 58", got)
	}
	if err := write(setpoint, objects.PropertyIdPresentValue, real(60), 16); err != nil {
		t.Fatalf("a write at 16: %v", err)
	}
	if got := present(setpoint); got != 58 {
		t.Errorf("a write at 16 showed over the one at 13: %v", got)
	}
	if err := write(setpoint, objects.PropertyIdPresentValue, null, 13); err != nil {
		t.Fatalf("a relinquish at 13: %v", err)
	}
	if got := present(setpoint); got != 60 {
		t.Errorf("after relinquishing 13 the present value is %v, want the 60 at 16", got)
	}
	if err := write(setpoint, objects.PropertyIdPresentValue, null, 16); err != nil {
		t.Fatalf("a relinquish at 16: %v", err)
	}
	if got := present(setpoint); got != 55 {
		t.Errorf("after relinquishing every slot the present value is %v, want the relinquish default 55", got)
	}
	if err := write(setpoint, objects.PropertyIdPresentValue, real(61), 0); err != nil {
		t.Fatalf("a write with no priority: %v", err)
	}
	if got := slot(16); got != 61 {
		t.Errorf("a write with no priority went to slot 16 as %v, want 61", got)
	}
	if err := write(setpoint, objects.PropertyIdPresentValue, null, 0); err != nil {
		t.Fatalf("a relinquish with no priority: %v", err)
	}
	if got := present(setpoint); got != 55 {
		t.Errorf("after a relinquish with no priority the present value is %v, want 55", got)
	}

	// The wrong kind, an array, an input, an unknown object and an unknown
	// property are refused as a device would.
	for _, tc := range []struct {
		name        string
		obj         client.ObjectID
		prop        uint16
		value       []*objects.Object
		class, code uint32
	}{
		{"the wrong kind", setpoint, objects.PropertyIdPresentValue, []*objects.Object{objects.EncEnumerated(1)}, 2, 9},
		{"the priority array", setpoint, objects.PropertyIdPriorityArray, real(1), 2, 40},
		{"an input", client.ObjectID{Type: objects.ObjectTypeAnalogInput}, objects.PropertyIdPresentValue, real(1), 2, 40},
		{"an unknown object", client.ObjectID{Type: objects.ObjectTypeAnalogValue, Instance: 99}, objects.PropertyIdPresentValue, real(1), 1, 31},
		{"an unknown property", setpoint, 9999, real(1), 2, 32},
	} {
		err := write(tc.obj, tc.prop, tc.value, 16)
		var be *client.BACnetError
		if !errors.As(err, &be) || be.Class != tc.class || be.Code != tc.code {
			t.Errorf("writing %s: %v, want error class %d code %d", tc.name, err, tc.class, tc.code)
		}
	}

	// out-of-service and object-name take a write and read back; a binary
	// and a multi-state take their own kinds.
	if err := write(fan, objects.PropertyIdOutOfService, []*objects.Object{objects.EncBoolean(true)}, 0); err != nil {
		t.Fatalf("out-of-service: %v", err)
	}
	if v := read(fan, objects.PropertyIdOutOfService); v.Kind != client.KindBoolean || !v.Bool {
		t.Errorf("out-of-service reads back as %+v, want true", v)
	}
	if err := write(setpoint, objects.PropertyIdObjectName, []*objects.Object{objects.EncString("AHU1.SAT.SP.2")}, 0); err != nil {
		t.Fatalf("object-name: %v", err)
	}
	if v := read(setpoint, objects.PropertyIdObjectName); v.Str != "AHU1.SAT.SP.2" {
		t.Errorf("object-name reads back as %+v", v)
	}
	if err := write(start, objects.PropertyIdPresentValue, []*objects.Object{objects.EncEnumerated(0)}, 8); err != nil {
		t.Fatalf("a binary output: %v", err)
	}
	if v := read(start, objects.PropertyIdPresentValue); v.Kind != client.KindEnumerated || v.Uint != 0 {
		t.Errorf("the binary output reads back as %+v, want inactive", v)
	}
	if err := write(mode, objects.PropertyIdPresentValue, []*objects.Object{objects.EncUnsignedInteger(4)}, 16); err != nil {
		t.Fatalf("a multi-state: %v", err)
	}
	if v := read(mode, objects.PropertyIdPresentValue); v.Kind != client.KindUnsigned || v.Uint != 4 {
		t.Errorf("the multi-state reads back as %+v, want 4", v)
	}

	// Everything the device took is listed, with the priority named and
	// the answer given: the 6 writes to the setpoint, the 5 refused, and
	// the 4 to the other properties and objects.
	writes := network.Writes()
	if len(writes) != 15 {
		t.Errorf("%d writes recorded, want 15", len(writes))
	}
	at13, refused, at8 := 0, 0, 0
	for _, w := range writes {
		if w.Device != 1001 || w.From != c.Local() {
			t.Errorf("a write recorded for device %d from %v", w.Device, w.From)
		}
		switch {
		case w.Object.ObjectType == setpoint.Type && w.Object.InstanceNumber == setpoint.Instance && w.Property == uint32(objects.PropertyIdPresentValue) && w.Priority == 13 && w.Class == 0 && w.Code == 0:
			at13++
		case w.Class != 0 || w.Code != 0:
			refused++
		case w.Object.ObjectType == start.Type && w.Priority == 8:
			at8++
		}
	}
	if at13 != 2 || refused != 5 || at8 != 1 {
		t.Errorf("recorded %d writes at 13, %d refused, %d at 8; want 2, 5, 1", at13, refused, at8)
	}

	// A LifeSafetyOperation, which the client never sends, is acknowledged
	// by the device and listed.
	p := newPeer(t)
	lso := plumbing.NewAPDU(plumbing.ConfirmedReq, 27, []objects.APDUPayload{
		objects.ContextTag(0, objects.EncUnsignedInteger(1)), objects.ContextTag(1, objects.EncString("operator")), objects.ContextTag(2, objects.EncEnumerated(0)),
	})
	lso.InvokeID, lso.MaxSize = 9, 5
	p.reply(network.Addr(), lso)
	buf := make([]byte, 2048)
	p.conn.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err := p.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		t.Fatal("no answer to the LifeSafetyOperation")
	}
	msg, err := bacnet.Parse(buf[:n])
	if ack, ok := msg.(*services.SimpleACK); err != nil || !ok || ack.APDU.InvokeID != 9 || ack.APDU.Service != 27 {
		t.Errorf("the LifeSafetyOperation was answered with %T %v, want a SimpleACK with invoke 9", msg, err)
	}
	if svcs := network.Services(); len(svcs) != 1 || svcs[0].Device != 1001 || svcs[0].Service != 27 || len(svcs[0].Tags) != 3 {
		t.Errorf("the recorded services are %+v, want one LifeSafetyOperation at device 1001 with 3 tags", svcs)
	}
}
