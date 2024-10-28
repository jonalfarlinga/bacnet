package bacnet

import (
	"log"

	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

const (
	DEFAULT_ACCEPTED_SIZE        = 1024
	DEFAULT_SEGMENTATION_SUPPORT = 0x3 // No segmentation
)

func NewWhois() ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncBroadcast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	u, _ := services.NewUnconfirmedWhoIs(bvlc, npdu)
	return u.MarshalBinary()
}

func NewIAm(deviceId uint32, vendorId uint16) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncBroadcast)

	npdu := plumbing.NewNPDU(false, true, false, false)
	npdu.DNET = 0xFFFF
	npdu.DLEN = 0
	npdu.Hop = 0xFF

	u, _ := services.NewUnconfirmedIAm(bvlc, npdu)

	u.APDU.Objects = services.IAmObjects(deviceId,
		DEFAULT_ACCEPTED_SIZE, DEFAULT_SEGMENTATION_SUPPORT, vendorId)
	u.SetLength()

	return u.MarshalBinary()
}

func NewCACK(service uint8, objectType uint16, instN uint32, propertyId uint8, value float32) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)

	c, _ := services.NewComplexACK(bvlc, npdu)

	c.APDU.Service = service
	c.APDU.InvokeID = 1
	c.APDU.Objects = services.ComplexACKObjects(objectType, instN, propertyId, value)

	c.SetLength()

	return c.MarshalBinary()
}

func NewSACK(service uint8) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)

	s, _ := services.NewSimpleACK(bvlc, npdu)

	s.APDU.Service = service
	s.APDU.InvokeID = 1

	s.SetLength()

	return s.MarshalBinary()
}

func NewError(service, errorClass, errorCode uint8) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)

	e, _ := services.NewError(bvlc, npdu)

	e.APDU.Service = service
	e.APDU.InvokeID = 1
	e.APDU.Objects = services.ErrorObjects(errorClass, errorCode)

	e.SetLength()

	return e.MarshalBinary()
}

func NewReadProperty(objectType uint16, instanceNumber uint32, propertyId uint8) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, true)

	c, _ := services.NewConfirmedReadProperty(bvlc, npdu)

	c.APDU.Service = services.ServiceConfirmedReadProperty
	c.APDU.MaxSize = 5
	c.APDU.InvokeID = 1
	c.APDU.Objects = services.ConfirmedReadPropertyObjects(objectType, instanceNumber, propertyId)

	c.SetLength()

	return c.MarshalBinary()
}

func NewReadRange(objectType uint16, instanceNumber uint32, propertyId uint8, rangeStart uint16, length int32) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, true)

	c, _ := services.NewConfirmedReadRange(bvlc, npdu)

	c.APDU.Service = services.ServiceConfirmedReadRange
	c.APDU.MaxSeg = 7
	c.APDU.MaxSize = 5
	c.APDU.InvokeID = 1
	c.APDU.Flags = 2
	c.APDU.Objects = services.ConfirmedReadRangeObjects(objectType, instanceNumber, propertyId, rangeStart, length)

	log.Printf("NewReadRange\n\tbvlc: Type: %d, Function: %d\n\tnpdu: DNET: %d, DLEN: %d, Hop: %d\n\tapdu: Service: %d, MaxSize: %d, InvokeID: %d\n\tobjects: %v\n",
		c.BVLC.Type, c.BVLC.Function, c.NPDU.DNET, c.NPDU.DLEN, c.NPDU.Hop, c.APDU.Service, c.APDU.MaxSize, c.APDU.InvokeID, c.APDU.Objects)
	c.SetLength()

	return c.MarshalBinary()
}

func NewWriteProperty(objectType uint16, instanceNumber uint32, propertyId uint8, value float32) ([]byte, error) {
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, true)

	c, _ := services.NewConfirmedWriteProperty(bvlc, npdu)

	c.APDU.Service = services.ServiceConfirmedWriteProperty
	c.APDU.MaxSize = 5
	c.APDU.InvokeID = 1
	c.APDU.Objects = services.ConfirmedWritePropertyObjects(objectType, instanceNumber, propertyId, value)

	c.SetLength()

	return c.MarshalBinary()
}
