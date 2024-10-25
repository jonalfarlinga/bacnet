package objects

import (
	"fmt"
	"time"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/pkg/errors"
)

type AppTag struct {
	TagNumber uint8
	TagClass  bool
	Length    uint8
	Value     interface{}
}

func NewAppTag(number uint8, class bool, value interface{}) *AppTag {
	return &AppTag{
		TagNumber: number,
		TagClass:  class,
		Value:     value,
	}
}

func (a *AppTag) UnmarshalBinary(b []byte) error {
	if l := len(b); l < objLenMin {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal AppTag - binary too short - %x", b),
		)
	}
	offset := 0

	a.TagNumber = b[offset] >> 4
	a.TagClass = common.IntToBool(int(b[offset]) & 0x8 >> 3)
	a.Length = uint8(b[offset] & 0x7)

	// Handle extended value case
	if a.Length == 5 {
		offset++
		a.Length = uint8(b[offset])
	}

	value, err := a.convertBACnetData(b, offset)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal AppTag - invalid data")
	}
	a.Value = value

	if l := len(b); l < 1 {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal AppTag - missing data - %v", a),
		)
	}

	return nil
}

func (a *AppTag) MarshalBinary() ([]byte, error) {
	b := make([]byte, a.MarshalLen())
	if err := a.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}

	return b, nil
}

func (a *AppTag) MarshalTo(b []byte) error {
	if len(b) < a.MarshalLen() {
		return errors.Wrap(common.ErrTooShortToMarshalBinary, "failed to marshall AppTag - marshal length too short")
	}
	offset := 0
	b[offset] = a.TagNumber << 4
	if a.Length < 5 {
		b[offset] |= a.Length
	} else {
		offset++
		b[offset] = a.Length
	}
	data, err := convertToBACnetData(a.Value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal AppTag - invalid data")
	}
	for i, d := range data {
		b[offset+i+1] = d
	}
	return nil
}

func (a *AppTag) MarshalLen() int {
	return int(a.Length) + 1
}

func convertDate(b []byte) (time.Time, error) {
	year := int(b[0]) + 1900
	month := time.Month(b[1])
	day := int(b[2])
	weekday := time.Weekday(b[3])
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if weekday != time.Sunday && date.Weekday() != weekday {
		return time.Time{}, errors.Wrap(common.ErrInvalidData, "failed to unmarshal AppTag - weekday mismatch")
	}
	return date, nil
}

func convertTime(b []byte) (time.Time, error) {
	hour := int(b[0])
	minute := int(b[1])
	second := int(b[2])
	hundredths := int(b[3])
	time := time.Date(0, 0, 0, hour, minute, second, hundredths*10000000, time.UTC)
	return time, nil
}

func convertBACnetID(b []byte) (string, error) {
	if len(b) < 4 {
		return "", errors.Wrap(common.ErrTooShortToParse, "failed to unmarshal AppTag - invalid object identifier length")
	}
	objectType := uint16(b[0])<<2 | uint16(b[1]&0xC0)>>6
	instance := uint32(b[1]&0x3F)<<16 | uint32(b[2])<<8 | uint32(b[3])
	oid := fmt.Sprintf("%d:%d", objectType, instance)
	return oid, nil
}

func (a *AppTag) convertBACnetData(b []byte, offset int) (interface{}, error) {
	switch a.TagNumber {
	case TagNull:
		return nil, nil
	case TagBoolean:
		return b[offset+1] == 1, nil
	case TagUnsignedInteger:
		return uint(b[offset+1]), nil
	case TagReal:
		return float32(b[offset+1]), nil
	case TagDouble:
		return float64(b[offset+1]), nil
	case TagOctetString:
		return b[offset+1 : offset+int(a.Length)+1], nil
	case TagCharacterString:
		return string(b[offset+1 : offset+int(a.Length)+1]), nil
	case TagBitString:
		return b[offset+1 : offset+int(a.Length)+1], nil
	case TagEnumerated:
		return uint(b[offset+1]), nil
	case TagDate:
		if len(b[offset+1:offset+int(a.Length)+1]) < 4 {
			return nil, errors.Wrap(common.ErrTooShortToParse, "failed to unmarshal AppTag - invalid date length")
		}
		date, err := convertDate(b[offset+1 : offset+int(a.Length)+1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal AppTag - invalid date")
		}
		return date, nil
	case TagTime:
		if len(b[offset+1:offset+int(a.Length)+1]) < 4 {
			return nil, errors.Wrap(common.ErrTooShortToParse, "failed to unmarshal AppTag - invalid time length")
		}
		time, err := convertTime(b[offset+1 : offset+int(a.Length)+1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal AppTag - invalid time")
		}
		return time, nil
	case TagBACnetObjectIdentifier:
		if len(b[offset+1:offset+int(a.Length)+1]) < 4 {
			return nil, errors.Wrap(common.ErrTooShortToParse, "failed to unmarshal AppTag - invalid object identifier length")
		}
		oid, err := convertBACnetID(b[offset+1 : offset+int(a.Length)+1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal AppTag - invalid object identifier")
		}
		a.Value = oid
	default:
		return nil, errors.Wrap(common.ErrInvalidData, "failed to unmarshal AppTag - invalid tag number")
	}
	return nil, nil
}

func convertToBACnetData(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return []byte{}, nil
	case bool:
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case uint:
		return []byte{byte(v)}, nil
	case float32:
		return []byte{byte(v)}, nil
	case float64:
		return []byte{byte(v)}, nil
	case string:
		return []byte(v), nil
	case time.Time:
		return []byte{byte(v.Year() - 1900), byte(v.Month()), byte(v.Day()), byte(v.Weekday())}, nil
	default:
		return nil, errors.Wrap(common.ErrInvalidData, "failed to marshal AppTag - invalid value type")
	}
}
