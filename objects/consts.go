package objects

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
	TagOpening uint8 = 0x3E
	TagClosing uint8 = 0x3F
)

// Be sure to check ../bacnet-stack/src/bacnet/bacenum.h for more!
const (
	ObjectTypeAnalogInput  uint16 = 0
	ObjectTypeAnalogOutput uint16 = 1
	ObjectTypeDevice       uint16 = 8
)

const (
	PropertyIdPresentValue uint8 = 85
	PropertyIdLogBuffer uint8 = 131
)

const (
	ErrorClassObject  uint8 = 1
	ErrorClassService uint8 = 5

	ErrorCodeUnknownObject        uint8 = 31
	ErrorCodeServiceRequestDenied uint8 = 29
)

func TagToString(t uint8) string {
	switch t {
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
		case TagOpening:
			return "Opening"
		case TagClosing:
			return "Closing"
		default:
			return "Unknown"
	}
}
