package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedReadRange is a BACnet message.
type ConfirmedReadRange struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ConfirmedReadRangeDec struct {
	ObjectType uint16
	InstanceId uint32
	PropertyId uint8
	Tags       []*objects.AppTag
}

func ConfirmedReadRangeObjects(objectType uint16, instN uint32, property uint8, index uint16, count int32) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 6)

	objs[0] = objects.EncObjectIdentifier(true, 0, objectType, instN)
	switch property {
	case objects.PropertyIdPresentValue:
		objs[1] = objects.EncPropertyIdentifier(true, 1, objects.PropertyIdPresentValue)
	case objects.PropertyIdLogBuffer:
		objs[1] = objects.EncPropertyIdentifier(true, 1, objects.PropertyIdLogBuffer)
		objs[2] = objects.EncOpeningTag(3)
		objs[3] = objects.EncUnsignedInteger(uint(index))
		objs[4] = objects.EncSignedInteger(int(count))
		objs[5] = objects.EncClosingTag(3)
	default:
		panic("Not Implemented")
	}

	return objs
}

func NewConfirmedReadRange(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) (*ConfirmedReadRange, uint8) {
	c := &ConfirmedReadRange{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ConfirmedReq, ServiceConfirmedReadRange, ConfirmedReadRangeObjects(
			0, 0, 131, 0, 0)),
	}
	c.SetLength()

	return c, c.APDU.Type
}

func (c *ConfirmedReadRange) MarshalLen() int {
	l := c.BVLC.MarshalLen()
	fmt.Println(l)
	l += c.NPDU.MarshalLen()
	fmt.Println(c.NPDU.MarshalLen())
	l += c.APDU.MarshalLen()
	fmt.Println(c.APDU.MarshalLen())

	return l
}

func (c *ConfirmedReadRange) SetLength() {
	c.BVLC.Length = uint16(c.MarshalLen())
}

func (c *ConfirmedReadRange) UnmarshalBinary(b []byte) error {
	if l := len(b); l < c.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal ConfirmedRP - marshal length %d binary length %d", c.MarshalLen(), l),
		)
	}

	var offset int = 0
	if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedRP %v", c),
		)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedRP %v", c),
		)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedRP %v", c),
		)
	}

	return nil
}

func (c *ConfirmedReadRange) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	fmt.Println(c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (c *ConfirmedReadRange) MarshalTo(b []byte) error {
	if len(b) < c.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal ConfirmedRP - marshal length %d binary length %d", c.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedRP")
	}
	fmt.Println(b[:offset+1])
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedRP")
	}
	fmt.Println(b[:offset+1])
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "failed to marshal ConfirmedRP")
	}
	fmt.Println(b[:offset+1])

	return nil
}

func (c *ConfirmedReadRange) Decode() (ConfirmedReadRangeDec, error) {
	decCRP := ConfirmedReadRangeDec{}

	if len(c.APDU.Objects) < 2 {
		return decCRP, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode ConfirmedRP - object count %d", len(c.APDU.Objects)),
		)
	}

	objs := make([]*objects.AppTag, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCRP, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACK object at index %d is not Object type", i),
			)
		}
		// log.Printf(
		// 	"\tObject i %d tagnum %d tagclass %v data %x\n",
		// 	i, enc_obj.TagNumber, enc_obj.TagClass, enc_obj.Data,
		// )
		if enc_obj.TagClass {
			switch enc_obj.TagNumber {
			case 0:
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Context object case 0")
				}
				decCRP.ObjectType = objId.ObjectType
				decCRP.InstanceId = objId.InstanceNumber
			case 1:
				propId, err := objects.DecPropertyIdentifier(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Context object case 1")
				}
				decCRP.PropertyId = propId
			}
		} else {
			// log.Println("TagNumber", enc_obj.TagNumber)
			switch enc_obj.TagNumber {
			case objects.TagUnsignedInteger:
				value, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Application object case 0")
				}
				objs = append(objs, &objects.AppTag{
					TagNumber: objects.TagUnsignedInteger,
					TagClass:  false,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			case objects.TagReal:
				value, err := objects.DecReal(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Application object case 4")
				}
				objs = append(objs, &objects.AppTag{
					TagNumber: objects.TagReal,
					TagClass:  false,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			case objects.TagCharacterString:
				value, err := objects.DecString(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Application object case 7")
				}
				objs = append(objs, &objects.AppTag{
					TagNumber: objects.TagCharacterString,
					TagClass:  false,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			case objects.TagBACnetObjectIdentifier:
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCRP, errors.Wrap(err, "decode Context object case 0")
				}
				objs = append(objs, &objects.AppTag{
					TagNumber: objects.TagBACnetObjectIdentifier,
					TagClass:  false,
					Length:    uint8(obj.MarshalLen()),
					Value:     fmt.Sprintf("%d:%d", objId.ObjectType, objId.InstanceNumber),
				})
			default:
				log.Println("\tnot encoded")
			}
		}
		decCRP.Tags = objs
	}

	return decCRP, nil
}
