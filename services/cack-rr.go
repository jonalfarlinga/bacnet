package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedIAm is a BACnet message.
type LogBufferCACK struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type LogBufferCACKDec struct {
	ObjectType uint16
	InstanceId uint32
	PropertyId uint16
	FirstItem  bool
	LastItem   bool
	MoreItems  bool
	ItemCount  uint32
	Tags       []*objects.Object
}

type StatusFlags struct {
	InAlarm      bool
	Fault        bool
	Overridden   bool
	OutOfService bool
}

// func LogBufferCACKObjects(instN uint32, propertyId uint16, value interface{}) []objects.APDUPayload {
// 	objs := make([]objects.APDUPayload, 5)
// 	// Not implemented for LogBufferCACK
// 	return objs
// }

// func NewLogBufferCACK(cack *ComplexACK) *LogBufferCACK {
// 	c := &LogBufferCACK{
// 		BVLC: cack.BVLC,
// 		NPDU: cack.NPDU,
// 		APDU: cack.APDU,
// 	}
// 	c.SetLength()
// 	return c
// }

// func (c *LogBufferCACK) UnmarshalBinary(b []byte) error {
// 	// Use ComplexACK, then convert using NewLogBufferCACK()
// 	return fmt.Errorf("unmarshal binary not implemented for LogBufferCACK")
// }

// func (c *LogBufferCACK) MarshalBinary() ([]byte, error) {
// 	b := make([]byte, c.MarshalLen())
// 	if err := c.MarshalTo(b); err != nil {
// 		return nil, errors.Wrap(err, "failed to marshal binary")
// 	}
// 	return b, nil
// }

// func (c *LogBufferCACK) MarshalTo(b []byte) error {
// 	if len(b) < c.MarshalLen() {
// 		return errors.Wrap(
// 			common.ErrTooShortToMarshalBinary,
// 			fmt.Sprintf("failed to marshal CACK %x - marshal length too short", b),
// 		)
// 	}
// 	var offset = 0
// 	if err := c.BVLC.MarshalTo(b[offset:]); err != nil {
// 		return errors.Wrap(err, "marshalling CACK")
// 	}
// 	offset += c.BVLC.MarshalLen()

// 	if err := c.NPDU.MarshalTo(b[offset:]); err != nil {
// 		return errors.Wrap(err, "marshalling CACK")
// 	}
// 	offset += c.NPDU.MarshalLen()

// 	if err := c.APDU.MarshalTo(b[offset:]); err != nil {
// 		return errors.Wrap(err, "marshalling CACK")
// 	}
// 	return nil
// }

// func (c *LogBufferCACK) MarshalLen() int {
// 	l := c.BVLC.MarshalLen()
// 	l += c.NPDU.MarshalLen()
// 	l += c.APDU.MarshalLen()

// 	return l
// }

// func (u *LogBufferCACK) SetLength() {
// 	u.BVLC.Length = uint16(u.MarshalLen())
// }

func (c *ComplexACK) DecodeRR() (LogBufferCACKDec, error) {
	decCACK := LogBufferCACKDec{}

	if len(c.APDU.Objects) < 3 {
		return decCACK, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode CACK - objects count: %d", len(c.APDU.Objects)),
		)
	}

	context := []uint8{8}
	objs := make([]*objects.Object, 0)
	for i, obj := range c.APDU.Objects {
		enc_obj, ok := obj.(*objects.Object)
		if !ok {
			return decCACK, errors.Wrap(
				common.ErrInvalidObjectType,
				fmt.Sprintf("LogBufferCACK object at index %d is not Object type", i),
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
				return decCACK, errors.Wrap(
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
				objId, err := objects.DecObjectIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				decCACK.ObjectType = objId.ObjectType
				decCACK.InstanceId = objId.InstanceNumber
			case combine(8, 1):
				propId, err := objects.DecPropertyIdentifier(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 1")
				}
				decCACK.PropertyId = propId
			case combine(8, 3):
				first, last, more, err := decResultsFlag(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 3")
				}
				decCACK.FirstItem = first
				decCACK.LastItem = last
				decCACK.MoreItems = more
			case combine(8, 4):
				data, err := objects.DecUnsignedInteger(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 4")
				}
				decCACK.ItemCount = data
			case combine(1, 2):
				value, err := objects.DecReal(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 2")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 2,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			case combine(5, 2):
				value, err := decStatusFlags(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 2")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 2,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			case combine(1, 0):
				value, err := objects.DecLogStatus(obj)
				if err != nil {
					return decCACK, errors.Wrap(err, "decode Context object case 0")
				}
				objs = append(objs, &objects.Object{
					TagNumber: 0,
					TagClass:  true,
					Length:    uint8(obj.MarshalLen()),
					Value:     value,
				})
			default:
				log.Printf("Unknown Context object tag class %t tag number %d\n", enc_obj.TagClass, enc_obj.TagNumber)
			}
		} else {
			// log.Println("TagNumber", enc_obj.TagNumber)
			tag, err := decodeTags(enc_obj, &obj)
			if err != nil {
				return decCACK, errors.Wrap(err, "decode Application Tag")
			}
			objs = append(objs, tag)
		}
	}
	decCACK.Tags = objs

	return decCACK, nil
}

func decResultsFlag(obj objects.APDUPayload) (bool, bool, bool, error) {
	var first, last, more bool
	enc_obj, ok := obj.(*objects.Object)
	if !ok {
		return false, false, false, common.ErrInvalidObjectType
	}
	first = enc_obj.Data[1]&0x80 == 0x80
	last = enc_obj.Data[1]&0x40 == 0x40
	more = enc_obj.Data[1]&0x20 == 0x20
	return first, last, more, nil
}

func decStatusFlags(obj objects.APDUPayload) (StatusFlags, error) {
	var status StatusFlags
	enc_obj, ok := obj.(*objects.Object)
	if !ok {
		return status, common.ErrInvalidObjectType
	}
	status.InAlarm = enc_obj.Data[1]&0x80 == 0x80
	status.Fault = enc_obj.Data[1]&0x40 == 0x40
	status.Overridden = enc_obj.Data[1]&0x20 == 0x20
	status.OutOfService = enc_obj.Data[1]&0x10 == 0x10
	return status, nil
}
