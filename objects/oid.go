package objects

import (
	"encoding/binary"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/pkg/errors"
)

type ObjectIdentifier struct {
	ObjectType     uint16
	InstanceNumber uint32
}

func DecObjectIdentifier(rawPayload APDUPayload) (ObjectIdentifier, error) {
	decObjectId := ObjectIdentifier{}

	rawObject, ok := rawPayload.(*Object)
	if !ok {
		return decObjectId, errors.Wrap(common.ErrWrongPayload, "failed to decode ObjectID")
	}

	switch rawObject.TagClass {
	case true:
		if rawObject.Length != 4 {
			return decObjectId, errors.Wrap(common.ErrWrongStructure, "failed to decode ObjectID - wrong binary length")
		}
	case false:
		if rawObject.Length != 4 || rawObject.TagNumber != TagBACnetObjectIdentifier {
			return decObjectId, errors.Wrap(common.ErrWrongStructure, "failed to decode ObjectID - wrong tag number")
		}
	}

	joinedData := binary.BigEndian.Uint32(rawObject.Data)
	decObjectId.ObjectType = uint16((joinedData >> 22) & 0x3FF)
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
