package objects

func EncOpeningTag(tagN uint8) *Object {
	oTag := Object{}

	oTag.TagClass = true
	oTag.TagNumber = tagN
	oTag.Kind = TagOpening

	return &oTag
}

func EncClosingTag(tagN uint8) *Object {
	cTag := Object{}

	cTag.TagClass = true
	cTag.TagNumber = tagN
	cTag.Kind = TagClosing

	return &cTag
}
