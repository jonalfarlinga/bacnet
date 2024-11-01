package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
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
			fmt.Sprintf("Unsupported PresentValue %v type %T\n", value, value),
		)
	}

	objs[4] = objects.EncClosingTag(3)
	return objs
}

func NewComplexACK(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) (*ComplexACK, uint8) {
	c := &ComplexACK{
		BVLC: bvlc,
		NPDU: npdu,
		// TODO: Consider to implement parameter struct to an argment of New functions.
		APDU: plumbing.NewAPDU(plumbing.ComplexAck, ServiceConfirmedReadProperty, ComplexACKObjects(
			objects.ObjectTypeAnalogOutput, 1, objects.PropertyIdPresentValue, 0)),
	}
	c.SetLength()
	return c, c.APDU.Type
}

func (c *ComplexACK) UnmarshalBinary(b []byte) error {
	var offset int = 0
	if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("unmarshalling CACK %v", c),
		)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("unmarshalling CACK %v", c),
		)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("unmarshalling CACK %v", c),
		)
	}
	return nil
}

func (c *ComplexACK) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (c *ComplexACK) MarshalTo(b []byte) error {
	if len(b) < c.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal CACK %x - marshal length too short", b),
		)
	}
	var offset = 0
	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling CACK")
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling CACK")
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling CACK")
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
		return decCACK, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode CACK - objects count: %d", len(c.APDU.Objects)),
		)
	}

	objs := make([]*objects.Object, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCACK, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACK object at index %d is not Object type", i),
			)
		}
		if enc_obj.TagClass {
			switch enc_obj.TagNumber {
			case 0:
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				decCACK.ObjectType = objId.ObjectType
				decCACK.InstanceId = objId.InstanceNumber
			case 1:
				propId, err := objects.DecPropertyIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 1")
				}
				if propId == objects.PropertyIdLogBuffer {
					return decCACK, fmt.Errorf("PropertyIdLogBuffer")
				}
				decCACK.PropertyId = propId
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
