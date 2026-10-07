package plumbing

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"github.com/jonalfarlinga/bacnet/common"
)

// BVLC function codes of the BBMD services a foreign device uses, and of the
// Broadcast Distribution Table read (ASHRAE 135 annex J.2).
const (
	BVLCFuncReadBroadcastDistributionTable    = 0x02
	BVLCFuncReadBroadcastDistributionTableAck = 0x03
	BVLCFuncRegisterForeignDevice             = 0x05
)

// ---- Register-Foreign-Device -----------------------------------------------

const registerForeignDeviceBodyLen = 2

// RegisterForeignDevice is a BVLC Register-Foreign-Device request: a device
// on another IP subnet asks a BBMD to forward broadcasts to it for TTL
// seconds. The BBMD answers with a BVLCResult. It carries no NPDU or APDU.
type RegisterForeignDevice struct {
	*BVLC
	// TTL is how long the registration lasts, in seconds. A BBMD keeps the
	// entry for 30 seconds longer than this.
	TTL uint16
}

// NewRegisterForeignDevice creates a RegisterForeignDevice request.
func NewRegisterForeignDevice(ttl uint16) *RegisterForeignDevice {
	r := &RegisterForeignDevice{
		BVLC: NewBVLC(BVLCFuncRegisterForeignDevice),
		TTL:  ttl,
	}
	r.SetLength()
	return r
}

// UnmarshalBinary sets the values retrieved from byte sequence in a
// RegisterForeignDevice frame.
func (r *RegisterForeignDevice) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling RegisterForeignDevice: %w", err)
	}
	r.BVLC = bvlc

	body := b[bvlc.MarshalLen():]
	if len(body) < registerForeignDeviceBodyLen {
		return fmt.Errorf(
			"unmarshalling RegisterForeignDevice - body length %d shorter than %d: %w",
			len(body), registerForeignDeviceBodyLen, common.ErrTooShortToParse,
		)
	}
	r.TTL = binary.BigEndian.Uint16(body[0:2])
	return nil
}

// MarshalBinary returns the byte sequence generated from a
// RegisterForeignDevice instance.
func (r *RegisterForeignDevice) MarshalBinary() ([]byte, error) {
	b := make([]byte, r.MarshalLen())
	if err := r.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (r *RegisterForeignDevice) MarshalTo(b []byte) error {
	if len(b) < r.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal RegisterForeignDevice - marshal length %d binary length %d: %v",
			r.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	if err := r.BVLC.MarshalTo(b); err != nil {
		return fmt.Errorf("marshalling RegisterForeignDevice: %v", err)
	}
	binary.BigEndian.PutUint16(b[r.BVLC.MarshalLen():], r.TTL)
	return nil
}

// MarshalLen returns the serial length of a RegisterForeignDevice.
func (r *RegisterForeignDevice) MarshalLen() int {
	return r.BVLC.MarshalLen() + registerForeignDeviceBodyLen
}

// SetLength sets the length in Length field.
func (r *RegisterForeignDevice) SetLength() {
	r.BVLC.Length = uint16(r.MarshalLen())
}

// GetService returns 0: Register-Foreign-Device has no application service
// choice.
func (r *RegisterForeignDevice) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (r *RegisterForeignDevice) GetType() uint8 {
	return r.BVLC.Function
}

// ---- Read-Broadcast-Distribution-Table -------------------------------------

const bdtEntryLen = 10

// BDTEntry is one entry of a BBMD's Broadcast Distribution Table: a peer
// BBMD's address and the broadcast distribution mask that says whether
// broadcasts reach it by directed broadcast (a mask with zero bits) or by
// unicast (all ones).
type BDTEntry struct {
	IP   [4]byte
	Port uint16
	Mask [4]byte
}

// AddrPort returns the entry's address.
func (e *BDTEntry) AddrPort() netip.AddrPort {
	return netip.AddrPortFrom(netip.AddrFrom4(e.IP), e.Port)
}

// UnmarshalBinary sets the values retrieved from byte sequence in a BDTEntry.
func (e *BDTEntry) UnmarshalBinary(b []byte) error {
	if l := len(b); l < bdtEntryLen {
		return fmt.Errorf(
			"failed to unmarshal BDTEntry - marshal length %d binary length %d: %w",
			bdtEntryLen, l,
			common.ErrTooShortToParse,
		)
	}
	copy(e.IP[:], b[0:4])
	e.Port = binary.BigEndian.Uint16(b[4:6])
	copy(e.Mask[:], b[6:10])
	return nil
}

// MarshalLen returns the serial length of a BDTEntry.
func (e *BDTEntry) MarshalLen() int {
	return bdtEntryLen
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (e *BDTEntry) MarshalTo(b []byte) error {
	if len(b) < e.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal BDTEntry - marshal length %d binary length %d: %v",
			e.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	copy(b[0:4], e.IP[:])
	binary.BigEndian.PutUint16(b[4:6], e.Port)
	copy(b[6:10], e.Mask[:])
	return nil
}

// ReadBroadcastDistributionTable is a BVLC Read-Broadcast-Distribution-Table
// request. Like Read-Foreign-Device-Table, the BVLC header is the entire
// message.
type ReadBroadcastDistributionTable struct {
	*BVLC
}

// NewReadBroadcastDistributionTable creates a ReadBroadcastDistributionTable
// request.
func NewReadBroadcastDistributionTable() *ReadBroadcastDistributionTable {
	return &ReadBroadcastDistributionTable{BVLC: NewBVLC(BVLCFuncReadBroadcastDistributionTable)}
}

// UnmarshalBinary sets the values retrieved from byte sequence in a
// ReadBroadcastDistributionTable frame.
func (r *ReadBroadcastDistributionTable) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling ReadBroadcastDistributionTable: %w", err)
	}
	r.BVLC = bvlc
	return nil
}

// GetService returns 0: Read-Broadcast-Distribution-Table has no application
// service choice.
func (r *ReadBroadcastDistributionTable) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (r *ReadBroadcastDistributionTable) GetType() uint8 {
	return r.BVLC.Function
}

// ReadBroadcastDistributionTableAck is the BVLC reply to a
// Read-Broadcast-Distribution-Table request, carrying every entry of the
// responder's table.
type ReadBroadcastDistributionTableAck struct {
	*BVLC
	Entries []BDTEntry
}

// NewReadBroadcastDistributionTableAck creates a
// ReadBroadcastDistributionTableAck.
func NewReadBroadcastDistributionTableAck(entries []BDTEntry) *ReadBroadcastDistributionTableAck {
	a := &ReadBroadcastDistributionTableAck{
		BVLC:    NewBVLC(BVLCFuncReadBroadcastDistributionTableAck),
		Entries: entries,
	}
	a.SetLength()
	return a
}

// UnmarshalBinary sets the values retrieved from byte sequence in a
// ReadBroadcastDistributionTableAck frame.
func (a *ReadBroadcastDistributionTableAck) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling ReadBroadcastDistributionTableAck: %w", err)
	}
	a.BVLC = bvlc

	body := b[bvlc.MarshalLen():]
	if len(body)%bdtEntryLen != 0 {
		return fmt.Errorf(
			"unmarshalling ReadBroadcastDistributionTableAck body length %d not a multiple of entry length %d: %w",
			len(body), bdtEntryLen, common.ErrInvalidData,
		)
	}

	a.Entries = make([]BDTEntry, len(body)/bdtEntryLen)
	for i := range a.Entries {
		offset := i * bdtEntryLen
		if err := a.Entries[i].UnmarshalBinary(body[offset : offset+bdtEntryLen]); err != nil {
			return fmt.Errorf("unmarshalling ReadBroadcastDistributionTableAck entry %d: %w", i, err)
		}
	}
	return nil
}

// MarshalBinary returns the byte sequence generated from a
// ReadBroadcastDistributionTableAck instance.
func (a *ReadBroadcastDistributionTableAck) MarshalBinary() ([]byte, error) {
	b := make([]byte, a.MarshalLen())
	if err := a.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (a *ReadBroadcastDistributionTableAck) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal ReadBroadcastDistributionTableAck - marshal length %d binary length %d: %v",
			a.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	if err := a.BVLC.MarshalTo(b); err != nil {
		return fmt.Errorf("marshalling ReadBroadcastDistributionTableAck: %v", err)
	}
	offset := a.BVLC.MarshalLen()
	for _, e := range a.Entries {
		if err := e.MarshalTo(b[offset:]); err != nil {
			return fmt.Errorf("marshalling ReadBroadcastDistributionTableAck: %v", err)
		}
		offset += e.MarshalLen()
	}
	return nil
}

// MarshalLen returns the serial length of a ReadBroadcastDistributionTableAck.
func (a *ReadBroadcastDistributionTableAck) MarshalLen() int {
	return a.BVLC.MarshalLen() + len(a.Entries)*bdtEntryLen
}

// SetLength sets the length in Length field.
func (a *ReadBroadcastDistributionTableAck) SetLength() {
	a.BVLC.Length = uint16(a.MarshalLen())
}

// GetService returns 0: Read-Broadcast-Distribution-Table-Ack has no
// application service choice.
func (a *ReadBroadcastDistributionTableAck) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (a *ReadBroadcastDistributionTableAck) GetType() uint8 {
	return a.BVLC.Function
}
