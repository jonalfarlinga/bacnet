package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// COVNotification is a BACnet message.
type COVNotification struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type COVNotificationDec struct {
	ProcessId      uint32
	DeviceType     uint16
	DevInstanceNum uint32
	ObjectType     uint16
	ObjInstanceNum uint32
	Lifetime       uint32
	Tags           []*objects.Object
}

// NewConfirmedCOV creates a UnconfirmedCOVNotification.
func NewUnconfirmedCOVNotification(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *COVNotification {
	u := &COVNotification{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.UnConfirmedReq, ServiceUnconfirmedCOVNotification,
			COVObjects(1, 1024, 0, true, 1)),
	}
	u.SetLength()

	return u
}

func NewConfirmedCOVNotification(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *COVNotification {
	u := &COVNotification{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ConfirmedReq, ServiceConfirmedCOVNotification,
			COVObjects(1, 1024, 0, true, 1)),
	}
	u.SetLength()

	return u
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedCOVNotification frame.
func (u *COVNotification) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal UnconfirmedCOVNotification - marshal length %d binary length %d: %v",
			u.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	// do I need to Unmarshal again?
	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedCOVNotification %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedCOVNotification %+v: %v",
			u, common.ErrTooShortToParse,
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling UnconfirmedCOVNotification %v: %v",
			u, common.ErrTooShortToParse,
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedCOVNotification instance.
func (u *COVNotification) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *COVNotification) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal UnconfirmedCOVNotification - marshal length %d binary length %d: %v",
			u.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedCOVNotification: %v", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedCOVNotification: %v", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedCOVNotification: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedCOVNotification.
func (u *COVNotification) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()
	return l
}

// SetLength sets the length in Length field.
func (u *COVNotification) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *COVNotification) Decode() (COVNotificationDec, error) {
	decCOV := COVNotificationDec{}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
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
				prop, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode ProcessId: %v", err)
				}
				decCOV.ProcessId = prop
			case combine(8, 1):
				prop, err := objects.DecObjectIdentifier(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode MonitoredObjID: %v", err)
				}
				decCOV.DeviceType = prop.ObjectType
				decCOV.DevInstanceNum = prop.InstanceNumber
			case combine(8, 2):
				prop, err := objects.DecObjectIdentifier(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode MonitoredObjID: %v", err)
				}
				decCOV.ObjectType = prop.ObjectType
				decCOV.ObjInstanceNum = prop.InstanceNumber
			case combine(8, 3):
				prop, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode Lifetime: %v", err)
				}
				decCOV.Lifetime = prop
			case combine(4, 0):
				prop, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, fmt.Errorf("decode PropertyId: %v", err)
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Value:     prop,
					Length:    uint8(enc_obj.MarshalLen()),
				})
			default:
				log.Printf("Unknown Context object: context %v tag class %t tag number %d\n", context, enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			// log.Println("TagNumber", enc_obj.TagNumber)
			tag, err := decodeAppTags(enc_obj, &obj)
			if err != nil {
				return decCOV, fmt.Errorf("decode Application Tag: %v", err)
			}
			objs = append(objs, tag)
		}
	}
	decCOV.Tags = objs
	return decCOV, nil
}

func (u *COVNotification) GetService() uint8 {
	return u.APDU.Service
}

func (u *COVNotification) GetType() uint8 {
	return u.APDU.Type
}
