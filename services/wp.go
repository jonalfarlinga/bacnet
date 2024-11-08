package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedIAm is a BACnet message.
type ConfirmedWriteProperty struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ConfirmedWritePropertyDec struct {
	ObjectType  uint16
	InstanceNum uint32
	PropertyId  uint16
	Value       float32
	Priority    uint8
}

func ConfirmedWritePropertyObjects(objectType uint16, instN uint32, propertyId uint16, data interface{}) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 6)

	objs[0] = objects.EncObjectIdentifier(true, 0, objectType, instN)
	objs[1] = objects.EncPropertyIdentifier(true, 1, propertyId)
	objs[2] = objects.EncOpeningTag(3)
	var obj *objects.Object
	switch data := data.(type) {
	case float32:
		obj = objects.EncReal(data)
	case uint:
		obj = objects.EncUnsignedInteger(data)
	case string:
		obj = objects.EncString(data)
	}
	obj.TagClass = true
	obj.TagNumber = 3
	objs[3] = obj
	objs[4] = objects.EncClosingTag(3)
	objs[5] = objects.EncPriority(true, 4, 16)

	return objs
}

func NewConfirmedWriteProperty(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *ConfirmedWriteProperty {
	c := &ConfirmedWriteProperty{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ConfirmedReq, ServiceConfirmedWriteProperty, nil),
	}
	c.SetLength()

	return c
}

func (c *ConfirmedWriteProperty) UnmarshalBinary(b []byte) error {
	if l := len(b); l < c.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal ConfirmedWP - marshal length %d binary length %d", c.MarshalLen(), l),
		)
	}

	var offset int = 0
	if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedWP %v", c),
		)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedWP %v", c),
		)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedWP %v", c),
		)
	}

	return nil
}

func (c *ConfirmedWriteProperty) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (c *ConfirmedWriteProperty) MarshalTo(b []byte) error {
	if len(b) < c.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal ConfirmedWP - marshal length %d binary length %d", c.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedWP")
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedWP")
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedWP")
	}

	return nil
}

func (c *ConfirmedWriteProperty) MarshalLen() int {
	l := c.BVLC.MarshalLen()
	l += c.NPDU.MarshalLen()
	l += c.APDU.MarshalLen()

	return l
}

func (c *ConfirmedWriteProperty) SetLength() {
	c.BVLC.Length = uint16(c.MarshalLen())
}

func (c *ConfirmedWriteProperty) Decode() (ConfirmedWritePropertyDec, error) {
	decCWP := ConfirmedWritePropertyDec{}

	if len(c.APDU.Objects) != 5 {
		return decCWP, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode ConfirmedWP - object count %d", len(c.APDU.Objects)),
		)
	}

	for i, obj := range c.APDU.Objects {
		switch i {
		case 0:
			objId, err := objects.DecObjectIdentifier(obj)
			if err != nil {
				return decCWP, errors.Wrap(err, "decoding ConfirmedWP")
			}
			decCWP.ObjectType = objId.ObjectType
			decCWP.InstanceNum = objId.InstanceNumber
		case 1:
			propId, err := objects.DecPropertyIdentifier(obj)
			if err != nil {
				return decCWP, errors.Wrap(err, "decoding ConfirmedWP")
			}
			decCWP.PropertyId = propId
		case 2:
			value, err := objects.DecReal(obj)
			if err != nil {
				return decCWP, errors.Wrap(err, "decoding ConfirmedWP")
			}
			decCWP.Value = value
		case 4:
			priority, err := objects.DecPriority(obj)
			if err != nil {
				return decCWP, errors.Wrap(err, "decoding ConfirmedWP")
			}
			decCWP.Priority = priority
		}
	}

	return decCWP, nil
}

func (u *ConfirmedWriteProperty) GetService() uint8 {
	return u.APDU.Service
}

func (u *ConfirmedWriteProperty) GetType() uint8 {
	return u.APDU.Type
}
