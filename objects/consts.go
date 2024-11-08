package objects

// Application Tag number
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

// Object types
const (
	ObjectTypeAnalogInput uint16 = iota
	ObjectTypeAnalogOutput
	ObjectTypeAnalogValue
	ObjectTypeBinaryInput
	ObjectTypeBinaryOutput
	ObjectTypeBinaryValue
	ObjectTypeCalendar
	ObjectTypeCommand
	ObjectTypeDevice
	ObjectTypeEventEnrollment
	ObjectTypeFile
	ObjectTypeGroup
	ObjectTypeLoop
	ObjectTypeMultiStateInput
	ObjectTypeMultiStateOutput
	ObjectTypeNotificationClass
	ObjectTypeProgram
	ObjectTypeSchedule
	ObjectTypeAveraging
	ObjectTypeMultiStateValue
	ObjectTypeTrendLog
	ObjectTypeLifeSafetyPoint
	ObjectTypeLifeSafetyZone
	ObjectTypeAccumulator
	ObjectTypePulseConverter
	ObjectTypeEventLog
	ObjectTypeGlobalGroup
	ObjectTypeTrendLogMultiple
	ObjectTypeLoadControl
	ObjectTypeStructuredView
	ObjectTypeAccessDoor
	ObjectTypeTimer
	ObjectTypeAccessCredential
	ObjectTypeAccessPoint
	ObjectTypeAccessRights
	ObjectTypeAccessUser
	ObjectTypeAccessZone
	ObjectTypeCredentialDataInput
	ObjectTypeNetworkSecurity
)

// Error classes
const (
	ErrorClassDevice   uint8 = iota
	ErrorClassObject
	ErrorClassProperty
	ErrorClassResources
	ErrorClassSecurity
	ErrorClassServices
	ErrorClassVT
	ErrorClassCommunication
	ErrorClassVendor
)

const (
	ErrorCodeUnknownObject        uint8 = 31
	ErrorCodeServiceRequestDenied uint8 = 29
)
