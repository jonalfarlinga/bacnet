package client_test

import (
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client/sim"
)

// A ranged Who-Is draws only the devices in the range, routed ones too.
func TestWhoIsRange(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, nil)

	if err := c.WhoIsRange(2100, 2110); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(heard()) < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	devices := heard()
	for _, want := range []uint32{2101, 2102, 2107} {
		if _, ok := devices[want]; !ok {
			t.Errorf("device %d did not answer a Who-Is for 2100 to 2110", want)
		}
	}
	for instance := range devices {
		if instance < 2100 || instance > 2110 {
			t.Errorf("device %d answered a Who-Is for 2100 to 2110", instance)
		}
	}

	if err := c.WhoIsRange(5, 1); err == nil {
		t.Error("a backwards range was sent")
	}
	if err := c.WhoIsRange(0, 1<<23); err == nil {
		t.Error("a range above the largest instance was sent")
	}
}
