package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// ConfirmedCOV is a BACnet message.
type ConfirmedCOV struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ConfirmedCOVDec struct {
	ProcessId       uint32
	MonitoredObjID  uint16
	MonitoredInstN  uint32
	ExpectConfirmed bool
	Lifetime        uint32
}

// IAmObjects creates an instance of ConfirmedCOV objects.
func COVObjects(pid uint, oid uint16, instN uint32, expect bool, life uint) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 4)

	obj := objects.EncUnsignedInteger(pid)
	obj.TagClass = true
	obj.TagNumber = 0
	objs[0] = obj
	objs[1] = objects.EncObjectIdentifier(true, 1, oid, instN)
	obj = &objects.Object{
		TagClass:  true,
		TagNumber: 2,
		Length:    1,
		Data:      []byte{byte(common.BoolToInt(expect))},
	}
	objs[2] = obj
	obj = objects.EncUnsignedInteger(life)
	obj.TagClass = true
	obj.TagNumber = 3
	objs[3] = obj

	return objs
}

func CancelCOVOBjects(pid uint, oid uint16, instN uint32) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 2)

	obj := objects.EncUnsignedInteger(pid)
	obj.TagClass = true
	obj.TagNumber = 0
	objs[0] = obj
	objs[1] = objects.EncObjectIdentifier(true, 1, oid, instN)

	return objs
}

// NewConfirmedSubscribeCOV creates a ConfirmedCOV.
func NewConfirmedSubscribeCOV(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) (*ConfirmedCOV, uint8) {
	u := &ConfirmedCOV{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ConfirmedReq, ServiceConfirmedSubscribeCOV, COVObjects(1, 1024, 0, true, 1)),
	}
	u.SetLength()

	return u, u.APDU.Type
}

// UnmarshalBinary sets the values retrieved from byte sequence in a ConfirmedCOV frame.
func (u *ConfirmedCOV) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal ConfirmedCOV - marshal length %d binary length %d", u.MarshalLen(), l),
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedCOV %v", u),
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedCOV %v", u),
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling ConfirmedCOV %v", u),
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a ConfirmedCOV instance.
func (u *ConfirmedCOV) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *ConfirmedCOV) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal ConfirmedCOV - marshal length %d binary length %d", u.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling ConfirmedCOV")
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling ConfirmedCOV")
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling ConfirmedCOV")
	}

	return nil
}

// MarshalLen returns the serial length of ConfirmedCOV.
func (u *ConfirmedCOV) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	// m := l
	l += u.NPDU.MarshalLen()
	// n := l - m
	l += u.APDU.MarshalLen()
	// o := l - m - n
	// fmt.Println("mlen", l, m, n, o)
	return l
}

// SetLength sets the length in Length field.
func (u *ConfirmedCOV) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *ConfirmedCOV) Decode() (ConfirmedCOVDec, error) {
	decCOV := ConfirmedCOVDec{}

	if len(u.APDU.Objects) != 4 {
		return decCOV, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode ConfirmedCOV %d - wrong object count", len(u.APDU.Objects)),
		)
	}

	// for i, obj := range u.APDU.Objects {
	// 	switch i {
	// 	case 0:
	// 	}
	// }
	return decCOV, nil
}
