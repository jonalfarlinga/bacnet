package bacnet

import (
	"errors"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// bacnetLenMin is the shortest frame with an NPDU: a BVLC header, an NPDU
// without addresses, and a one-byte network layer message.
const bacnetLenMin = 7

// Parse decodes the given bytes, which must be one whole BACnet/IP datagram
// starting at the BVLC header. A frame captured without one goes to ParseNPDU.
func Parse(b []byte) (plumbing.BACnet, error) {
	if len(b) > 0 && b[0] != plumbing.BVLCType {
		return nil, fmt.Errorf(
			"parsing BVLC type %#x, want %#x: %w", b[0], plumbing.BVLCType, common.ErrInvalidData,
		)
	}

	// BVLC-only messages (BVLC-Result, Register-Foreign-Device, and the
	// table reads and their acks) carry no NPDU/APDU, so they can be as short as 4-6 bytes -
	// shorter than bacnetLenMin, which assumes an NPDU follows.
	// Peek at the BVLC function code before applying that length floor.
	if len(b) >= 4 {
		var bvlcPeek plumbing.BVLC
		if err := bvlcPeek.UnmarshalBinary(b); err == nil {
			switch bvlcPeek.Function {
			case plumbing.BVLCFuncResult:
				r := &plumbing.BVLCResult{}
				if err := r.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing BVLCResult %x: %v", b, err)
				}
				return r, nil
			case plumbing.BVLCFuncReadForeignDeviceTable:
				r := &plumbing.ReadForeignDeviceTable{}
				if err := r.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing ReadForeignDeviceTable %x: %v", b, err)
				}
				return r, nil
			case plumbing.BVLCFuncReadForeignDeviceTableAck:
				a := &plumbing.ReadForeignDeviceTableAck{}
				if err := a.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing ReadForeignDeviceTableAck %x: %v", b, err)
				}
				return a, nil
			case plumbing.BVLCFuncRegisterForeignDevice:
				r := &plumbing.RegisterForeignDevice{}
				if err := r.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing RegisterForeignDevice %x: %w", b, err)
				}
				return r, nil
			case plumbing.BVLCFuncReadBroadcastDistributionTable:
				r := &plumbing.ReadBroadcastDistributionTable{}
				if err := r.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing ReadBroadcastDistributionTable %x: %w", b, err)
				}
				return r, nil
			case plumbing.BVLCFuncReadBroadcastDistributionTableAck:
				a := &plumbing.ReadBroadcastDistributionTableAck{}
				if err := a.UnmarshalBinary(b); err != nil {
					return nil, fmt.Errorf("parsing ReadBroadcastDistributionTableAck %x: %w", b, err)
				}
				return a, nil
			}
		}
	}

	if len(b) < bacnetLenMin {
		return nil, fmt.Errorf(
			"parsing length %d: %w", len(b), common.ErrTooShortToParse,
		)
	}

	var bvlc plumbing.BVLC
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("parsing BVLC %x: %w", b, err)
	}
	offset := bvlc.MarshalLen()

	var npdu plumbing.NPDU
	if err := npdu.UnmarshalBinary(b[offset:]); err != nil {
		return nil, fmt.Errorf("parsing NPDU %x: %w", b[offset:], err)
	}
	offset += npdu.MarshalLen()

	if (npdu.Control & plumbing.NetworkLayerMessage) != 0 {
		m := &plumbing.NetworkMessage{}
		if err := m.UnmarshalBinary(b); err != nil {
			return nil, fmt.Errorf("parsing network layer message %x: %w", b, err)
		}
		return m, nil
	}

	return parseAPDU(&bvlc, &npdu, b[offset:])
}

// ParseNPDU decodes a message that starts at the NPDU, with no BVLC in front
// of it: what a capture shows for BACnet over 802.3 with an LLC header. b
// must end where the NPDU ends, so padding that the data link added has to
// be cut off first.
//
// The message returned has an Original-Unicast BVLC header added, since the
// message types cannot be marshalled without one. The NPDU bytes are what
// its MarshalBinary returns after the first 4.
func ParseNPDU(b []byte) (plumbing.BACnet, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	total := bvlc.MarshalLen() + len(b)
	if total > 0xFFFF {
		return nil, fmt.Errorf("parsing NPDU of %d bytes: %w", len(b), common.ErrTooBigValue)
	}
	bvlc.Length = uint16(total)

	framed := make([]byte, total)
	if err := bvlc.MarshalTo(framed); err != nil {
		return nil, err
	}
	copy(framed[bvlc.MarshalLen():], b)
	return Parse(framed)
}

// ParseWithFallback first attempts strict decoding with Parse and, if that fails,
// retries reading the NPDU in the layout that versions of this library wrote
// before the field order was corrected (see parseNPDULegacy).
// It skips the fallback when the strict parse already reached a fully-decoded NPDU/APDU
// and only failed because the service or PDU type isn't implemented: a different NPDU
// parsing strategy cannot change which service byte the APDU carries.
func ParseWithFallback(b []byte) (plumbing.BACnet, error) {
	strictMsg, strictErr := Parse(b)
	if strictErr == nil {
		return strictMsg, nil
	}

	if errors.Is(strictErr, common.ErrNotImplemented) {
		return nil, strictErr
	}

	fallbackMsg, fallbackErr := parseLegacyRoutedNPDU(b)
	if fallbackErr == nil {
		return fallbackMsg, nil
	}

	return nil, fmt.Errorf("strict parse failed: %w; fallback parse failed: %v", strictErr, fallbackErr)
}

func parseLegacyRoutedNPDU(b []byte) (plumbing.BACnet, error) {
	if len(b) > 0 && b[0] != plumbing.BVLCType {
		return nil, fmt.Errorf(
			"parsing BVLC type %#x, want %#x: %w", b[0], plumbing.BVLCType, common.ErrInvalidData,
		)
	}

	var bvlc plumbing.BVLC
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("parsing BVLC %x: %w", b, err)
	}
	offset := bvlc.MarshalLen()

	npdu, npduLen, err := parseNPDULegacy(b[offset:])
	if err != nil {
		return nil, fmt.Errorf("legacy NPDU parse %x: %v", b[offset:], err)
	}
	offset += npduLen

	// Those versions never wrote a network layer message.
	if (npdu.Control & plumbing.NetworkLayerMessage) != 0 {
		return nil, fmt.Errorf(
			"parsing NPDU control %x in the legacy layout: %w", npdu.Control, common.ErrInvalidData,
		)
	}

	return parseAPDU(&bvlc, &npdu, b[offset:])
}

func parseAPDU(bvlc *plumbing.BVLC, npdu *plumbing.NPDU, b []byte) (plumbing.BACnet, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("parsing APDU: no bytes after the NPDU: %w", common.ErrTooShortToParse)
	}

	var apdu plumbing.APDU
	if err := apdu.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("parsing APDU %x: %w", b, err)
	}

	return wrapParsedMessage(bvlc, npdu, &apdu)
}

// parseNPDULegacy reads an NPDU that has both a destination and a source,
// with the hop count ahead of the source instead of after it. The standard
// puts the hop count last, and plumbing.NPDU reads and writes it there. This
// library wrote it ahead of the source until that was corrected, so a peer
// running an older version still sends this layout. It returns the NPDU and
// the number of bytes it occupies.
func parseNPDULegacy(b []byte) (plumbing.NPDU, int, error) {
	var npdu plumbing.NPDU
	if len(b) < 2 {
		return npdu, 0, fmt.Errorf("NPDU header too short")
	}

	npdu.Version = b[0]
	npdu.Control = b[1]
	offset := 2

	if (npdu.Control&0x20) == 0 || (npdu.Control&0x08) == 0 {
		return npdu, 0, fmt.Errorf("NPDU control %x has only the standard layout", npdu.Control)
	}

	if len(b) < offset+3 {
		return npdu, 0, fmt.Errorf("NPDU destination fields too short")
	}
	npdu.DNET = uint16(b[offset])<<8 | uint16(b[offset+1])
	offset += 2
	npdu.DLEN = b[offset]
	offset++

	if len(b) < offset+int(npdu.DLEN) {
		return npdu, 0, fmt.Errorf("NPDU DADR too short")
	}
	if npdu.DLEN > 0 {
		npdu.DADR = append([]byte(nil), b[offset:offset+int(npdu.DLEN)]...)
	}
	offset += int(npdu.DLEN)

	if len(b) <= offset {
		return npdu, 0, fmt.Errorf("NPDU hop too short")
	}
	npdu.Hop = b[offset]
	offset++

	if len(b) < offset+3 {
		return npdu, 0, fmt.Errorf("NPDU source fields too short")
	}
	npdu.SNET = uint16(b[offset])<<8 | uint16(b[offset+1])
	offset += 2
	npdu.SLEN = b[offset]
	offset++

	if len(b) < offset+int(npdu.SLEN) {
		return npdu, 0, fmt.Errorf("NPDU SADR too short")
	}
	if npdu.SLEN > 0 {
		npdu.SADR = append([]byte(nil), b[offset:offset+int(npdu.SLEN)]...)
	}
	offset += int(npdu.SLEN)

	return npdu, offset, nil
}

func wrapParsedMessage(bvlc *plumbing.BVLC, npdu *plumbing.NPDU, apdu *plumbing.APDU) (plumbing.BACnet, error) {
	var bacnet plumbing.BACnet

	switch apdu.Type {
	case plumbing.UnConfirmedReq:
		switch apdu.Service {
		case services.ServiceUnconfirmedWhoIs:
			bacnet = &services.UnconfirmedWhoIs{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceUnconfirmedIAm:
			bacnet = &services.UnconfirmedIAm{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceUnconfirmedWhoHas:
			bacnet = &services.UnconfirmedWhoHas{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceUnconfirmedCOVNotification:
			bacnet = &services.COVNotification{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		}
	case plumbing.ConfirmedReq:
		if apdu.IsSegmented() {
			// Every segment (including segment 0) is routed uniformly
			// through SegmentedAPDU: every segment repeats the service
			// choice, but a lone segment's parameters are incomplete.
			// Callers reassemble the full byte stream and re-parse it
			// to get a concrete, fully-decoded service type.
			bacnet = &services.SegmentedAPDU{BVLC: bvlc, NPDU: npdu, APDU: apdu}
			break
		}
		switch apdu.Service {
		case services.ServiceConfirmedCOVNotification:
			bacnet = &services.COVNotification{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceConfirmedReadProperty, services.ServiceConfirmedReadPropMultiple:
			bacnet = &services.ConfirmedReadProperty{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceConfirmedReadRange:
			bacnet = &services.ConfirmedReadRange{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceConfirmedSubscribeCOV:
			bacnet = &services.ConfirmedCOV{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		case services.ServiceConfirmedWriteProperty:
			bacnet = &services.ConfirmedWriteProperty{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		}
	case plumbing.ComplexAck:
		bacnet = &services.ComplexACK{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	case plumbing.SimpleAck:
		bacnet = &services.SimpleACK{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	case plumbing.Error:
		bacnet = &services.Error{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	case plumbing.Reject:
		bacnet = &services.Reject{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	case plumbing.Abort:
		bacnet = &services.Abort{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	case plumbing.SegmentAck:
		bacnet = &services.SegmentACK{BVLC: bvlc, NPDU: npdu, APDU: apdu}
	default:
		return nil, fmt.Errorf("parsing APDU type %x: %w", apdu.Type, common.ErrNotImplemented)
	}

	if bacnet == nil {
		return nil, fmt.Errorf("parsing service %x: %w", uint16(apdu.Type)<<8|uint16(apdu.Service), common.ErrNotImplemented)
	}

	return bacnet, nil
}
