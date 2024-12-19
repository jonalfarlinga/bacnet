package objects

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/jonalfarlinga/bacnet/common"
)

var TagMap = map[uint8]string{
	TagNull:                   "Null",
	TagBoolean:                "Boolean",
	TagUnsignedInteger:        "UnsignedInteger",
	TagSignedInteger:          "SignedInteger",
	TagReal:                   "Real",
	TagDouble:                 "Double",
	TagOctetString:            "OctetString",
	TagCharacterString:        "CharacterString",
	TagBitString:              "BitString",
	TagEnumerated:             "Enumerated",
	TagDate:                   "Date",
	TagTime:                   "Time",
	TagBACnetObjectIdentifier: "BACnetObjectIdentifier",
}

// TagNumber 0
func DecNull(rawPayload APDUPayload) error {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return fmt.Errorf(
			"failed to decode Null - %+v: %s", rawPayload, common.ErrWrongPayload,
		)
	}

	if rawObject.TagNumber != TagNull && !rawObject.TagClass {
		return fmt.Errorf(
			"failed to decode Null - %+v: %v", rawObject.TagNumber, common.ErrWrongStructure,
		)
	}

	return nil
}

func EncNull() *Object {
	newObj := Object{}

	newObj.TagNumber = TagNull
	newObj.TagClass = false
	newObj.Data = nil
	newObj.Length = 0

	return &newObj
}

// TagNumber 1
func DecBoolean(rawPayload APDUPayload) (bool, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return false, fmt.Errorf(
			"DecBoolean not ok - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}
	return rawObject.Length == 1, nil
}

func EncBoolean(value bool) *Object {
	newObj := Object{}

	newObj.TagNumber = TagBoolean
	newObj.TagClass = false
	if value {
		newObj.Length = 1
	} else {
		newObj.Length = 0
	}
	return &newObj
}

// TagNumber 2
func DecUnsignedInteger(rawPayload APDUPayload) (uint32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, fmt.Errorf(
			"failed to decode UnsignedInteger - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}

	if rawObject.TagNumber != TagUnsignedInteger && !rawObject.TagClass {
		return 0, fmt.Errorf(
			"failed to decode UnsignedInteger - %+v: %v", rawObject.TagNumber, common.ErrWrongStructure,
		)
	}

	switch rawObject.Length {
	case 1:
		return uint32(rawObject.Data[0]), nil
	case 2:
		return uint32(binary.BigEndian.Uint16(rawObject.Data)), nil
	case 3:
		return uint32(uint16(uint32(rawObject.Data[0])<<16) | binary.BigEndian.Uint16(rawObject.Data[1:])), nil
	case 4:
		return binary.BigEndian.Uint32(rawObject.Data), nil
	}

	return 0, fmt.Errorf(
		"failed to decode UnsignedInteger - %+v: %v", rawObject.Data, common.ErrNotImplemented,
	)
}

func EncUnsignedInteger(value uint) *Object {
	newObj := Object{}
	var data []byte
	switch {
	case value <= 255:
		data = make([]byte, 1)
		data[0] = byte(value)
	case value <= 65535:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data[:], uint16(value))
	case value <= 4294967295:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data[:], uint32(value))
	default:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data[:], uint64(value))
	}
	newObj.TagNumber = TagUnsignedInteger
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 3
func DecSignedInteger(rawPayload APDUPayload) (int, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, fmt.Errorf(
			"failed to decode SignedInteger - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}
	switch rawObject.Length {
	case 1:
		return int(int8(rawObject.Data[0])), nil
	case 2:
		return int(int16(binary.BigEndian.Uint16(rawObject.Data))), nil
	case 3:
		return int(int32(uint32(rawObject.Data[0])<<16) | int32(binary.BigEndian.Uint16(rawObject.Data[1:]))), nil
	case 4:
		return int(binary.BigEndian.Uint32(rawObject.Data)), nil
	}
	return 0, fmt.Errorf(
		"failed to decode SignedInteger - %+v: %v", rawObject.Data, common.ErrNotImplemented,
	)
}

func EncSignedInteger(value int) *Object {
	newObj := Object{}

	var data []byte
	switch {
	case value >= -128 && value <= 127:
		data = make([]byte, 1)
		data[0] = byte(value)
	case value >= -32768 && value <= 32767:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, uint16(value))
	case value >= -8388608 && value <= 8388607:
		data = make([]byte, 3)
		data[0] = byte(value >> 16)
		binary.BigEndian.PutUint16(data[1:], uint16(value))
	default:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(value))
	}

	newObj.TagNumber = TagSignedInteger
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 4
func DecReal(rawPayload APDUPayload) (float32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, fmt.Errorf(
			"failed to decode Real - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}

	if rawObject.TagNumber != TagReal && !rawObject.TagClass {
		return 0, fmt.Errorf(
			"failed to decode real - %+v: %v", rawObject.TagNumber, common.ErrWrongStructure,
		)
	}

	return math.Float32frombits(binary.BigEndian.Uint32(rawObject.Data)), nil
}

func EncReal(value float32) *Object {
	newObj := Object{}

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data[:], math.Float32bits(value))

	newObj.TagNumber = TagReal
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 5
func DecDouble(rawPayload APDUPayload) (float64, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, fmt.Errorf(
			"failed to decode Double - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}
	return math.Float64frombits(binary.BigEndian.Uint64(rawObject.Data)), nil
}

func EncDouble(value float64) *Object {
	newObj := Object{}

	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data[:], math.Float64bits(value))

	newObj.TagNumber = TagDouble
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 6
func DecOctetString(rawPayload APDUPayload) ([]byte, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return nil, fmt.Errorf(
			"failed to decode OctetString - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}
	return rawObject.Data, nil
}

func EncOctetString(value []byte) *Object {
	newObj := Object{}

	newObj.TagNumber = TagOctetString
	newObj.TagClass = false
	newObj.Data = value
	newObj.Length = uint8(len(value))

	return &newObj
}

// TagNumber 7
func DecString(rawPayload APDUPayload) (string, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return "", fmt.Errorf(
			"DecString not ok - %+v: %v", rawPayload,
			common.ErrWrongPayload,
		)
	}
	if rawObject.TagNumber != TagCharacterString || rawObject.TagClass {
		return "", fmt.Errorf(
			"DecString wrong tag number - %+v: %v", rawObject.TagNumber, common.ErrWrongStructure,
		)
	}
	return string(rawObject.Data[1:]), nil
}

func EncString(value string) *Object {
	newObj := Object{}
	newObj.TagNumber = TagCharacterString
	newObj.TagClass = false
	newObj.Data = append([]byte{0}, []byte(value)...)
	newObj.Length = uint8(len(newObj.Data))
	return &newObj
}

// TagNumber 8
func DecBitString(rawPayload APDUPayload) ([]bool, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return nil, fmt.Errorf(
			"failed to decode BitString - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}
	unused := int(rawObject.Data[0])
	var bits []bool
	for i := 1; i < len(rawObject.Data); i++ {
		for j := 0; j < 8; j++ {
			bit := (rawObject.Data[i] >> uint8(7-j)) & 0x01
			bits = append(bits, bit == 1)
		}
	}
	return bits[:len(bits)-unused], nil
}

func EncBitString(value []bool) *Object {
	newObj := Object{}
	data := make([]byte, 1)
	var unused = 8 - (len(value) % 8)
	if unused == 8 {
		unused = 0
	}
	data[0] = byte(unused)

	for i := 0; i < len(value); i += 8 {
		var byteValue byte
		for j := 0; j < 8; j++ {
			if i+j < len(value) {
				if value[i+j] {
					byteValue |= (1 << uint8(7-j))
				}
			}
		}
		data = append(data, byteValue)
	}

	newObj.TagNumber = TagBitString
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 9
func DecEnumerated(rawPayload APDUPayload) (uint32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, fmt.Errorf(
			"failed to decode EnumObject - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}

	if rawObject.TagNumber != TagEnumerated && !rawObject.TagClass {
		return 0, fmt.Errorf(
			"failed to decode EnumObject - %+v: %v", rawObject.TagNumber, common.ErrWrongStructure,
		)
	}

	switch rawObject.Length {
	case 1:
		return uint32(rawObject.Data[0]), nil
	case 2:
		return uint32(binary.BigEndian.Uint16(rawObject.Data)), nil
	case 3:
		return uint32(uint16(uint32(rawObject.Data[0])<<16) | binary.BigEndian.Uint16(rawObject.Data[1:])), nil
	case 4:
		return binary.BigEndian.Uint32(rawObject.Data), nil
	}

	return 0, fmt.Errorf(
		"failed to decode EnumObject - %+v: %v", rawObject.Data, common.ErrNotImplemented,
	)
}

func EncEnumerated(value uint8) *Object {
	newObj := Object{}

	data := make([]byte, 1)
	data[0] = value

	newObj.TagNumber = TagEnumerated
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

// TagNumber 10
func DecDate(rawPayload APDUPayload) (time.Time, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return time.Time{}, fmt.Errorf(
			"failed to decode Date - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}

	year := int(rawObject.Data[0]) + 1900
	month := time.Month(rawObject.Data[1])
	day := int(rawObject.Data[2])
	weekday := time.Weekday(rawObject.Data[3])
	if weekday == 7 {
		weekday = time.Weekday(0)
	}
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	return date, nil
}

// TagNumber 11
func DecTime(rawPayload APDUPayload) (time.Time, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return time.Time{}, fmt.Errorf(
			"failed to decode Time - %+v: %v", rawPayload, common.ErrWrongPayload,
		)
	}

	if rawObject.Length != 4 {
		return time.Time{}, fmt.Errorf(
			"failed to decode Time - wrong length - %+v: %v", rawObject.Length,
			common.ErrWrongStructure,
		)
	}

	hour := int(rawObject.Data[0])
	minute := int(rawObject.Data[1])
	second := int(rawObject.Data[2])
	hundredths := int(rawObject.Data[3])

	return time.Date(0, 1, 1, hour, minute, second, hundredths*10_000_000, time.UTC), nil
}
