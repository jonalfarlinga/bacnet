package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// SegmentedAPDU wraps a single segment of a segmented ConfirmedReq message.
// Only the first segment (APDU.SequenceNumber == 0) carries a service
// choice, so the concrete service type generally can't be determined until
// every segment has arrived - callers should feed each SegmentedAPDU to a
// reassembler keyed on (source address, InvokeID, APDU type) and re-parse
// the reassembled bytes once complete, rather than trying to Decode() this
// type directly.
type SegmentedAPDU struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

// UnmarshalBinary sets the values retrieved from byte sequence in a SegmentedAPDU frame.
func (s *SegmentedAPDU) UnmarshalBinary(b []byte) error {
	if l := len(b); l < s.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal SegmentedAPDU - marshal length %d binary length %d: %v",
			s.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := s.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentedAPDU: %v", err)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentedAPDU: %v", err)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling SegmentedAPDU: %v", err)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a SegmentedAPDU instance.
func (s *SegmentedAPDU) MarshalBinary() ([]byte, error) {
	b := make([]byte, s.MarshalLen())
	if err := s.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (s *SegmentedAPDU) MarshalTo(b []byte) error {
	if len(b) < s.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal SegmentedAPDU - marshal length %d binary length %d: %v",
			s.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := s.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal SegmentedAPDU: %v", err)
	}
	offset += s.BVLC.MarshalLen()

	if err := s.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal SegmentedAPDU: %v", err)
	}
	offset += s.NPDU.MarshalLen()

	if err := s.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal SegmentedAPDU: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of a SegmentedAPDU.
func (s *SegmentedAPDU) MarshalLen() int {
	l := s.BVLC.MarshalLen()
	l += s.NPDU.MarshalLen()
	l += s.APDU.MarshalLen()
	return l
}

func (s *SegmentedAPDU) GetService() uint8 {
	return s.APDU.Service
}

func (s *SegmentedAPDU) GetType() uint8 {
	return s.APDU.Type
}
