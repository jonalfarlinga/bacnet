package client_test

import (
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// twoNetworks is the default sim with FCU-2-07 moved behind the air handler
// on a network of its own, 2002, so that two routers each serve one network:
// the plant router 2001 with VAV-2-01 and VAV-2-02.
func twoNetworks(t *testing.T) *sim.Network {
	t.Helper()
	stations := sim.Default()
	fcu := stations[1][3]
	fcu.Net = 2002
	stations[1] = stations[1][:3]
	stations[0] = append(stations[0], fcu)
	network, err := sim.StartDevices(stations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	return network
}

// sentLog records where each datagram the client sends goes, and how.
type sentLog struct {
	mu   sync.Mutex
	sent []sentFrame
}

type sentFrame struct {
	to       netip.AddrPort
	function uint8
}

func (l *sentLog) record(e client.PacketEvent) {
	if e.Inbound || len(e.Data) < 2 {
		return
	}
	l.mu.Lock()
	l.sent = append(l.sent, sentFrame{to: e.Peer, function: e.Data[1]})
	l.mu.Unlock()
}

func (l *sentLog) last() sentFrame {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sent[len(l.sent)-1]
}

// awaitHeard waits for the instances want, then a little longer for any that
// should not come.
func awaitHeard(heard func() map[uint32]client.Device, want int) map[uint32]client.Device {
	deadline := time.Now().Add(2 * time.Second)
	for len(heard()) < want && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	return heard()
}

func sameInstances(got map[uint32]client.Device, want ...uint32) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			return false
		}
	}
	return true
}

// A Who-Is for one network draws only the devices on it. With no route
// known it goes as a broadcast with the network set; the I-Ams teach the
// route, and the next goes to the router alone.
func TestWhoIsNetwork(t *testing.T) {
	network := twoNetworks(t)
	var log sentLog
	c, heard := openOn(t, network, func(cfg *client.Config) { cfg.OnPacket = log.record })

	if err := c.WhoIsNetwork(2001); err != nil {
		t.Fatal(err)
	}
	if f := log.last(); f.to != network.Addr() || f.function != plumbing.BVLCFuncBroadcast {
		t.Errorf("with no route known the Who-Is went to %v as BVLC function %#x, want a broadcast to %v", f.to, f.function, network.Addr())
	}
	if got := awaitHeard(heard, 2); !sameInstances(got, 2101, 2102) {
		t.Fatalf("a Who-Is for network 2001 drew %v, want 2101 and 2102 alone", keys(got))
	}
	router, ok := c.Routes()[2001]
	if !ok {
		t.Fatal("the I-Ams from network 2001 taught no route to it")
	}

	if err := c.WhoIsNetwork(2001); err != nil {
		t.Fatal(err)
	}
	if f := log.last(); f.to != router || f.function != plumbing.BVLCFuncUnicast {
		t.Errorf("with the route known the Who-Is went to %v as BVLC function %#x, want a unicast to the router at %v", f.to, f.function, router)
	}

	c2, heard2 := openOn(t, network, nil)
	if err := c2.WhoIsNetwork(2002); err != nil {
		t.Fatal(err)
	}
	if got := awaitHeard(heard2, 1); !sameInstances(got, 2107) {
		t.Errorf("a Who-Is for network 2002 drew %v, want 2107 alone", keys(got))
	}
}

func TestWhoIsNetworkRange(t *testing.T) {
	network := twoNetworks(t)
	c, heard := openOn(t, network, nil)

	if err := c.WhoIsNetworkRange(2001, 2102, 2200); err != nil {
		t.Fatal(err)
	}
	if got := awaitHeard(heard, 1); !sameInstances(got, 2102) {
		t.Errorf("a Who-Is for 2102 to 2200 on network 2001 drew %v, want 2102 alone", keys(got))
	}

	if err := c.WhoIsNetworkRange(2001, 5, 1); err == nil {
		t.Error("a backwards range was sent")
	}
	for _, net := range []uint16{0, 0xFFFF} {
		if err := c.WhoIsNetwork(net); err == nil {
			t.Errorf("a Who-Is for network %#x was sent as one remote network", net)
		}
	}
}

func keys(m map[uint32]client.Device) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
