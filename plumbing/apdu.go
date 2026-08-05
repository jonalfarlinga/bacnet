package plumbing

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
)

// APDU is a Application protocol DAta Units.
type APDU struct {
	Type               uint8
	Flags              uint8
	MaxSeg             uint8
	MaxSize            uint8
	InvokeID           uint8
	SequenceNumber     uint8
	ProposedWindowSize uint8
	Service            uint8
	Objects            []objects.APDUPayload

	// Reason holds the abort-reason (BACnetAbortReason) or reject-reason
	// (BACnetRejectReason) enumerated value for Abort/Reject PDUs.
	Reason uint8

	// Payload holds the raw, undecoded bytes following the fixed header of
	// a segmented ConfirmedReq/ComplexAck APDU (everything after the
	// service choice on segment 0, or after the sequence-number/window-size
	// pair on later segments, which carry no service choice of their own).
	// A single segment's parameters are generally incomplete, so Objects is
	// left unpopulated for segmented APDUs - callers must reassemble
	// Payload across segments and re-parse the combined buffer.
	Payload []byte

	// ActualWindowSize is the window size carried by a Segment-ACK PDU:
	// how many segments the acknowledging device is willing to receive in
	// the next window.
	ActualWindowSize uint8
}

// NewAPDU creates an APDU.
func NewAPDU(t, s uint8, objs []objects.APDUPayload) *APDU {
	return &APDU{
		Type:    t,
		Service: s,
		Objects: objs,
	}
}

// IsSegmented reports whether the SEG bit is set: this APDU is one segment
// of a larger segmented message.
func (a *APDU) IsSegmented() bool {
	return a.Flags&0x8 != 0
}

// HasMoreSegments reports whether the MOR bit is set: at least one more
// segment follows this one.
func (a *APDU) HasMoreSegments() bool {
	return a.Flags&0x4 != 0
}

// SegmentedResponseAccepted reports whether the SA bit is set.
func (a *APDU) SegmentedResponseAccepted() bool {
	return a.Flags&0x2 != 0
}

// IsNegativeAck reports whether a Segment-ACK's NAK bit is set: the
// acknowledging device detected a problem (e.g. missed segment) rather than
// simply confirming receipt.
func (a *APDU) IsNegativeAck() bool {
	return a.Flags&0x2 != 0
}

// IsServerAck reports whether a Segment-ACK's "server" bit is set: the
// device sending this ack is itself acting as the server for the
// transaction (acknowledging a segmented ConfirmedReq sent to it by a
// client). It's false when the ack-sender is the client, acknowledging a
// segmented ComplexAck reply to its own request.
func (a *APDU) IsServerAck() bool {
	return a.Flags&0x1 != 0
}

// UnmarshalBinary sets the values retrieved from byte sequence in a APDU frame.
func (a *APDU) UnmarshalBinary(b []byte) error {
	a.Type = b[0] >> 4
	a.Flags = b[0] & 0xF

	var offset int = 1
	switch a.Type {
	case UnConfirmedReq:
		a.Service = b[offset]
		offset++
		if len(b) > 2 {
			a.Objects = parseObjectTags(b, offset, false)
		}
	case ConfirmedReq:
		a.MaxSeg = b[offset] >> 4
		a.MaxSize = b[offset] & 0xF
		offset++
		a.InvokeID = b[offset]
		offset++

		if a.IsSegmented() {
			return a.unmarshalSegmentHeader(b, offset)
		}

		a.Service = b[offset]
		offset++
		if len(b) > 2 {
			a.Objects = parseObjectTags(b, offset, false)
		}
	case ComplexAck:
		a.InvokeID = b[offset]
		offset++

		if a.IsSegmented() {
			return a.unmarshalSegmentHeader(b, offset)
		}

		a.Service = b[offset]
		offset++
		a.Objects = parseObjectTags(b, offset, true)
	case SimpleAck, Error:
		a.InvokeID = b[offset]
		offset++
		a.Service = b[offset]
		offset++
		a.Objects = parseObjectTags(b, offset, true)
	case Reject, Abort:
		if len(b) <= offset {
			return fmt.Errorf("unmarshal APDU: missing InvokeID: %s", common.ErrTooShortToParse)
		}
		a.InvokeID = b[offset]
		offset++

		if len(b) <= offset {
			return fmt.Errorf("unmarshal APDU: missing reason: %s", common.ErrTooShortToParse)
		}
		a.Reason = b[offset]
	case SegmentAck:
		if len(b) < offset+3 {
			return fmt.Errorf("unmarshal APDU: segment ack too short: %s", common.ErrTooShortToParse)
		}
		a.InvokeID = b[offset]
		offset++
		a.SequenceNumber = b[offset]
		offset++
		a.ActualWindowSize = b[offset]
	default:
		return fmt.Errorf("unmarshal APDU: %s", common.ErrNotImplemented)
	}

	return nil
}

// unmarshalSegmentHeader parses the sequence-number/proposed-window-size
// pair that follows InvokeID on every segment of a segmented ConfirmedReq or
// ComplexAck. Only the first segment (SequenceNumber == 0) carries a service
// choice byte; everything after the fixed header is stashed in Payload
// rather than tag-parsed, since a lone segment's parameters are generally
// incomplete.
func (a *APDU) unmarshalSegmentHeader(b []byte, offset int) error {
	if len(b) < offset+2 {
		return fmt.Errorf(
			"unmarshal APDU: segmented header at offset %d: %v",
			offset, common.ErrTooShortToParse,
		)
	}
	a.SequenceNumber = b[offset]
	offset++
	a.ProposedWindowSize = b[offset]
	offset++

	if a.SequenceNumber == 0 {
		if len(b) <= offset {
			return fmt.Errorf(
				"unmarshal APDU: segmented service choice at offset %d: %v",
				offset, common.ErrTooShortToParse,
			)
		}
		a.Service = b[offset]
		offset++
	}

	a.Payload = append([]byte(nil), b[offset:]...)
	return nil
}

// parseObjectTags decodes a run of application/context tags starting at
// offset. When decodeBoolean is true, application-tagged booleans (tag
// number 1) are decoded inline as their length nibble, matching the
// ComplexAck/SimpleAck/Error encoding.
func parseObjectTags(b []byte, offset int, decodeBoolean bool) []objects.APDUPayload {
	objs := []objects.APDUPayload{}
	for offset < len(b) {
		o := objects.Object{
			TagNumber: b[offset] >> 4,
			TagClass:  common.IntToBool(int(b[offset]) & 0x8 >> 3),
			Length:    b[offset] & 0x7,
		}

		// Handle extended value case
		if o.Length == 5 {
			offset++
			o.Length = uint8(b[offset])
		} else if o.Length > 5 {
			offset++
			objs = append(objs, &o)
			continue
		}

		// Handle boolean data
		if decodeBoolean && !o.TagClass && o.TagNumber == 1 {
			o.Value = o.Length
			objs = append(objs, &o)
			offset++
			continue
		}

		o.Data = b[offset+1 : offset+int(o.Length)+1]
		objs = append(objs, &o)
		offset += int(o.Length) + 1

		if offset >= len(b) {
			break
		}
	}
	return objs
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (a *APDU) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal APDU - marshall length %d binary length %d: %s",
			a.MarshalLen(),
			len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}

	var offset int = 0
	b[offset] = a.Type<<4 | a.Flags
	offset++

	if a.Flags&0x2 == 1 {
		b[offset] = (a.MaxSeg & 0x7 << 4) | (a.MaxSize & 0xF)
		offset++
	}

	switch a.Type {
	case UnConfirmedReq:
		b[offset] = a.Service
		offset++
		if a.MarshalLen() > 2 {
			for _, o := range a.Objects {
				ob, err := o.MarshalBinary()
				if err != nil {
					return fmt.Errorf("failed to marshal UnconfirmedReq: %v", err)
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += int(o.MarshalLen())

				if offset > a.MarshalLen() {
					return fmt.Errorf(
						"failed to marshal UnconfirmedReq marshal length %d binary length %d: %v",
						a.MarshalLen(),
						len(b),
						common.ErrTooShortToMarshalBinary,
					)
				}
			}
		}
	case Reject, Abort:
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.Reason
		offset++
	case SegmentAck:
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.SequenceNumber
		offset++
		b[offset] = a.ActualWindowSize
		offset++
	case ComplexAck, SimpleAck, Error:
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.Service
		offset++
		if a.MarshalLen() > 4 {
			for _, o := range a.Objects {
				ob, err := o.MarshalBinary()
				if err != nil {
					return fmt.Errorf("failed to marshal CACK/SACK/ERROR: %s", err)
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += o.MarshalLen()

				if offset > a.MarshalLen() {
					return fmt.Errorf(
						"failed to marshal CACK/SACK/ERROR - binary overflow at offset %d: %v",
						offset,
						common.ErrTooShortToMarshalBinary,
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
					return fmt.Errorf("failed to marshal ConfirmedReq: %v", err)
				}

				copy(b[offset:offset+o.MarshalLen()], ob)
				offset += o.MarshalLen()

				if offset > a.MarshalLen() {
					return fmt.Errorf(
						"failed to marshal ConfirmedReq - binary overflow at offset %d: %v",
						offset,
						common.ErrTooShortToMarshalBinary,
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
	case ComplexAck, SimpleAck, Error, Reject, Abort:
		l += 3
	case SegmentAck:
		l += 4
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
