package bacnet

import (
	"testing"

	"github.com/jonalfarlinga/bacnet/test_utils"
)

func TestParseWhois(t *testing.T) {
	test_utils.TestParseWhois(t, Parse)
}
func TestParseReadProperty(t *testing.T) {
	test_utils.TestParseReadProperty(t, Parse)
}
func TestParseIam(t *testing.T) {
	test_utils.TestParseIam(t, Parse)
}
func TestParseReadPropertyMultiple(t *testing.T) {
	test_utils.TestParseReadPropertyMultiple(t, Parse)
}
