package objects

import "github.com/jonalfarlinga/bacnet/common"

type LogStatus struct {
	LogDisabled  bool
	BufferPurged bool
	LogFull      bool
}

func DecLogStatus(obj APDUPayload) (*LogStatus, error) {
	enc_obj, ok := obj.(*Object)
	if !ok {
		return nil, common.ErrInvalidObjectType
	}
	return &LogStatus{
		LogDisabled:  enc_obj.Data[1]&0x80 == 0x80,
		BufferPurged: enc_obj.Data[1]&0x40 == 0x40,
		LogFull:      enc_obj.Data[1]&0x20 == 0x20,
	}, nil
}
