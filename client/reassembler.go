package client

import (
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
	nextSeq  uint8
	touched  time.Time
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
			return r.finish(key)
		}
		return FeedResult{Accepted: true}
	}

	buf, ok := r.active[key]
	if !ok {
		return FeedResult{Dropped: fmt.Sprintf(
			"segment %d of invoke ID %d from %s: no reassembly under way, its first segment was not seen",
			seq, apdu.InvokeID, peer)}
	}
	if seq < buf.nextSeq {
		// A retransmitted duplicate of a segment already taken: acknowledged
		// again, not appended again.
		buf.touched = time.Now()
		return FeedResult{Accepted: true}
	}
	if seq > buf.nextSeq {
		delete(r.active, key)
		return FeedResult{Dropped: fmt.Sprintf(
			"reassembly of invoke ID %d from %s: segment %d arrived where %d was expected",
			apdu.InvokeID, peer, seq, buf.nextSeq)}
	}

	buf.data = append(buf.data, apdu.Payload...)
	buf.nextSeq++
	buf.touched = time.Now()
	if !more {
		return r.finish(key)
	}
	return FeedResult{Accepted: true}
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
