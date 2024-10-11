package plumbing

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/pkg/errors"
)

// APDU is a Application protocol DAta Units.
type APDU struct {
	Type     uint8
	Flags    uint8
	MaxSeg   uint8
	MaxSize  uint8
	InvokeID uint8
	Service  uint8
	Objects  []objects.APDUPayload
}

// NewAPDU creates an APDU.
func NewAPDU(t, s uint8, objs []objects.APDUPayload) *APDU {
	return &APDU{
		Type:    t,
		Service: s,
		Objects: objs,
	}
}

// UnmarshalBinary sets the values retrieved from byte sequence in a APDU frame.
func (a *APDU) UnmarshalBinary(b []byte) error {
	if l := len(b); l < a.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal APDU - marshal length %d binary length %d", a.MarshalLen(), l),
		)
	}

	a.Type = b[0] >> 4
	a.Flags = b[0] & 0x7

	var offset int = 1
	switch a.Type {
	case UnConfirmedReq:
		a.Service = b[offset]
		offset++
		if len(b) > 2 {
			objs := []objects.APDUPayload{}
			for {
				o := objects.Object{
					TagNumber: b[offset] >> 4,
					TagClass:  common.IntToBool(int(b[offset]) & 0x8 >> 3),
					Length:    b[offset] & 0x7,
				}

				// Handle extended value case
				if o.Length == 5 {
					offset++
					o.Length = uint8(b[offset])
				}

				o.Data = b[offset+1 : offset+int(o.Length)+1]
				objs = append(objs, &o)
				offset += int(o.Length) + 1

				if offset >= len(b) {
					break
				}
			}
			a.Objects = objs
		}
	case ConfirmedReq:
		offset++
		a.InvokeID = b[offset]
		offset++
		a.Service = b[offset]
		offset++
		if len(b) > 2 {
			objs := []objects.APDUPayload{}
			for {
				o := objects.Object{
					TagNumber: b[offset] >> 4,
					TagClass:  common.IntToBool(int(b[offset]) & 0x8 >> 3),
					Length:    b[offset] & 0x7,
				}

				// Handle extended value case
				if o.Length == 5 {
					offset++
					o.Length = uint8(b[offset])
				}

				// Drop tags so that they don't get in the way!
				if b[offset] == objects.TagOpening || b[offset] == objects.TagClosing {
					offset++
					if offset >= len(b) {
						break
					}
					continue
				}

				o.Data = b[offset+1 : offset+int(o.Length)+1]
				objs = append(objs, &o)
				offset += int(o.Length) + 1

				if offset >= len(b) {
					break
				}
			}
			a.Objects = objs
		}
	case ComplexAck, SimpleAck, Error:
		a.InvokeID = b[offset]
		offset++
		a.Service = b[offset]
		offset++
		if len(b) > 3 {
			objs := []objects.APDUPayload{}
			for {
				o := objects.Object{
					TagNumber: b[offset] >> 4,
					TagClass:  common.IntToBool(int(b[offset]) & 0x8 >> 3),
					Length:    b[offset] & 0x7,
				}

				// Handle extended value case
				if o.Length == 5 {
					offset++
					o.Length = uint8(b[offset])
				}

				// Drop tags so that they don't get in the way!
				if b[offset] == objects.TagOpening || b[offset] == objects.TagClosing {
					fmt.Print("tag opening/closing\n")
					offset++
					if offset >= len(b) {
						break
					}
					continue
				}

				o.Data = b[offset+1 : offset+int(o.Length)+1]
				objs = append(objs, &o)
				offset += int(o.Length) + 1

				if offset >= len(b) {
					break
				}
			}
			a.Objects = objs
		}
	}

	return nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (a *APDU) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal APDU - marshall length %d binary length %d", a.MarshalLen(), len(b)),
		)
	}

	var offset int = 0
	b[offset] = a.Type<<4 | a.Flags
	offset++

	switch a.Type {
	case UnConfirmedReq:
		b[offset] = a.Service
		offset++
		if a.MarshalLen() > 2 {
			for _, o := range a.Objects {
				ob, err := o.MarshalBinary()
				if err != nil {
					return errors.Wrap(err, "failed to marshal UnconfirmedReq")
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += int(o.MarshalLen())

				if offset > a.MarshalLen() {
					return errors.Wrap(
						common.ErrTooShortToMarshalBinary,
						fmt.Sprintf("failed to marshal UnconfirmedReq marshal length %d binary length %d", a.MarshalLen(), len(b)),
					)
				}
			}
		}
	case ComplexAck, SimpleAck, Error:
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.Service
		offset++
		if a.MarshalLen() > 4 {
			for _, o := range a.Objects {
				ob, err := o.MarshalBinary()
				if err != nil {
					return errors.Wrap(err, "failed to marshal CACK/SACK/ERROR")
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += o.MarshalLen()

				if offset > a.MarshalLen() {
					return errors.Wrap(
						common.ErrTooShortToMarshalBinary,
						fmt.Sprintf("failed to marshal CACK/SACK/ERROR - binary overflow at offset %d", offset),
					)
				}
			}
		}
	case ConfirmedReq:
		b[offset] |= (a.MaxSeg & 0x7 << 4) | (a.MaxSize & 0xF)
		offset++
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.Service
		offset++
		if a.MarshalLen() > 4 {
			for _, o := range a.Objects {
				ob, err := o.MarshalBinary()
				if err != nil {
					return errors.Wrap(err, "failed to marshal ConfirmedReq")
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += o.MarshalLen()

				if offset > a.MarshalLen() {
					return errors.Wrap(
						common.ErrTooShortToMarshalBinary,
						fmt.Sprintf("failed to marshal ConfirmedReq - binary overflow at offset %d", offset),
					)
				}
			}
		}
	}

	return nil
}

// MarshalLen returns the serial length of APDU.
func (a *APDU) MarshalLen() int {
	var l int = 0
	switch a.Type {
	case ConfirmedReq:
		l += 4
	case ComplexAck, SimpleAck, Error:
		l += 3
	case UnConfirmedReq:
		l += 2
	}
	for _, o := range a.Objects {
		l += o.MarshalLen()
	}
	return l
}

// SetAPDUFlags sets APDU Flags to APDU.
func (a *APDU) SetAPDUFlags(sa, moreSegments, segmentedReq bool) {
	a.Flags = uint8(
		common.BoolToInt(sa)<<1 | common.BoolToInt(moreSegments)<<2 | common.BoolToInt(segmentedReq)<<3,
	)
}
