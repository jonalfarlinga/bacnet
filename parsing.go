package bacnet

import (
	"errors"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

const bacnetLenMin = 8

// Parse decodes the given bytes.
func Parse(b []byte) (plumbing.BACnet, error) {

	// BVLC-only messages (BVLC-Result, Read-Foreign-Device-Table and its
	// ack) carry no NPDU/APDU, so they can be as short as 4-6 bytes -
	// shorter than bacnetLenMin, which assumes a full NPDU+APDU follows.
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
			}
		}
	}

	if len(b) < bacnetLenMin {
		return nil, fmt.Errorf(
			"parsing length %d: %v", len(b), common.ErrTooShortToParse,
		)
	}

	bvlc, parseBuf, err := normalizeForwardedBVLC(b)
	if err != nil {
		return nil, err
	}

	var npdu plumbing.NPDU
	offset := bvlc.MarshalLen()

	if err := npdu.UnmarshalBinary(parseBuf[offset:]); err != nil {
		return nil, fmt.Errorf("parsing NPDU %x: %v", parseBuf[offset:], err)
	}
	offset += npdu.MarshalLen()

	if (npdu.Control & 0x80) != 0 {
		return nil, fmt.Errorf(
			"parsing NPDU control %x: %w",
			npdu.Control,
			common.ErrNotImplemented,
		)
	}

	if len(parseBuf) <= offset {
		return nil, fmt.Errorf(
			"parsing APDU offset %d in %d bytes: %v",
			offset,
			len(parseBuf),
			common.ErrTooShortToParse,
		)
	}

	var apdu plumbing.APDU
	if err := apdu.UnmarshalBinary(parseBuf[offset:]); err != nil {
		return nil, fmt.Errorf("parsing APDU %x: %v", parseBuf[offset:], err)
	}

	return wrapParsedMessage(&bvlc, &npdu, &apdu)
}

// ParseWithFallback first attempts strict decoding with Parse and, if that fails,
// retries with a bounded lenient parser for known non-canonical routed NPDU layouts.
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

	fallbackMsg, fallbackErr := parseLenientRoutedNPDU(b)
	if fallbackErr == nil {
		return fallbackMsg, nil
	}

	return nil, fmt.Errorf("strict parse failed: %v; fallback parse failed: %v", strictErr, fallbackErr)
}

func parseLenientRoutedNPDU(b []byte) (plumbing.BACnet, error) {
	bvlc, parseBuf, err := normalizeForwardedBVLC(b)
	if err != nil {
		return nil, err
	}

	offset := bvlc.MarshalLen()
	npdu, npduLen, err := parseNPDULenient(parseBuf[offset:])
	if err != nil {
		return nil, fmt.Errorf("lenient NPDU parse %x: %v", parseBuf[offset:], err)
	}
	offset += npduLen

	if (npdu.Control & 0x80) != 0 {
		return nil, fmt.Errorf(
			"parsing NPDU control %x: %w",
			npdu.Control,
			common.ErrNotImplemented,
		)
	}

	if len(parseBuf) <= offset {
		return nil, fmt.Errorf(
			"parsing APDU offset %d in %d bytes: %v",
			offset,
			len(parseBuf),
			common.ErrTooShortToParse,
		)
	}

	var apdu plumbing.APDU
	if err := apdu.UnmarshalBinary(parseBuf[offset:]); err != nil {
		return nil, fmt.Errorf("parsing APDU %x: %v", parseBuf[offset:], err)
	}

	return wrapParsedMessage(&bvlc, &npdu, &apdu)
}

func normalizeForwardedBVLC(b []byte) (plumbing.BVLC, []byte, error) {
	var bvlc plumbing.BVLC
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return bvlc, nil, fmt.Errorf("parsing BVLC %x: %v", b, err)
	}

	parseBuf := b
	if bvlc.Function == plumbing.BVLCFuncForwardedNPDU {
		forwardedHdrOffset := bvlc.MarshalLen()
		if len(b) < forwardedHdrOffset+6 {
			return bvlc, nil, fmt.Errorf(
				"parsing BVLC forwarded header %x: %v",
				b,
				common.ErrTooShortToParse,
			)
		}

		parseBuf = make([]byte, 0, len(b)-6)
		parseBuf = append(parseBuf, b[:forwardedHdrOffset]...)
		parseBuf = append(parseBuf, b[forwardedHdrOffset+6:]...)
	}

	return bvlc, parseBuf, nil
}

func parseNPDULenient(b []byte) (plumbing.NPDU, int, error) {
	var npdu plumbing.NPDU
	if len(b) < 2 {
		return npdu, 0, fmt.Errorf("NPDU header too short")
	}

	npdu.Version = b[0]
	npdu.Control = b[1]
	offset := 2

	hasDest := (npdu.Control & 0x20) != 0
	hasSource := (npdu.Control & 0x08) != 0

	if hasDest {
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
		npdu.DADR = append([]byte(nil), b[offset:offset+int(npdu.DLEN)]...)
		offset += int(npdu.DLEN)

		if hasSource {
			canon, canonOff, canonErr := parseAfterDestCanonical(npdu, b, offset)
			if canonErr == nil {
				return canon, canonOff, nil
			}

			alt, altOff, altErr := parseAfterDestSourceThenHop(npdu, b, offset)
			if altErr == nil {
				return alt, altOff, nil
			}

			return npdu, 0, fmt.Errorf("canonical: %v; source-then-hop: %v", canonErr, altErr)
		}

		if len(b) <= offset {
			return npdu, 0, fmt.Errorf("NPDU hop too short")
		}
		npdu.Hop = b[offset]
		offset++
		return npdu, offset, nil
	}

	if hasSource {
		npduWithSource, nextOffset, err := parseSourceFields(npdu, b, offset)
		if err != nil {
			return npdu, 0, err
		}
		return npduWithSource, nextOffset, nil
	}

	return npdu, offset, nil
}

func parseAfterDestCanonical(npdu plumbing.NPDU, b []byte, offset int) (plumbing.NPDU, int, error) {
	if len(b) <= offset {
		return npdu, 0, fmt.Errorf("NPDU hop too short")
	}
	npdu.Hop = b[offset]
	offset++
	return parseSourceFields(npdu, b, offset)
}

func parseAfterDestSourceThenHop(npdu plumbing.NPDU, b []byte, offset int) (plumbing.NPDU, int, error) {
	npduWithSource, nextOffset, err := parseSourceFields(npdu, b, offset)
	if err != nil {
		return npdu, 0, err
	}
	if len(b) <= nextOffset {
		return npdu, 0, fmt.Errorf("NPDU trailing hop too short")
	}
	npduWithSource.Hop = b[nextOffset]
	nextOffset++
	return npduWithSource, nextOffset, nil
}

func parseSourceFields(npdu plumbing.NPDU, b []byte, offset int) (plumbing.NPDU, int, error) {
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
	npdu.SADR = append([]byte(nil), b[offset:offset+int(npdu.SLEN)]...)
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
		case services.ServiceUnconfirmedCOVNotification:
			bacnet = &services.COVNotification{BVLC: bvlc, NPDU: npdu, APDU: apdu}
		}
	case plumbing.ConfirmedReq:
		if apdu.IsSegmented() {
			// Every segment (including segment 0) is routed uniformly
			// through SegmentedAPDU: a continuation segment has no service
			// choice of its own to switch on, since only segment 0 carries
			// one. Callers reassemble the full byte stream and re-parse it
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
