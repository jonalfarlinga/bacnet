package objects

import (
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

type OpenCloseTag struct {
	TagNumber uint8
	TagClass  bool
	Name      uint8
}

func (n *OpenCloseTag) UnmarshalBinary(b []byte) error {
	if l := len(b); l < objLenMin {
		return fmt.Errorf(
			"failed to unmarshal NamedTag %x: %s", b, common.ErrTooShortToParse,
		)
	}
	n.TagNumber = b[0] >> 4
	n.TagClass = common.IntToBool(int(b[0]) & 0x8 >> 3)
	n.Name = b[0] & 0x7

	if l := len(b); l < 1 {
		return fmt.Errorf(
			"failed to unmarshal NamedTag %+v: %v", n,
			common.ErrTooShortToParse,
		)
	}

	return nil
}

func (n *OpenCloseTag) MarshalBinary() ([]byte, error) {
	b := make([]byte, n.MarshalLen())
	if err := n.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}

	return b, nil
}

func (n *OpenCloseTag) MarshalTo(b []byte) error {
	if len(b) < n.MarshalLen() {
		return fmt.Errorf("failed to marshall NamedTag %+v: %v", n, common.ErrTooShortToMarshalBinary)
	}
	b[0] = n.TagNumber<<4 | uint8(common.BoolToInt(n.TagClass))<<3 | n.Name
	return nil
}

func (n *OpenCloseTag) MarshalLen() int {
	return 1
}

func DecOpeningTab(rawPayload APDUPayload) (bool, error) {
	rawTag, ok := rawPayload.(*OpenCloseTag)
	if !ok {
		return false, fmt.Errorf("failed to decode OpeningTab %+v: %s", rawPayload, common.ErrWrongPayload)
	}
	return rawTag.Name == 0x6 && rawTag.TagClass, nil
}

func EncOpeningTag(tagN uint8) *OpenCloseTag {
	oTag := OpenCloseTag{}

	oTag.TagClass = true
	oTag.TagNumber = tagN
	oTag.Name = 0x6

	return &oTag
}

func DecClosingTab(rawPayload APDUPayload) (bool, error) {
	rawTag, ok := rawPayload.(*OpenCloseTag)
	if !ok {
		return false, fmt.Errorf("failed to decode ClosingTab %+v: %v", rawPayload, common.ErrWrongPayload)
	}
	return rawTag.Name == 0x7 && rawTag.TagClass, nil
}

func EncClosingTag(tagN uint8) *OpenCloseTag {
	cTag := OpenCloseTag{}

	cTag.TagClass = true
	cTag.TagNumber = tagN
	cTag.Name = 0x7

	return &cTag
}
