package plumbing

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

// NPDU is a Network Protocol Data Units.
type NPDU struct {
	Version uint8
	Control uint8
	DNET    uint16
	DLEN    uint8
	DADR    []byte
	SNET    uint16
	SLEN    uint8
	SADR    []byte
	Hop     uint8
}

// NewNPDU creates a NPDU.
func NewNPDU(nsduContain bool, dstSpecifier bool, srcSpecifier bool, expectingReply bool) *NPDU {
	n := &NPDU{
		Version: 1,
	}
	n.SetControlFlags(nsduContain, dstSpecifier, srcSpecifier, expectingReply)
	return n
}

// SetControlFlags sets control flags to NPDU.
func (n *NPDU) SetControlFlags(nsduContain bool, dstSpecifier bool, srcSpecifier bool, expectingReply bool) {
	n.Control = uint8(
		common.BoolToInt(nsduContain)<<7 | common.BoolToInt(dstSpecifier)<<5 |
			common.BoolToInt(srcSpecifier)<<3 | common.BoolToInt(expectingReply)<<2,
	)
}

// UnmarshalBinary sets the values retrieved from byte sequence in a NPDU frame.
func (n *NPDU) UnmarshalBinary(b []byte) error {
	if l := len(b); l < npduLenMin {
		return fmt.Errorf(
			"failed to unmarshal NPDU - marshal length %d binary length %d: %v",
			npduLenMin, l,
			common.ErrTooShortToParse,
		)
	}
	n.Version = b[0]
	n.Control = b[1]

	offset := npduLenMin
	if flagDNET := (n.Control & 0x20) >> 5; flagDNET == 1 {
		if len(b) < offset+3 {
			return fmt.Errorf(
				"failed to unmarshal NPDU destination info at offset %d: %v",
				offset,
				common.ErrTooShortToParse,
			)
		}
		n.DNET = binary.BigEndian.Uint16(b[offset : offset+2])
		offset += 2
		n.DLEN = b[offset]
		offset++

		n.DADR = nil
		if n.DLEN > 0 {
			if len(b) < offset+int(n.DLEN) {
				return fmt.Errorf(
					"failed to unmarshal NPDU DADR at offset %d length %d: %v",
					offset,
					n.DLEN,
					common.ErrTooShortToParse,
				)
			}
			n.DADR = append([]byte(nil), b[offset:offset+int(n.DLEN)]...)
			offset += int(n.DLEN)
		}

		if len(b) <= offset {
			return fmt.Errorf(
				"failed to unmarshal NPDU hop count at offset %d: %v",
				offset,
				common.ErrTooShortToParse,
			)
		}
		n.Hop = b[offset]
		offset++
	}

	if flagSNET := (n.Control & 0x08) >> 3; flagSNET == 1 {
		if len(b) < offset+3 {
			return fmt.Errorf(
				"failed to unmarshal NPDU source info at offset %d: %v",
				offset,
				common.ErrTooShortToParse,
			)
		}
		n.SNET = binary.BigEndian.Uint16(b[offset : offset+2])
		offset += 2
		n.SLEN = b[offset]
		offset++

		n.SADR = nil
		if n.SLEN > 0 {
			if len(b) < offset+int(n.SLEN) {
				return fmt.Errorf(
					"failed to unmarshal NPDU SADR at offset %d length %d: %v",
					offset,
					n.SLEN,
					common.ErrTooShortToParse,
				)
			}
			n.SADR = append([]byte(nil), b[offset:offset+int(n.SLEN)]...)
			offset += int(n.SLEN)
		}
	}

	return nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (n *NPDU) MarshalTo(b []byte) error {
	if len(b) < n.MarshalLen() {
		return fmt.Errorf(
			"failed to marshall NPDU - marshal length %d binary length %d: %v",
			n.MarshalLen(),
			len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	b[0] = n.Version
	b[1] = n.Control
	if flagDNET := (n.Control & 0x20) >> 5; flagDNET == 1 {
		offset := npduLenMin
		binary.BigEndian.PutUint16(b[offset:offset+2], n.DNET)
		offset += 2
		b[offset] = n.DLEN
		offset++

		if n.DLEN > 0 {
			copy(b[offset:offset+int(n.DLEN)], n.DADR)
			offset += int(n.DLEN)
		}

		b[offset] = n.Hop
		offset++

		if flagSNET := (n.Control & 0x08) >> 3; flagSNET == 1 {
			binary.BigEndian.PutUint16(b[offset:offset+2], n.SNET)
			offset += 2
			b[offset] = n.SLEN
			offset++

			if n.SLEN > 0 {
				copy(b[offset:offset+int(n.SLEN)], n.SADR)
			}
		}
		return nil
	}

	if flagSNET := (n.Control & 0x08) >> 3; flagSNET == 1 {
		offset := npduLenMin
		binary.BigEndian.PutUint16(b[offset:offset+2], n.SNET)
		offset += 2
		b[offset] = n.SLEN
		offset++

		if n.SLEN > 0 {
			copy(b[offset:offset+int(n.SLEN)], n.SADR)
		}
	}
	return nil
}

const npduLenMin = 2

// MarshalLen returns the serial length of NPDU.
func (n *NPDU) MarshalLen() int {
	l := npduLenMin
	if flagDNET := (n.Control & 0x20) >> 5; flagDNET == 1 {
		l += 2 // DNET
		l += 1 // DLEN
		l += int(n.DLEN)
		l += 1 // Hop count
	}
	if flagSNET := (n.Control & 0x08) >> 3; flagSNET == 1 {
		l += 2 // SNET
		l += 1 // SLEN
		l += int(n.SLEN)
	}
	return l
}
