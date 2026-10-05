package plumbing

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"github.com/jonalfarlinga/bacnet/common"
)

// BVLCType is used for BACnet/IP in BVLL.
const BVLCType = 0x81

// BVLCFunc determines unicast or broadcast in BVLL.
const (
	BVLCFuncForwardedNPDU = 0x04
	BVLCFuncUnicast       = 0x0a
	BVLCFuncBroadcast     = 0x0b
)

// BVLC is a BVLC frame.
type BVLC struct {
	Type     uint8
	Function uint8
	Length   uint16

	// Origin is the B/IP address (4-byte IPv4 address, 2-byte port) of the
	// device that first sent the message. It is on the wire, and meaningful,
	// only for a Forwarded-NPDU: the datagram's own source is then the BBMD
	// that forwarded it.
	Origin [bvlcOriginLen]byte
}

// NewBVLC creates a BVLC.
func NewBVLC(f uint8) *BVLC {
	bvlc := &BVLC{
		Type:     BVLCType,
		Function: f,
		Length:   4,
	}
	return bvlc
}

// UnmarshalBinary sets the values retrieved from byte sequence in a BVLC frame.
func (bvlc *BVLC) UnmarshalBinary(b []byte) error {
	if l := len(b); l < bvlclen {
		return fmt.Errorf(
			"failed to unmarshal BVLC - marshal length %d binary length %d: %w",
			bvlclen, l,
			common.ErrTooShortToParse,
		)
	}
	bvlc.Type = b[0]
	bvlc.Function = b[1]
	bvlc.Length = binary.BigEndian.Uint16(b[2:4])

	bvlc.Origin = [bvlcOriginLen]byte{}
	if bvlc.Function == BVLCFuncForwardedNPDU {
		if l := len(b); l < bvlclen+bvlcOriginLen {
			return fmt.Errorf(
				"failed to unmarshal BVLC - forwarded header length %d binary length %d: %w",
				bvlclen+bvlcOriginLen, l,
				common.ErrTooShortToParse,
			)
		}
		copy(bvlc.Origin[:], b[bvlclen:])
	}

	return nil
}

// OriginAddrPort returns Origin as an address and port. ok is false unless
// the frame is a Forwarded-NPDU.
func (bvlc *BVLC) OriginAddrPort() (origin netip.AddrPort, ok bool) {
	if bvlc.Function != BVLCFuncForwardedNPDU {
		return netip.AddrPort{}, false
	}
	addr := netip.AddrFrom4([4]byte(bvlc.Origin[:4]))
	return netip.AddrPortFrom(addr, binary.BigEndian.Uint16(bvlc.Origin[4:])), true
}

// MarshalBinary returns the byte sequence generated from a BVLC instance.
func (bvlc *BVLC) MarshalBinary() ([]byte, error) {
	b := make([]byte, bvlc.MarshalLen())
	if err := bvlc.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}

	return b, nil
}

const (
	bvlclen       = 4
	bvlcOriginLen = 6
)

// MarshalLen returns the serial length of BVLC: the 4-byte header, plus the
// originating address when the frame is a Forwarded-NPDU.
func (bvlc *BVLC) MarshalLen() int {
	if bvlc.Function == BVLCFuncForwardedNPDU {
		return bvlclen + bvlcOriginLen
	}
	return bvlclen
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (bvlc *BVLC) MarshalTo(b []byte) error {
	if len(b) < bvlc.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal BVLC - marshal length %d binary length %d: %v",
			bvlc.MarshalLen(),
			len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	b[0] = byte(bvlc.Type)
	b[1] = byte(bvlc.Function)
	binary.BigEndian.PutUint16(b[2:4], bvlc.Length)
	if bvlc.Function == BVLCFuncForwardedNPDU {
		copy(b[bvlclen:], bvlc.Origin[:])
	}
	return nil
}
