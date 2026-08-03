package plumbing

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

// BVLC function codes for Foreign Device Table maintenance.
const (
	BVLCFuncReadForeignDeviceTable    = 0x06
	BVLCFuncReadForeignDeviceTableAck = 0x07
)

const fdtEntryLen = 10

// FDTEntry is a single entry of a BACnet/IP Foreign Device Table: the
// registrant's address, its requested TTL, and the time left before the
// entry expires.
type FDTEntry struct {
	IP            [4]byte
	Port          uint16
	TTL           uint16
	RemainingTime uint16
}

// UnmarshalBinary sets the values retrieved from byte sequence in a FDTEntry.
func (e *FDTEntry) UnmarshalBinary(b []byte) error {
	if l := len(b); l < fdtEntryLen {
		return fmt.Errorf(
			"failed to unmarshal FDTEntry - marshal length %d binary length %d: %v",
			fdtEntryLen, l,
			common.ErrTooShortToParse,
		)
	}
	copy(e.IP[:], b[0:4])
	e.Port = binary.BigEndian.Uint16(b[4:6])
	e.TTL = binary.BigEndian.Uint16(b[6:8])
	e.RemainingTime = binary.BigEndian.Uint16(b[8:10])
	return nil
}

// MarshalLen returns the serial length of a FDTEntry.
func (e *FDTEntry) MarshalLen() int {
	return fdtEntryLen
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (e *FDTEntry) MarshalTo(b []byte) error {
	if len(b) < e.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal FDTEntry - marshal length %d binary length %d: %v",
			e.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	copy(b[0:4], e.IP[:])
	binary.BigEndian.PutUint16(b[4:6], e.Port)
	binary.BigEndian.PutUint16(b[6:8], e.TTL)
	binary.BigEndian.PutUint16(b[8:10], e.RemainingTime)
	return nil
}

// ReadForeignDeviceTable is a BVLC Read-Foreign-Device-Table request. Unlike
// application services, it carries no NPDU or APDU: the BVLC header is the
// entire message.
type ReadForeignDeviceTable struct {
	*BVLC
}

// NewReadForeignDeviceTable creates a ReadForeignDeviceTable request.
func NewReadForeignDeviceTable() *ReadForeignDeviceTable {
	return &ReadForeignDeviceTable{BVLC: NewBVLC(BVLCFuncReadForeignDeviceTable)}
}

// GetService returns 0: Read-Foreign-Device-Table has no application service choice.
func (r *ReadForeignDeviceTable) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (r *ReadForeignDeviceTable) GetType() uint8 {
	return r.BVLC.Function
}

// ReadForeignDeviceTableAck is the BVLC reply to a Read-Foreign-Device-Table
// request, carrying every entry in the responder's Foreign Device Table.
type ReadForeignDeviceTableAck struct {
	*BVLC
	Entries []FDTEntry
}

// NewReadForeignDeviceTableAck creates a ReadForeignDeviceTableAck.
func NewReadForeignDeviceTableAck(entries []FDTEntry) *ReadForeignDeviceTableAck {
	a := &ReadForeignDeviceTableAck{
		BVLC:    NewBVLC(BVLCFuncReadForeignDeviceTableAck),
		Entries: entries,
	}
	a.SetLength()
	return a
}

// UnmarshalBinary sets the values retrieved from byte sequence in a ReadForeignDeviceTableAck frame.
func (a *ReadForeignDeviceTableAck) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling ReadForeignDeviceTableAck: %v", err)
	}
	a.BVLC = bvlc

	body := b[bvlc.MarshalLen():]
	if len(body)%fdtEntryLen != 0 {
		return fmt.Errorf(
			"unmarshalling ReadForeignDeviceTableAck body length %d not a multiple of entry length %d: %v",
			len(body), fdtEntryLen, common.ErrTooShortToParse,
		)
	}

	a.Entries = make([]FDTEntry, len(body)/fdtEntryLen)
	for i := range a.Entries {
		offset := i * fdtEntryLen
		if err := a.Entries[i].UnmarshalBinary(body[offset : offset+fdtEntryLen]); err != nil {
			return fmt.Errorf("unmarshalling ReadForeignDeviceTableAck entry %d: %v", i, err)
		}
	}
	return nil
}

// MarshalBinary returns the byte sequence generated from a ReadForeignDeviceTableAck instance.
func (a *ReadForeignDeviceTableAck) MarshalBinary() ([]byte, error) {
	b := make([]byte, a.MarshalLen())
	if err := a.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (a *ReadForeignDeviceTableAck) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal ReadForeignDeviceTableAck - marshal length %d binary length %d: %v",
			a.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	if err := a.BVLC.MarshalTo(b); err != nil {
		return fmt.Errorf("marshalling ReadForeignDeviceTableAck: %v", err)
	}
	offset := a.BVLC.MarshalLen()
	for _, e := range a.Entries {
		if err := e.MarshalTo(b[offset:]); err != nil {
			return fmt.Errorf("marshalling ReadForeignDeviceTableAck: %v", err)
		}
		offset += e.MarshalLen()
	}
	return nil
}

// MarshalLen returns the serial length of a ReadForeignDeviceTableAck.
func (a *ReadForeignDeviceTableAck) MarshalLen() int {
	return a.BVLC.MarshalLen() + len(a.Entries)*fdtEntryLen
}

// SetLength sets the length in Length field.
func (a *ReadForeignDeviceTableAck) SetLength() {
	a.BVLC.Length = uint16(a.MarshalLen())
}

// GetService returns 0: Read-Foreign-Device-Table-Ack has no application service choice.
func (a *ReadForeignDeviceTableAck) GetService() uint8 {
	return 0
}

// GetType returns the BVLC function code of the message.
func (a *ReadForeignDeviceTableAck) GetType() uint8 {
	return a.BVLC.Function
}
