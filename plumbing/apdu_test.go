package plumbing_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/internal/testutil"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// The tags of a ReadPropertyMultiple ack: object-identifier and object-name
// results for device,101.
var rpmAckTags = []byte{
	0x0c, 0x02, 0x00, 0x00, 0x65, 0x1e, 0x29, 0x4b, 0x4e, 0xc4, 0x02, 0x00,
	0x00, 0x65, 0x4f, 0x29, 0x4d, 0x4e, 0x75, 0x10, 0x00, 0x55, 0x4f, 0x2d,
	0x41, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x74, 0x2d, 0x54, 0x65, 0x73, 0x74,
	0x4f, 0x1f,
}

func TestParseObjectTags(t *testing.T) {
	want := []objects.APDUPayload{
		objects.EncObjectIdentifier(true, 0, objects.ObjectTypeDevice, 101),
		objects.EncOpeningTag(1),
		objects.ContextTag(2, objects.EncUnsignedInteger(uint(objects.PropertyIdObjectIdentifier))),
		objects.EncOpeningTag(4),
		objects.EncObjectIdentifier(false, objects.TagBACnetObjectIdentifier, objects.ObjectTypeDevice, 101),
		objects.EncClosingTag(4),
		objects.ContextTag(2, objects.EncUnsignedInteger(uint(objects.PropertyIdObjectName))),
		objects.EncOpeningTag(4),
		objects.EncString("UO-Analyst-Test"),
		objects.EncClosingTag(4),
		objects.EncClosingTag(1),
	}

	// Leading bytes stand in for the APDU header the tags follow.
	b := append([]byte{0x30, 0x01, 0x0e}, rpmAckTags...)
	got, err := plumbing.ParseObjectTags(b, 3)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

func TestParseObjectTagsNone(t *testing.T) {
	got, err := plumbing.ParseObjectTags([]byte{0x20, 0x01, 0x0f}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d tags, want none", len(got))
	}
}

// Cutting a tag run short must fail, or stop at a tag boundary, without
// reading the bytes beyond the cut.
func TestParseObjectTagsTruncated(t *testing.T) {
	boundaries := map[int]bool{}
	for offset := 0; offset < len(rpmAckTags); {
		_, n, err := objects.DecodeTag(rpmAckTags[offset:])
		if err != nil {
			t.Fatal(err)
		}
		offset += n
		boundaries[offset] = true
	}

	for n := 1; n < len(rpmAckTags); n++ {
		_, err := plumbing.ParseObjectTags(rpmAckTags[:n], 0)
		switch {
		case boundaries[n] && err != nil:
			t.Errorf("%d bytes end on a tag boundary: %v", n, err)
		case !boundaries[n] && !errors.Is(err, common.ErrTooShortToParse):
			t.Errorf("%d bytes end inside a tag: got error %v, want %v", n, err, common.ErrTooShortToParse)
		}
	}
}

func TestAPDUUnmarshalTruncated(t *testing.T) {
	testcases := []struct {
		description string
		serialized  []byte
		headerLen   int // bytes before the tags
	}{
		{"unconfirmed request", []byte{0x10, 0x08, 0x09, 0x7b, 0x19, 0x7b}, 2},
		{"confirmed request", []byte{0x02, 0x75, 0x01, 0x0c, 0x0c, 0x00, 0x40, 0x00, 0x01, 0x19, 0x55}, 4},
		{"segmented confirmed request", []byte{0x0c, 0x75, 0x01, 0x00, 0x04, 0x0f, 0x0c, 0x00}, 6},
		{"simple ack", []byte{0x20, 0x01, 0x0f}, 3},
		{"complex ack", []byte{0x30, 0x01, 0x0c, 0x0c, 0x00, 0x40, 0x00, 0x01, 0x19, 0x55}, 3},
		{"segmented complex ack", []byte{0x3c, 0x01, 0x00, 0x04, 0x0c, 0x0c, 0x02}, 5},
		{"segment ack", []byte{0x40, 0x01, 0x03, 0x10}, 4},
		{"error", []byte{0x50, 0x01, 0x0c, 0x91, 0x01, 0x91, 0x1f}, 3},
		{"reject", []byte{0x60, 0x01, 0x09}, 3},
		{"abort", []byte{0x71, 0x01, 0x04}, 3},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			if err := new(plumbing.APDU).UnmarshalBinary(c.serialized); err != nil {
				t.Fatalf("whole APDU: %v", err)
			}
			for n := 0; n < c.headerLen; n++ {
				err := new(plumbing.APDU).UnmarshalBinary(c.serialized[:n])
				if !errors.Is(err, common.ErrTooShortToParse) {
					t.Errorf("%d bytes: got error %v, want %v", n, err, common.ErrTooShortToParse)
				}
			}
		})
	}
}

func TestAPDUMarshalConfirmedRequest(t *testing.T) {
	a := plumbing.NewAPDU(plumbing.ConfirmedReq, 0x0c, []objects.APDUPayload{
		objects.EncObjectIdentifier(true, 0, objects.ObjectTypeAnalogOutput, 1),
		objects.ContextTag(1, objects.EncUnsignedInteger(uint(objects.PropertyIdPresentValue))),
	})
	a.SetAPDUFlags(true, false, false)
	a.MaxSeg = 7
	a.MaxSize = 5
	a.InvokeID = 1

	want := []byte{0x02, 0x75, 0x01, 0x0c, 0x0c, 0x00, 0x40, 0x00, 0x01, 0x19, 0x55}
	// A dirty buffer shows that every header byte is written, not merged.
	got := bytes.Repeat([]byte{0xff}, a.MarshalLen())
	if err := a.MarshalTo(got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("differs: want %x, got %x", want, got)
	}

	parsed := new(plumbing.APDU)
	if err := parsed.UnmarshalBinary(got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(a, parsed); diff != "" {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

// apduTags returns the tag run of a fixture frame, or nil if the frame has
// none (BVLC-only messages, segments, and PDUs without tags).
func apduTags(frame []byte) []byte {
	var bvlc plumbing.BVLC
	if bvlc.UnmarshalBinary(frame) != nil || bvlc.Function == plumbing.BVLCFuncForwardedNPDU {
		return nil
	}
	offset := bvlc.MarshalLen()

	var npdu plumbing.NPDU
	if npdu.UnmarshalBinary(frame[offset:]) != nil {
		return nil
	}
	offset += npdu.MarshalLen()
	if len(frame) <= offset {
		return nil
	}

	var apdu plumbing.APDU
	if apdu.UnmarshalBinary(frame[offset:]) != nil || apdu.IsSegmented() {
		return nil
	}
	switch apdu.Type {
	case plumbing.UnConfirmedReq:
		offset += 2
	case plumbing.ConfirmedReq:
		offset += 4
	case plumbing.SimpleAck, plumbing.ComplexAck, plumbing.Error:
		offset += 3
	default:
		return nil
	}
	if len(frame) <= offset {
		return nil
	}
	return frame[offset:]
}

// FuzzObjectTags checks that the tag parser never panics, that it leaves its
// input alone, and that whatever it accepts survives a marshal/parse round
// trip unchanged.
func FuzzObjectTags(f *testing.F) {
	f.Add(rpmAckTags)
	for _, frame := range testutil.LoadHexFrames(f, "../testdata/*.hex") {
		if tags := apduTags(frame.Data); tags != nil {
			f.Add(tags)
		}
	}
	// One seed per length form and for tag-number extension.
	for _, obj := range []*objects.Object{
		objects.EncNull(),
		objects.EncBoolean(true),
		objects.EncReal(1.1),
		objects.EncString("extended length"),
		objects.EncOctetString(make([]byte, 254)),
		objects.EncOctetString(make([]byte, 65536)),
		objects.ContextTag(254, objects.EncUnsignedInteger(1)),
		objects.EncOpeningTag(100),
		objects.EncClosingTag(100),
	} {
		b, err := obj.MarshalBinary()
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		in := bytes.Clone(b)
		objs, err := plumbing.ParseObjectTags(b, 0)
		if !bytes.Equal(in, b) {
			t.Fatalf("parsing changed its input: was %x, now %x", in, b)
		}
		if err != nil {
			if !errors.Is(err, common.ErrTooShortToParse) && !errors.Is(err, common.ErrInvalidData) {
				t.Fatalf("error wraps neither ErrTooShortToParse nor ErrInvalidData: %v", err)
			}
			return
		}

		var out []byte
		for i, obj := range objs {
			ob, err := obj.MarshalBinary()
			if err != nil {
				t.Fatalf("marshalling tag %d %+v: %v", i, obj, err)
			}
			if len(ob) != obj.MarshalLen() {
				t.Fatalf("tag %d %+v: MarshalLen %d but marshalled %d bytes", i, obj, obj.MarshalLen(), len(ob))
			}
			out = append(out, ob...)
		}
		// The input may use non-minimal length forms, so the bytes can
		// differ; the tags they decode to cannot.
		if len(out) > len(in) {
			t.Fatalf("marshalled %d bytes from %d", len(out), len(in))
		}
		again, err := plumbing.ParseObjectTags(out, 0)
		if err != nil {
			t.Fatalf("parsing marshalled tags %x: %v", out, err)
		}
		if diff := cmp.Diff(objs, again); diff != "" {
			t.Fatalf("round trip differs: (-first +second)\n%s", diff)
		}
	})
}

// A segment is written back as it was read: sequence number, window size,
// service choice and the raw payload.
func TestAPDUSegmentRoundTrip(t *testing.T) {
	testcases := []struct {
		description string
		serialized  []byte
		service     uint8
		sequence    uint8
		payload     []byte
	}{
		{
			"confirmed request, first segment",
			[]byte{0x0c, 0x75, 0x01, 0x00, 0x04, 0x0f, 0x0c, 0x00},
			0x0f, 0, []byte{0x0c, 0x00},
		},
		{
			"complex ack, first segment",
			[]byte{0x3c, 0x01, 0x00, 0x04, 0x0c, 0x0c, 0x02},
			0x0c, 0, []byte{0x0c, 0x02},
		},
		{
			"complex ack, last segment",
			[]byte{0x38, 0x01, 0x02, 0x04, 0x0c, 0x3f},
			0x0c, 2, []byte{0x3f},
		},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			a := new(plumbing.APDU)
			if err := a.UnmarshalBinary(c.serialized); err != nil {
				t.Fatal(err)
			}
			if a.Service != c.service || a.SequenceNumber != c.sequence || a.ProposedWindowSize != 4 {
				t.Errorf("got service %#x sequence %d window %d, want %#x, %d and 4",
					a.Service, a.SequenceNumber, a.ProposedWindowSize, c.service, c.sequence)
			}
			if !bytes.Equal(c.payload, a.Payload) {
				t.Errorf("payload differs: want %x, got %x", c.payload, a.Payload)
			}

			if l := a.MarshalLen(); l != len(c.serialized) {
				t.Fatalf("got marshal length %d, want %d", l, len(c.serialized))
			}
			got := bytes.Repeat([]byte{0xff}, a.MarshalLen())
			if err := a.MarshalTo(got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c.serialized, got) {
				t.Errorf("differs: want %x, got %x", c.serialized, got)
			}
		})
	}
}
