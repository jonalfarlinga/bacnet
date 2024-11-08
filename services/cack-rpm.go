package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// ComplexACKRPM is a BACnet message.
type ComplexACKRPM struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ComplexACKRPMDec struct {
	ObjectType uint16
	InstanceId uint32
	Tags       []*objects.Object
}

func (c *ComplexACK) DecodeRPM() (ComplexACKRPMDec, error) {
	decCACK := ComplexACKRPMDec{}
	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCACK, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACKRPM object at index %d is not Object type", i),
			)
		}

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 && enc_obj.Data == nil {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 && enc_obj.Data == nil {
			if len(context) == 0 {
				return decCACK, errors.Wrap(
					common.ErrInvalidObjectType,
					fmt.Sprintf("LogBufferCACK object at index %d has mismatched closing tag", i),
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
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				decCACK.ObjectType = objId.ObjectType
				decCACK.InstanceId = objId.InstanceNumber
			case combine(1, 2):
				propId, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 1")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 2,
					TagClass:  true,
					Value:     propId,
					Data:      enc_obj.Data,
					Length:    enc_obj.Length,
				})
			case combine(4, 0):
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Value:     objId,
					Length:    uint8(obj.MarshalLen()),
				})
			case combine(4, 1):
				value, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 1")
				}
				propId := uint16(value)
				objs = append(objs, &objects.Object{
					TagNumber: 1,
					TagClass:  true,
					Value:     propId,
					Length:    uint8(obj.MarshalLen()),
				})
			case combine(4, 3):
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 3,
					TagClass:  true,
					Value:     objId,
					Length:    uint8(obj.MarshalLen()),
				})
			default:
				log.Println("context", context, "TagNumber", enc_obj.TagNumber)
			}
		} else {
			tag, err := decodeTags(enc_obj, &obj)
			if err != nil {
				return decCACK, errors.Wrap(err, "decode Application Tag")
			}
			objs = append(objs, tag)
		}
	}
	decCACK.Tags = objs

	return decCACK, nil
}
