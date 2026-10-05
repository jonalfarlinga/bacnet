package objects

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

func ContextTag(tagNumber uint8, o *Object) *Object {
	o.TagClass = true
	o.TagNumber = tagNumber
	return o
}

func EncContextBool(tagNumber uint8, value bool) *Object {
	obj := &Object{
		TagClass:  true,
		TagNumber: tagNumber,
		Length:    1,
		Data:      []byte{byte(common.BoolToInt(value))},
	}
	return obj
}

func DecContextBool(rawPayload APDUPayload) (bool, error) {
	encObj, ok := rawPayload.(*Object)
	if !ok {
		return false, common.ErrInvalidObjectType
	}
	if len(encObj.Data) != 1 {
		return false, fmt.Errorf(
			"failed to decode context Boolean - data length %d: %w", len(encObj.Data), common.ErrInvalidData,
		)
	}
	return common.IntToBool(int(encObj.Data[0] & 0x01)), nil
}
