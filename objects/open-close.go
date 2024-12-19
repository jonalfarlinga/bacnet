package objects

func EncOpeningTag(tagN uint8) *Object {
	oTag := Object{}

	oTag.TagClass = true
	oTag.TagNumber = tagN
	oTag.Length = 0x6

	return &oTag
}

func EncClosingTag(tagN uint8) *Object {
	cTag := Object{}

	cTag.TagClass = true
	cTag.TagNumber = tagN
	cTag.Length = 0x7

	return &cTag
}
