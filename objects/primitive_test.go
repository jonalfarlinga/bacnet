package objects_test

import (
	"bytes"
	"testing"

	"github.com/jonalfarlinga/bacnet/objects"
)

func TestUnsignedInteger(t *testing.T) {
	testcases := []struct {
		description string
		data        []byte
		value       uint32
	}{
		{"1 byte", []byte{0x48}, 72},
		{"2 bytes", []byte{0x05, 0xc4}, 1476},
		{"3 bytes", []byte{0x3f, 0xff, 0xff}, 4194303},
		{"4 bytes", []byte{0xff, 0xff, 0xff, 0xff}, 4294967295},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			obj := objects.NewObject(objects.TagUnsignedInteger, false, c.data)
			got, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.value {
				t.Errorf("got %d, want %d", got, c.value)
			}
		})
	}
}

func TestSignedInteger(t *testing.T) {
	testcases := []struct {
		description string
		data        []byte
		value       int
	}{
		{"1 byte positive", []byte{0x48}, 72},
		{"1 byte negative", []byte{0xff}, -1},
		{"2 bytes positive", []byte{0x7f, 0xff}, 32767},
		{"2 bytes negative", []byte{0x80, 0x00}, -32768},
		{"3 bytes positive", []byte{0x7f, 0xff, 0xff}, 8388607},
		{"3 bytes negative", []byte{0x80, 0x00, 0x00}, -8388608},
		{"4 bytes positive", []byte{0x7f, 0xff, 0xff, 0xff}, 2147483647},
		{"4 bytes negative", []byte{0x80, 0x00, 0x00, 0x00}, -2147483648},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			t.Run("Decode", func(t *testing.T) {
				obj := objects.NewObject(objects.TagSignedInteger, false, c.data)
				got, err := objects.DecSignedInteger(obj)
				if err != nil {
					t.Fatal(err)
				}
				if got != c.value {
					t.Errorf("got %d, want %d", got, c.value)
				}
			})
			t.Run("Encode", func(t *testing.T) {
				if got := objects.EncSignedInteger(c.value).Data; !bytes.Equal(c.data, got) {
					t.Errorf("got %x, want %x", got, c.data)
				}
			})
		})
	}
}

func TestEnumerated(t *testing.T) {
	testcases := []struct {
		description string
		value       uint32
		serialized  []byte
	}{
		{"1 byte", 3, []byte{0x91, 0x03}},
		{"1 byte maximum", 255, []byte{0x91, 0xff}},
		{"2 bytes", 256, []byte{0x92, 0x01, 0x00}},
		{"2 bytes maximum", 65535, []byte{0x92, 0xff, 0xff}},
		{"3 bytes", 65536, []byte{0x93, 0x01, 0x00, 0x00}},
		{"4 bytes", 4294967295, []byte{0x94, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			b, err := objects.EncEnumerated(c.value).MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c.serialized, b) {
				t.Fatalf("serialized %x, want %x", b, c.serialized)
			}

			obj, _, err := objects.DecodeTag(b)
			if err != nil {
				t.Fatal(err)
			}
			got, err := objects.DecEnumerated(obj)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.value {
				t.Errorf("got %d, want %d", got, c.value)
			}
		})
	}
}

// Decoders see data of any length off the network: each must return an error,
// not panic, when the data is shorter than its type needs.
func TestDecodeShortData(t *testing.T) {
	decoders := []struct {
		description string
		tagNumber   uint8
		minLen      int
		decode      func(objects.APDUPayload) error
	}{
		{"UnsignedInteger", objects.TagUnsignedInteger, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecUnsignedInteger(o)
			return err
		}},
		{"SignedInteger", objects.TagSignedInteger, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecSignedInteger(o)
			return err
		}},
		{"Real", objects.TagReal, 4, func(o objects.APDUPayload) error {
			_, err := objects.DecReal(o)
			return err
		}},
		{"Double", objects.TagDouble, 8, func(o objects.APDUPayload) error {
			_, err := objects.DecDouble(o)
			return err
		}},
		{"String", objects.TagCharacterString, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecString(o)
			return err
		}},
		{"BitString", objects.TagBitString, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecBitString(o)
			return err
		}},
		{"Enumerated", objects.TagEnumerated, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecEnumerated(o)
			return err
		}},
		{"Date", objects.TagDate, 4, func(o objects.APDUPayload) error {
			_, err := objects.DecDate(o)
			return err
		}},
		{"Time", objects.TagTime, 4, func(o objects.APDUPayload) error {
			_, err := objects.DecTime(o)
			return err
		}},
		{"ObjectIdentifier", objects.TagBACnetObjectIdentifier, 4, func(o objects.APDUPayload) error {
			_, err := objects.DecObjectIdentifier(o)
			return err
		}},
		{"LogStatus", objects.TagBitString, 2, func(o objects.APDUPayload) error {
			_, err := objects.DecLogStatus(o)
			return err
		}},
		{"ContextBool", objects.TagBoolean, 1, func(o objects.APDUPayload) error {
			_, err := objects.DecContextBool(objects.ContextTag(2, o.(*objects.Object)))
			return err
		}},
	}

	for _, d := range decoders {
		t.Run(d.description, func(t *testing.T) {
			for n := 0; n < d.minLen; n++ {
				obj := objects.NewObject(d.tagNumber, false, make([]byte, n))
				if err := d.decode(obj); err == nil {
					t.Errorf("%d data bytes: got no error", n)
				}
			}
			obj := objects.NewObject(d.tagNumber, false, make([]byte, d.minLen))
			if err := d.decode(obj); err != nil {
				t.Errorf("%d data bytes: %v", d.minLen, err)
			}
		})
	}
}
