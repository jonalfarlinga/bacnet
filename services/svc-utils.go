package services

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/pkg/errors"
)

func decodeTags(enc_obj *objects.Object, obj *objects.APDUPayload) (*objects.AppTag, error) {
	switch enc_obj.TagNumber {
	case objects.TagNull:
		err := objects.DecNull(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 0")
		}
		return &objects.AppTag{
			TagNumber: objects.TagNull,
			TagClass:  false,
			Length:    0,
			Value:     nil,
		}, nil
	case objects.TagBoolean:
		value, err := objects.DecBoolean(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 1")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagBoolean,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagUnsignedInteger:
		value, err := objects.DecUnsignedInteger(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 0")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagUnsignedInteger,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagReal:
		value, err := objects.DecReal(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 4")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagReal,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagCharacterString:
		value, err := objects.DecString(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 7")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagCharacterString,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagBitString:
		value, err := objects.DecBitString(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 5")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagBitString,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagEnumerated:
		value, err := objects.DecEnumerated(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 8")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagEnumerated,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagDate:
		value, err := objects.DecDate(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 9")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagDate,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagTime:
		value, err := objects.DecTime(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 8")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagTime,
			TagClass:  false,
			Length:    uint8(length),
			Value:     value,
		}, nil
	case objects.TagBACnetObjectIdentifier:
		objId, err := objects.DecObjectIdentifier(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Context object case 0")
		}
		length := (*obj).MarshalLen()
		return &objects.AppTag{
			TagNumber: objects.TagBACnetObjectIdentifier,
			TagClass:  false,
			Length:    uint8(length),
			Value:     fmt.Sprintf("%d:%d", objId.ObjectType, objId.InstanceNumber),
		}, nil
	default:
		log.Printf("\tnot encoded tag class %t tag number %d\n", enc_obj.TagClass, enc_obj.TagNumber)
		return nil, errors.Wrap(common.ErrNotImplemented, "decode Application object case default")
	}
}

func combine(t, s uint8) uint16 {
	// 0001, 0010
	// return:
	// 0001 0000
	// BINOR
	// 0000 0010
	// _________
	// 0001 0010
	return uint16(t)<<8 | uint16(s)
}
