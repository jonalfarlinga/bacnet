package test_utils

import (
	"fmt"
	"testing"

	"github.com/jonalfarlinga/bacnet/objects"
)

func AssertEqualTag(t *testing.T, expected *objects.Object, actual objects.APDUPayload) {
	errs := []string{}
	actualTag, ok := actual.(*objects.Object)
	if !ok {
		t.Errorf("Expected Object, got %T", actual)
	}
	if expected.TagNumber != actualTag.TagNumber {
		errs = append(errs, fmt.Sprintf("TagNumber: expected %v, got %v", expected.TagNumber, actualTag.TagNumber))
	}
	if expected.TagClass != actualTag.TagClass {
		errs = append(errs, fmt.Sprintf("TagClass: expected %v, got %v", expected.TagClass, actualTag.TagClass))
	}
	if expected.Length != actualTag.Length {
		errs = append(errs, fmt.Sprintf("Length: expected %v, got %v", expected.Length, actualTag.Length))
	}
	for i, b := range expected.Data {
		if b != actualTag.Data[i] {
			errs = append(errs, fmt.Sprintf("Data: expected %v, got %v", expected.Data, actualTag.Data))
		}
	}
	if len(errs) > 0 {
		t.Errorf("Errors: %v", errs)
	}
}

func AssertEqual(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}
