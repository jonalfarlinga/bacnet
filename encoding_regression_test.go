package bacnet_test

import (
	"testing"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func TestWritePropertyBuilderDecode(t *testing.T) {
	for _, priority := range []bool{false, true} {
		raw, err := bacnet.NewWriteProperty(objects.ObjectTypeAnalogOutput, 0, 85, float32(12.5))
		if err != nil {
			t.Fatal(err)
		}
		msg, err := bacnet.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		wp := msg.(*services.ConfirmedWriteProperty)
		if !priority {
			wp.APDU.Objects = wp.APDU.Objects[:5]
		}
		decoded, err := wp.Decode()
		if err != nil {
			t.Fatal(err)
		}
		wantPriority := uint8(0)
		if priority {
			wantPriority = 16
		}
		if decoded.Value != 12.5 || decoded.Priority != wantPriority || len(decoded.Tags) != 1 {
			t.Fatalf("decoded %+v", decoded)
		}
		if _, err := decoded.Tags[0].MarshalBinary(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWritePropertyConstructedValue(t *testing.T) {
	wp := services.NewConfirmedWriteProperty(plumbing.NewBVLC(10), plumbing.NewNPDU(false, false, false, true))
	wp.APDU.Objects = []objects.APDUPayload{
		objects.EncObjectIdentifier(true, 0, objects.ObjectTypeAnalogOutput, 1),
		objects.ContextTag(1, objects.EncUnsignedInteger(85)),
		objects.ContextTag(2, objects.EncUnsignedInteger(3)),
		objects.EncOpeningTag(3), objects.EncOpeningTag(0), objects.EncReal(2), objects.EncClosingTag(0), objects.EncClosingTag(3),
		objects.ContextTag(4, objects.EncUnsignedInteger(8)),
	}
	decoded, err := wp.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ArrayIndex == nil || *decoded.ArrayIndex != 3 || decoded.Priority != 8 || len(decoded.Tags) != 3 || decoded.Tags[1].Value != float32(2) {
		t.Fatalf("decoded %+v", decoded)
	}
	wp.APDU.Objects[6] = objects.EncClosingTag(1)
	if _, err := wp.Decode(); err == nil {
		t.Fatal("mismatched closing tag was accepted")
	}
	wp.APDU.Objects[6] = objects.EncClosingTag(0)
	wp.APDU.Objects[8] = objects.ContextTag(4, objects.EncUnsignedInteger(17))
	if _, err := wp.Decode(); err == nil {
		t.Fatal("priority 17 was accepted")
	}
}

func TestRPMConstructorService(t *testing.T) {
	msg := services.NewConfirmedReadPropertyMultiple(plumbing.NewBVLC(10), plumbing.NewNPDU(false, false, false, true))
	msg.APDU.Objects = services.ConfirmedReadPropertyMultipleObjects(8, 101, []uint16{77})
	msg.SetLength()
	raw, err := msg.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := bacnet.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.GetService() != services.ServiceConfirmedReadPropMultiple {
		t.Fatalf("service %d", parsed.GetService())
	}
}

func TestSingleByteAckPayload(t *testing.T) {
	for _, kind := range []uint8{plumbing.ComplexAck, plumbing.Error} {
		a := plumbing.NewAPDU(kind, 12, []objects.APDUPayload{objects.EncBoolean(true)})
		b := make([]byte, a.MarshalLen())
		if err := a.MarshalTo(b); err != nil {
			t.Fatal(err)
		}
		var decoded plumbing.APDU
		if err := decoded.UnmarshalBinary(b); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Objects) != 1 {
			t.Fatalf("payload missing in %x", b)
		}
		value, err := objects.DecBoolean(decoded.Objects[0])
		if err != nil || !value || decoded.Objects[0].(*objects.Object).TagNumber != objects.TagBoolean {
			t.Fatalf("boolean replaced in %x", b)
		}
	}
}
