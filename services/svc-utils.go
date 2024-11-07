package services

import (
	"log"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/pkg/errors"
)

func decodeTags(enc_obj *objects.Object, obj *objects.APDUPayload) (*objects.Object, error) {
	var value interface{}
	var length int
	switch enc_obj.TagNumber {
	case objects.TagNull:
		err := objects.DecNull(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 0")
		}
		length = 0
		value = nil
	case objects.TagBoolean:
		data, err := objects.DecBoolean(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 1")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagUnsignedInteger:
		data, err := objects.DecUnsignedInteger(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 0")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagSignedInteger:
		data, err := objects.DecSignedInteger(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 1")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagReal:
		data, err := objects.DecReal(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 4")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagDouble:
		data, err := objects.DecDouble(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 5")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagOctetString:
		data, err := objects.DecOctetString(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 6")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagCharacterString:
		data, err := objects.DecString(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 7")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagBitString:
		data, err := objects.DecBitString(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 5")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagEnumerated:
		data, err := objects.DecEnumerated(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 8")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagDate:
		data, err := objects.DecDate(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 9")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagTime:
		data, err := objects.DecTime(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Application object case 8")
		}
		length = (*obj).MarshalLen()
		value = data
	case objects.TagBACnetObjectIdentifier:
		objId, err := objects.DecObjectIdentifier(*obj)
		if err != nil {
			return nil, errors.Wrap(err, "decode Context object case 0")
		}
		length = (*obj).MarshalLen()
		value = objId
	default:
		log.Printf("\tnot encoded tag class %t tag number %d\n", enc_obj.TagClass, enc_obj.TagNumber)
	}
	return &objects.Object{
		TagNumber: enc_obj.TagNumber,
		TagClass:  false,
		Length:    uint8(length),
		Data:      enc_obj.Data,
		Value:     value,
	}, nil
}

func combine(t, s uint8) uint16 {
	return uint16(t)<<8 | uint16(s)
}
