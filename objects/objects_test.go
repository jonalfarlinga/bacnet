package objects_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
)

// fill returns n bytes of a repeating, non-zero pattern.
func fill(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i%251) + 1
	}
	return b
}

// join concatenates a tag header and its data.
func join(header []byte, data []byte) []byte {
	return append(append([]byte(nil), header...), data...)
}

func TestTagRoundTrip(t *testing.T) {
	testcases := []struct {
		description string
		structured  *objects.Object
		serialized  []byte
	}{
		{
			description: "application null, length 0",
			structured:  objects.EncNull(),
			serialized:  []byte{0x00},
		},
		{
			description: "context tag, length 0",
			structured:  &objects.Object{TagNumber: 2, TagClass: true},
			serialized:  []byte{0x28},
		},
		{
			description: "length 1",
			structured:  objects.EncUnsignedInteger(72),
			serialized:  []byte{0x21, 0x48},
		},
		{
			description: "length 2",
			structured:  objects.EncUnsignedInteger(1476),
			serialized:  []byte{0x22, 0x05, 0xc4},
		},
		{
			description: "length 3",
			structured:  objects.ContextTag(1, objects.NewObject(0, false, []byte{0x3f, 0xff, 0xff})),
			serialized:  []byte{0x1b, 0x3f, 0xff, 0xff},
		},
		{
			description: "length 4",
			structured:  objects.EncReal(1.1),
			serialized:  []byte{0x44, 0x3f, 0x8c, 0xcc, 0xcd},
		},
		{
			description: "extended length, 1 byte, minimum 5",
			structured:  objects.EncString("AO-1"),
			serialized:  []byte{0x75, 0x05, 0x00, 'A', 'O', '-', '1'},
		},
		{
			description: "extended length, 1 byte, maximum 253",
			structured:  objects.EncOctetString(fill(253)),
			serialized:  join([]byte{0x65, 0xfd}, fill(253)),
		},
		{
			description: "extended length, 2 bytes, minimum 254",
			structured:  objects.EncOctetString(fill(254)),
			serialized:  join([]byte{0x65, 0xfe, 0x00, 0xfe}, fill(254)),
		},
		{
			description: "extended length, 2 bytes, maximum 65535",
			structured:  objects.EncOctetString(fill(65535)),
			serialized:  join([]byte{0x65, 0xfe, 0xff, 0xff}, fill(65535)),
		},
		{
			description: "extended length, 4 bytes, minimum 65536",
			structured:  objects.EncOctetString(fill(65536)),
			serialized:  join([]byte{0x65, 0xff, 0x00, 0x01, 0x00, 0x00}, fill(65536)),
		},
		{
			description: "boolean false, value in the length field",
			structured:  objects.EncBoolean(false),
			serialized:  []byte{0x10},
		},
		{
			description: "boolean true, value in the length field",
			structured:  objects.EncBoolean(true),
			serialized:  []byte{0x11},
		},
		{
			description: "context boolean carries a data byte",
			structured:  objects.EncContextBool(2, true),
			serialized:  []byte{0x29, 0x01},
		},
		{
			description: "opening tag",
			structured:  objects.EncOpeningTag(3),
			serialized:  []byte{0x3e},
		},
		{
			description: "closing tag",
			structured:  objects.EncClosingTag(3),
			serialized:  []byte{0x3f},
		},
		{
			description: "highest tag number without extension",
			structured:  objects.ContextTag(14, objects.EncUnsignedInteger(1)),
			serialized:  []byte{0xe9, 0x01},
		},
		{
			description: "extended tag number, minimum 15",
			structured:  objects.ContextTag(15, objects.EncUnsignedInteger(1)),
			serialized:  []byte{0xf9, 0x0f, 0x01},
		},
		{
			description: "extended tag number, maximum 254",
			structured:  objects.ContextTag(254, objects.EncUnsignedInteger(1)),
			serialized:  []byte{0xf9, 0xfe, 0x01},
		},
		{
			description: "extended tag number with extended length",
			structured:  objects.ContextTag(200, objects.EncOctetString(fill(300))),
			serialized:  join([]byte{0xfd, 0xc8, 0xfe, 0x01, 0x2c}, fill(300)),
		},
		{
			description: "extended tag number, opening tag",
			structured:  objects.EncOpeningTag(100),
			serialized:  []byte{0xfe, 0x64},
		},
		{
			description: "extended tag number, closing tag",
			structured:  objects.EncClosingTag(100),
			serialized:  []byte{0xff, 0x64},
		},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			t.Run("Decode", func(t *testing.T) {
				got, n, err := objects.DecodeTag(c.serialized)
				if err != nil {
					t.Fatal(err)
				}
				if n != len(c.serialized) {
					t.Errorf("consumed %d bytes, want %d", n, len(c.serialized))
				}
				if diff := cmp.Diff(c.structured, got); diff != "" {
					t.Errorf("differs: (-want +got)\n%s", diff)
				}
			})
			t.Run("Serialize", func(t *testing.T) {
				if got, want := c.structured.MarshalLen(), len(c.serialized); got != want {
					t.Errorf("MarshalLen %d, want %d", got, want)
				}
				got, err := c.structured.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(c.serialized, got) {
					t.Errorf("differs: want %x, got %x", c.serialized, got)
				}
			})
		})
	}
}

func TestDecodeTagErrors(t *testing.T) {
	testcases := []struct {
		description string
		serialized  []byte
		want        error
	}{
		{"empty", []byte{}, common.ErrTooShortToParse},
		{"data shorter than length", []byte{0x22, 0x05}, common.ErrTooShortToParse},
		{"length 4 with no data", []byte{0x44}, common.ErrTooShortToParse},
		{"missing extended tag number", []byte{0xf9}, common.ErrTooShortToParse},
		{"missing extended length", []byte{0x75}, common.ErrTooShortToParse},
		{"data shorter than extended length", []byte{0x75, 0x10, 0x00, 0x55}, common.ErrTooShortToParse},
		{"missing 2-byte length", []byte{0x65, 0xfe, 0x01}, common.ErrTooShortToParse},
		{"data shorter than 2-byte length", []byte{0x65, 0xfe, 0x01, 0x00, 0xaa}, common.ErrTooShortToParse},
		{"missing 4-byte length", []byte{0x65, 0xff, 0x00, 0x00, 0x01}, common.ErrTooShortToParse},
		{"4-byte length far past the buffer", []byte{0x65, 0xff, 0xff, 0xff, 0xff, 0xff}, common.ErrTooShortToParse},
		{"reserved tag number 255", []byte{0xf9, 0xff, 0x01}, common.ErrInvalidData},
		{"application tag with opening length field", []byte{0x26}, common.ErrInvalidData},
		{"application tag with closing length field", []byte{0x27}, common.ErrInvalidData},
		{"boolean with value 2", []byte{0x12}, common.ErrInvalidData},
		{"boolean with extended length field", []byte{0x15, 0x01}, common.ErrInvalidData},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			obj, n, err := objects.DecodeTag(c.serialized)
			if !errors.Is(err, c.want) {
				t.Fatalf("got object %+v, n %d, error %v; want error %v", obj, n, err, c.want)
			}
			if err := new(objects.Object).UnmarshalBinary(c.serialized); !errors.Is(err, c.want) {
				t.Errorf("UnmarshalBinary: got error %v, want %v", err, c.want)
			}
		})
	}
}

// A decoder that accepts a non-minimal length or tag-number form must still
// re-encode it in the minimal form.
func TestDecodeTagNonCanonical(t *testing.T) {
	testcases := []struct {
		description string
		serialized  []byte
		canonical   []byte
	}{
		{"1-byte extended length for 2 bytes", []byte{0x25, 0x02, 0x05, 0xc4}, []byte{0x22, 0x05, 0xc4}},
		{"2-byte extended length for 2 bytes", []byte{0x25, 0xfe, 0x00, 0x02, 0x05, 0xc4}, []byte{0x22, 0x05, 0xc4}},
		{"4-byte extended length for 2 bytes", []byte{0x25, 0xff, 0x00, 0x00, 0x00, 0x02, 0x05, 0xc4}, []byte{0x22, 0x05, 0xc4}},
		{"extended tag number below 15", []byte{0xf9, 0x03, 0x01}, []byte{0x39, 0x01}},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			obj, n, err := objects.DecodeTag(c.serialized)
			if err != nil {
				t.Fatal(err)
			}
			if n != len(c.serialized) {
				t.Errorf("consumed %d bytes, want %d", n, len(c.serialized))
			}
			got, err := obj.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c.canonical, got) {
				t.Errorf("differs: want %x, got %x", c.canonical, got)
			}
		})
	}
}

func TestDecodeTagStopsAtTagEnd(t *testing.T) {
	obj, n, err := objects.DecodeTag([]byte{0x21, 0x48, 0x91, 0x00})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("consumed %d bytes, want 2", n)
	}
	if diff := cmp.Diff(objects.EncUnsignedInteger(72), obj); diff != "" {
		t.Errorf("differs: (-want +got)\n%s", diff)
	}
}

// The client reuses its read buffer, so decoded data must be a copy.
func TestDecodeTagCopiesData(t *testing.T) {
	buf := []byte{0x75, 0x05, 0x00, 'A', 'O', '-', '1'}
	obj, _, err := objects.DecodeTag(buf)
	if err != nil {
		t.Fatal(err)
	}
	for i := range buf {
		buf[i] = 0xee
	}
	if want := []byte{0x00, 'A', 'O', '-', '1'}; !bytes.Equal(want, obj.Data) {
		t.Errorf("data changed with the source buffer: want %x, got %x", want, obj.Data)
	}
}

func TestMarshalInvalidObject(t *testing.T) {
	testcases := []struct {
		description string
		structured  *objects.Object
		want        error
	}{
		{
			"length longer than data",
			&objects.Object{TagNumber: objects.TagUnsignedInteger, Length: 2, Data: []byte{0x01}},
			common.ErrInvalidData,
		},
		{
			"length shorter than data",
			&objects.Object{TagNumber: objects.TagUnsignedInteger, Length: 1, Data: []byte{0x01, 0x02}},
			common.ErrInvalidData,
		},
		{
			// the opening-tag sentinel from before Kind existed
			"length 6 without data",
			&objects.Object{TagNumber: 3, TagClass: true, Length: 6},
			common.ErrInvalidData,
		},
		{
			"application opening tag",
			&objects.Object{TagNumber: 3, Kind: objects.TagOpening},
			common.ErrInvalidData,
		},
		{
			"boolean with value 2",
			&objects.Object{TagNumber: objects.TagBoolean, Length: 2},
			common.ErrInvalidData,
		},
		{
			"reserved tag number 255",
			&objects.Object{TagNumber: 255, TagClass: true, Length: 1, Data: []byte{0x01}},
			common.ErrInvalidData,
		},
		{
			"unknown kind",
			&objects.Object{TagNumber: 3, TagClass: true, Kind: objects.TagClosing + 1},
			common.ErrInvalidData,
		},
	}

	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			b, err := c.structured.MarshalBinary()
			if !errors.Is(err, c.want) {
				t.Fatalf("got %x, error %v; want error %v", b, err, c.want)
			}
		})
	}

	t.Run("buffer shorter than MarshalLen", func(t *testing.T) {
		obj := objects.EncReal(1.1)
		err := obj.MarshalTo(make([]byte, obj.MarshalLen()-1))
		if !errors.Is(err, common.ErrTooShortToMarshalBinary) {
			t.Errorf("got error %v, want %v", err, common.ErrTooShortToMarshalBinary)
		}
	})
}
