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
	Tags       []*objects.Object
}

func ComplexACKRPMObjects(objectType uint16, instN uint32, propertyId uint16, value interface{}) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 5)
	// Not Implemented for ComplexACKRPM
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
	// Use ComplexACK, then convert using NewComplexACKRPM()
	return fmt.Errorf("UnmarshalBinary not implemented for ComplexACKRPM")
}

func (c *ComplexACKRPM) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (c *ComplexACKRPM) MarshalTo(b []byte) error {
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
				propId, err := objects.DecPropertyIdentifier(obj)
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
