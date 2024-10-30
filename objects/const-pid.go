package objects

const (
	PropertyIdAckedTransitions uint16 = iota
	PropertyIdAckRequired
	PropertyIdAction
	PropertyIdActionText
	PropertyIdActiveText
	PropertyIdActiveVtSessions
	PropertyIdAlarmValue
	PropertyIdAlarmValues
	PropertyIdAll
	PropertyIdAllWritesSuccessful
	PropertyIdApduSegmentTimeout
	PropertyIdApduTimeout
	PropertyIdApplicationSoftwareVersion
	PropertyIdArchive
	PropertyIdBias
	PropertyIdChangeOfStateCount
	PropertyIdChangeOfStateTime
	PropertyIdNotificationClass
	PropertyIdThisPropertyDeleted
	PropertyIdControlledVariableReference
	PropertyIdControlledVariableUnits
	PropertyIdControlledVariableValue
	PropertyIdCovIncrement
	PropertyIdDateList
	PropertyIdDaylightSavingsStatus
	PropertyIdDeadband
	PropertyIdDerivativeConstant
	PropertyIdDerivativeConstantUnits
	PropertyIdDescription
	PropertyIdDescriptionOfHalt
	PropertyIdDeviceAddressBinding
	PropertyIdDeviceType
	PropertyIdEffectivePeriod
	PropertyIdElapsedActiveTime
	PropertyIdErrorLimit
	PropertyIdEventEnable
	PropertyIdEventState
	PropertyIdEventType
	PropertyIdExceptionSchedule
	PropertyIdFaultValues
	PropertyIdFeedbackValue
	PropertyIdFileAccessMethod
	PropertyIdFileSize
	PropertyIdFileType
	PropertyIdFirmwareRevision
	PropertyIdHighLimit
	PropertyIdInactiveText
	PropertyIdInProcess
	PropertyIdInstanceOf
	PropertyIdIntegralConstant
	PropertyIdIntegralConstantUnits
	PropertyIdRemovedInVersion1Revision4_51
	PropertyIdLimitEnable
	PropertyIdListOfGroupMembers
	PropertyIdListOfObjectPropertyReferences
	PropertyIdUnassigned_55
	PropertyIdLocalDate
	PropertyIdLocalTime
	PropertyIdLocation
	PropertyIdLowLimit
	PropertyIdManipulatedVariableReference
	PropertyIdMaximumOutput
	PropertyIdMaxApduLengthAccepted
	PropertyIdMaxInfoFrames
	PropertyIdMaxMaster
	PropertyIdMaxPresValue
	PropertyIdMinimumOffTime
	PropertyIdMinimumOnTime
	PropertyIdMinimumOutput
	PropertyIdMinPresValue
	PropertyIdModelName
	PropertyIdModificationDate
	PropertyIdNotifyType
	PropertyIdNumberOfApduRetries
	PropertyIdNumberOfStates
	PropertyIdObjectIdentifier
	PropertyIdObjectList
	PropertyIdObjectName
	PropertyIdObjectPropertyReference
	PropertyIdObjectType
	PropertyIdOptional
	PropertyIdOutOfService
	PropertyIdOutputUnits
	PropertyIdEventParameters
	PropertyIdPolarity
	PropertyIdPresentValue
	PropertyIdPriority
	PropertyIdPriorityArray
	PropertyIdPriorityForWriting
	PropertyIdProcessIdentifier
	PropertyIdProgramChange
	PropertyIdProgramLocation
	PropertyIdProgramState
	PropertyIdProportionalConstant
	PropertyIdProportionalConstantUnits
	PropertyIdRemovedInVersion1Revision2_95
	PropertyIdProtocolObjectTypesSupported
	PropertyIdProtocolServicesSupported
	PropertyIdProtocolVersion
	PropertyIdReadOnly
	PropertyIdReasonForHalt
	PropertyIdRemovedInVersion1Revision4_101
	PropertyIdRecipientList
	PropertyIdReliability
	PropertyIdRelinquishDefault
	PropertyIdRequired
	PropertyIdResolution
	PropertyIdSegmentationSupported
	PropertyIdSetpoint
	PropertyIdSetpointReference
	PropertyIdStateText
	PropertyIdStatusFlags
	PropertyIdSystemStatus
	PropertyIdTimeDelay
	PropertyIdTimeOfActiveTimeReset
	PropertyIdTimeOfStateCountReset
	PropertyIdTimeSynchronizationRecipients
	PropertyIdUnits
	PropertyIdUpdateInterval
	PropertyIdUtcOffset
	PropertyIdVendorIdentifier
	PropertyIdVendorName
	PropertyIdVtClassesSupported
	PropertyIdWeeklySchedule
	PropertyIdAttemptedSamples
	PropertyIdAverageValue
	PropertyIdBufferSize
	PropertyIdClientCovIncrement
	PropertyIdCovResubscriptionInterval
	PropertyIdRemovedInVersion1Revision3_129
	PropertyIdEventTimeStamps
	PropertyIdLogBuffer
	PropertyIdLogDeviceObjectProperty
	PropertyIdEnable
	PropertyIdLogInterval
	PropertyIdMaximumValue
	PropertyIdMinimumValue
	PropertyIdNotificationThreshold
	PropertyIdRemovedInVersion1Revision3_138
	PropertyIdProtocolRevision
	PropertyIdRecordsSinceNotification
	PropertyIdRecordCount
	PropertyIdStartTime
	PropertyIdStopTime
	PropertyIdStopWhenFull
	PropertyIdTotalRecordCount
	PropertyIdValidSamples
	PropertyIdWindowInterval
	PropertyIdWindowSamples
	PropertyIdMaximumValueTimestamp
	PropertyIdMinimumValueTimestamp
	PropertyIdVarianceValue
	PropertyIdActiveCovSubscriptions
	PropertyIdBackupFailureTimeout
	PropertyIdConfigurationFiles
	PropertyIdDatabaseRevision
	PropertyIdDirectReading
	PropertyIdLastRestoreTime
	PropertyIdMaintenanceRequired
	PropertyIdMemberOf
	PropertyIdMode
	PropertyIdOperationExpected
	PropertyIdSetting
	PropertyIdSilenced
	PropertyIdTrackingValue
	PropertyIdZoneMembers
	PropertyIdLifeSafetyAlarmValues
	PropertyIdMaxSegmentsAccepted
	PropertyIdProfileName
	PropertyIdAutoSlaveDiscovery
	PropertyIdManualSlaveAddressBinding
	PropertyIdSlaveAddressBinding
	PropertyIdSlaveProxyEnable
	PropertyIdLastNotifyRecord
	PropertyIdScheduleDefault
	PropertyIdAcceptedModes
	PropertyIdAdjustValue
	PropertyIdCount
	PropertyIdCountBeforeChange
	PropertyIdCountChangeTime
	PropertyIdCovPeriod
	PropertyIdInputReference
	PropertyIdLimitMonitoringInterval
	PropertyIdLoggingObject
	PropertyIdLoggingRecord
	PropertyIdPrescale
	PropertyIdPulseRate
	PropertyIdScale
	PropertyIdScaleFactor
	PropertyIdUpdateTime
	PropertyIdValueBeforeChange
	PropertyIdValueSet
	PropertyIdValueChangeTime
	PropertyIdAlignIntervals
	PropertyIdUnassigned_194
	PropertyIdIntervalOffset
	PropertyIdLastRestartReason
	PropertyIdLoggingType
	PropertyIdUnassigned_198
	PropertyIdUnassigned_199
	PropertyIdUnassigned_200
	PropertyIdUnassigned_201
	PropertyIdRestartNotificationRecipients
	PropertyIdTimeOfDeviceRestart
	PropertyIdTimeSynchronizationInterval
	PropertyIdTrigger
	PropertyIdUtcTimeSynchronizationRecipients
	PropertyIdNodeSubtype
	PropertyIdNodeType
	PropertyIdStructuredObjectList
	PropertyIdSubordinateAnnotations
	PropertyIdSubordinateList
	PropertyIdActualShedLevel
	PropertyIdDutyWindow
	PropertyIdExpectedShedLevel
	PropertyIdFullDutyBaseline
	PropertyIdUnassigned_216
	PropertyIdUnassigned_217
	PropertyIdRequestedShedLevel
	PropertyIdShedDuration
	PropertyIdShedLevelDescriptions
	PropertyIdShedLevels
	PropertyIdStateDescription
	PropertyIdUnassigned_223
	PropertyIdUnassigned_224
	PropertyIdUnassigned_225
	PropertyIdDoorAlarmState
	PropertyIdDoorExtendedPulseTime
	PropertyIdDoorMembers
	PropertyIdDoorOpenTooLongTime
	PropertyIdDoorPulseTime
	PropertyIdDoorStatus
	PropertyIdDoorUnlockDelayTime
	PropertyIdLockStatus
	PropertyIdMaskedAlarmValues
	PropertyIdSecuredStatus
	PropertyIdUnassigned_236
	PropertyIdUnassigned_237
	PropertyIdUnassigned_238
	PropertyIdUnassigned_239
	PropertyIdUnassigned_240
	PropertyIdUnassigned_241
	PropertyIdUnassigned_242
	PropertyIdUnassigned_243
	PropertyIdAbsenteeLimit
	PropertyIdAccessAlarmEvents
	PropertyIdAccessDoors
	PropertyIdAccessEvent
	PropertyIdAccessEventAuthenticationFactor
	PropertyIdAccessEventCredential
	PropertyIdAccessEventTime
	PropertyIdAccessTransactionEvents
	PropertyIdAccompaniment
	PropertyIdAccompanimentTime
	PropertyIdActivationTime
	PropertyIdActiveAuthenticationPolicy
	PropertyIdAssignedAccessRights
	PropertyIdAuthenticationFactors
	PropertyIdAuthenticationPolicyList
	PropertyIdAuthenticationPolicyNames
	PropertyIdAuthenticationStatus
	PropertyIdAuthorizationMode
	PropertyIdBelongsTo
	PropertyIdCredentialDisable
	PropertyIdCredentialStatus
	PropertyIdCredentials
	PropertyIdCredentialsInZone
	PropertyIdDaysRemaining
	PropertyIdEntryPoints
	PropertyIdExitPoints
	PropertyIdExpiryTime
	PropertyIdExtendedTimeEnable
	PropertyIdFailedAttemptEvents
	PropertyIdFailedAttempts
	PropertyIdFailedAttemptsTime
	PropertyIdLastAccessEvent
	PropertyIdLastAccessPoint
	PropertyIdLastCredentialAdded
	PropertyIdLastCredentialAddedTime
	PropertyIdLastCredentialRemoved
	PropertyIdLastCredentialRemovedTime
	PropertyIdLastUseTime
	PropertyIdLockout
	PropertyIdLockoutRelinquishTime
	PropertyIdRemovedInVersion1Revision13_284
	PropertyIdMaxFailedAttempts
	PropertyIdMembers
	PropertyIdMusterPoint
	PropertyIdNegativeAccessRules
	PropertyIdNumberOfAuthenticationPolicies
	PropertyIdOccupancyCount
	PropertyIdOccupancyCountAdjust
	PropertyIdOccupancyCountEnable
	PropertyIdRemovedInVersion1Revision13_293
	PropertyIdOccupancyLowerLimit
	PropertyIdOccupancyLowerLimitEnforced
	PropertyIdOccupancyState
	PropertyIdOccupancyUpperLimit
	PropertyIdOccupancyUpperLimitEnforced
	PropertyIdRemovedInVersion1Revision13_299
	PropertyIdPassbackMode
	PropertyIdPassbackTimeout
	PropertyIdPositiveAccessRules
	PropertyIdReasonForDisable
	PropertyIdSupportedFormats
	PropertyIdSupportedFormatClasses
	PropertyIdThreatAuthority
	PropertyIdThreatLevel
	PropertyIdTraceFlag
	PropertyIdTransactionNotificationClass
	PropertyIdUserExternalIdentifier
	PropertyIdUserInformationReference
	PropertyIdUnassigned_312
	PropertyIdUnassigned_313
	PropertyIdUnassigned_314
	PropertyIdUnassigned_315
	PropertyIdUnassigned_316
	PropertyIdUserName
	PropertyIdUserType
	PropertyIdUsesRemaining
	PropertyIdZoneFrom
	PropertyIdZoneTo
	PropertyIdAccessEventTag
	PropertyIdGlobalIdentifier
	PropertyIdUnassigned_324
	PropertyIdUnassigned_325
	PropertyIdVerificationTime
	PropertyIdBaseDeviceSecurityPolicy
	PropertyIdDistributionKeyRevision
	PropertyIdDoNotHide
	PropertyIdKeySets
	PropertyIdLastKeyServer
	PropertyIdNetworkAccessSecurityPolicies
	PropertyIdPacketReorderTime
	PropertyIdSecurityPduTimeout
	PropertyIdSecurityTimeWindow
	PropertyIdSupportedSecurityAlgorithms
	PropertyIdUpdateKeySetTimeout
	PropertyIdBackupAndRestoreState
	PropertyIdBackupPreparationTime
	PropertyIdRestoreCompletionTime
	PropertyIdRestorePreparationTime
	PropertyIdBitMask
	PropertyIdBitText
	PropertyIdIsUtc
	PropertyIdGroupMembers
	PropertyIdGroupMemberNames
	PropertyIdMemberStatusFlags
	PropertyIdRequestedUpdateInterval
	PropertyIdCovuPeriod
	PropertyIdCovuRecipients
	PropertyIdEventMessageTexts
	PropertyIdEventMessageTextsConfig
	PropertyIdEventDetectionEnable
	PropertyIdEventAlgorithmInhibit
	PropertyIdEventAlgorithmInhibitRef
	PropertyIdTimeDelayNormal
	PropertyIdReliabilityEvaluationInhibit
	PropertyIdFaultParameters
	PropertyIdFaultType
	PropertyIdLocalForwardingOnly
	PropertyIdProcessIdentifierFilter
	PropertyIdSubscribedRecipients
	PropertyIdPortFilter
	PropertyIdAuthorizationExemptions
	PropertyIdAllowGroupDelayInhibit
	PropertyIdChannelNumber
	PropertyIdControlGroups
	PropertyIdExecutionDelay
	PropertyIdLastPriority
	PropertyIdWriteStatus
	PropertyIdPropertyList
	PropertyIdSerialNumber
	PropertyIdBlinkWarnEnable
	PropertyIdDefaultFadeTime
	PropertyIdDefaultRampRate
	PropertyIdDefaultStepIncrement
	PropertyIdEgressTime
	PropertyIdInProgress
	PropertyIdInstantaneousPower
	PropertyIdLightingCommand
	PropertyIdLightingCommandDefaultPriority
	PropertyIdMaxActualValue
	PropertyIdMinActualValue
	PropertyIdPower
	PropertyIdTransition
	PropertyIdEgressActive
)

var PropertyMap = map[uint16]string{
	PropertyIdAckedTransitions: "AckedTransitions",
	PropertyIdAckRequired: "AckRequired",
	PropertyIdAction: "Action",
	PropertyIdActionText: "ActionText",
	PropertyIdActiveText: "ActiveText",
	PropertyIdActiveVtSessions: "ActiveVtSessions",
	PropertyIdAlarmValue: "AlarmValue",
	PropertyIdAlarmValues: "AlarmValues",
	PropertyIdAll: "All",
	PropertyIdAllWritesSuccessful: "AllWritesSuccessful",
	PropertyIdApduSegmentTimeout: "ApduSegmentTimeout",
	PropertyIdApduTimeout: "ApduTimeout",
	PropertyIdApplicationSoftwareVersion: "ApplicationSoftwareVersion",
	PropertyIdArchive: "Archive",
	PropertyIdBias: "Bias",
	PropertyIdChangeOfStateCount: "ChangeOfStateCount",
	PropertyIdChangeOfStateTime: "ChangeOfStateTime",
	PropertyIdNotificationClass: "NotificationClass",
	PropertyIdThisPropertyDeleted: "ThisPropertyDeleted",
	PropertyIdControlledVariableReference: "ControlledVariableReference",
	PropertyIdControlledVariableUnits: "ControlledVariableUnits",
	PropertyIdControlledVariableValue: "ControlledVariableValue",
	PropertyIdCovIncrement: "CovIncrement",
	PropertyIdDateList: "DateList",
	PropertyIdDaylightSavingsStatus: "DaylightSavingsStatus",
	PropertyIdDeadband: "Deadband",
	PropertyIdDerivativeConstant: "DerivativeConstant",
	PropertyIdDerivativeConstantUnits: "DerivativeConstantUnits",
	PropertyIdDescription: "Description",
	PropertyIdDescriptionOfHalt: "DescriptionOfHalt",
	PropertyIdDeviceAddressBinding: "DeviceAddressBinding",
	PropertyIdDeviceType: "DeviceType",
	PropertyIdEffectivePeriod: "EffectivePeriod",
	PropertyIdElapsedActiveTime: "ElapsedActiveTime",
	PropertyIdErrorLimit: "ErrorLimit",
	PropertyIdEventEnable: "EventEnable",
	PropertyIdEventState: "EventState",
	PropertyIdEventType: "EventType",
	PropertyIdExceptionSchedule: "ExceptionSchedule",
	PropertyIdFaultValues: "FaultValues",
	PropertyIdFeedbackValue: "FeedbackValue",
	PropertyIdFileAccessMethod: "FileAccessMethod",
	PropertyIdFileSize: "FileSize",
	PropertyIdFileType: "FileType",
	PropertyIdFirmwareRevision: "FirmwareRevision",
	PropertyIdHighLimit: "HighLimit",
	PropertyIdInactiveText: "InactiveText",
	PropertyIdInProcess: "InProcess",
	PropertyIdInstanceOf: "InstanceOf",
	PropertyIdIntegralConstant: "IntegralConstant",
	PropertyIdIntegralConstantUnits: "IntegralConstantUnits",
	PropertyIdRemovedInVersion1Revision4_51: "RemovedInVersion1Revision4_51",
	PropertyIdLimitEnable: "LimitEnable",
	PropertyIdListOfGroupMembers: "ListOfGroupMembers",
	PropertyIdListOfObjectPropertyReferences: "ListOfObjectPropertyReferences",
	PropertyIdUnassigned_55: "Unassigned_55",
	PropertyIdLocalDate: "LocalDate",
	PropertyIdLocalTime: "LocalTime",
	PropertyIdLocation: "Location",
	PropertyIdLowLimit: "LowLimit",
	PropertyIdManipulatedVariableReference: "ManipulatedVariableReference",
	PropertyIdMaximumOutput: "MaximumOutput",
	PropertyIdMaxApduLengthAccepted: "MaxApduLengthAccepted",
	PropertyIdMaxInfoFrames: "MaxInfoFrames",
	PropertyIdMaxMaster: "MaxMaster",
	PropertyIdMaxPresValue: "MaxPresValue",
	PropertyIdMinimumOffTime: "MinimumOffTime",
	PropertyIdMinimumOnTime: "MinimumOnTime",
	PropertyIdMinimumOutput: "MinimumOutput",
	PropertyIdMinPresValue: "MinPresValue",
	PropertyIdModelName: "ModelName",
	PropertyIdModificationDate: "ModificationDate",
	PropertyIdNotifyType: "NotifyType",
	PropertyIdNumberOfApduRetries: "NumberOfApduRetries",
	PropertyIdNumberOfStates: "NumberOfStates",
	PropertyIdObjectIdentifier: "ObjectIdentifier",
	PropertyIdObjectList: "ObjectList",
	PropertyIdObjectName: "ObjectName",
	PropertyIdObjectPropertyReference: "ObjectPropertyReference",
	PropertyIdObjectType: "ObjectType",
	PropertyIdOptional: "Optional",
	PropertyIdOutOfService: "OutOfService",
	PropertyIdOutputUnits: "OutputUnits",
	PropertyIdEventParameters: "EventParameters",
	PropertyIdPolarity: "Polarity",
	PropertyIdPresentValue: "PresentValue",
	PropertyIdPriority: "Priority",
	PropertyIdPriorityArray: "PriorityArray",
	PropertyIdPriorityForWriting: "PriorityForWriting",
	PropertyIdProcessIdentifier: "ProcessIdentifier",
	PropertyIdProgramChange: "ProgramChange",
	PropertyIdProgramLocation: "ProgramLocation",
	PropertyIdProgramState: "ProgramState",
	PropertyIdProportionalConstant: "ProportionalConstant",
	PropertyIdProportionalConstantUnits: "ProportionalConstantUnits",
	PropertyIdRemovedInVersion1Revision2_95: "RemovedInVersion1Revision2_95",
	PropertyIdProtocolObjectTypesSupported: "ProtocolObjectTypesSupported",
	PropertyIdProtocolServicesSupported: "ProtocolServicesSupported",
	PropertyIdProtocolVersion: "ProtocolVersion",
	PropertyIdReadOnly: "ReadOnly",
	PropertyIdReasonForHalt: "ReasonForHalt",
	PropertyIdRemovedInVersion1Revision4_101: "RemovedInVersion1Revision4_101",
	PropertyIdRecipientList: "RecipientList",
	PropertyIdReliability: "Reliability",
	PropertyIdRelinquishDefault: "RelinquishDefault",
	PropertyIdRequired: "Required",
	PropertyIdResolution: "Resolution",
	PropertyIdSegmentationSupported: "SegmentationSupported",
	PropertyIdSetpoint: "Setpoint",
	PropertyIdSetpointReference: "SetpointReference",
	PropertyIdStateText: "StateText",
	PropertyIdStatusFlags: "StatusFlags",
	PropertyIdSystemStatus: "SystemStatus",
	PropertyIdTimeDelay: "TimeDelay",
	PropertyIdTimeOfActiveTimeReset: "TimeOfActiveTimeReset",
	PropertyIdTimeOfStateCountReset: "TimeOfStateCountReset",
	PropertyIdTimeSynchronizationRecipients: "TimeSynchronizationRecipients",
	PropertyIdUnits: "Units",
	PropertyIdUpdateInterval: "UpdateInterval",
	PropertyIdUtcOffset: "UtcOffset",
	PropertyIdVendorIdentifier: "VendorIdentifier",
	PropertyIdVendorName: "VendorName",
	PropertyIdVtClassesSupported: "VtClassesSupported",
	PropertyIdWeeklySchedule: "WeeklySchedule",
	PropertyIdAttemptedSamples: "AttemptedSamples",
	PropertyIdAverageValue: "AverageValue",
	PropertyIdBufferSize: "BufferSize",
	PropertyIdClientCovIncrement: "ClientCovIncrement",
	PropertyIdCovResubscriptionInterval: "CovResubscriptionInterval",
	PropertyIdRemovedInVersion1Revision3_129: "RemovedInVersion1Revision3_129",
	PropertyIdEventTimeStamps: "EventTimeStamps",
	PropertyIdLogBuffer: "LogBuffer",
	PropertyIdLogDeviceObjectProperty: "LogDeviceObjectProperty",
	PropertyIdEnable: "Enable",
	PropertyIdLogInterval: "LogInterval",
	PropertyIdMaximumValue: "MaximumValue",
	PropertyIdMinimumValue: "MinimumValue",
	PropertyIdNotificationThreshold: "NotificationThreshold",
	PropertyIdRemovedInVersion1Revision3_138: "RemovedInVersion1Revision3_138",
	PropertyIdProtocolRevision: "ProtocolRevision",
	PropertyIdRecordsSinceNotification: "RecordsSinceNotification",
	PropertyIdRecordCount: "RecordCount",
	PropertyIdStartTime: "StartTime",
	PropertyIdStopTime: "StopTime",
	PropertyIdStopWhenFull: "StopWhenFull",
	PropertyIdTotalRecordCount: "TotalRecordCount",
	PropertyIdValidSamples: "ValidSamples",
	PropertyIdWindowInterval: "WindowInterval",
	PropertyIdWindowSamples: "WindowSamples",
	PropertyIdMaximumValueTimestamp: "MaximumValueTimestamp",
	PropertyIdMinimumValueTimestamp: "MinimumValueTimestamp",
	PropertyIdVarianceValue: "VarianceValue",
	PropertyIdActiveCovSubscriptions: "ActiveCovSubscriptions",
	PropertyIdBackupFailureTimeout: "BackupFailureTimeout",
	PropertyIdConfigurationFiles: "ConfigurationFiles",
	PropertyIdDatabaseRevision: "DatabaseRevision",
	PropertyIdDirectReading: "DirectReading",
	PropertyIdLastRestoreTime: "LastRestoreTime",
	PropertyIdMaintenanceRequired: "MaintenanceRequired",
	PropertyIdMemberOf: "MemberOf",
	PropertyIdMode: "Mode",
	PropertyIdOperationExpected: "OperationExpected",
	PropertyIdSetting: "Setting",
	PropertyIdSilenced: "Silenced",
	PropertyIdTrackingValue: "TrackingValue",
	PropertyIdZoneMembers: "ZoneMembers",
	PropertyIdLifeSafetyAlarmValues: "LifeSafetyAlarmValues",
	PropertyIdMaxSegmentsAccepted: "MaxSegmentsAccepted",
	PropertyIdProfileName: "ProfileName",
	PropertyIdAutoSlaveDiscovery: "AutoSlaveDiscovery",
	PropertyIdManualSlaveAddressBinding: "ManualSlaveAddressBinding",
	PropertyIdSlaveAddressBinding: "SlaveAddressBinding",
	PropertyIdSlaveProxyEnable: "SlaveProxyEnable",
	PropertyIdLastNotifyRecord: "LastNotifyRecord",
	PropertyIdScheduleDefault: "ScheduleDefault",
	PropertyIdAcceptedModes: "AcceptedModes",
	PropertyIdAdjustValue: "AdjustValue",
	PropertyIdCount: "Count",
	PropertyIdCountBeforeChange: "CountBeforeChange",
	PropertyIdCountChangeTime: "CountChangeTime",
	PropertyIdCovPeriod: "CovPeriod",
	PropertyIdInputReference: "InputReference",
	PropertyIdLimitMonitoringInterval: "LimitMonitoringInterval",
	PropertyIdLoggingObject: "LoggingObject",
	PropertyIdLoggingRecord: "LoggingRecord",
	PropertyIdPrescale: "Prescale",
	PropertyIdPulseRate: "PulseRate",
	PropertyIdScale: "Scale",
	PropertyIdScaleFactor: "ScaleFactor",
	PropertyIdUpdateTime: "UpdateTime",
	PropertyIdValueBeforeChange: "ValueBeforeChange",
	PropertyIdValueSet: "ValueSet",
	PropertyIdValueChangeTime: "ValueChangeTime",
	PropertyIdAlignIntervals: "AlignIntervals",
	PropertyIdUnassigned_194: "Unassigned_194",
	PropertyIdIntervalOffset: "IntervalOffset",
	PropertyIdLastRestartReason: "LastRestartReason",
	PropertyIdLoggingType: "LoggingType",
	PropertyIdUnassigned_198: "Unassigned_198",
	PropertyIdUnassigned_199: "Unassigned_199",
	PropertyIdUnassigned_200: "Unassigned_200",
	PropertyIdUnassigned_201: "Unassigned_201",
	PropertyIdRestartNotificationRecipients: "RestartNotificationRecipients",
	PropertyIdTimeOfDeviceRestart: "TimeOfDeviceRestart",
	PropertyIdTimeSynchronizationInterval: "TimeSynchronizationInterval",
	PropertyIdTrigger: "Trigger",
	PropertyIdUtcTimeSynchronizationRecipients: "UtcTimeSynchronizationRecipients",
	PropertyIdNodeSubtype: "NodeSubtype",
	PropertyIdNodeType: "NodeType",
	PropertyIdStructuredObjectList: "StructuredObjectList",
	PropertyIdSubordinateAnnotations: "SubordinateAnnotations",
	PropertyIdSubordinateList: "SubordinateList",
	PropertyIdActualShedLevel: "ActualShedLevel",
	PropertyIdDutyWindow: "DutyWindow",
	PropertyIdExpectedShedLevel: "ExpectedShedLevel",
	PropertyIdFullDutyBaseline: "FullDutyBaseline",
	PropertyIdUnassigned_216: "Unassigned_216",
	PropertyIdUnassigned_217: "Unassigned_217",
	PropertyIdRequestedShedLevel: "RequestedShedLevel",
	PropertyIdShedDuration: "ShedDuration",
	PropertyIdShedLevelDescriptions: "ShedLevelDescriptions",
	PropertyIdShedLevels: "ShedLevels",
	PropertyIdStateDescription: "StateDescription",
	PropertyIdUnassigned_223: "Unassigned_223",
	PropertyIdUnassigned_224: "Unassigned_224",
	PropertyIdUnassigned_225: "Unassigned_225",
	PropertyIdDoorAlarmState: "DoorAlarmState",
	PropertyIdDoorExtendedPulseTime: "DoorExtendedPulseTime",
	PropertyIdDoorMembers: "DoorMembers",
	PropertyIdDoorOpenTooLongTime: "DoorOpenTooLongTime",
	PropertyIdDoorPulseTime: "DoorPulseTime",
	PropertyIdDoorStatus: "DoorStatus",
	PropertyIdDoorUnlockDelayTime: "DoorUnlockDelayTime",
	PropertyIdLockStatus: "LockStatus",
	PropertyIdMaskedAlarmValues: "MaskedAlarmValues",
	PropertyIdSecuredStatus: "SecuredStatus",
	PropertyIdUnassigned_236: "Unassigned_236",
	PropertyIdUnassigned_237: "Unassigned_237",
	PropertyIdUnassigned_238: "Unassigned_238",
	PropertyIdUnassigned_239: "Unassigned_239",
	PropertyIdUnassigned_240: "Unassigned_240",
	PropertyIdUnassigned_241: "Unassigned_241",
	PropertyIdUnassigned_242: "Unassigned_242",
	PropertyIdUnassigned_243: "Unassigned_243",
	PropertyIdAbsenteeLimit: "AbsenteeLimit",
	PropertyIdAccessAlarmEvents: "AccessAlarmEvents",
	PropertyIdAccessDoors: "AccessDoors",
	PropertyIdAccessEvent: "AccessEvent",
	PropertyIdAccessEventAuthenticationFactor: "AccessEventAuthenticationFactor",
	PropertyIdAccessEventCredential: "AccessEventCredential",
	PropertyIdAccessEventTime: "AccessEventTime",
	PropertyIdAccessTransactionEvents: "AccessTransactionEvents",
	PropertyIdAccompaniment: "Accompaniment",
	PropertyIdAccompanimentTime: "AccompanimentTime",
	PropertyIdActivationTime: "ActivationTime",
	PropertyIdActiveAuthenticationPolicy: "ActiveAuthenticationPolicy",
	PropertyIdAssignedAccessRights: "AssignedAccessRights",
	PropertyIdAuthenticationFactors: "AuthenticationFactors",
	PropertyIdAuthenticationPolicyList: "AuthenticationPolicyList",
	PropertyIdAuthenticationPolicyNames: "AuthenticationPolicyNames",
	PropertyIdAuthenticationStatus: "AuthenticationStatus",
	PropertyIdAuthorizationMode: "AuthorizationMode",
	PropertyIdBelongsTo: "BelongsTo",
	PropertyIdCredentialDisable: "CredentialDisable",
	PropertyIdCredentialStatus: "CredentialStatus",
	PropertyIdCredentials: "Credentials",
	PropertyIdCredentialsInZone: "CredentialsInZone",
	PropertyIdDaysRemaining: "DaysRemaining",
	PropertyIdEntryPoints: "EntryPoints",
	PropertyIdExitPoints: "ExitPoints",
	PropertyIdExpiryTime: "ExpiryTime",
	PropertyIdExtendedTimeEnable: "ExtendedTimeEnable",
	PropertyIdFailedAttemptEvents: "FailedAttemptEvents",
	PropertyIdFailedAttempts: "FailedAttempts",
	PropertyIdFailedAttemptsTime: "FailedAttemptsTime",
	PropertyIdLastAccessEvent: "LastAccessEvent",
	PropertyIdLastAccessPoint: "LastAccessPoint",
	PropertyIdLastCredentialAdded: "LastCredentialAdded",
	PropertyIdLastCredentialAddedTime: "LastCredentialAddedTime",
	PropertyIdLastCredentialRemoved: "LastCredentialRemoved",
	PropertyIdLastCredentialRemovedTime: "LastCredentialRemovedTime",
	PropertyIdLastUseTime: "LastUseTime",
	PropertyIdLockout: "Lockout",
	PropertyIdLockoutRelinquishTime: "LockoutRelinquishTime",
	PropertyIdRemovedInVersion1Revision13_284: "RemovedInVersion1Revision13_284",
	PropertyIdMaxFailedAttempts: "MaxFailedAttempts",
	PropertyIdMembers: "Members",
	PropertyIdMusterPoint: "MusterPoint",
	PropertyIdNegativeAccessRules: "NegativeAccessRules",
	PropertyIdNumberOfAuthenticationPolicies: "NumberOfAuthenticationPolicies",
	PropertyIdOccupancyCount: "OccupancyCount",
	PropertyIdOccupancyCountAdjust: "OccupancyCountAdjust",
	PropertyIdOccupancyCountEnable: "OccupancyCountEnable",
	PropertyIdRemovedInVersion1Revision13_293: "RemovedInVersion1Revision13_293",
	PropertyIdOccupancyLowerLimit: "OccupancyLowerLimit",
	PropertyIdOccupancyLowerLimitEnforced: "OccupancyLowerLimitEnforced",
	PropertyIdOccupancyState: "OccupancyState",
	PropertyIdOccupancyUpperLimit: "OccupancyUpperLimit",
	PropertyIdOccupancyUpperLimitEnforced: "OccupancyUpperLimitEnforced",
	PropertyIdRemovedInVersion1Revision13_299: "RemovedInVersion1Revision13_299",
	PropertyIdPassbackMode: "PassbackMode",
	PropertyIdPassbackTimeout: "PassbackTimeout",
	PropertyIdPositiveAccessRules: "PositiveAccessRules",
	PropertyIdReasonForDisable: "ReasonForDisable",
	PropertyIdSupportedFormats: "SupportedFormats",
	PropertyIdSupportedFormatClasses: "SupportedFormatClasses",
	PropertyIdThreatAuthority: "ThreatAuthority",
	PropertyIdThreatLevel: "ThreatLevel",
	PropertyIdTraceFlag: "TraceFlag",
	PropertyIdTransactionNotificationClass: "TransactionNotificationClass",
	PropertyIdUserExternalIdentifier: "UserExternalIdentifier",
	PropertyIdUserInformationReference: "UserInformationReference",
	PropertyIdUnassigned_312: "Unassigned_312",
	PropertyIdUnassigned_313: "Unassigned_313",
	PropertyIdUnassigned_314: "Unassigned_314",
	PropertyIdUnassigned_315: "Unassigned_315",
	PropertyIdUnassigned_316: "Unassigned_316",
	PropertyIdUserName: "UserName",
	PropertyIdUserType: "UserType",
	PropertyIdUsesRemaining: "UsesRemaining",
	PropertyIdZoneFrom: "ZoneFrom",
	PropertyIdZoneTo: "ZoneTo",
	PropertyIdAccessEventTag: "AccessEventTag",
	PropertyIdGlobalIdentifier: "GlobalIdentifier",
	PropertyIdUnassigned_324: "Unassigned_324",
	PropertyIdUnassigned_325: "Unassigned_325",
	PropertyIdVerificationTime: "VerificationTime",
	PropertyIdBaseDeviceSecurityPolicy: "BaseDeviceSecurityPolicy",
	PropertyIdDistributionKeyRevision: "DistributionKeyRevision",
	PropertyIdDoNotHide: "DoNotHide",
	PropertyIdKeySets: "KeySets",
	PropertyIdLastKeyServer: "LastKeyServer",
	PropertyIdNetworkAccessSecurityPolicies: "NetworkAccessSecurityPolicies",
	PropertyIdPacketReorderTime: "PacketReorderTime",
	PropertyIdSecurityPduTimeout: "SecurityPduTimeout",
	PropertyIdSecurityTimeWindow: "SecurityTimeWindow",
	PropertyIdSupportedSecurityAlgorithms: "SupportedSecurityAlgorithms",
	PropertyIdUpdateKeySetTimeout: "UpdateKeySetTimeout",
	PropertyIdBackupAndRestoreState: "BackupAndRestoreState",
	PropertyIdBackupPreparationTime: "BackupPreparationTime",
	PropertyIdRestoreCompletionTime: "RestoreCompletionTime",
	PropertyIdRestorePreparationTime: "RestorePreparationTime",
	PropertyIdBitMask: "BitMask",
	PropertyIdBitText: "BitText",
	PropertyIdIsUtc: "IsUtc",
	PropertyIdGroupMembers: "GroupMembers",
	PropertyIdGroupMemberNames: "GroupMemberNames",
	PropertyIdMemberStatusFlags: "MemberStatusFlags",
	PropertyIdRequestedUpdateInterval: "RequestedUpdateInterval",
	PropertyIdCovuPeriod: "CovuPeriod",
	PropertyIdCovuRecipients: "CovuRecipients",
	PropertyIdEventMessageTexts: "EventMessageTexts",
	PropertyIdEventMessageTextsConfig: "EventMessageTextsConfig",
	PropertyIdEventDetectionEnable: "EventDetectionEnable",
	PropertyIdEventAlgorithmInhibit: "EventAlgorithmInhibit",
	PropertyIdEventAlgorithmInhibitRef: "EventAlgorithmInhibitRef",
	PropertyIdTimeDelayNormal: "TimeDelayNormal",
	PropertyIdReliabilityEvaluationInhibit: "ReliabilityEvaluationInhibit",
	PropertyIdFaultParameters: "FaultParameters",
	PropertyIdFaultType: "FaultType",
	PropertyIdLocalForwardingOnly: "LocalForwardingOnly",
	PropertyIdProcessIdentifierFilter: "ProcessIdentifierFilter",
	PropertyIdSubscribedRecipients: "SubscribedRecipients",
	PropertyIdPortFilter: "PortFilter",
	PropertyIdAuthorizationExemptions: "AuthorizationExemptions",
	PropertyIdAllowGroupDelayInhibit: "AllowGroupDelayInhibit",
	PropertyIdChannelNumber: "ChannelNumber",
	PropertyIdControlGroups: "ControlGroups",
	PropertyIdExecutionDelay: "ExecutionDelay",
	PropertyIdLastPriority: "LastPriority",
	PropertyIdWriteStatus: "WriteStatus",
	PropertyIdPropertyList: "PropertyList",
	PropertyIdSerialNumber: "SerialNumber",
	PropertyIdBlinkWarnEnable: "BlinkWarnEnable",
	PropertyIdDefaultFadeTime: "DefaultFadeTime",
	PropertyIdDefaultRampRate: "DefaultRampRate",
	PropertyIdDefaultStepIncrement: "DefaultStepIncrement",
	PropertyIdEgressTime: "EgressTime",
	PropertyIdInProgress: "InProgress",
	PropertyIdInstantaneousPower: "InstantaneousPower",
	PropertyIdLightingCommand: "LightingCommand",
	PropertyIdLightingCommandDefaultPriority: "LightingCommandDefaultPriority",
	PropertyIdMaxActualValue: "MaxActualValue",
	PropertyIdMinActualValue: "MinActualValue",
	PropertyIdPower: "Power",
	PropertyIdTransition: "Transition",
	PropertyIdEgressActive: "EgressActive",
}
