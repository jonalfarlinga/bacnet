package services

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/pkg/errors"
)

// UnconfirmedIAm is a BACnet message.
type Error struct {
	*plumbing.BVLC
	*plumbing.NPDU
	*plumbing.APDU
}

type ErrorDec struct {
	ErrorClass uint8
	ErrorCode  uint8
}

func ErrorObjects(errClass, errCode uint8) []objects.APDUPayload {
	objs := make([]objects.APDUPayload, 2)

	objs[0] = objects.EncEnumerated(errClass)
	objs[1] = objects.EncEnumerated(errCode)

	return objs
}

func NewError(bvlc *plumbing.BVLC, npdu *plumbing.NPDU) *Error {
	e := &Error{
		BVLC: bvlc,
		NPDU: npdu,
		// TODO: Consider to implement parameter struct to an argment of New functions.
		APDU: plumbing.NewAPDU(plumbing.Error, ServiceConfirmedReadProperty, ErrorObjects(1, 31)),
	}
	e.SetLength()

	return e
}

// UnmarshalBinary sets the values retrieved from byte sequence in a UnconfirmedIAm frame.
func (e *Error) UnmarshalBinary(b []byte) error {
	if l := len(b); l < e.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("failed to unmarshal Error - marshal length %d binary length %d", e.MarshalLen(), l),
		)
	}

	var offset int = 0

	if err := e.BVLC.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling Error %v", e),
		)
	}
	offset += e.BVLC.MarshalLen()

	if err := e.NPDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling Error %v", e),
		)
	}
	offset += e.NPDU.MarshalLen()

	if err := e.APDU.UnmarshalBinary(b[offset:]); err != nil {
		return errors.Wrap(
			common.ErrTooShortToParse,
			fmt.Sprintf("unmarshalling Error %v", e),
		)
	}

	return nil
}

// MarshalBinary returns the byte sequence generated from a UnconfirmedIAm instance.
func (e *Error) MarshalBinary() ([]byte, error) {
	b := make([]byte, e.MarshalLen())
	if err := e.MarshalTo(b); err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary")
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (e *Error) MarshalTo(b []byte) error {
	if len(b) < e.MarshalLen() {
		return errors.Wrap(
			common.ErrTooShortToMarshalBinary,
			fmt.Sprintf("failed to marshal Error - marshal length %d binary length %d", e.MarshalLen(), len(b)),
		)
	}
	var offset = 0
	if err := e.BVLC.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling Error")
	}
	offset += e.BVLC.MarshalLen()

	if err := e.NPDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling Error")
	}
	offset += e.NPDU.MarshalLen()

	if err := e.APDU.MarshalTo(b[offset:]); err != nil {
		return errors.Wrap(err, "marshalling Error")
	}

	return nil
}

// MarshalLen returns the serial length of UnconfirmedIAm.
func (e *Error) MarshalLen() int {
	l := e.BVLC.MarshalLen()
	l += e.NPDU.MarshalLen()
	l += e.APDU.MarshalLen()

	return l
}

// SetLength sets the length in Length field.
func (e *Error) SetLength() {
	e.BVLC.Length = uint16(e.MarshalLen())
}

func (e *Error) Decode() (ErrorDec, error) {
	decErr := ErrorDec{}

	if len(e.APDU.Objects) != 2 {
		return decErr, errors.Wrap(
			common.ErrWrongObjectCount,
			fmt.Sprintf("failed to decode Error - object count: %d", len(e.APDU.Objects)),
		)
	}

	for i, obj := range e.APDU.Objects {
		switch i {
		case 0:
			errClass, err := objects.DecEnumerated(obj)
			if err != nil {
				return decErr, errors.Wrap(err, "failed to decode Enumerated Object")
			}
			decErr.ErrorClass = uint8(errClass)
		case 1:
			errCode, err := objects.DecEnumerated(obj)
			if err != nil {
				return decErr, errors.Wrap(err, "failed to decode Enumerated Object")
			}
			decErr.ErrorCode = uint8(errCode)
		}
	}

	return decErr, nil
}

func (u *Error) GetService() uint8 {
	return u.APDU.Service
}

func (u *Error) GetType() uint8 {
	return u.APDU.Type
}
