package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// Abort is a BACnet Abort-PDU: sent instead of a normal reply when the
// responder can't complete a confirmed request in its current form (e.g.
// segmentation-not-supported when the requester declined a segmented
// reply). The reason code lives on APDU.Reason - see objects.AbortReasonMap
// for human-readable labels.
type Abort struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

func NewAbort(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *Abort {
	a := &Abort{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.Abort, 0, nil),
	}
	a.SetLength()

	return a
}

func (a *Abort) UnmarshalBinary(b []byte) error {
	if l := len(b); l < a.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal Abort - marshal length %d binary length %d: %v",
			a.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := a.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Abort: %v", common.ErrTooShortToParse)
	}
	offset += a.BVLC.MarshalLen()

	if err := a.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Abort: %v", common.ErrTooShortToParse)
	}
	offset += a.NPDU.MarshalLen()

	if err := a.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling Abort: %v", common.ErrTooShortToParse)
	}

	return nil
}

func (a *Abort) MarshalBinary() ([]byte, error) {
	b := make([]byte, a.MarshalLen())
	if err := a.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (a *Abort) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal Abort - marshal length %d binary length %d: %v",
			a.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := a.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Abort: %v", err)
	}
	offset += a.BVLC.MarshalLen()

	if err := a.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Abort: %v", err)
	}
	offset += a.NPDU.MarshalLen()

	if err := a.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling Abort: %v", err)
	}

	return nil
}

func (a *Abort) MarshalLen() int {
	l := a.BVLC.MarshalLen()
	l += a.NPDU.MarshalLen()
	l += a.APDU.MarshalLen()

	return l
}

func (a *Abort) SetLength() {
	a.BVLC.Length = uint16(a.MarshalLen())
}

func (a *Abort) GetService() uint8 {
	return a.APDU.Service
}

func (a *Abort) GetType() uint8 {
	return a.APDU.Type
}
