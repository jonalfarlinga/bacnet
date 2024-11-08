package objects

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/pkg/errors"
)

func DecPropertyIdentifier(rawPayload APDUPayload) (uint16, error) {
	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return 0, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("failed to decode PropertyID - %v", rawPayload),
		)
	}
	if rawObject.Length == 1 {
		return uint16(rawObject.Data[0]), nil
	}
	return binary.BigEndian.Uint16(rawObject.Data), nil
}

// func EncPropertyIdentifier(contextTag bool, tagN uint8, propId uint16) *Object {
// 	newObj := Object{}
// 	var data []byte
// 	if propId < 256 {
// 		data = []byte{uint8(propId)}
// 	} else {
// 		data = []byte{uint8(propId >> 8), uint8(propId)}
// 	}
// 	newObj.TagNumber = tagN
// 	newObj.TagClass = contextTag
// 	newObj.Data = data
// 	newObj.Length = uint8(len(data))

// 	return &newObj
// }
