package objects

import "fmt"

// Tag number
const (
	TagNull uint8 = iota
	TagBoolean
	TagUnsignedInteger
	TagSignedInteger
	TagReal
	TagDouble
	TagOctetString
	TagCharacterString
	TagBitString
	TagEnumerated
	TagDate
	TagTime
	TagBACnetObjectIdentifier
)

// Be sure to check ../bacnet-stack/src/bacnet/bacenum.h for more!
const (
	ObjectTypeAnalogInput  uint16 = 0
	ObjectTypeAnalogOutput uint16 = 1
	ObjectTypeDevice       uint16 = 8
	ObjectTrendLog         uint16 = 20
)

const (
	ErrorClassObject  uint8 = 1
	ErrorClassService uint8 = 5

	ErrorCodeUnknownObject        uint8 = 31
	ErrorCodeServiceRequestDenied uint8 = 29
)

func TagToString(t *Object) string {
	if t.TagClass {
		return fmt.Sprintf("Context %v", t.TagNumber)
	}
	switch t.TagNumber {
	case TagNull:
		return "Null"
	case TagBoolean:
		return "Boolean"
	case TagUnsignedInteger:
		return "UnsignedInteger"
	case TagSignedInteger:
		return "SignedInteger"
	case TagReal:
		return "Real"
	case TagDouble:
		return "Double"
	case TagOctetString:
		return "OctetString"
	case TagCharacterString:
		return "CharacterString"
	case TagBitString:
		return "BitString"
	case TagEnumerated:
		return "Enumerated"
	case TagDate:
		return "Date"
	case TagTime:
		return "Time"
	case TagBACnetObjectIdentifier:
		return "BACnetObjectIdentifier"
	default:
		return "Unknown"
	}
}
