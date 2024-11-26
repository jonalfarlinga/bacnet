package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// UnconfirmedIAm is a BACnet message.
type ComplexACK struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ComplexACKDec struct {
	ObjectType uint16
	InstanceId uint32
	PropertyId uint16
	Tags       []*objects.Object
}

func ComplexACKObjects(objectType uint16, instN uint32, propertyId uint16, value interface{}) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 5)
	objs[0] = objects.EncObjectIdentifier(true, 0, objectType, instN)
	objs[1] = objects.ContextTag(1, objects.EncUnsignedInteger(uint(propertyId)))
	objs[2] = objects.EncOpeningTag(3)

	switch v := value.(type) {
	case int:
		objs[3] = objects.EncReal(float32(v))
	case uint8:
		objs[3] = objects.EncUnsignedInteger(uint(v))
	case uint16:
		objs[3] = objects.EncUnsignedInteger(uint(v))
	case float32:
		objs[3] = objects.EncReal(v)
	case string:
		objs[3] = objects.EncString(v)
	default:
		panic(
			fmt.Sprintf("Unsupported PresentValue %v type %T\n", value, value),
		)
	}

	objs[4] = objects.EncClosingTag(3)
	return objs
}

func NewComplexACK(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *ComplexACK {
	c := &ComplexACK{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ComplexAck, ServiceConfirmedReadProperty, ComplexACKObjects(
			objects.ObjectTypeAnalogOutput, 1, objects.PropertyIdPresentValue, 0)),
	}
	c.SetLength()
	return c
}

func (c *ComplexACK) UnmarshalBinary(b []byte) error {
	var offset int = 0
	if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling CACK %+v: %v", c, err)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling CACK %+v: %v", c, err)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling CACK %+v: %v", c, err)
	}
	return nil
}

func (c *ComplexACK) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (c *ComplexACK) MarshalTo(b []byte) error {
	if len(b) < c.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal CACK - marshal length too short %x: %v",
			b, common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling CACK: %v", err)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling CACK: %v", err)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling CACK: %v", err)
	}
	return nil
}

func (c *ComplexACK) MarshalLen() int {
	l := c.BVLC.MarshalLen()
	l += c.NPDU.MarshalLen()
	l += c.APDU.MarshalLen()

	return l
}

func (u *ComplexACK) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (c *ComplexACK) Decode() (ComplexACKDec, error) {
	decCACK := ComplexACKDec{}

	if len(c.APDU.Objects) < 3 {
		return decCACK, fmt.Errorf(
			"failed to decode CACK - objects count %d: %v",
			len(c.APDU.Objects),
			common.ErrWrongObjectCount,
		)
	}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCACK, fmt.Errorf(
				"ComplexACK object at index %d is not Object type: %v",
				i, common.ErrInvalidObjectType,
			)
		}

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 {
			if len(context) == 0 {
				return decCACK, fmt.Errorf(
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
					return decCACK, fmt.Errorf("decode Context object case 0: %v", err)
				}
				decCACK.ObjectType = objId.ObjectType
				decCACK.InstanceId = objId.InstanceNumber
			case combine(8, 1):
				value, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCACK, fmt.Errorf("decode Context object case 1: %v", err)
				}
				propId := uint16(value)
				if propId == objects.PropertyIdLogBuffer {
					return decCACK, fmt.Errorf("PropertyIdLogBuffer should use ComplexACK.DecodeRR()")
				}
				decCACK.PropertyId = propId
			case combine(3, 0):
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, fmt.Errorf("decode Context object case 0: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     objId,
				})
			case combine(3, 1):
				propId, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCACK, fmt.Errorf("decode Context object case 1: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 1,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     propId,
				})
			case combine(3, 3):
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, fmt.Errorf("decode Context object case 0: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 3,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     objId,
				})
			default:
				log.Printf("Unknown Context object: context %v tag class %t tag number %d\n", context, enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			tag, err := decodeAppTags(enc_obj, &obj)
			if err != nil {
				return decCACK, fmt.Errorf("decode Application Tag: %v", err)
			}
			objs = append(objs, tag)
		}
	}
	decCACK.Tags = objs

	return decCACK, nil
}

func (u *ComplexACK) GetService() uint8 {
	return u.APDU.Service
}

func (u *ComplexACK) GetType() uint8 {
	return u.APDU.Type
}
