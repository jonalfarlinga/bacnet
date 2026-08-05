package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// Reject is a BACnet Reject-PDU: sent instead of a normal reply when the
// request APDU itself is malformed (bad tag, wrong parameter count, etc.)
// rather than merely unable to be completed. The reason code lives on
// APDU.Reason - see objects.RejectReasonMap for human-readable labels.
type Reject struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

func NewReject(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *Reject {
	r := &Reject{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.Reject, 0, nil),
	}
	r.SetLength()

	return r
}

func (r *Reject) UnmarshalBinary(b []byte) error {
	if l := len(b); l < r.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal Reject - marshal length %d binary length %d: %v",
			r.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := r.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Reject: %v", common.ErrTooShortToParse)
	}
	offset += r.BVLC.MarshalLen()

	if err := r.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Reject: %v", common.ErrTooShortToParse)
	}
	offset += r.NPDU.MarshalLen()

	if err := r.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Reject: %v", common.ErrTooShortToParse)
	}

	return nil
}

func (r *Reject) MarshalBinary() ([]byte, error) {
	b := make([]byte, r.MarshalLen())
	if err := r.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (r *Reject) MarshalTo(b []byte) error {
	if len(b) < r.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal Reject - marshal length %d binary length %d: %v",
			r.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := r.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Reject: %v", err)
	}
	offset += r.BVLC.MarshalLen()

	if err := r.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Reject: %v", err)
	}
	offset += r.NPDU.MarshalLen()

	if err := r.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Reject: %v", err)
	}

	return nil
}

func (r *Reject) MarshalLen() int {
	l := r.BVLC.MarshalLen()
	l += r.NPDU.MarshalLen()
	l += r.APDU.MarshalLen()

	return l
}

func (r *Reject) SetLength() {
	r.BVLC.Length = uint16(r.MarshalLen())
}

func (r *Reject) GetService() uint8 {
	return r.APDU.Service
}

func (r *Reject) GetType() uint8 {
	return r.APDU.Type
}
