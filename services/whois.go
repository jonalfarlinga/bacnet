package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
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
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal UnconfirmedWhoIs - marshal length %d binary length %d", u.MarshalLen(), l),
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedWhoIs %v", u),
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedWhoIs %v", u),
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedWhoIs %v", u),
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedWhoIs instance.
func (u *UnconfirmedWhoIs) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *UnconfirmedWhoIs) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal UnconfirmedWhoIs - marshal length %d binary length %d", u.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedWhoIs")
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedWhoIs")
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedWhoIs")
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
			return decWhois, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACK object at index %d is not Object type", i),
			)
		}

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 {
			if len(context) == 0 {
				return decWhois, errors.Wrap(
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
				lowRange, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decWhois, errors.Wrap(err, "decode Context object case 0")
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
					return decWhois, errors.Wrap(err, "decode Context object case 1")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 1,
					TagClass:  true,
					Value:     highRange,
					Length:    uint8(obj.MarshalLen()),
				})
			default:
				log.Printf("Unknown tag %+v\n", enc_obj)
			}
		} else {
			tag, err := decodeTags(enc_obj, &obj)
			if err != nil {
				return decWhois, errors.Wrap(err, "decode Application Tag")
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
