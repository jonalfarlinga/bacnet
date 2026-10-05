package objects

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

type LogStatus struct {
	LogDisabled  bool
	BufferPurged bool
	LogFull      bool
}

// Log Status tags for TrendLogs
func DecLogStatus(obj APDUPayload) (*LogStatus, error) {
	enc_obj, ok := obj.(*Object)
	if !ok {
		return nil, common.ErrInvalidObjectType
	}
	if len(enc_obj.Data) < 2 {
		return nil, fmt.Errorf(
			"failed to decode LogStatus - data length %d: %w", len(enc_obj.Data), common.ErrTooShortToParse,
		)
	}
	return &LogStatus{
		LogDisabled:  enc_obj.Data[1]&0x80 == 0x80,
		BufferPurged: enc_obj.Data[1]&0x40 == 0x40,
		LogFull:      enc_obj.Data[1]&0x20 == 0x20,
	}, nil
}
