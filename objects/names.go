package objects

import "fmt"

// ObjectTypeMap names the standard object types.
var ObjectTypeMap = map[uint16]string{
	0: "analog-input", 1: "analog-output", 2: "analog-value", 3: "binary-input", 4: "binary-output",
	5: "binary-value", 6: "calendar", 7: "command", 8: "device", 9: "event-enrollment", 10: "file",
	11: "group", 12: "loop", 13: "multi-state-input", 14: "multi-state-output", 15: "notification-class",
	16: "program", 17: "schedule", 18: "averaging", 19: "multi-state-value", 20: "trend-log",
	21: "life-safety-point", 22: "life-safety-zone", 23: "accumulator", 24: "pulse-converter",
	25: "event-log", 26: "global-group", 27: "trend-log-multiple", 28: "load-control", 29: "structured-view",
	30: "access-door", 31: "timer", 32: "access-credential", 33: "access-point", 34: "access-rights",
	35: "access-user", 36: "access-zone", 37: "credential-data-input", 38: "network-security",
	39: "bitstring-value", 40: "characterstring-value", 41: "date-pattern-value", 42: "date-value",
	43: "datetime-pattern-value", 44: "datetime-value", 45: "integer-value", 46: "large-analog-value",
	47: "octetstring-value", 48: "positive-integer-value", 49: "time-pattern-value", 50: "time-value",
	51: "notification-forwarder", 52: "alert-enrollment", 53: "channel", 54: "lighting-output",
	55: "binary-lighting-output", 56: "network-port", 57: "elevator-group", 58: "escalator", 59: "lift",
}

// ObjectTypeName names an object type, for display. Types from 128 up are
// a vendor's own.
func ObjectTypeName(t uint16) string {
	if name, ok := ObjectTypeMap[t]; ok {
		return name
	}
	if t >= 128 {
		return fmt.Sprintf("proprietary(%d)", t)
	}
	return fmt.Sprintf("unknown(%d)", t)
}

// PropertyName names a property identifier, for display. Identifiers from
// 512 up are a vendor's own.
func PropertyName(id uint32) string {
	if id <= 0xFFFF {
		if name, ok := PropertyMap[uint16(id)]; ok {
			return name
		}
	}
	if id >= 512 {
		return fmt.Sprintf("proprietary(%d)", id)
	}
	return fmt.Sprintf("unknown(%d)", id)
}

// UnitName names an engineering unit, for display. Units from 256 up are a
// vendor's own.
func UnitName(u uint16) string {
	if name, ok := UnitMap[u]; ok {
		return name
	}
	if u >= 256 {
		return fmt.Sprintf("proprietary(%d)", u)
	}
	return fmt.Sprintf("unknown(%d)", u)
}
