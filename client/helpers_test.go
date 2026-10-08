package client_test

import (
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
)

// openOn opens a client on a simulated network, with a change to the
// configuration if one is given, and returns it with a function that gives
// the devices whose I-Am it has heard so far, by instance.
func openOn(t *testing.T, network *sim.Network, change func(*client.Config)) (*client.Client, func() map[uint32]client.Device) {
	t.Helper()
	var mu sync.Mutex
	found := map[uint32]client.Device{}
	cfg := client.Config{
		Local:     netip.MustParseAddrPort("127.0.0.1:0"),
		Broadcast: network.Addr(),
		Timeout:   time.Second,
		Retries:   1,
		OnIAm: func(d client.Device) {
			mu.Lock()
			found[d.Instance] = d
			mu.Unlock()
		},
	}
	if change != nil {
		change(&cfg)
	}
	c, err := client.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c, func() map[uint32]client.Device {
		mu.Lock()
		defer mu.Unlock()
		devices := make(map[uint32]client.Device, len(found))
		for k, v := range found {
			devices[k] = v
		}
		return devices
	}
}

// awaitDevices sends a Who-Is and waits until the instances given have all
// answered, or fails. It returns every device heard.
func awaitDevices(t *testing.T, c *client.Client, heard func() map[uint32]client.Device, instances ...uint32) map[uint32]client.Device {
	t.Helper()
	if err := c.WhoIs(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		devices := heard()
		all := true
		for _, i := range instances {
			if _, ok := devices[i]; !ok {
				all = false
			}
		}
		if all {
			return devices
		}
		if time.Now().After(deadline) {
			t.Fatalf("heard %v, want %v among them", devices, instances)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
