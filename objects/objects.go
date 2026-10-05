package objects

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

type APDUPayload interface {
	UnmarshalBinary([]byte) error
	MarshalBinary() ([]byte, error)
	MarshalTo([]byte) error
	MarshalLen() int
}

// TagKind tells a primitive tag, which carries data, from the opening and
// closing tags that bracket constructed data.
type TagKind uint8

const (
	TagPrimitive TagKind = iota
	TagOpening
	TagClosing
)

// Object is an object in APDU.
type Object struct {
	TagNumber uint8
	TagClass  bool // true = context
	Kind      TagKind
	// Length is the data length in bytes. The one exception is an
	// application-tagged Boolean: it has no data, and its value (0 or 1) is
	// carried in the tag's length field, so it is held here.
	Length uint32
	Data   []byte
	Value  interface{}
}

// NewObject creates an Object.
func NewObject(number uint8, class bool, data []byte) *Object {
	obj := &Object{
		TagNumber: number,
		TagClass:  class,
		Length:    uint32(len(data)),
		Data:      data,
	}

	return obj
}

// Values of the length/value/type field in a tag's initial octet.
const (
	lvtExtended uint8 = 5 // the length follows in 1, 2 or 4 more octets
	lvtOpening  uint8 = 6
	lvtClosing  uint8 = 7
)

// tagNumberExtended in the initial octet's tag field means the tag number
// (15-254) follows in the next octet.
const tagNumberExtended uint8 = 15

// Values of the first extended-length octet that announce a wider length.
const (
	extLen16 uint8 = 254
	extLen32 uint8 = 255
)

// DecodeTag decodes the single tag at the start of b and returns it with the
// number of bytes it occupies. Data is copied, so the Object stays valid when
// b is reused.
func DecodeTag(b []byte) (*Object, int, error) {
	if len(b) == 0 {
		return nil, 0, fmt.Errorf("decode tag: no tag octet: %w", common.ErrTooShortToParse)
	}

	o := &Object{
		TagNumber: b[0] >> 4,
		TagClass:  b[0]&0x8 != 0,
	}
	lvt := b[0] & 0x7
	offset := 1

	if o.TagNumber == tagNumberExtended {
		if len(b) <= offset {
			return nil, 0, fmt.Errorf("decode tag: missing extended tag number: %w", common.ErrTooShortToParse)
		}
		o.TagNumber = b[offset]
		offset++
		if o.TagNumber == 0xFF {
			return nil, 0, fmt.Errorf("decode tag: reserved tag number 255: %w", common.ErrInvalidData)
		}
	}

	switch {
	case o.TagClass && lvt == lvtOpening:
		o.Kind = TagOpening
		return o, offset, nil
	case o.TagClass && lvt == lvtClosing:
		o.Kind = TagClosing
		return o, offset, nil
	case o.isAppBoolean():
		if lvt > 1 {
			return nil, 0, fmt.Errorf("decode tag: boolean value %d: %w", lvt, common.ErrInvalidData)
		}
		o.Length = uint32(lvt)
		return o, offset, nil
	case !o.TagClass && lvt > lvtExtended:
		return nil, 0, fmt.Errorf(
			"decode tag: application tag %d with length field %d: %w", o.TagNumber, lvt, common.ErrInvalidData,
		)
	}

	length := uint32(lvt)
	if lvt == lvtExtended {
		if len(b) <= offset {
			return nil, 0, fmt.Errorf("decode tag: missing extended length: %w", common.ErrTooShortToParse)
		}
		first := b[offset]
		offset++
		switch first {
		case extLen16:
			if len(b) < offset+2 {
				return nil, 0, fmt.Errorf("decode tag: missing 2-byte length: %w", common.ErrTooShortToParse)
			}
			length = uint32(binary.BigEndian.Uint16(b[offset:]))
			offset += 2
		case extLen32:
			if len(b) < offset+4 {
				return nil, 0, fmt.Errorf("decode tag: missing 4-byte length: %w", common.ErrTooShortToParse)
			}
			length = binary.BigEndian.Uint32(b[offset:])
			offset += 4
		default:
			length = uint32(first)
		}
	}

	if uint64(length) > uint64(len(b)-offset) {
		return nil, 0, fmt.Errorf(
			"decode tag: data length %d with %d bytes left: %w", length, len(b)-offset, common.ErrTooShortToParse,
		)
	}
	o.Length = length
	o.Data = append([]byte(nil), b[offset:offset+int(length)]...)

	return o, offset + int(length), nil
}

// UnmarshalBinary sets the values retrieved from the tag at the start of b.
// Bytes after that tag are ignored; use DecodeTag to learn how many were used.
func (o *Object) UnmarshalBinary(b []byte) error {
	dec, _, err := DecodeTag(b)
	if err != nil {
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}
	*o = *dec
	return nil
}

// MarshalBinary returns the byte sequence generated from a Object instance.
func (o *Object) MarshalBinary() ([]byte, error) {
	b := make([]byte, o.MarshalLen())
	if err := o.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal object: %w", err)
	}

	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (o *Object) MarshalTo(b []byte) error {
	if len(b) < o.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal object - marshal length %d binary length %d: %w",
			o.MarshalLen(), len(b), common.ErrTooShortToMarshalBinary,
		)
	}
	if o.TagNumber == 0xFF {
		return fmt.Errorf("failed to marshal object - reserved tag number 255: %w", common.ErrInvalidData)
	}

	var lvt uint8
	switch o.Kind {
	case TagPrimitive:
		switch {
		case o.isAppBoolean():
			if o.Length > 1 {
				return fmt.Errorf("failed to marshal object - boolean value %d: %w", o.Length, common.ErrInvalidData)
			}
			lvt = uint8(o.Length)
		case uint64(len(o.Data)) != uint64(o.Length):
			return fmt.Errorf(
				"failed to marshal object - length %d with %d data bytes: %w",
				o.Length, len(o.Data), common.ErrInvalidData,
			)
		case o.Length < uint32(lvtExtended):
			lvt = uint8(o.Length)
		default:
			lvt = lvtExtended
		}
	case TagOpening, TagClosing:
		if !o.TagClass {
			return fmt.Errorf(
				"failed to marshal object - opening/closing tag with application class: %w", common.ErrInvalidData,
			)
		}
		lvt = lvtOpening
		if o.Kind == TagClosing {
			lvt = lvtClosing
		}
	default:
		return fmt.Errorf("failed to marshal object - tag kind %d: %w", o.Kind, common.ErrInvalidData)
	}

	tagField := o.TagNumber
	if tagField > tagNumberExtended {
		tagField = tagNumberExtended
	}
	b[0] = tagField<<4 | uint8(common.BoolToInt(o.TagClass))<<3 | lvt
	offset := 1
	if o.TagNumber >= tagNumberExtended {
		b[offset] = o.TagNumber
		offset++
	}
	if !o.hasData() {
		return nil
	}

	switch {
	case o.Length < uint32(lvtExtended):
	case o.Length < uint32(extLen16):
		b[offset] = uint8(o.Length)
		offset++
	case o.Length <= 0xFFFF:
		b[offset] = extLen16
		binary.BigEndian.PutUint16(b[offset+1:], uint16(o.Length))
		offset += 3
	default:
		b[offset] = extLen32
		binary.BigEndian.PutUint32(b[offset+1:], o.Length)
		offset += 5
	}
	copy(b[offset:], o.Data)
	return nil
}

// MarshalLen returns the serial length of Object.
func (o *Object) MarshalLen() int {
	l := 1
	if o.TagNumber >= tagNumberExtended {
		l++
	}
	if !o.hasData() {
		return l
	}

	switch {
	case o.Length < uint32(lvtExtended):
	case o.Length < uint32(extLen16):
		l++
	case o.Length <= 0xFFFF:
		l += 3
	default:
		l += 5
	}
	return l + int(o.Length)
}

// isAppBoolean reports whether o is an application-tagged Boolean, whose
// value sits in the length field.
func (o *Object) isAppBoolean() bool {
	return !o.TagClass && o.TagNumber == TagBoolean
}

// hasData reports whether data bytes follow the tag header on the wire.
func (o *Object) hasData() bool {
	return o.Kind == TagPrimitive && !o.isAppBoolean()
}
