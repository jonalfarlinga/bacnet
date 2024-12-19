package test_utils

import (
	"testing"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func TestParseWhois(t *testing.T, Parse func([]byte) (plumbing.BACnet, error)) {
	result, err := Parse([]byte{
		0x81, 0x0a, 0x00, 0x0e, 0x01, 0x00, 0x10,
		0x08, 0x09, 0x00, 0x1b, 0x3f, 0xff, 0xff,
	})
	if err != nil {
		t.Errorf("Error parsing: %v", err)
	}
	resultWhois, ok := result.(*services.UnconfirmedWhoIs)
	if !ok {
		t.Errorf("Didn't get Whois: %v", result)
	}
	rangeLow := &objects.Object{
		TagNumber: 0,
		TagClass:  true,
		Data:      []byte{0x00},
		Length:    1,
	}
	rangeHigh := &objects.Object{
		TagNumber: 1,
		TagClass:  true,
		Data:      []byte{0x3f, 0xff, 0xff},
		Length:    3,
	}
	AssertEqual(t, uint8(0x81), resultWhois.BVLC.Type)
	AssertEqual(t, uint8(0x0a), resultWhois.BVLC.Function)
	AssertEqual(t, uint16(0x0e), resultWhois.BVLC.Length)
	AssertEqual(t, uint8(0x01), resultWhois.NPDU.Version)
	AssertEqual(t, plumbing.UnConfirmedReq, resultWhois.APDU.Type)
	AssertEqual(t, services.ServiceUnconfirmedWhoIs, resultWhois.APDU.Service)
	AssertEqual(t, 2, len(resultWhois.APDU.Objects))
	AssertEqualTag(t, rangeLow, resultWhois.APDU.Objects[0])
	AssertEqualTag(t, rangeHigh, resultWhois.APDU.Objects[1])
}

func TestParseReadProperty(t *testing.T, Parse func([]byte) (plumbing.BACnet, error)) {
	result, err := Parse([]byte{
		0x81, 0x0a, 0x00, 0x17, 0x01, 0x00, 0x30, 0x01, 0x0c, 0x0c, 0x02,
		0x00, 0x00, 0x65, 0x19, 0x4b, 0x3e, 0xc4, 0x02, 0x00, 0x00, 0x65, 0x3f,
	})
	if err != nil {
		t.Errorf("Error parsing: %v", err)
	}
	resultReadProp, ok := result.(*services.ComplexACK)
	if !ok {
		t.Errorf("Didn't get ConfirmedReadProperty: %v", result)
	}
	objectId := &objects.Object{
		TagNumber: 0,
		TagClass:  true,
		Data:      []byte{0x02, 0x00, 0x00, 0x65},
		Length:    4,
	}
	propId := &objects.Object{
		TagNumber: 1,
		TagClass:  true,
		Data:      []byte{0x4b},
		Length:    1,
	}
	propValue := &objects.Object{
		TagNumber: 12,
		TagClass:  false,
		Data:      []byte{0x02, 0x00, 0x00, 0x65},
		Length:    4,
	}
	open := &objects.Object{
		TagNumber: 3,
		TagClass:  true,
		Data: []byte{},
		Length: 6,
	}
	close := &objects.Object{
		TagNumber: 3,
		TagClass:  true,
		Data: []byte{},
		Length: 7,
	}
	AssertEqual(t, uint8(0x81), resultReadProp.BVLC.Type)
	AssertEqual(t, uint8(0x0a), resultReadProp.BVLC.Function)
	AssertEqual(t, uint16(0x17), resultReadProp.BVLC.Length)
	AssertEqual(t, uint8(0x01), resultReadProp.NPDU.Version)
	AssertEqual(t, plumbing.ComplexAck, resultReadProp.APDU.Type)
	AssertEqual(t, services.ServiceConfirmedReadProperty, resultReadProp.APDU.Service)
	AssertEqual(t, 5, len(resultReadProp.APDU.Objects))
	AssertEqualTag(t, objectId, resultReadProp.APDU.Objects[0])
	AssertEqualTag(t, propId, resultReadProp.APDU.Objects[1])
	AssertEqualTag(t, open, resultReadProp.APDU.Objects[2])
	AssertEqualTag(t, propValue, resultReadProp.APDU.Objects[3])
	AssertEqualTag(t, close, resultReadProp.APDU.Objects[4])
}

func TestParseIam(t *testing.T, Parse func([]byte) (plumbing.BACnet, error)) {
	result, err := Parse([]byte{
		0x81, 0x0a, 0x00, 0x14, 0x01, 0x00, 0x10, 0x00, 0xc4, 0x02,
		0x00, 0x00, 0x65, 0x22, 0x05, 0xc4, 0x91, 0x00, 0x21, 0x07,
	})
	if err != nil {
		t.Errorf("Error parsing: %v", err)
	}
	resultIam, ok := result.(*services.UnconfirmedIAm)
	if !ok {
		t.Errorf("Didn't get IAm: %v", result)
	}
	oid := &objects.Object{
		TagNumber: 12,
		TagClass:  false,
		Data:      []byte{0x02, 0x00, 0x00, 0x65},
		Length:    4,
	}
	maxAPDU := &objects.Object{
		TagNumber: 2,
		TagClass:  false,
		Data:      []byte{0x05, 0xc4},
		Length:    2,
	}
	seg := &objects.Object{
		TagNumber: 9,
		TagClass:  false,
		Data:      []byte{0x00},
		Length:    1,
	}
	vendor := &objects.Object{
		TagNumber: 2,
		TagClass:  false,
		Data:      []byte{0x07},
		Length:    1,
	}
	AssertEqual(t, uint8(0x81), resultIam.BVLC.Type)
	AssertEqual(t, uint8(0x0a), resultIam.BVLC.Function)
	AssertEqual(t, uint16(0x14), resultIam.BVLC.Length)
	AssertEqual(t, uint8(0x01), resultIam.NPDU.Version)
	AssertEqual(t, plumbing.UnConfirmedReq, resultIam.APDU.Type)
	AssertEqual(t, services.ServiceUnconfirmedIAm, resultIam.APDU.Service)
	AssertEqual(t, 4, len(resultIam.APDU.Objects))
	AssertEqualTag(t, oid, resultIam.APDU.Objects[0])
	AssertEqualTag(t, maxAPDU, resultIam.APDU.Objects[1])
	AssertEqualTag(t, seg, resultIam.APDU.Objects[2])
	AssertEqualTag(t, vendor, resultIam.APDU.Objects[3])

}

func TestParseReadPropertyMultiple(t *testing.T, Parse func([]byte) (plumbing.BACnet, error)) {
	result, err := Parse([]byte{
		0x81, 0x0a, 0x00, 0x2f, 0x01, 0x00, 0x30, 0x01, 0x0e, 0x0c, 0x02, 0x00,
		0x00, 0x65, 0x1e, 0x29, 0x4b, 0x4e, 0xc4, 0x02, 0x00, 0x00, 0x65, 0x4f,
		0x29, 0x4d, 0x4e, 0x75, 0x10, 0x00, 0x55, 0x4f, 0x2d, 0x41, 0x6e, 0x61,
		0x6c, 0x79, 0x73, 0x74, 0x2d, 0x54, 0x65, 0x73, 0x74, 0x4f, 0x1f,
	})
	if err != nil {
		t.Errorf("Error parsing: %v", err)
	}
	resultReadPropMultiple, ok := result.(*services.ComplexACK)
	if !ok {
		t.Errorf("Didn't get ConfirmedReadPropertyMultiple: %v", result)
	}
	oid := &objects.Object{
		TagNumber: 0,
		TagClass:  true,
		Data:      []byte{0x02, 0x00, 0x00, 0x65},
		Length:    4,
	}
	open1 := &objects.Object{
		TagNumber: 1,
		TagClass:  true,
		Data:      []byte{},
		Length:    6,
	}
	close1 := &objects.Object{
		TagNumber: 1,
		TagClass:  true,
		Data:      []byte{},
		Length:    7,
	}
	open4 := &objects.Object{
		TagNumber: 4,
		TagClass:  true,
		Data:      []byte{},
		Length:    6,
	}
	close4 := &objects.Object{
		TagNumber: 4,
		TagClass:  true,
		Data:      []byte{},
		Length:    7,
	}
	pid1 := &objects.Object{
		TagNumber: 2,
		TagClass:  true,
		Data:      []byte{0x4b},
		Length:    1,
	}
	pid2 := &objects.Object{
		TagNumber: 2,
		TagClass:  true,
		Data:      []byte{0x4d},
		Length:    1,
	}
	objectId := &objects.Object{
		TagNumber: 12,
		TagClass:  false,
		Data:      []byte{0x02, 0x00, 0x00, 0x65},
		Length:    4,
	}
	objectName:= &objects.Object{
		TagNumber: 7,
		TagClass:  false,
		Data: []byte{0x00, 0x55, 0x4f, 0x2d, 0x41, 0x6e, 0x61, 0x6c, 0x79, 0x73,
			0x74, 0x2d, 0x54, 0x65, 0x73, 0x74},
		Length: 16,
	}
	AssertEqual(t, uint8(0x81), resultReadPropMultiple.BVLC.Type)
	AssertEqual(t, uint8(0x0a), resultReadPropMultiple.BVLC.Function)
	AssertEqual(t, uint16(0x2f), resultReadPropMultiple.BVLC.Length)
	AssertEqual(t, uint8(0x01), resultReadPropMultiple.NPDU.Version)
	AssertEqual(t, plumbing.ComplexAck, resultReadPropMultiple.APDU.Type)
	AssertEqual(t, services.ServiceConfirmedReadPropMultiple, resultReadPropMultiple.APDU.Service)
	AssertEqual(t, 11, len(resultReadPropMultiple.APDU.Objects))
	AssertEqualTag(t, oid, resultReadPropMultiple.APDU.Objects[0])
	AssertEqualTag(t, open1, resultReadPropMultiple.APDU.Objects[1])
	AssertEqualTag(t, pid1, resultReadPropMultiple.APDU.Objects[2])
	AssertEqualTag(t, open4, resultReadPropMultiple.APDU.Objects[3])
	AssertEqualTag(t, objectId, resultReadPropMultiple.APDU.Objects[4])
	AssertEqualTag(t, close4, resultReadPropMultiple.APDU.Objects[5])
	AssertEqualTag(t, pid2, resultReadPropMultiple.APDU.Objects[6])
	AssertEqualTag(t, open4, resultReadPropMultiple.APDU.Objects[7])
	AssertEqualTag(t, objectName, resultReadPropMultiple.APDU.Objects[8])
	AssertEqualTag(t, close4, resultReadPropMultiple.APDU.Objects[9])
	AssertEqualTag(t, close1, resultReadPropMultiple.APDU.Objects[10])
}
