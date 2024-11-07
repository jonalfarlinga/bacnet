package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedWhoIs is a BACnet message.
type UnconfirmedWhoIs struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
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
