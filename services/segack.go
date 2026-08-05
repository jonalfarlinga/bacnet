package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// SegmentACK is a BACnet Segment-ACK-PDU: sent by the receiving side of a
// segmented transfer to confirm a segment (or window of segments) and let
// the sender continue with the next window.
type SegmentACK struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

func NewSegmentACK(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *SegmentACK {
	s := &SegmentACK{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.SegmentAck, 0, nil),
	}
	s.SetLength()

	return s
}

func (s *SegmentACK) UnmarshalBinary(b []byte) error {
	if l := len(b); l < s.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal SegmentACK - marshal length %d binary length %d: %v",
			s.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := s.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentACK: %v", common.ErrTooShortToParse)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentACK: %v", common.ErrTooShortToParse)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentACK: %v", common.ErrTooShortToParse)
	}

	return nil
}

func (s *SegmentACK) MarshalBinary() ([]byte, error) {
	b := make([]byte, s.MarshalLen())
	if err := s.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (s *SegmentACK) MarshalTo(b []byte) error {
	if len(b) < s.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal SegmentACK - marshal length %d binary length %d: %v",
			s.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := s.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SegmentACK: %v", err)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SegmentACK: %v", err)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling SegmentACK: %v", err)
	}

	return nil
}

func (s *SegmentACK) MarshalLen() int {
	l := s.BVLC.MarshalLen()
	l += s.NPDU.MarshalLen()
	l += s.APDU.MarshalLen()

	return l
}

func (s *SegmentACK) SetLength() {
	s.BVLC.Length = uint16(s.MarshalLen())
}

func (s *SegmentACK) GetService() uint8 {
	return s.APDU.Service
}

func (s *SegmentACK) GetType() uint8 {
	return s.APDU.Type
}
