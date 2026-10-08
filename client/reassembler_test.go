package client

import (
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func segment(seq uint8, more bool, payload []byte) *plumbing.APDU {
	a := &plumbing.APDU{
		Type: plumbing.ComplexAck, Flags: 0x8, InvokeID: 7, SequenceNumber: seq, ProposedWindowSize: 4,
		Service: services.ServiceConfirmedReadProperty, Payload: payload,
	}
	if more {
		a.Flags |= 0x4
	}
	return a
}

func TestReassembler(t *testing.T) {
	// A ReadProperty ack split in the middle of its tags.
	whole, err := objects.EncString("a reply long enough to be sent in two segments").MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	head := []*objects.Object{
		objects.EncObjectIdentifier(true, 0, objects.ObjectTypeDevice, 9),
		objects.ContextTag(1, objects.EncUnsignedInteger(uint(objects.PropertyIdObjectName))),
		objects.EncOpeningTag(3),
	}
	var payload []byte
	for _, o := range head {
		b, _ := o.MarshalBinary()
		payload = append(payload, b...)
	}
	payload = append(payload, whole...)
	b, _ := objects.EncClosingTag(3).MarshalBinary()
	payload = append(payload, b...)
	cut := len(payload) / 2

	r := NewReassembler(time.Minute)
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)

	res := r.Feed("dev", bvlc, npdu, segment(0, true, payload[:cut]))
	if !res.Accepted || res.Complete || res.Dropped != "" {
		t.Fatalf("the first segment gave %+v, want it accepted and incomplete", res)
	}
	// A segment from the future is a gap: the reassembly is dropped.
	if res := r.Feed("dev", bvlc, npdu, segment(2, false, nil)); res.Accepted || res.Dropped == "" {
		t.Errorf("a segment ahead of the next gave %+v, want it dropped", res)
	}
	// Start again; a duplicate of the first is acknowledged but not added.
	r.Feed("dev", bvlc, npdu, segment(0, true, payload[:cut]))
	if res := r.Feed("dev", bvlc, npdu, segment(0, true, payload[:cut])); !res.Accepted || res.Complete {
		t.Errorf("a duplicate segment gave %+v, want it accepted only", res)
	}
	res = r.Feed("dev", bvlc, npdu, segment(1, false, payload[cut:]))
	if !res.Complete || !res.Accepted {
		t.Fatalf("the last segment gave %+v, want the whole message", res)
	}

	msg, err := bacnet.Parse(res.Raw)
	if err != nil {
		t.Fatal(err)
	}
	ack, ok := msg.(*services.ComplexACK)
	if !ok || ack.APDU.IsSegmented() || ack.APDU.InvokeID != 7 {
		t.Fatalf("the whole message parsed as %T %+v, want an unsegmented ComplexACK with invoke ID 7", msg, msg)
	}
	tags, err := toTags(ack.APDU.Objects)
	if err != nil {
		t.Fatal(err)
	}
	values, rest, err := decodeUntilClose(tags[3:])
	if err != nil || len(values) != 1 || values[0].Str != "a reply long enough to be sent in two segments" || len(rest) != 1 {
		t.Errorf("the reassembled value is %+v (rest %d, %v)", values, len(rest), err)
	}
	// A segment of a message nobody started is dropped, not acknowledged.
	if res := r.Feed("other", bvlc, npdu, segment(1, false, nil)); res.Accepted || res.Dropped == "" {
		t.Errorf("a continuation without a start gave %+v, want it dropped", res)
	}
}
