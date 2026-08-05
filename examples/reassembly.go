package main

import (
	"log"
	"net"
	"time"

	"github.com/jonalfarlinga/bacnet/plumbing"
)

// segmentReassemblyTTL bounds how long a reassembly can sit incomplete
// (e.g. a lost final segment) before it's dropped to avoid leaking memory.
const segmentReassemblyTTL = 30 * time.Second

type segmentKey struct {
	addr     string
	invokeID uint8
	pduType  uint8
}

type segmentBuffer struct {
	bvlc     *plumbing.BVLC
	npdu     *plumbing.NPDU
	pduType  uint8
	flags    uint8
	maxSeg   uint8
	maxSize  uint8
	invokeID uint8
	service  uint8
	data     []byte
	nextSeq  uint8
	touched  time.Time
}

// SegmentReassembler buffers segmented ConfirmedReq/ComplexAck messages
// keyed by (remote address, invoke ID, PDU type) and reconstitutes the raw
// wire bytes of the complete message once every segment has arrived.
type SegmentReassembler struct {
	active map[segmentKey]*segmentBuffer
	ttl    time.Duration
}

// NewSegmentReassembler creates a SegmentReassembler. Reassemblies left
// incomplete for longer than ttl are dropped; ttl <= 0 disables eviction.
func NewSegmentReassembler(ttl time.Duration) *SegmentReassembler {
	return &SegmentReassembler{active: make(map[segmentKey]*segmentBuffer), ttl: ttl}
}

// Feed processes one already-parsed segment. It returns (raw, complete,
// accepted). accepted is true whenever the segment warrants a Segment-ACK -
// either it advanced the reassembly (whether or not that completes it), or
// it's a retransmitted duplicate of a segment already consumed (re-acked,
// but not re-appended to avoid corrupting the buffered data). complete is
// true when this segment finishes the reassembly, in which case raw holds
// the reconstructed wire bytes of the full, unsegmented message (SEG/MOR
// cleared), ready to be re-parsed like any ordinary packet. accepted is
// false when a segment is discarded outright: an out-of-order arrival
// (sequence number ahead of what's expected), or a continuation segment for
// which no segment 0 was ever recorded - most commonly because the
// reassembly's first segment arrived before this orchestrator started
// listening. No ack is sent for discarded segments.
func (r *SegmentReassembler) Feed(remoteAddr net.Addr, bvlc *plumbing.BVLC, npdu *plumbing.NPDU, apdu *plumbing.APDU) ([]byte, bool, bool) {
	r.evictStale()

	key := segmentKey{addr: remoteAddr.String(), invokeID: apdu.InvokeID, pduType: apdu.Type}
	seq := apdu.SequenceNumber
	more := apdu.HasMoreSegments()

	if seq == 0 {
		r.active[key] = &segmentBuffer{
			bvlc:     bvlc,
			npdu:     npdu,
			pduType:  apdu.Type,
			flags:    apdu.Flags,
			maxSeg:   apdu.MaxSeg,
			maxSize:  apdu.MaxSize,
			invokeID: apdu.InvokeID,
			service:  apdu.Service,
			data:     append([]byte(nil), apdu.Payload...),
			nextSeq:  1,
			touched:  time.Now(),
		}
		if !more {
			raw, complete := r.finish(key)
			return raw, complete, true
		}
		return nil, false, true
	}

	buf, ok := r.active[key]
	if !ok {
		log.Printf(
			"discarding segment seq=%d from %s invokeID=%d: no active reassembly (its first segment likely arrived before this orchestrator started listening)",
			seq, remoteAddr, apdu.InvokeID,
		)
		return nil, false, false
	}

	if seq < buf.nextSeq {
		// A retransmitted duplicate of a segment we've already consumed.
		// This must NOT be appended again (that would duplicate/corrupt
		// buf.data) and must NOT discard progress either - it's harmless,
		// so just re-ack it and leave buf.data untouched.
		log.Printf(
			"duplicate segment seq=%d from %s invokeID=%d (already have through seq=%d): re-acking without re-appending",
			seq, remoteAddr, apdu.InvokeID, buf.nextSeq-1,
		)
		buf.touched = time.Now()
		return nil, false, true
	}

	if seq > buf.nextSeq {
		log.Printf(
			"discarding reassembly from %s invokeID=%d: out-of-order segment (expected seq=%d, got %d)",
			remoteAddr, apdu.InvokeID, buf.nextSeq, seq,
		)
		delete(r.active, key)
		return nil, false, false
	}

	buf.data = append(buf.data, apdu.Payload...)
	buf.nextSeq++
	buf.touched = time.Now()

	if !more {
		raw, complete := r.finish(key)
		return raw, complete, true
	}
	return nil, false, true
}

// finish rebuilds the raw wire bytes of the fully reassembled message from
// the buffered segment data. It clears the SEG/MOR flags so the result
// decodes as an ordinary, unsegmented message on re-parse.
func (r *SegmentReassembler) finish(key segmentKey) ([]byte, bool) {
	buf := r.active[key]
	delete(r.active, key)

	flags := buf.flags &^ 0xC // clear SEG (0x8) and MOR (0x4); keep SA
	apduBytes := make([]byte, 0, 4+len(buf.data))
	apduBytes = append(apduBytes, buf.pduType<<4|flags)
	if buf.pduType == plumbing.ConfirmedReq {
		apduBytes = append(apduBytes, buf.maxSeg<<4|buf.maxSize)
	}
	apduBytes = append(apduBytes, buf.invokeID, buf.service)
	apduBytes = append(apduBytes, buf.data...)

	bvlcLen := buf.bvlc.MarshalLen()
	npduLen := buf.npdu.MarshalLen()
	buf.bvlc.Length = uint16(bvlcLen + npduLen + len(apduBytes))

	raw := make([]byte, bvlcLen+npduLen+len(apduBytes))
	if err := buf.bvlc.MarshalTo(raw[:bvlcLen]); err != nil {
		log.Printf("reassembly: failed to marshal BVLC: %v", err)
		return nil, false
	}
	if err := buf.npdu.MarshalTo(raw[bvlcLen : bvlcLen+npduLen]); err != nil {
		log.Printf("reassembly: failed to marshal NPDU: %v", err)
		return nil, false
	}
	copy(raw[bvlcLen+npduLen:], apduBytes)

	return raw, true
}

// evictStale drops reassemblies that haven't seen a segment in longer than
// ttl, so a lost final segment doesn't leak memory forever.
func (r *SegmentReassembler) evictStale() {
	if r.ttl <= 0 {
		return
	}
	cutoff := time.Now().Add(-r.ttl)
	for key, buf := range r.active {
		if buf.touched.Before(cutoff) {
			log.Printf("dropping stale reassembly for invokeID=%d (%s): timed out after %s", buf.invokeID, key.addr, r.ttl)
			delete(r.active, key)
		}
	}
}
