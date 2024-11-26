package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// SimpleACK is a BACnet message.
type SimpleACK struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

func NewSimpleACK(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *SimpleACK {
	s := &SimpleACK{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.SimpleAck, ServiceConfirmedReadProperty, nil),
	}
	s.SetLength()

	return s
}

func (s *SimpleACK) UnmarshalBinary(b []byte) error {
	if l := len(b); l < s.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal SACK - marshal length %d binary length %d: %v",
			s.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := s.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling SACK %+v: %v", s, common.ErrTooShortToParse,
		)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling SACK %+v: %v", s, common.ErrTooShortToParse,
		)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling SACK %+v: %v", s, common.ErrTooShortToParse,
		)
	}

	return nil
}

func (s *SimpleACK) MarshalBinary() ([]byte, error) {
	b := make([]byte, s.MarshalLen())
	if err := s.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (s *SimpleACK) MarshalTo(b []byte) error {
	if len(b) < s.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal SACK - marshal length %d binary length %d: %v",
			s.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := s.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SACK: %v", err)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SACK: %v", err)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SACK: %v", err)
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
