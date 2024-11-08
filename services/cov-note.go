package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedCOVNotification is a BACnet message.
type UnconfirmedCOVNotification struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type UnconfirmedCOVNotificationDec struct {
	ProcessId      uint32
	DeviceType     uint16
	DevInstanceNum uint32
	ObjectType     uint16
	ObjInstanceNum uint32
	Lifetime       uint32
	Tags           []*objects.Object
}

// NewConfirmedCOV creates a UnconfirmedCOVNotification.
func NewUnconfirmedCOVNotification(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *UnconfirmedCOVNotification {
	u := &UnconfirmedCOVNotification{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.UnConfirmedReq, ServiceConfirmedSubscribeCOV,
			COVObjects(1, 1024, 0, true, 1)),
	}
	u.SetLength()

	return u
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedCOVNotification frame.
func (u *UnconfirmedCOVNotification) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal UnconfirmedCOVNotification - marshal length %d binary length %d", u.MarshalLen(), l),
		)
	}

	// do I need to Unmarshal again?
	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedCOVNotification %v", u),
		)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedCOVNotification %v", u),
		)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling UnconfirmedCOVNotification %v", u),
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedCOVNotification instance.
func (u *UnconfirmedCOVNotification) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *UnconfirmedCOVNotification) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal UnconfirmedCOVNotification - marshal length %d binary length %d", u.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedCOVNotification")
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedCOVNotification")
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling UnconfirmedCOVNotification")
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedCOVNotification.
func (u *UnconfirmedCOVNotification) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()
	return l
}

// SetLength sets the length in Length field.
func (u *UnconfirmedCOVNotification) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *UnconfirmedCOVNotification) Decode() (UnconfirmedCOVNotificationDec, error) {
	decCOV := UnconfirmedCOVNotificationDec{}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range u.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCOV, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("ComplexACK object at index %d is not Object type", i),
			)
		}
		// log.Printf(
		// 	"\tObject i %d tagnum %d tagclass %v data %x\n",
		// 	i, enc_obj.TagNumber, enc_obj.TagClass, enc_obj.Data,
		// )

		// add or remove context based on opening and closing tags
		if enc_obj.Length == 6 {
			context = append(context, enc_obj.TagNumber)
			continue
		}
		if enc_obj.Length == 7 {
			if len(context) == 0 {
				return decCOV, errors.Wrap(
					common.ErrInvalidObjectType,
					fmt.Sprintf("LogBufferCACK object at index %d has mismatched closing tag", i),
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
					return decCOV, errors.Wrap(err, "decode ProcessId")
				}
				decCOV.ProcessId = prop
			case combine(8, 1):
				prop, err := objects.DecObjectIdentifier(enc_obj)
				if err != nil {
					return decCOV, errors.Wrap(err, "decode MonitoredObjID")
				}
				decCOV.DeviceType = prop.ObjectType
				decCOV.DevInstanceNum = prop.InstanceNumber
			case combine(8, 2):
				prop, err := objects.DecObjectIdentifier(enc_obj)
				if err != nil {
					return decCOV, errors.Wrap(err, "decode MonitoredObjID")
				}
				decCOV.ObjectType = prop.ObjectType
				decCOV.ObjInstanceNum = prop.InstanceNumber
			case combine(8, 3):
				prop, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, errors.Wrap(err, "decode Lifetime")
				}

				decCOV.Lifetime = prop
			case combine(4, 0):
				prop, err := objects.DecUnsignedInteger(enc_obj)
				if err != nil {
					return decCOV, errors.Wrap(err, "decode PropertyId")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Value:     prop,
					Length:    uint8(enc_obj.MarshalLen()),
				})
			}

		} else {
			// log.Println("TagNumber", enc_obj.TagNumber)
			tag, err := decodeTags(enc_obj, &obj)
			if err != nil {
				return decCOV, errors.Wrap(err, "decode Application Tag")
			}
			objs = append(objs, tag)
		}
	}
	decCOV.Tags = objs
	return decCOV, nil
}

func (u *UnconfirmedCOVNotification) GetService() uint8 {
	return u.APDU.Service
}

func (u *UnconfirmedCOVNotification) GetType() uint8 {
	return u.APDU.Type
}
