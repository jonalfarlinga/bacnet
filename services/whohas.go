package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// UnconfirmedWhoHas is a BACnet message.
type UnconfirmedWhoHas struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type UnconfirmedWhoHasDec struct {
	// HasLimits tells whether the request names a device instance range.
	// Without one, every device that has the object answers.
	HasLimits bool
	LowLimit  uint32
	HighLimit uint32
	// A request asks for an object by identifier or by name, never both.
	// ObjectID is nil when it asks by name.
	ObjectID   *objects.ObjectIdentifier
	ObjectName string
	// Skipped holds the tags Decode did not recognise and left undecoded.
	Skipped []*objects.Object
}

// WhoHasObjects creates the objects of a Who-Has request without a device
// instance range. It asks by name when objectName is not empty, and by
// identifier otherwise.
func WhoHasObjects(objectType uint16, instN uint32, objectName string) []objects.APDUPayload {
	if objectName != "" {
		return []objects.APDUPayload{objects.ContextTag(3, objects.EncString(objectName))}
	}
	return []objects.APDUPayload{objects.EncObjectIdentifier(true, 2, objectType, instN)}
}

// NewUnconfirmedWhoHas creates a UnconfirmedWhoHas.
func NewUnconfirmedWhoHas(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *UnconfirmedWhoHas {
	u := &UnconfirmedWhoHas{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.UnConfirmedReq, ServiceUnconfirmedWhoHas, nil),
	}
	u.SetLength()
	return u
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedWhoHas frame.
func (u *UnconfirmedWhoHas) UnmarshalBinary(b []byte) error {
	if l := len(b); l < u.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal UnconfirmedWhoHas - marshal length %d binary length %d: %w",
			u.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := u.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling UnconfirmedWhoHas: %w", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling UnconfirmedWhoHas: %w", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling UnconfirmedWhoHas: %w", err)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedWhoHas instance.
func (u *UnconfirmedWhoHas) MarshalBinary() ([]byte, error) {
	b := make([]byte, u.MarshalLen())
	if err := u.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (u *UnconfirmedWhoHas) MarshalTo(b []byte) error {
	if len(b) < u.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal UnconfirmedWhoHas - marshal length %d binary length %d: %v",
			u.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := u.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoHas: %v", err)
	}
	offset += u.BVLC.MarshalLen()

	if err := u.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoHas: %v", err)
	}
	offset += u.NPDU.MarshalLen()

	if err := u.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling UnconfirmedWhoHas: %v", err)
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedWhoHas.
func (u *UnconfirmedWhoHas) MarshalLen() int {
	l := u.BVLC.MarshalLen()
	l += u.NPDU.MarshalLen()
	l += u.APDU.MarshalLen()

	return l
}

func (u *UnconfirmedWhoHas) Decode() (UnconfirmedWhoHasDec, error) {
	decWhoHas := UnconfirmedWhoHasDec{}

	var hasLow, hasHigh, hasName bool
	for i, obj := range u.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decWhoHas, fmt.Errorf(
				"WhoHas object at index %d is not Object type: %v",
				i, common.ErrInvalidObjectType,
			)
		}

		if !enc_obj.TagClass || enc_obj.Kind != objects.TagPrimitive {
			decWhoHas.Skipped = append(decWhoHas.Skipped, enc_obj)
			continue
		}

		switch enc_obj.TagNumber {
		case 0:
			low, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				return decWhoHas, fmt.Errorf("decode device instance low limit: %v", err)
			}
			decWhoHas.LowLimit = low
			hasLow = true
		case 1:
			high, err := objects.DecUnsignedInteger(obj)
			if err != nil {
				return decWhoHas, fmt.Errorf("decode device instance high limit: %v", err)
			}
			decWhoHas.HighLimit = high
			hasHigh = true
		case 2:
			objId, err := objects.DecObjectIdentifier(obj)
			if err != nil {
				return decWhoHas, fmt.Errorf("decode object identifier: %v", err)
			}
			decWhoHas.ObjectID = &objId
		case 3:
			// A character string: the character set octet, then the text.
			if len(enc_obj.Data) == 0 {
				return decWhoHas, fmt.Errorf(
					"decode object name - missing character set octet: %w", common.ErrTooShortToParse,
				)
			}
			decWhoHas.ObjectName = string(enc_obj.Data[1:])
			hasName = true
		default:
			decWhoHas.Skipped = append(decWhoHas.Skipped, enc_obj)
		}
	}

	if hasLow != hasHigh {
		return decWhoHas, fmt.Errorf(
			"failed to decode WhoHas - one device instance limit without the other: %w",
			common.ErrWrongStructure,
		)
	}
	decWhoHas.HasLimits = hasLow
	if (decWhoHas.ObjectID != nil) == hasName {
		return decWhoHas, fmt.Errorf(
			"failed to decode WhoHas - want an object identifier or an object name: %w",
			common.ErrWrongStructure,
		)
	}

	return decWhoHas, nil
}

// SetLength sets the length in Length field.
func (u *UnconfirmedWhoHas) SetLength() {
	u.BVLC.Length = uint16(u.MarshalLen())
}

func (u *UnconfirmedWhoHas) GetService() uint8 {
	return u.APDU.Service
}

func (u *UnconfirmedWhoHas) GetType() uint8 {
	return u.APDU.Type
}
