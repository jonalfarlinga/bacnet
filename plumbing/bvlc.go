package plumbing

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/pkg/errors"
)

// BVLCType is used for BACnet/IP in BVLL.
const BVLCType = 0x81

// BVLCFunc determines unicast or broadcast in BVLL.
const (
	BVLCFuncUnicast   = 0x0a
	BVLCFuncBroadcast = 0x0b
)

// BVLC is a BVLC frame.
type BVLC struct {
	Type     uint8
	Function uint8
	Length   uint16
}

// NewBVLC creates a BVLC.
func NewBVLC(f uint8) *BVLC {
	bvlc := &BVLC{
		Type:     BVLCType,
		Function: f,
		Length: 4,
	}
	return bvlc
}

// UnmarshalBinary sets the values retrieved from byte sequence in a BVLC frame.
func (bvlc *BVLC) UnmarshalBinary(b []byte) error {
	if l := len(b); l < bvlc.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal BVLC marshal length %d binary length %d", bvlc.MarshalLen(), l),
		)
	}
	bvlc.Type = b[0]
	bvlc.Function = b[1]
	bvlc.Length = binary.BigEndian.Uint16(b[2:4])

	return nil
}

// MarshalBinary returns the byte sequence generated from a BVLC instance.
func (bvlc *BVLC) MarshalBinary() ([]byte, error) {
	b := make([]byte, bvlc.MarshalLen())
	if err := bvlc.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}

	return b, nil
}

const bvlclen = 4

// MarshalLen returns the serial length of BVLC.
func (bvlc *BVLC) MarshalLen() int {
	return bvlclen
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (bvlc *BVLC) MarshalTo(b []byte) error {
	if len(b) < bvlc.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal BVLC - marshal length %d binary length %d", bvlc.MarshalLen(), len(b)),
		)
	}
	b[0] = byte(bvlc.Type)
	b[1] = byte(bvlc.Function)
	binary.BigEndian.PutUint16(b[2:4], bvlc.Length)
	return nil
}
