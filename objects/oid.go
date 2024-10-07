package objects

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/ulbios/bacnet/common"
)

type ObjectIdentifier struct {
	ObjectType     uint16
	InstanceNumber uint32
}

func DecObjectIdentifier(rawPayload APDUPayload) (ObjectIdentifier, error) {
	decObjectId := ObjectIdentifier{}

	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return decObjectId, errors.Wrap(
			common.ErrWrongPayload,
			fmt.Sprintf("DecObjectIdentifier not ok: %T", rawPayload),
		)
	}

	switch rawObject.TagClass {
	case true:
		if rawObject.Length != 4 {
			return decObjectId, errors.Wrap(
				common.ErrWrongStructure,
				fmt.Sprintf("DecObjectIdentifier length: %d, tag class: %v", rawObject.Length, rawObject.TagClass),
			)
		}
	case false:
		if rawObject.Length != 4 || rawObject.TagNumber != TagBACnetObjectIdentifier {
			return decObjectId, errors.Wrap(
				common.ErrWrongStructure,
				fmt.Sprintf("DecObjectIdentifier length: %d, tag class: %v", rawObject.Length, rawObject.TagClass),
			)
		}
	}

	joinedData := binary.BigEndian.Uint32(rawObject.Data)
	decObjectId.ObjectType = uint16(joinedData & (uint32(0xFFC) << 20) >> 20)
	decObjectId.InstanceNumber = uint32(joinedData & 0x3FFFFF)

	return decObjectId, nil
}

func EncObjectIdentifier(contextTag bool, tagN uint8, objType uint16, instN uint32) *Object {
	newObj := Object{}
	data := make([]byte, 4)

	binary.BigEndian.PutUint32(data[:], uint32(objType)<<22|instN)

	newObj.TagNumber = tagN
	newObj.TagClass = contextTag
	newObj.Data = data
	newObj.Length = uint8(len(data))

	return &newObj
}
