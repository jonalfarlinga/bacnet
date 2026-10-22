package client

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

func openSim(t *testing.T, change func(*Config)) (*Client, *sim.Network) {
	t.Helper()
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	cfg := Config{
		Local:     netip.MustParseAddrPort("127.0.0.1:0"),
		Broadcast: network.Addr(),
		Timeout:   time.Second,
	}
	if change != nil {
		change(&cfg)
	}
	c, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c, network
}

// The in-flight limit holds the requests beyond it until one finishes.
func TestInFlightLimit(t *testing.T) {
	c, _ := openSim(t, func(cfg *Config) {
		cfg.MaxInFlightDevice = func(Address) int { return 1 }
		cfg.MaxInFlightNet = func(uint16) int { return 2 }
	})
	direct := Address{IP: netip.MustParseAddrPort("127.0.0.1:47808")}
	routed := Address{Net: 7, MAC: []byte{1}}
	other := Address{Net: 7, MAC: []byte{2}}

	release, err := c.acquire(context.Background(), direct)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.acquire(ctx, direct); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a second request to a device with one slot gave %v, want it to wait", err)
	}
	release()
	if release, err = c.acquire(context.Background(), direct); err != nil {
		t.Errorf("after the first request finished, the next gave %v", err)
	} else {
		release()
	}

	// The devices of one routed network share its slots.
	r1, err := c.acquire(context.Background(), routed)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := c.acquire(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.acquire(ctx, routed); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a third request to a network with two slots gave %v, want it to wait", err)
	}
	r1()
	r2()
}

// The rate limit lets a burst through, then paces the rest.
func TestRateLimit(t *testing.T) {
	c, _ := openSim(t, func(cfg *Config) { cfg.MaxRequestsPerSec = 10 })
	dev := Address{IP: c.cfg.Broadcast}
	obj := ObjectID{Type: objects.ObjectTypeDevice, Instance: 1001}

	start := time.Now()
	for range 15 {
		if _, err := c.ReadProperty(context.Background(), dev, obj, objects.PropertyIdObjectName); err != nil {
			t.Fatal(err)
		}
	}
	// 10 go at once, then 5 more at 10 a second.
	if d := time.Since(start); d < 400*time.Millisecond {
		t.Errorf("15 requests at 10 a second took %s, want about half a second", d)
	}
	if st := c.Stats().Peers[dev.String()]; st.Sent != 15 || st.Retries != 0 || st.Timeouts != 0 {
		t.Errorf("counted %+v, want 15 sent and nothing else", st)
	}

	// Below zero is no limit.
	c2, _ := openSim(t, func(cfg *Config) { cfg.MaxRequestsPerSec = -1 })
	if c2.limiter != nil {
		t.Error("a rate below zero still made a limiter")
	}
}

// Invoke IDs rotate, and skip one still in flight.
func TestInvokeIDs(t *testing.T) {
	c, _ := openSim(t, nil)
	to := Address{IP: netip.MustParseAddrPort("127.0.0.1:47808")}
	first := plumbing.NewAPDU(plumbing.ConfirmedReq, 12, nil)
	key, _ := c.begin(to, first)
	second := plumbing.NewAPDU(plumbing.ConfirmedReq, 12, nil)
	c.begin(to, second)
	if first.InvokeID == second.InvokeID {
		t.Errorf("two requests in flight got invoke ID %d", first.InvokeID)
	}
	c.mu.Lock()
	c.invokes[keyOf(to, 0)] = first.InvokeID - 1 // the next would be first's
	c.mu.Unlock()
	third := plumbing.NewAPDU(plumbing.ConfirmedReq, 12, nil)
	c.begin(to, third)
	if third.InvokeID == first.InvokeID {
		t.Errorf("a request got invoke ID %d while one in flight has it", third.InvokeID)
	}
	c.mu.Lock()
	delete(c.pending, key)
	c.mu.Unlock()
}
