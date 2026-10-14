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
	ErrorClassDevice uint8 = iota
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

// Abort reasons (BACnetAbortReason)
const (
	AbortReasonOther uint8 = iota
	AbortReasonBufferOverflow
	AbortReasonInvalidAPDUInThisState
	AbortReasonPreemptedByHigherPriorityTask
	AbortReasonSegmentationNotSupported
	AbortReasonSecurityError
	AbortReasonInsufficientSecurity
	AbortReasonWindowSizeOutOfRange
	AbortReasonApplicationExceededReplyTime
	AbortReasonOutOfResources
	AbortReasonTSMTimeout
	AbortReasonAPDUTooLong
)

// AbortReasonMap gives a human-readable label for each BACnetAbortReason value.
var AbortReasonMap = map[uint8]string{
	AbortReasonOther:                         "Other",
	AbortReasonBufferOverflow:                "Buffer overflow",
	AbortReasonInvalidAPDUInThisState:        "Invalid APDU in this state",
	AbortReasonPreemptedByHigherPriorityTask: "Preempted by higher priority task",
	AbortReasonSegmentationNotSupported:      "Segmentation not supported",
	AbortReasonSecurityError:                 "Security error",
	AbortReasonInsufficientSecurity:          "Insufficient security",
	AbortReasonWindowSizeOutOfRange:          "Window size out of range",
	AbortReasonApplicationExceededReplyTime:  "Application exceeded reply time",
	AbortReasonOutOfResources:                "Out of resources",
	AbortReasonTSMTimeout:                    "TSM timeout",
	AbortReasonAPDUTooLong:                   "APDU too long",
}

// Reject reasons (BACnetRejectReason)
const (
	RejectReasonOther uint8 = iota
	RejectReasonBufferOverflow
	RejectReasonInconsistentParameters
	RejectReasonInvalidParameterDataType
	RejectReasonInvalidTag
	RejectReasonMissingRequiredParameter
	RejectReasonParameterOutOfRange
	RejectReasonTooManyArguments
	RejectReasonUndefinedEnumeration
	RejectReasonUnrecognizedService
)

// RejectReasonMap gives a human-readable label for each BACnetRejectReason value.
var RejectReasonMap = map[uint8]string{
	RejectReasonOther:                    "Other",
	RejectReasonBufferOverflow:           "Buffer overflow",
	RejectReasonInconsistentParameters:   "Inconsistent parameters",
	RejectReasonInvalidParameterDataType: "Invalid parameter data type",
	RejectReasonInvalidTag:               "Invalid tag",
	RejectReasonMissingRequiredParameter: "Missing required parameter",
	RejectReasonParameterOutOfRange:      "Parameter out of range",
	RejectReasonTooManyArguments:         "Too many arguments",
	RejectReasonUndefinedEnumeration:     "Undefined enumeration",
	RejectReasonUnrecognizedService:      "Unrecognized service",
}
