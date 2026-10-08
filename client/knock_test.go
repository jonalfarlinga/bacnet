package client

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
)

// A knock names the device at an address, and leaves nothing in the client
// for an address where nobody is.
func TestKnock(t *testing.T) {
	c, network := openSim(t, func(cfg *Config) {
		cfg.Broadcast = netip.MustParseAddrPort("127.0.0.1:9") // no I-Am is heard
		cfg.Retries = 2
	})
	ctx := context.Background()
	stations := network.Addrs()

	instance, err := c.Knock(ctx, stations[3])
	if err != nil || instance != 4000 {
		t.Fatalf("knocking on the legacy panel gave %d, %v, want device 4000", instance, err)
	}
	if st := c.Stats().Peers[stations[3].String()]; st.Sent != 1 {
		t.Errorf("the device that answered counts %+v, want one request sent", st)
	}

	nobody := netip.MustParseAddrPort("127.0.0.1:9")
	start := time.Now()
	if _, err := c.Knock(ctx, nobody); !errors.Is(err, ErrTimeout) {
		t.Errorf("knocking on nobody gave %v, want a timeout", err)
	}
	// One second for each send: a resend would make it two or three.
	if d := time.Since(start); d > 1800*time.Millisecond {
		t.Errorf("knocking on nobody took %s: the request was sent again", d)
	}
	if st, ok := c.Stats().Peers[nobody.String()]; ok {
		t.Errorf("nobody counts %+v, want nothing kept", st)
	}
	c.mu.Lock()
	_, invoke := c.invokes[keyOf(Address{IP: nobody}, 0)]
	slots := len(c.inflight)
	c.mu.Unlock()
	if invoke || slots != 0 {
		t.Errorf("after the knocks the client holds an invoke ID for nobody (%t) and %d sets of slots, want neither", invoke, slots)
	}

	// A device the client has read from is counted as any other when it
	// stops answering.
	network.Silence(3, true)
	if _, err := c.Knock(ctx, stations[3]); !errors.Is(err, ErrTimeout) {
		t.Errorf("knocking on a silent device gave %v, want a timeout", err)
	}
	if st := c.Stats().Peers[stations[3].String()]; st.Sent != 2 || st.Timeouts != 1 || st.Retries != 0 {
		t.Errorf("the device that fell silent counts %+v, want two sent, one timed out, none resent", st)
	}
}

// A device older than the wildcard instance declines the knock, and says
// which it is to a Who-Is sent to it alone.
func TestKnockDeclined(t *testing.T) {
	stations := sim.Default()
	stations[3][0].NoWildcard = true
	network, err := sim.StartDevices(stations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	var mu sync.Mutex
	var heard []Device
	c, err := Open(Config{
		Local:     netip.MustParseAddrPort("127.0.0.1:0"),
		Broadcast: netip.MustParseAddrPort("127.0.0.1:9"),
		Timeout:   time.Second,
		OnIAm: func(d Device) {
			mu.Lock()
			heard = append(heard, d)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	old := network.Addrs()[3]

	if _, err := c.Knock(context.Background(), old); !Declined(err) {
		t.Fatalf("knocking on a device without the wildcard instance gave %v, want it declined", err)
	}
	if err := c.WhoIsAt(old); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		got := append([]Device(nil), heard...)
		mu.Unlock()
		if len(got) == 1 && got[0].Instance == 4000 && got[0].Addr.IP == old {
			break
		}
		if len(got) > 1 || time.Now().After(deadline) {
			t.Fatalf("a Who-Is to one device drew %+v, want the I-Am of device 4000 alone", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A device denied by its instance, at an address the client had not heard
// it from, is told by its answer and left alone from then on.
func TestKnockDenied(t *testing.T) {
	instance := uint32(4000)
	c, network := openSim(t, func(cfg *Config) {
		cfg.Broadcast = netip.MustParseAddrPort("127.0.0.1:9")
		cfg.Denied = []SourceRule{{Instance: &instance}}
	})
	ctx := context.Background()
	at := network.Addrs()[3]

	if _, err := c.Knock(ctx, at); !errors.Is(err, ErrSourceDenied) {
		t.Errorf("knocking on a denied device gave %v, want it denied", err)
	}
	_, err := c.ReadProperty(ctx, Address{IP: at}, ObjectID{Type: objects.ObjectTypeDevice, Instance: 4000}, objects.PropertyIdObjectName)
	if !errors.Is(err, ErrSourceDenied) {
		t.Errorf("reading the denied device after the knock gave %v, want it denied", err)
	}
	if _, err := c.Knock(ctx, at); !errors.Is(err, ErrSourceDenied) {
		t.Errorf("a second knock gave %v, want it denied", err)
	}
	if st := c.Stats().Peers[at.String()]; st.Sent != 1 {
		t.Errorf("the denied device counts %+v, want the one request that found it out", st)
	}
}
