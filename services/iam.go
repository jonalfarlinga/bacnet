package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// UnconfirmedIAm is a BACnet message.
type UnconfirmedIAm struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type UnconfirmedIAmDec struct {
	InstanceNum           uint32
	DeviceType            uint16
	MaxAPDULength         uint16
	SegmentationSupported uint8
	VendorId              uint16
}

// IAmObjects creates an instance of UnconfirmedIAm objects.
func IAmObjects(instN uint32, acceptedSize uint16, supportedSeg uint8, vendorID uint16) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 4)

	objs[0] = objects.EncObjectIdentifier(false, objects.TagBACnetObjectIdentifier, 8, instN)
	objs[1] = objects.EncUnsignedInteger(uint(acceptedSize))
	objs[2] = objects.EncEnumerated(supportedSeg)
	objs[3] = objects.EncUnsignedInteger(uint(vendorID))

	return objs
}

// NewUnconfirmedIAm creates a UnconfirmedIam.
func NewUnconfirmedIAm(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *UnconfirmedIAm {
	u := &UnconfirmedIAm{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.UnConfirmedReq, ServiceUnconfirmedIAm, IAmObjects(1, 1024, 0, 1)),
	}
	u.SetLength()

	return u
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedIAm frame.
func (u *UnconfirmedIAm) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal UnconfirmedIAm - marshal length %d binary length %d: %v",
			u.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedIAm %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedIAm %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedIAm %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedIAm instance.
func (u *UnconfirmedIAm) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *UnconfirmedIAm) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal UnconfirmedIAm - marshal length %d binary length %d: %v",
			u.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedIAm: %v", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedIAm: %v", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedIAm: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedIAm.
func (u *UnconfirmedIAm) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()
	return l
}

// SetLength sets the length in Length field.
func (u *UnconfirmedIAm) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *UnconfirmedIAm) Decode() (UnconfirmedIAmDec, error) {
	decIAm := UnconfirmedIAmDec{}

	if len(u.APDU.Objects) != 4 {
		return decIAm, fmt.Errorf(
			"failed to decode UnconfirmedIAm - number of objects %d: %v",
			len(u.APDU.Objects),
			common.ErrWrongObjectCount,
		)
	}

	for i, obj := range u.APDU.Objects {
		switch i {
		case 0:
			objId, err := objects.DecObjectIdentifier(obj)
			if err != nil {
				return decIAm, fmt.Errorf("decoding UnconfirmedIAm: %v", err)
			}
			decIAm.DeviceType = objId.ObjectType
			decIAm.InstanceNum = objId.InstanceNumber
		case 1:
			maxLen, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				return decIAm, fmt.Errorf("decoding UnconfirmedIAm: %v", err)
			}
			decIAm.MaxAPDULength = uint16(maxLen)
		case 2:
			segSupport, err := objects.DecEnumerated(obj)
			if err != nil {
				return decIAm, fmt.Errorf("decoding UnconfirmedIAm: %v", err)
			}
			decIAm.SegmentationSupported = uint8(segSupport)
		case 3:
			vendorId, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				return decIAm, fmt.Errorf("decoding UnconfirmedIAm: %v", err)
			}
			decIAm.VendorId = uint16(vendorId)
		}
	}

	return decIAm, nil
}

func (u *UnconfirmedIAm) GetService() uint8 {
	return u.APDU.Service
}

func (u *UnconfirmedIAm) GetType() uint8 {
	return u.APDU.Type
}
