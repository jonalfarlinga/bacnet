package bacnet

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/jonalfarlinga/bacnet/test_utils"
)

// Every frame in testdata must parse without error.
func TestFixtureFramesParse(t *testing.T) {
	for _, frame := range test_utils.LoadHexFrames(t, "testdata/*.hex") {
		t.Run(frame.Name, func(t *testing.T) {
			if _, err := Parse(frame.Data); err != nil {
				t.Errorf("parsing %x: %v", frame.Data, err)
			}
		})
	}
}

// The captured frames are plain, unsegmented messages, so marshalling the
// parsed message must give back the captured bytes.
func TestCaptureFramesRoundTrip(t *testing.T) {
	for _, frame := range test_utils.LoadHexFrames(t, "testdata/BACnet_*.hex") {
		t.Run(frame.Name, func(t *testing.T) {
			msg, err := Parse(frame.Data)
			if err != nil {
				t.Fatalf("parsing %x: %v", frame.Data, err)
			}
			got, err := msg.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(frame.Data, got) {
				t.Errorf("differs: want %x, got %x", frame.Data, got)
			}
		})
	}
}

// A truncated datagram must give an error or a shorter message, and never
// read past the end of what was received.
func TestFixtureFramesTruncated(t *testing.T) {
	for _, frame := range test_utils.LoadHexFrames(t, "testdata/*.hex") {
		t.Run(frame.Name, func(t *testing.T) {
			for n := 0; n < len(frame.Data); n++ {
				// The backing array still holds the rest of the frame, as a
				// reused receive buffer would.
				msg, err := ParseWithFallback(frame.Data[:n])
				if err != nil {
					continue
				}
				decodeAll(msg)
				if l := msg.MarshalLen(); l > n {
					t.Errorf("%d of %d bytes parsed into a %d-byte message", n, len(frame.Data), l)
				}
			}
		})
	}
}

func TestParseTruncatedTag(t *testing.T) {
	// ReadProperty ack whose object-identifier value is cut off after 2 of
	// its 4 bytes.
	b := []byte{
		0x81, 0x0a, 0x00, 0x14, 0x01, 0x00, 0x30, 0x01, 0x0c, 0x0c, 0x02,
		0x00, 0x00, 0x65, 0x19, 0x4b, 0x3e, 0xc4, 0x02, 0x00,
	}
	if _, err := Parse(b); !errors.Is(err, common.ErrTooShortToParse) {
		t.Errorf("got error %v, want %v", err, common.ErrTooShortToParse)
	}
}

// A value longer than 253 bytes needs the 2-byte extended length form.
func TestLongStringRoundTrip(t *testing.T) {
	want := strings.Repeat("BACnet ", 60)

	c := services.NewComplexACK(
		plumbing.NewBVLC(plumbing.BVLCFuncUnicast),
		plumbing.NewNPDU(false, false, false, false),
	)
	c.APDU.InvokeID = 1
	c.APDU.Objects = services.ComplexACKObjects(
		objects.ObjectTypeDevice, 101, objects.PropertyIdDescription, want,
	)
	c.SetLength()
	b, err := c.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	msg, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	cack, ok := msg.(*services.ComplexACK)
	if !ok {
		t.Fatalf("got %T, want *services.ComplexACK", msg)
	}
	dec, err := cack.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if len(dec.Tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(dec.Tags))
	}
	if got := dec.Tags[0].Value; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// decodeAll runs every decoder a parsed message offers. Decoders may reject
// the message; the tests calling this only care that they do not panic.
func decodeAll(msg plumbing.BACnet) {
	switch m := msg.(type) {
	case *services.UnconfirmedWhoIs:
		m.Decode()
	case *services.UnconfirmedIAm:
		m.Decode()
	case *services.COVNotification:
		m.Decode()
	case *services.ConfirmedReadProperty:
		m.Decode()
		m.DecodeRPM()
	case *services.ConfirmedReadRange:
		m.Decode()
	case *services.ConfirmedCOV:
		m.Decode()
	case *services.ConfirmedWriteProperty:
		m.Decode()
	case *services.ComplexACK:
		m.Decode()
		m.DecodeRPM()
		m.DecodeRR()
	case *services.Error:
		m.Decode()
	}
}
