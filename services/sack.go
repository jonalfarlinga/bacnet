package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// SimpleACK is a BACnet message.
type SimpleACK struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

func NewSimpleACK(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) (*SimpleACK, uint8) {
	s := &SimpleACK{
		BVLC: bvlc,
		NPDU: npdu,
		// TODO: Consider to implement parameter struct to an argment of New functions.
		APDU: plumbing.NewAPDU(plumbing.SimpleAck, ServiceConfirmedReadProperty, nil),
	}
	s.SetLength()

	return s, s.APDU.Type
}

func (s *SimpleACK) UnmarshalBinary(b []byte) error {
	if l := len(b); l < s.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal SACK - marshal length %d binary length %d", s.MarshalLen(), l),
		)
	}

	var offset int = 0
	if err := s.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling SACK %v", s),
		)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling SACK %v", s),
		)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling SACK %v", s),
		)
	}

	return nil
}

func (s *SimpleACK) MarshalBinary() ([]byte, error) {
	b := make([]byte, s.MarshalLen())
	if err := s.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

func (s *SimpleACK) MarshalTo(b []byte) error {
	if len(b) < s.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal SACK - marshal length %d binary length %d", s.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := s.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling SACK")
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling SACK")
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling SACK")
	}

	return nil
}

func (s *SimpleACK) MarshalLen() int {
	l := s.BVLC.MarshalLen()
	l += s.NPDU.MarshalLen()
	l += s.APDU.MarshalLen()

	return l
}

func (s *SimpleACK) SetLength() {
	s.BVLC.Length = uint16(s.MarshalLen())
}

func (u *SimpleACK) GetService() uint8 {
	return u.APDU.Service
}

func (u *SimpleACK) GetType() uint8 {
	return u.APDU.Type
}
