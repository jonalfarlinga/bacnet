package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
)

type ConfirmedReadPropMultDec struct {
	ObjectType  uint16
	InstanceNum uint32
	Tags        []*objects.Object
}

func (c *ConfirmedReadProperty) DecodeRPM() (ConfirmedReadPropMultDec, error) {
	decRPM := ConfirmedReadPropMultDec{}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decRPM, fmt.Errorf(
				"ComplexACKRPM object at index %d is not Object type: %v",
				i, common.ErrInvalidObjectType,
			)
		}

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 && enc_obj.Data == nil {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 && enc_obj.Data == nil {
			if len(context) == 0 {
				return decRPM, fmt.Errorf(
					"LogBufferCACK object at index %d has mismatched closing tag: %v",
					i, common.ErrInvalidObjectType,
				)
			}
			context = context[:len(context)-1]
			continue
		}

		if enc_obj.TagClass {
			c := combine(context[len(context)-1], enc_obj.TagNumber)
			switch c {
			case combine(8, 0):
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decRPM, fmt.Errorf("decode Context object case 0: %v", err)
				}
				decRPM.ObjectType = objId.ObjectType
				decRPM.InstanceNum = objId.InstanceNumber
			case combine(1, 0):
				propId, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decRPM, fmt.Errorf("decode Context object case 0: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     propId,
				})
			default:
				log.Printf("Unknown Context object: context %v tag class %t tag number %d\n", context, enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			tag, err := decodeAppTags(enc_obj, &obj)
			if err != nil {
				return decRPM, fmt.Errorf("decode Application Tag: %v", err)
			}
			objs = append(objs, tag)
		}
	}
	decRPM.Tags = objs

	return decRPM, nil
}
