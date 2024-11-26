package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// ConfirmedCOV is a BACnet message.
type ConfirmedCOV struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ConfirmedCOVDec struct {
	ProcessId        uint32
	MonitoredObjType uint16
	MonitoredInstNum uint32
	ExpectConfirmed  bool
	Lifetime         uint32
}

// IAmObjects creates an instance of ConfirmedCOV objects.
func COVObjects(pid uint, oid uint16, instN uint32, expect bool, life uint) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 4)

	objs[0] = objects.ContextTag(0, objects.EncUnsignedInteger(pid))
	objs[1] = objects.EncObjectIdentifier(true, 1, oid, instN)
	objs[2] = objects.EncContextBool(2, expect)
	objs[3] = objects.ContextTag(3, objects.EncUnsignedInteger(life))

	return objs
}

func CancelCOVOBjects(pid uint, oid uint16, instN uint32) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 2)

	objs[0] = objects.ContextTag(0, objects.EncUnsignedInteger(pid))
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
		return fmt.Errorf(
			"failed to unmarshal ConfirmedCOV - marshal length %d binary length %d: %v",
			u.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedCOV %v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedCOV %v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedCOV %v: %v",
			u, common.ErrTooShortToParse,
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a ConfirmedCOV instance.
func (u *ConfirmedCOV) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *ConfirmedCOV) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal ConfirmedCOV - marshal length %d binary length %d: %v",
			u.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling ConfirmedCOV: %v", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling ConfirmedCOV: %v", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling ConfirmedCOV: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of ConfirmedCOV.
func (u *ConfirmedCOV) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()
	return l
}

// SetLength sets the length in Length field.
func (u *ConfirmedCOV) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *ConfirmedCOV) Decode() (ConfirmedCOVDec, error) {
	decCOV := ConfirmedCOVDec{}

	if len(u.APDU.Objects) != 4 {
		return decCOV, fmt.Errorf(
			"failed to decode ConfirmedCOV - number of objects %d: %v",
			len(u.APDU.Objects),
			common.ErrWrongObjectCount,
		)
	}

	context := []uint8{8}
	for i, obj := range u.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCOV, fmt.Errorf(
				"ComplexACK object at index %d is not Object type: %v",
				i, common.ErrInvalidObjectType,
			)
		}

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 {
			if len(context) == 0 {
				return decCOV, fmt.Errorf(
					"LogBufferCACK object at index %d has mismatched closing tag: %v",
					i, common.ErrInvalidObjectType,
				)
			}
			context = context[:len(context)-1]
			continue
		}

		if enc_obj.TagClass {
			c := combine(context[len(context)-1], enc_obj.TagNumber)
			switch c {
			case combine(8, 0):
				proc, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode ProcessId: %v", err)
				}
				decCOV.ProcessId = proc
			case combine(8, 1):
				objId, err := objects.DecObjectIdentifier(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode MonitoredObjID: %v", err)
				}
				decCOV.MonitoredObjType = objId.ObjectType
				decCOV.MonitoredInstNum = objId.InstanceNumber
			case combine(8, 2):
				if len(enc_obj.Data) != 1 {
					return decCOV, fmt.Errorf(
						"LogBufferCACK object at index %d has invalid data length: %v",
						i, common.ErrInvalidObjectType,
					)
				}
				decCOV.ExpectConfirmed = common.IntToBool(int(enc_obj.Data[0]))
			case combine(8, 3):
				life, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode Lifetime: %v", err)
				}
				decCOV.Lifetime = life
			default:
				log.Printf("Unknown Context object: context %v tag class %t tag number %d\n", context, enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			log.Printf("Not encoded object: %+v\n", enc_obj)
		}
	}

	return decCOV, nil
}
