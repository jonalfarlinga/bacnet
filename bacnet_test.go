package bacnet

import (
	"testing"

	"github.com/jonalfarlinga/bacnet/internal/testutil"
)

func TestParseWhois(t *testing.T) {
	testutil.TestParseWhois(t, Parse)
}
func TestParseReadProperty(t *testing.T) {
	testutil.TestParseReadProperty(t, Parse)
}
func TestParseIam(t *testing.T) {
	testutil.TestParseIam(t, Parse)
}
func TestParseReadPropertyMultiple(t *testing.T) {
	testutil.TestParseReadPropertyMultiple(t, Parse)
}
