package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// UnconfirmedWhoIs is a BACnet message.
type UnconfirmedWhoIs struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type UnconfirmedWhoIsDec struct {
	Tags []*objects.Object
}

// NewUnconfirmedWhoIs creates a UnconfirmedWhoIs.
func NewUnconfirmedWhoIs(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *UnconfirmedWhoIs {
	u := &UnconfirmedWhoIs{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.UnConfirmedReq, ServiceUnconfirmedWhoIs, nil),
	}
	u.SetLength()
	return u
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedWhoIs frame.
func (u *UnconfirmedWhoIs) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal UnconfirmedWhoIs - marshal length %d binary length %d: %v",
			u.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedWhoIs %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedWhoIs %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedWhoIs %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedWhoIs instance.
func (u *UnconfirmedWhoIs) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *UnconfirmedWhoIs) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal UnconfirmedWhoIs - marshal length %d binary length %d: %v",
			u.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoIs: %v", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoIs: %v", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoIs: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedWhoIs.
func (u *UnconfirmedWhoIs) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()

	return l
}

func (u *UnconfirmedWhoIs) Decode() (UnconfirmedWhoIsDec, error) {
	decWhois := UnconfirmedWhoIsDec{}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range u.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decWhois, fmt.Errorf(
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
				return decWhois, fmt.Errorf(
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
				lowRange, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decWhois, fmt.Errorf("decode Context object case 0: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Value:     lowRange,
					Length:    uint8(obj.MarshalLen()),
				})
			case combine(8, 1):
				highRange, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decWhois, fmt.Errorf("decode Context object case 1: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 1,
					TagClass:  true,
					Value:     highRange,
					Length:    uint8(obj.MarshalLen()),
				})
			default:
				log.Printf("Unknown Context object: context %v tag class %t tag number %d\n", context, enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			tag, err := decodeAppTags(enc_obj, &obj)
			if err != nil {
				return decWhois, fmt.Errorf("decode Application Tag: %v", err)
			}
			objs = append(objs, tag)
		}
	}
	decWhois.Tags = objs

	return decWhois, nil
}

// SetLength sets the length in Length field.
func (u *UnconfirmedWhoIs) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *UnconfirmedWhoIs) GetService() uint8 {
	return u.APDU.Service
}

func (u *UnconfirmedWhoIs) GetType() uint8 {
	return u.APDU.Type
}
