package client

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jonalfarlinga/bacnet/plumbing"
)

// Reassembler puts segmented ConfirmedReq and ComplexAck messages back
// together for a program that reads raw datagrams itself, as the
// orchestrator example does. The client's own reads reassemble their replies
// inline and do not use it.
//
// Segments are kept by peer, invoke ID and PDU type until the last one
// arrives; the result is the wire bytes of the whole message with the
// segmentation flags cleared, ready to be parsed like any other datagram.
type Reassembler struct {
	active map[segmentKey]*segmentBuffer
	ttl    time.Duration
}

// FeedResult is what became of one segment.
type FeedResult struct {
	// Raw is the whole message, once Complete.
	Raw      []byte
	Complete bool
	// Accepted says the segment is worth a Segment-ACK: it advanced the
	// reassembly, or it is a duplicate of one that did.
	Accepted bool
	// Dropped says why the segment, or the reassembly it belonged to, was
	// discarded; empty otherwise.
	Dropped string
}

type segmentKey struct {
	peer     string
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
	// lastLen is the length of the last segment taken, the one a device
	// sends again when its Segment-ACK is lost.
	lastLen int
	nextSeq uint8
	touched time.Time
}

// NewReassembler returns a Reassembler that drops a reassembly left
// incomplete for longer than ttl, as one whose last segment was lost. A ttl
// of 0 or less keeps them forever.
func NewReassembler(ttl time.Duration) *Reassembler {
	return &Reassembler{active: map[segmentKey]*segmentBuffer{}, ttl: ttl}
}

// Feed takes one parsed segment from a peer, named however the caller tells
// peers apart.
func (r *Reassembler) Feed(peer string, bvlc *plumbing.BVLC, npdu *plumbing.NPDU, apdu *plumbing.APDU) FeedResult {
	r.evictStale()

	key := segmentKey{peer: peer, invokeID: apdu.InvokeID, pduType: apdu.Type}
	seq := apdu.SequenceNumber
	more := apdu.HasMoreSegments()
	buf, ok := r.active[key]
	if seq == 0 && ok && buf.nextSeq != 0 && !buf.resent(apdu.Payload) {
		// A first segment while a reassembly is under way, and not the one
		// already taken sent again: a new message on the invoke ID of a
		// reassembly that never completed. It starts over.
		ok = false
	}

	if seq == 0 && !ok {
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
			lastLen:  len(apdu.Payload),
			nextSeq:  1,
			touched:  time.Now(),
		}
		if !more {
			return r.finish(key)
		}
		return FeedResult{Accepted: true}
	}

	if !ok {
		return FeedResult{Dropped: fmt.Sprintf(
			"segment %d of invoke ID %d from %s: no reassembly under way, its first segment was not seen",
			seq, apdu.InvokeID, peer)}
	}
	// Sequence numbers wrap at 256. A recently received sequence behind
	// nextSeq is a duplicate even when its numeric value is larger.
	if seq != buf.nextSeq && uint8(buf.nextSeq-seq) < 128 {
		// A retransmitted duplicate of a segment already taken: acknowledged
		// again, not appended again.
		buf.touched = time.Now()
		return FeedResult{Accepted: true}
	}
	if seq != buf.nextSeq {
		delete(r.active, key)
		return FeedResult{Dropped: fmt.Sprintf(
			"reassembly of invoke ID %d from %s: segment %d arrived where %d was expected",
			apdu.InvokeID, peer, seq, buf.nextSeq)}
	}

	buf.data = append(buf.data, apdu.Payload...)
	buf.lastLen = len(apdu.Payload)
	buf.nextSeq++
	buf.touched = time.Now()
	if !more {
		return r.finish(key)
	}
	return FeedResult{Accepted: true}
}

// resent reports whether a first segment is the one the buffer holds, sent
// again. A device sends its first segment again only while it waits for the
// Segment-ACK of that one, so nothing after it has been taken yet; the
// sequence number wraps, so the segment is the last one taken.
func (b *segmentBuffer) resent(payload []byte) bool {
	return b.nextSeq == 1 && bytes.Equal(payload, b.data[len(b.data)-b.lastLen:])
}

// finish rebuilds the wire bytes of the whole message from the segments,
// with the SEG and MOR flags cleared so that it parses as an ordinary one.
func (r *Reassembler) finish(key segmentKey) FeedResult {
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
	if bvlcLen+npduLen+len(apduBytes) > 0xFFFF {
		return FeedResult{Accepted: true, Dropped: "the reassembled message exceeds the BVLC length limit"}
	}
	buf.bvlc.Length = uint16(bvlcLen + npduLen + len(apduBytes))

	raw := make([]byte, bvlcLen+npduLen+len(apduBytes))
	if err := buf.bvlc.MarshalTo(raw[:bvlcLen]); err != nil {
		return FeedResult{Accepted: true, Dropped: "the reassembled message could not be marshalled: " + err.Error()}
	}
	if err := buf.npdu.MarshalTo(raw[bvlcLen : bvlcLen+npduLen]); err != nil {
		return FeedResult{Accepted: true, Dropped: "the reassembled message could not be marshalled: " + err.Error()}
	}
	copy(raw[bvlcLen+npduLen:], apduBytes)
	return FeedResult{Raw: raw, Complete: true, Accepted: true}
}

// evictStale drops the reassemblies that have had no segment for longer
// than the TTL.
func (r *Reassembler) evictStale() {
	if r.ttl <= 0 {
		return
	}
	cutoff := time.Now().Add(-r.ttl)
	for key, buf := range r.active {
		if buf.touched.Before(cutoff) {
			delete(r.active, key)
		}
	}
}
