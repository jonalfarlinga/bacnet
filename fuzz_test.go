package bacnet

import (
	"bytes"
	"testing"

	"github.com/jonalfarlinga/bacnet/test_utils"
)

// FuzzParse feeds arbitrary datagrams through parsing, every decoder, and
// marshalling. None of them may panic or change the input.
func FuzzParse(f *testing.F) {
	for _, frame := range test_utils.LoadHexFrames(f, "testdata/*.hex") {
		f.Add(frame.Data)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		in := bytes.Clone(b)

		strictMsg, strictErr := Parse(b)
		msg, err := ParseWithFallback(b)
		if !bytes.Equal(in, b) {
			t.Fatalf("parsing changed its input: was %x, now %x", in, b)
		}
		if strictErr == nil {
			decodeAll(strictMsg)
		}
		if err != nil {
			if strictErr == nil {
				t.Fatalf("Parse accepted %x but ParseWithFallback failed: %v", b, err)
			}
			return
		}

		if npduMsg, err := ParseNPDU(b); err == nil {
			decodeAll(npduMsg)
		}
		if !bytes.Equal(in, b) {
			t.Fatalf("ParseNPDU changed its input: was %x, now %x", in, b)
		}

		decodeAll(msg)
		if _, err := msg.MarshalBinary(); err != nil {
			// A parsed message may be one this library cannot write back.
			t.Skip(err)
		}
	})
}
