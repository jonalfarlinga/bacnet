package bacnet

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// These tests replay frames exchanged with a Siemens field panel on the test
// bench. testdata/bench_siemens.hex holds the same frames.

// Every segment of a segmented ComplexACK repeats the service choice. The
// payloads, joined, must be the parameters of the unsegmented reply.
func TestBenchSegmentedObjectList(t *testing.T) {
	whole := mustHex(t, "810a008f 0100 30010c"+
		"0c02000065194c3ec400000000c400800001c400800002c400800003c400800004c400802710c400802711"+
		"c400802712c400802713c400802714c40080271bc40080271cc40080271dc400802725c400802726c40080272f"+
		"c400802730c400802731c400802739c40080273ac40080273bc40080274cc402000065c402800000c4050000003f")
	segments := []string{
		"810a0038 0104 3c020010 0c 0c02000065194c3ec400000000c400800001c400800002c400800003c400800004c400802710c400802711c400",
		"810a0038 0104 3c020110 0c 802712c400802713c400802714c40080271bc40080271cc40080271dc400802725c400802726c40080272fc400",
		"810a0037 0104 38020210 0c 802730c400802731c400802739c40080273ac40080273bc40080274cc402000065c402800000c4050000003f",
	}

	var params []byte
	for i, s := range segments {
		msg, err := Parse(mustHex(t, s))
		if err != nil {
			t.Fatalf("segment %d: %v", i, err)
		}
		cack, ok := msg.(*services.ComplexACK)
		if !ok {
			t.Fatalf("segment %d: got %T, want *services.ComplexACK", i, msg)
		}
		a := cack.APDU
		if !a.IsSegmented() || int(a.SequenceNumber) != i || a.ProposedWindowSize != 16 {
			t.Errorf("segment %d: segmented %t, sequence %d, window %d", i, a.IsSegmented(), a.SequenceNumber, a.ProposedWindowSize)
		}
		if a.Service != services.ServiceConfirmedReadProperty {
			t.Errorf("segment %d: got service %d, want %d", i, a.Service, services.ServiceConfirmedReadProperty)
		}
		if more := i < len(segments)-1; a.HasMoreSegments() != more {
			t.Errorf("segment %d: more-follows is %t, want %t", i, a.HasMoreSegments(), more)
		}
		params = append(params, a.Payload...)
	}

	// BVLC, NPDU, then PDU type, invoke ID and service choice.
	if want := whole[4+2+3:]; !bytes.Equal(want, params) {
		t.Fatalf("joined segments differ from the unsegmented reply:\nwant %x\ngot  %x", want, params)
	}

	// Rebuilt as one unsegmented reply, the segments decode to the 25 objects.
	rebuilt := append(append([]byte(nil), whole[:4+2+3]...), params...)
	msg, err := Parse(rebuilt)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := msg.(*services.ComplexACK).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if len(dec.Tags) != 25 {
		t.Errorf("got %d objects, want 25", len(dec.Tags))
	}
}

// A ReadProperty request names the property in context tag 1; tag 2 is the
// optional array index.
func TestBenchReadPropertyRequest(t *testing.T) {
	t.Run("from the panel", func(t *testing.T) {
		msg, err := Parse(mustHex(t, "810a0011 0104 0205a00c 0c02000141 1961"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := msg.(*services.ConfirmedReadProperty).Decode()
		if err != nil {
			t.Fatal(err)
		}
		want := services.ConfirmedReadPropertyDec{
			ObjectType:  objects.ObjectTypeDevice,
			InstanceNum: 321,
			PropertyId:  objects.PropertyIdProtocolServicesSupported,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("differs: (-want +got)\n%s", diff)
		}
	})

	t.Run("with an array index", func(t *testing.T) {
		b, err := NewReadPropertyIndexed(objects.ObjectTypeDevice, 101, objects.PropertyIdObjectList, 3)
		if err != nil {
			t.Fatal(err)
		}
		msg, err := Parse(b)
		if err != nil {
			t.Fatal(err)
		}
		got, err := msg.(*services.ConfirmedReadProperty).Decode()
		if err != nil {
			t.Fatal(err)
		}
		index := uint32(3)
		want := services.ConfirmedReadPropertyDec{
			ObjectType:  objects.ObjectTypeDevice,
			InstanceNum: 101,
			PropertyId:  objects.PropertyIdObjectList,
			ArrayIndex:  &index,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("differs: (-want +got)\n%s", diff)
		}
	})
}

// Active_COV_Subscriptions is a list of constructed values whose context tags
// Decode does not know. It must skip them, not fail.
func TestBenchActiveCOVSubscriptions(t *testing.T) {
	msg, err := Parse(mustHex(t, "810a0052 0100 308d0c 0c0200006519983e"+
		"0e0e1e21006506c0a80165bac01f0f1a1bbe0f1e0c0000000019551f2901393c"+
		"0e0e1e21006506c0a80165bac01f0f1a1bbd0f1e0c0000000019551f2900393c 3f"))
	if err != nil {
		t.Fatal(err)
	}
	dec, err := msg.(*services.ComplexACK).Decode()
	if err != nil {
		t.Fatal(err)
	}
	if dec.ObjectType != objects.ObjectTypeDevice || dec.InstanceId != 101 || dec.PropertyId != 152 {
		t.Errorf("got %d,%d property %d, want device,101 property 152", dec.ObjectType, dec.InstanceId, dec.PropertyId)
	}
	if len(dec.Skipped) == 0 {
		t.Error("no tags skipped: the subscription fields were decoded as something else")
	}
}

// A confirmed COV notification carries the invoke ID that its SimpleACK must
// echo.
func TestBenchConfirmedCOVNotification(t *testing.T) {
	msg, err := Parse(mustHex(t, "810a002c 0104 0205f801 0a1bc3 1c02000065 2c00800003 3a012c"+
		"4e09552e44000000002f096f2e8204002f4f"))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := msg.(*services.COVNotification)
	if !ok {
		t.Fatalf("got %T, want *services.COVNotification", msg)
	}
	if n.APDU.Type != plumbing.ConfirmedReq || n.APDU.InvokeID != 248 {
		t.Errorf("got PDU type %d invoke ID %d, want %d and 248", n.APDU.Type, n.APDU.InvokeID, plumbing.ConfirmedReq)
	}
	dec, err := n.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if dec.ProcessId != 7107 || dec.DevInstanceNum != 101 ||
		dec.ObjectType != objects.ObjectTypeAnalogValue || dec.ObjInstanceNum != 3 || dec.Lifetime != 300 {
		t.Errorf("got %+v", dec)
	}
}

// A reply has to carry the invoke ID of the request it answers.
func TestReplyBuildersEchoInvokeID(t *testing.T) {
	const invokeID = 248

	sack, err := NewSACKReply(invokeID, services.ServiceConfirmedCOVNotification)
	if err != nil {
		t.Fatal(err)
	}
	// The SimpleACK the panel accepted for its confirmed COV notification.
	if want := mustHex(t, "810a0009 0100 20f801"); !bytes.Equal(want, sack) {
		t.Errorf("SimpleACK differs: want %x, got %x", want, sack)
	}

	errReply, err := NewErrorReply(invokeID, services.ServiceConfirmedReadProperty, objects.ErrorClassProperty, 32)
	if err != nil {
		t.Fatal(err)
	}
	cack, err := NewCACKReply(invokeID, services.ServiceConfirmedReadProperty,
		objects.ObjectTypeAnalogOutput, 1, objects.PropertyIdPresentValue, 1.1)
	if err != nil {
		t.Fatal(err)
	}
	for name, reply := range map[string][]byte{"SimpleACK": sack, "Error": errReply, "ComplexACK": cack} {
		msg, err := Parse(reply)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var apdu *plumbing.APDU
		switch m := msg.(type) {
		case *services.SimpleACK:
			apdu = m.APDU
		case *services.Error:
			apdu = m.APDU
		case *services.ComplexACK:
			apdu = m.APDU
		default:
			t.Fatalf("%s: parsed as %T", name, msg)
		}
		if apdu.InvokeID != invokeID {
			t.Errorf("%s: got invoke ID %d, want %d", name, apdu.InvokeID, invokeID)
		}
	}

	// The builders without an invoke ID keep writing 1.
	old, err := NewSACK(services.ServiceConfirmedWriteProperty)
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := NewSACKReply(1, services.ServiceConfirmedWriteProperty); !bytes.Equal(want, old) {
		t.Errorf("NewSACK differs from NewSACKReply(1): want %x, got %x", want, old)
	}
}
