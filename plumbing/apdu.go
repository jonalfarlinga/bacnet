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
	// a segmented ConfirmedReq/ComplexAck APDU: everything after the
	// service choice, which every segment carries.
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
	if len(b) == 0 {
		return fmt.Errorf("unmarshal APDU: missing PDU type: %w", common.ErrTooShortToParse)
	}
	a.Type = b[0] >> 4
	a.Flags = b[0] & 0xF

	var offset int = 1
	var err error
	switch a.Type {
	case UnConfirmedReq:
		if len(b) < offset+1 {
			return fmt.Errorf("unmarshal APDU: missing service choice: %w", common.ErrTooShortToParse)
		}
		a.Service = b[offset]
		offset++
		if len(b) > 2 {
			if a.Objects, err = parseObjectTags(b, offset); err != nil {
				return fmt.Errorf("unmarshal APDU: %w", err)
			}
		}
	case ConfirmedReq:
		if len(b) < offset+2 {
			return fmt.Errorf("unmarshal APDU: confirmed request header: %w", common.ErrTooShortToParse)
		}
		a.MaxSeg = b[offset] >> 4
		a.MaxSize = b[offset] & 0xF
		offset++
		a.InvokeID = b[offset]
		offset++

		if a.IsSegmented() {
			return a.unmarshalSegmentHeader(b, offset)
		}

		if len(b) < offset+1 {
			return fmt.Errorf("unmarshal APDU: missing service choice: %w", common.ErrTooShortToParse)
		}
		a.Service = b[offset]
		offset++
		if a.Objects, err = parseObjectTags(b, offset); err != nil {
			return fmt.Errorf("unmarshal APDU: %w", err)
		}
	case ComplexAck:
		if len(b) < offset+1 {
			return fmt.Errorf("unmarshal APDU: missing InvokeID: %w", common.ErrTooShortToParse)
		}
		a.InvokeID = b[offset]
		offset++

		if a.IsSegmented() {
			return a.unmarshalSegmentHeader(b, offset)
		}

		if len(b) < offset+1 {
			return fmt.Errorf("unmarshal APDU: missing service choice: %w", common.ErrTooShortToParse)
		}
		a.Service = b[offset]
		offset++
		if a.Objects, err = parseObjectTags(b, offset); err != nil {
			return fmt.Errorf("unmarshal APDU: %w", err)
		}
	case SimpleAck, Error:
		if len(b) < offset+2 {
			return fmt.Errorf("unmarshal APDU: missing InvokeID or service choice: %w", common.ErrTooShortToParse)
		}
		a.InvokeID = b[offset]
		offset++
		a.Service = b[offset]
		offset++
		if a.Objects, err = parseObjectTags(b, offset); err != nil {
			return fmt.Errorf("unmarshal APDU: %w", err)
		}
	case Reject, Abort:
		if len(b) <= offset {
			return fmt.Errorf("unmarshal APDU: missing InvokeID: %w", common.ErrTooShortToParse)
		}
		a.InvokeID = b[offset]
		offset++

		if len(b) <= offset {
			return fmt.Errorf("unmarshal APDU: missing reason: %w", common.ErrTooShortToParse)
		}
		a.Reason = b[offset]
	case SegmentAck:
		if len(b) < offset+3 {
			return fmt.Errorf("unmarshal APDU: segment ack too short: %w", common.ErrTooShortToParse)
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
// ComplexAck, and the service choice after it. Every segment repeats the
// whole header, service choice included (ASHRAE 135 clause 20.1.2 and
// 20.1.5); everything after it is stashed in Payload rather than tag-parsed,
// since a lone segment's parameters are generally incomplete.
func (a *APDU) unmarshalSegmentHeader(b []byte, offset int) error {
	if len(b) < offset+2 {
		return fmt.Errorf(
			"unmarshal APDU: segmented header at offset %d: %w",
			offset, common.ErrTooShortToParse,
		)
	}
	a.SequenceNumber = b[offset]
	offset++
	a.ProposedWindowSize = b[offset]
	offset++

	if len(b) <= offset {
		return fmt.Errorf(
			"unmarshal APDU: segmented service choice at offset %d: %w",
			offset, common.ErrTooShortToParse,
		)
	}
	a.Service = b[offset]
	offset++

	a.Payload = append([]byte(nil), b[offset:]...)
	return nil
}

// parseObjectTags decodes the run of application/context tags from offset to
// the end of b. The decoded tags do not alias b.
func parseObjectTags(b []byte, offset int) ([]objects.APDUPayload, error) {
	objs := []objects.APDUPayload{}
	for offset < len(b) {
		o, n, err := objects.DecodeTag(b[offset:])
		if err != nil {
			return nil, fmt.Errorf("tag %d at offset %d: %w", len(objs), offset, err)
		}
		objs = append(objs, o)
		offset += n
	}
	return objs, nil
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

	if a.isSegment() {
		if a.Type == ConfirmedReq {
			b[offset] = (a.MaxSeg & 0x7 << 4) | (a.MaxSize & 0xF)
			offset++
		}
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.SequenceNumber
		offset++
		b[offset] = a.ProposedWindowSize
		offset++
		b[offset] = a.Service
		offset++
		copy(b[offset:], a.Payload)
		return nil
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
	case SegmentAck:
		b[offset] = a.InvokeID
		offset++
		b[offset] = a.SequenceNumber
		offset++
		b[offset] = a.ActualWindowSize
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
		b[offset] = (a.MaxSeg & 0x7 << 4) | (a.MaxSize & 0xF)
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

// isSegment reports whether the APDU is one segment of a ConfirmedReq or a
// ComplexAck, the only PDU types that can be segmented. A segment is
// marshalled from Payload, not from Objects.
func (a *APDU) isSegment() bool {
	return a.IsSegmented() && (a.Type == ConfirmedReq || a.Type == ComplexAck)
}

// MarshalLen returns the serial length of APDU.
func (a *APDU) MarshalLen() int {
	if a.isSegment() {
		// Type and flags, invoke ID, sequence number, window size and
		// service choice; a ConfirmedReq also has its max-segments octet.
		l := 5 + len(a.Payload)
		if a.Type == ConfirmedReq {
			l++
		}
		return l
	}

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
