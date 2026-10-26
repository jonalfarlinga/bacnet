package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// UnconfirmedIAm is a BACnet message.
type ConfirmedWriteProperty struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ConfirmedWritePropertyDec struct {
	ObjectType  uint16
	InstanceNum uint32
	PropertyId  uint16
	ArrayIndex  *uint32
	Value       float32
	Priority    uint8
	Tags        []*objects.Object
}

func ConfirmedWritePropertyObjects(objectType uint16, instN uint32, propertyId uint16, data interface{}) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 6)

	objs[0] = objects.EncObjectIdentifier(true, 0, objectType, instN)
	objs[1] = objects.ContextTag(1, objects.EncUnsignedInteger(uint(propertyId)))
	objs[2] = objects.EncOpeningTag(3)
	switch data := data.(type) {
	case float32:
		objs[3] = objects.EncReal(data)
	case uint:
		objs[3] = objects.EncUnsignedInteger(data)
	case string:
		objs[3] = objects.EncString(data)
	}
	objs[4] = objects.EncClosingTag(3)
	objs[5] = objects.ContextTag(4, objects.EncUnsignedInteger(16))

	return objs
}

func NewConfirmedWriteProperty(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *ConfirmedWriteProperty {
	c := &ConfirmedWriteProperty{
		BVLC: bvlc,
		NPDU: npdu,
		APDU: plumbing.NewAPDU(plumbing.ConfirmedReq, ServiceConfirmedWriteProperty, nil),
	}
	c.SetLength()

	return c
}

func (c *ConfirmedWriteProperty) UnmarshalBinary(b []byte) error {
	if l := len(b); l < c.MarshalLen() {
		return fmt.Errorf(
			"failed to unmarshal ConfirmedWP - marshal length %d binary length %d: %v",
			c.MarshalLen(), l,
			common.ErrTooShortToParse,
		)
	}

	var offset int = 0
	if err := c.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedWP %+v: %v", c, common.ErrTooShortToParse,
		)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedWP %+v: %v", c, common.ErrTooShortToParse,
		)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf(
			"unmarshalling ConfirmedWP %+v: %v", c, common.ErrTooShortToParse,
		)
	}

	return nil
}

func (c *ConfirmedWriteProperty) MarshalBinary() ([]byte, error) {
	b := make([]byte, c.MarshalLen())
	if err := c.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

func (c *ConfirmedWriteProperty) MarshalTo(b []byte) error {
	if len(b) < c.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal ConfirmedWP - marshal length %d binary length %d: %v",
			c.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	var offset = 0
	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal ConfirmedWP: %v", err)
	}
	offset += c.BVLC.MarshalLen()

	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal ConfirmedWP: %v", err)
	}
	offset += c.NPDU.MarshalLen()

	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("failed to marshal ConfirmedWP: %v", err)
	}

	return nil
}

func (c *ConfirmedWriteProperty) MarshalLen() int {
	l := c.BVLC.MarshalLen()
	l += c.NPDU.MarshalLen()
	l += c.APDU.MarshalLen()

	return l
}

func (c *ConfirmedWriteProperty) SetLength() {
	c.BVLC.Length = uint16(c.MarshalLen())
}

func (c *ConfirmedWriteProperty) Decode() (ConfirmedWritePropertyDec, error) {
	decCWP := ConfirmedWritePropertyDec{}

	if len(c.APDU.Objects) < 5 {
		return decCWP, fmt.Errorf(
			"failed to decode ConfirmedWP - object count %d: %v",
			len(c.APDU.Objects),
			common.ErrWrongObjectCount,
		)
	}

	tags := make([]*objects.Object, len(c.APDU.Objects))
	for i, obj := range c.APDU.Objects {
		var ok bool
		if tags[i], ok = obj.(*objects.Object); !ok || tags[i] == nil {
			return decCWP, fmt.Errorf("WriteProperty object %d: %w", i, common.ErrInvalidObjectType)
		}
	}
	contextTag := func(t *objects.Object, number uint8) bool {
		return t.TagClass && t.Kind == objects.TagPrimitive && t.TagNumber == number
	}
	if !contextTag(tags[0], 0) || !contextTag(tags[1], 1) {
		return decCWP, fmt.Errorf("WriteProperty requires object and property identifiers: %w", common.ErrWrongStructure)
	}
	id, err := objects.DecObjectIdentifier(tags[0])
	if err != nil {
		return decCWP, err
	}
	prop, err := objects.DecUnsignedInteger(tags[1])
	if err != nil || prop > 0xFFFF {
		return decCWP, fmt.Errorf("invalid WriteProperty property identifier: %w", common.ErrWrongStructure)
	}
	decCWP.ObjectType, decCWP.InstanceNum, decCWP.PropertyId = id.ObjectType, id.InstanceNumber, uint16(prop)
	rest := tags[2:]
	if contextTag(rest[0], 2) {
		index, err := objects.DecUnsignedInteger(rest[0])
		if err != nil {
			return decCWP, err
		}
		decCWP.ArrayIndex = &index
		rest = rest[1:]
	}
	if len(rest) == 0 || rest[0].Kind != objects.TagOpening || rest[0].TagNumber != 3 {
		return decCWP, fmt.Errorf("WriteProperty requires a value: %w", common.ErrWrongStructure)
	}
	stack := []uint8{3}
	i := 1
	for ; i < len(rest); i++ {
		t := rest[i]
		switch t.Kind {
		case objects.TagOpening:
			stack = append(stack, t.TagNumber)
		case objects.TagClosing:
			if t.TagNumber != stack[len(stack)-1] {
				return decCWP, fmt.Errorf("mismatched WriteProperty value tags: %w", common.ErrWrongStructure)
			}
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			break
		}
		if !t.TagClass && t.Kind == objects.TagPrimitive {
			var payload objects.APDUPayload = t
			decoded, err := decodeAppTags(t, &payload)
			if err != nil {
				return decCWP, err
			}
			decoded.Length = t.Length
			decCWP.Tags = append(decCWP.Tags, decoded)
		} else {
			decCWP.Tags = append(decCWP.Tags, t)
		}
	}
	if len(stack) != 0 || i == 1 {
		return decCWP, fmt.Errorf("missing or unclosed WriteProperty value: %w", common.ErrWrongStructure)
	}
	if len(decCWP.Tags) == 1 {
		decCWP.Value, _ = decCWP.Tags[0].Value.(float32)
	}
	rest = rest[i+1:]
	if len(rest) > 0 {
		if len(rest) != 1 || !contextTag(rest[0], 4) {
			return decCWP, fmt.Errorf("unexpected tags after WriteProperty value: %w", common.ErrWrongStructure)
		}
		priority, err := objects.DecUnsignedInteger(rest[0])
		if err != nil || priority < 1 || priority > 16 {
			return decCWP, fmt.Errorf("invalid WriteProperty priority: %w", common.ErrWrongStructure)
		}
		decCWP.Priority = uint8(priority)
	}

	return decCWP, nil
}

func (u *ConfirmedWriteProperty) GetService() uint8 {
	return u.APDU.Service
}

func (u *ConfirmedWriteProperty) GetType() uint8 {
	return u.APDU.Type
}
