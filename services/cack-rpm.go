package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedIAm is a BACnet message.
type ComplexACKRPM struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ComplexACKRPMDec struct {
	ObjectType uint16
	InstanceId uint32
	Tags       []*objects.AppTag
}

func ComplexACKRPMObjects(objectType uint16, instN uint32, propertyId uint16, value interface{}) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 5)
	objs[0] = objects.EncObjectIdentifier(true, 0, objectType, instN)
	objs[1] = objects.EncPropertyIdentifier(true, 1, propertyId)
	objs[2] = objects.EncOpeningTag(3)

	switch v := value.(type) {
	case int:
		objs[3] = objects.EncReal(float32(v))
	case uint8:
		objs[3] = objects.EncUnsignedInteger8(v)
	case uint16:
		objs[3] = objects.EncUnsignedInteger16(v)
	case float32:
		objs[3] = objects.EncReal(v)
	case string:
		objs[3] = objects.EncString(v)
	default:
		panic(
			fmt.Sprintf("Unsupported PresentValue type %T", value),
		)
	}

	objs[4] = objects.EncClosingTag(3)
	return objs
}

func NewComplexACKRPM(cack *ComplexACK) *ComplexACKRPM {
	c := &ComplexACKRPM{
		BVLC: cack.BVLC,
		NPDU: cack.NPDU,
		APDU: cack.APDU,
	}
	c.SetLength()
	return c
}

func (c *ComplexACKRPM) UnmarshalBinary(b []byte) error {
	// if l := len(b); l < c.MarshalLen()-2 {
	// 	return errors.Wrap(
	// 		common.ErrTooShortToParse,
	// 		fmt.Sprintf("failed to unmarshal CACK %v - marshal length %d binary length %d", c, c.MarshalLen(), l),
	// 	)
	// }

	// var offset int = 0
	// if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
	// 	return errors.Wrap(
	// 		err,
	// 		fmt.Sprintf("unmarshalling CACK %v", c),
	// 	)
	// }
	// offset += c.BVLC.MarshalLen()

	// if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
	// 	return errors.Wrap(
	// 		err,
	// 		fmt.Sprintf("unmarshalling CACK %v", c),
	// 	)
	// }
	// offset += c.NPDU.MarshalLen()

	// if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
	// 	return errors.Wrap(
	// 		err,
	// 		fmt.Sprintf("unmarshalling CACK %v", c),
	// 	)
	// }

	return nil
}

func (c *ComplexACKRPM) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (c *ComplexACKRPM) MarshalTo(b []byte) error {
	// if len(b) < c.MarshalLen() {
	// 	return errors.Wrap(
	// 		common.ErrTooShortToMarshalBinary,
	// 		fmt.Sprintf("failed to marshal CACK %x - marshal length too short", b),
	// 	)
	// }
	// var offset = 0
	// if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
	// 	return errors.Wrap(err, "marshalling CACK")
	// }
	// offset += c.BVLC.MarshalLen()

	// if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
	// 	return errors.Wrap(err, "marshalling CACK")
	// }
	// offset += c.NPDU.MarshalLen()

	// if err := c.APDU.MarshalTo(b[offset:]); err != nil {
	// 	return errors.Wrap(err, "marshalling CACK")
	// }

	return nil
}

func (c *ComplexACKRPM) MarshalLen() int {
	l := c.BVLC.MarshalLen()
	l += c.NPDU.MarshalLen()
	l += c.APDU.MarshalLen()

	return l
}

func (u *ComplexACKRPM) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (c *ComplexACKRPM) Decode() (ComplexACKRPMDec, error) {
	decCACK := ComplexACKRPMDec{}

	if len(c.APDU.Objects) < 3 {
		return decCACK, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode CACK - objects count: %d", len(c.APDU.Objects)),
		)
	}

	context := []uint8{8}
	objs := make([]*objects.AppTag, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCACK, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACKRPM object at index %d is not Object type", i),
			)
		}
		// log.Printf(
		// 	"\tObject i %d tagnum %d tagclass %v data %x\n",
		// 	i, enc_obj.TagNumber, enc_obj.TagClass, enc_obj.Data,
		// )

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

		// log.Printf("%+v", objs)
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
				propId, err := objects.DecPropertyIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 1")
				}
				if propId == objects.PropertyIdLogBuffer {
					return decCACK, fmt.Errorf("PropertyIdLogBuffer")
				}
				objs = append(objs, &objects.AppTag{
					TagNumber: 2,
					TagClass:  true,
					Value:     propId,
					Length:    enc_obj.Length,
				})
			}
		} else {
			// log.Println("TagNumber", enc_obj.TagNumber)
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
