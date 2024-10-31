package objects

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/pkg/errors"
)

func DecString(rawPayload APDUPayload) (string, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return "", errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("DecString not ok: %v", rawPayload),
		)
	}
	if rawObject.TagNumber != TagCharacterString || rawObject.TagClass {
		return "", errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("DecString wrong tag number: %v", rawObject.TagNumber),
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

func DecUnsignedInteger(rawPayload APDUPayload) (uint32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode UnsignedInteger - %v", rawPayload),
		)
	}

	if rawObject.TagNumber != TagUnsignedInteger && !rawObject.TagClass {
		return 0, errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode UnsignedInteger - wrong tag number - %v", rawObject.TagNumber),
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

	return 0, errors.Wrap(
		common.ErrNotImplemented,
		fmt.Sprintf("failed to decode UnsignedInteger - %v", rawObject.Data),
	)
}

func EncUnsignedInteger(value uint) *Object {
	switch {
	case value <= 255:
		return EncUnsignedInteger8(uint8(value))
	case value <= 65535:
		return EncUnsignedInteger16(uint16(value))
	default:
		panic("EncUnsignedInteger not implemented for 32-bit  and 64-bit")
	}
}

func EncUnsignedInteger8(value uint8) *Object {
	newObj := Object{}

	data := make([]byte, 1)
	data[0] = value

	newObj.TagNumber = TagUnsignedInteger
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

func EncUnsignedInteger16(value uint16) *Object {
	newObj := Object{}

	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data[:], value)

	newObj.TagNumber = TagUnsignedInteger
	newObj.TagClass = false
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}

func DecEnumerated(rawPayload APDUPayload) (uint32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode EnumObject - %v", rawPayload),
		)
	}

	if rawObject.TagNumber != TagEnumerated && !rawObject.TagClass {
		return 0, errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode EnumObject - wrong tag number - %v", rawObject.TagNumber),
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

	return 0, errors.Wrap(
		common.ErrNotImplemented,
		fmt.Sprintf("failed to decode EnumObject - %v", rawObject.Data),
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

func DecReal(rawPayload APDUPayload) (float32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode Real - %v", rawPayload),
		)
	}

	if rawObject.TagNumber != TagReal && !rawObject.TagClass {
		return 0, errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode real - wrong tag number - %v", rawObject.TagNumber),
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

func DecNull(rawPayload APDUPayload) error {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode Null - %v", rawPayload),
		)
	}

	if rawObject.TagNumber != TagNull && !rawObject.TagClass {
		return errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode Null - wrong tag number - %v", rawObject.TagNumber),
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

func DecTime(rawPayload APDUPayload) (time.Time, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return time.Time{}, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode Time - %v", rawPayload),
		)
	}

	if rawObject.Length != 4 {
		return time.Time{}, errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode Time - wrong length - %v", rawObject.Length),
		)
	}

	hour := int(rawObject.Data[0])
	minute := int(rawObject.Data[1])
	second := int(rawObject.Data[2])
	hundredths := int(rawObject.Data[3])

	return time.Date(0, 1, 1, hour, minute, second, hundredths*10_000_000, time.UTC), nil
}

func DecDate(rawPayload APDUPayload) (time.Time, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return time.Time{}, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode Date - %v", rawPayload),
		)
	}
	if rawObject.Length != 4 {
		return time.Time{}, errors.Wrap(
			common.ErrWrongStructure,
			fmt.Sprintf("failed to decode Date - wrong length - %v", rawObject.Length),
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
	if date.Weekday() != weekday {
		return time.Time{}, errors.Wrap(
			common.ErrInvalidData,
			fmt.Sprintf("failed to decode Date - weekday mismatch - %v %v", weekday, date.Weekday()),
		)
	}

	return date, nil
}

func DecBitString(rawPayload APDUPayload) (uint32, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode BitString - %v", rawPayload),
		)
	}
	unused := int(rawObject.Data[0])
	var bits uint32
	for i := len(rawObject.Data) - 1; i > 0; i-- {
		bits = bits<<8 | uint32(rawObject.Data[i])
	}
	return bits >> unused, nil
}
