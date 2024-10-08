package bacnet

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/pkg/errors"
)

const bacnetLenMin = 8

func combine(t, s uint8) uint16 {
	return uint16(t)<<8 | uint16(s)
}

// Parse decodes the given bytes.
func Parse(b []byte) (plumbing.BACnet, error) {
	if len(b) < bacnetLenMin {
		return nil, errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("Parsing length %d", len(b)),
		)
	}

	var bvlc plumbing.BVLC
	var npdu plumbing.NPDU
	var bacnet plumbing.BACnet

	offset := 0

	if err := bvlc.UnmarshalBinary(b); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Parsing BVLC %x", b))
	}
	offset += bvlc.MarshalLen()
	log.Printf("BVLC %x\n", b[:offset])
	if err := npdu.UnmarshalBinary(b[offset:]); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Parsing NPDU %x", b[offset:]))
	}
	offset += npdu.MarshalLen()
	log.Printf("NPDU %x\n", b[:offset])
	log.Printf("APDU %x\n", b[offset:])
	var c uint16
	switch b[offset] >> 4 & 0xFF {
	case plumbing.UnConfirmedReq:
		c = combine(b[offset], b[offset+1])
	case plumbing.ConfirmedReq:
		c = combine(b[offset], b[offset+3]) // We need to skip the PDU flags and the InvokeID
	case plumbing.ComplexAck, plumbing.SimpleAck, plumbing.Error:
		c = combine(b[offset], 0) // We need to skip the PDU flags and the InvokeID
	}
	fmt.Printf("switch: %x\n", c)
	switch c {
	case combine(plumbing.UnConfirmedReq<<4, services.ServiceUnconfirmedWhoIs):
		bacnet = services.NewUnconfirmedWhoIs(&bvlc, &npdu)
	case combine(plumbing.UnConfirmedReq<<4, services.ServiceUnconfirmedIAm):
		bacnet = services.NewUnconfirmedIAm(&bvlc, &npdu)
	case combine(plumbing.ConfirmedReq<<4, services.ServiceConfirmedReadProperty):
		bacnet = services.NewConfirmedReadProperty(&bvlc, &npdu)
	case combine(plumbing.ConfirmedReq<<4, services.ServiceConfirmedWriteProperty):
		bacnet = services.NewConfirmedWriteProperty(&bvlc, &npdu)
	case combine(plumbing.ComplexAck<<4, 0):
		bacnet = services.NewComplexACK(&bvlc, &npdu)
	case combine(plumbing.SimpleAck<<4, 0):
		bacnet = services.NewSimpleACK(&bvlc, &npdu)
	case combine(plumbing.Error<<4, 0):
		bacnet = services.NewError(&bvlc, &npdu)
	default:
		return nil, errors.Wrap(
			common.ErrNotImplemented,
			fmt.Sprintf("Parsing service: %x", c),
		)
	}
	if err := bacnet.UnmarshalBinary(b); err != nil {
		return nil, errors.Wrap(
			err,
			fmt.Sprintf("Parsing BACnet %x", b[offset:]),
		)
	}

	return bacnet, nil
}
