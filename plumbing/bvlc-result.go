package plumbing

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

// BVLCFuncResult is the BVLC function code for BVLC-Result.
const BVLCFuncResult = 0x00

// BVLC-Result codes, returned in reply to BBMD administrative requests
// (Write/Read-Broadcast-Distribution-Table, Register-Foreign-Device,
// Read-Foreign-Device-Table, Delete-Foreign-Device-Table-Entry,
// Distribute-Broadcast-To-Network).
const (
	BVLCResultSuccess                          uint16 = 0x0000
	BVLCResultWriteBroadcastDistributionNAK    uint16 = 0x0010
	BVLCResultReadBroadcastDistributionNAK     uint16 = 0x0020
	BVLCResultRegisterForeignDeviceNAK         uint16 = 0x0030
	BVLCResultReadForeignDeviceTableNAK        uint16 = 0x0040
	BVLCResultDeleteForeignDeviceTableEntryNAK uint16 = 0x0050
	BVLCResultDistributeBroadcastToNetworkNAK  uint16 = 0x0060
)

// BVLCResultCodeMap gives a human-readable label for each BVLC-Result code.
var BVLCResultCodeMap = map[uint16]string{
	BVLCResultSuccess:                          "Successful completion",
	BVLCResultWriteBroadcastDistributionNAK:    "Write-Broadcast-Distribution-Table NAK",
	BVLCResultReadBroadcastDistributionNAK:     "Read-Broadcast-Distribution-Table NAK",
	BVLCResultRegisterForeignDeviceNAK:         "Register-Foreign-Device NAK",
	BVLCResultReadForeignDeviceTableNAK:        "Read-Foreign-Device-Table NAK",
	BVLCResultDeleteForeignDeviceTableEntryNAK: "Delete-Foreign-Device-Table-Entry NAK",
	BVLCResultDistributeBroadcastToNetworkNAK:  "Distribute-Broadcast-To-Network NAK",
}

const bvlcResultBodyLen = 2

// BVLCResult is a BVLC-Result reply. Like ReadForeignDeviceTable, it carries
// no NPDU or APDU: a 2-byte result code follows the BVLC header directly.
type BVLCResult struct {
	*BVLC
	ResultCode uint16
}

// NewBVLCResult creates a BVLCResult.
func NewBVLCResult(code uint16) *BVLCResult {
	r := &BVLCResult{
		BVLC:       NewBVLC(BVLCFuncResult),
		ResultCode: code,
	}
	r.SetLength()
	return r
}

// UnmarshalBinary sets the values retrieved from byte sequence in a BVLCResult frame.
func (r *BVLCResult) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling BVLCResult: %v", err)
	}
	r.BVLC = bvlc

	body := b[bvlc.MarshalLen():]
	if len(body) < bvlcResultBodyLen {
		return fmt.Errorf(
			"unmarshalling BVLCResult - body length %d shorter than %d: %v",
			len(body), bvlcResultBodyLen, common.ErrTooShortToParse,
		)
	}
	r.ResultCode = binary.BigEndian.Uint16(body[0:2])
	return nil
}

// MarshalBinary returns the byte sequence generated from a BVLCResult instance.
func (r *BVLCResult) MarshalBinary() ([]byte, error) {
	b := make([]byte, r.MarshalLen())
	if err := r.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (r *BVLCResult) MarshalTo(b []byte) error {
	if len(b) < r.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal BVLCResult - marshal length %d binary length %d: %v",
			r.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	if err := r.BVLC.MarshalTo(b); err != nil {
		return fmt.Errorf("marshalling BVLCResult: %v", err)
	}
	binary.BigEndian.PutUint16(b[r.BVLC.MarshalLen():], r.ResultCode)
	return nil
}

// MarshalLen returns the serial length of a BVLCResult.
func (r *BVLCResult) MarshalLen() int {
	return r.BVLC.MarshalLen() + bvlcResultBodyLen
}

// SetLength sets the length in Length field.
func (r *BVLCResult) SetLength() {
	r.BVLC.Length = uint16(r.MarshalLen())
}

// GetService returns 0: BVLC-Result has no application service choice.
func (r *BVLCResult) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (r *BVLCResult) GetType() uint8 {
	return r.BVLC.Function
}
