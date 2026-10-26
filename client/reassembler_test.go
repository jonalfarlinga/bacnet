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

// A first segment that is not the one already taken, sent again, starts a
// new reassembly: one that never completed is not continued with the
// segments of the message that reuses its invoke ID.
func TestReassemblerNewMessageReplacesStale(t *testing.T) {
	r := NewReassembler(time.Minute)
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	enumerated := func(v byte) []byte { return []byte{0x91, v} } // one Enumerated tag
	values := func(t *testing.T, raw []byte) []byte {
		t.Helper()
		msg, err := bacnet.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		var got []byte
		for _, o := range msg.(*services.ComplexACK).APDU.Objects {
			got = append(got, o.(*objects.Object).Data...)
		}
		return got
	}

	// A reply that stops after three segments, then a different reply on
	// the same invoke ID.
	for seq := range uint8(3) {
		if res := r.Feed("dev", bvlc, npdu, segment(seq, true, enumerated(seq))); !res.Accepted || res.Complete {
			t.Fatalf("segment %d of the stale reply gave %+v", seq, res)
		}
	}
	if res := r.Feed("dev", bvlc, npdu, segment(0, true, enumerated(10))); !res.Accepted || res.Complete || res.Dropped != "" {
		t.Fatalf("the first segment of the new reply gave %+v, want it accepted and incomplete", res)
	}
	res := r.Feed("dev", bvlc, npdu, segment(1, false, enumerated(11)))
	if !res.Complete {
		t.Fatalf("the last segment of the new reply gave %+v, want the whole message", res)
	}
	if got := values(t, res.Raw); string(got) != string([]byte{10, 11}) {
		t.Errorf("the new reply reassembled as %v, want [10 11] without the stale segments", got)
	}

	// A stale reply of one segment is replaced as well when the first
	// segment differs, and kept when it is the same one sent again.
	r.Feed("dev", bvlc, npdu, segment(0, true, enumerated(20)))
	if res := r.Feed("dev", bvlc, npdu, segment(0, true, enumerated(30))); !res.Accepted || res.Complete {
		t.Fatalf("a different first segment gave %+v, want it accepted", res)
	}
	if res := r.Feed("dev", bvlc, npdu, segment(0, true, enumerated(30))); !res.Accepted || res.Complete {
		t.Fatalf("the first segment sent again gave %+v, want it accepted", res)
	}
	res = r.Feed("dev", bvlc, npdu, segment(1, false, enumerated(31)))
	if got := values(t, res.Raw); !res.Complete || string(got) != string([]byte{30, 31}) {
		t.Errorf("the reply reassembled as %v (complete %v), want [30 31]", got, res.Complete)
	}
}
