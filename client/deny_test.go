package client_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
)

func ptr[T any](v T) *T { return &v }

func deviceObject(instance uint32) client.ObjectID {
	return client.ObjectID{Type: objects.ObjectTypeDevice, Instance: instance}
}

// A device denied by instance gets no request, and its I-Am is dropped.
func TestDeniedByInstance(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, nil)
	devices := awaitDevices(t, c, heard, 1001, 2000)
	ctx := context.Background()

	rule := client.SourceRule{Instance: ptr(uint32(1001))}
	if err := c.SetDenied(rule, true); err != nil {
		t.Fatal(err)
	}
	_, err = c.ReadProperty(ctx, devices[1001].Addr, deviceObject(1001), objects.PropertyIdObjectName)
	if !errors.Is(err, client.ErrSourceDenied) {
		t.Errorf("reading a denied device gave %v, want it denied", err)
	}
	if _, err := c.ReadProperty(ctx, devices[2000].Addr, deviceObject(2000), objects.PropertyIdObjectName); err != nil {
		t.Errorf("reading another device failed: %v", err)
	}
	if st := c.Stats().Peers[devices[1001].Addr.String()]; st.Denied != 1 {
		t.Errorf("device 1001 counts %+v, want one denied request", st)
	}

	// The next Who-Is draws an I-Am from the denied device, which is
	// dropped; the client restarted from nothing would still not list it.
	c2, heard2 := openOn(t, network, func(cfg *client.Config) { cfg.Denied = []client.SourceRule{rule} })
	found := awaitDevices(t, c2, heard2, 2000, 3000, 4000)
	time.Sleep(50 * time.Millisecond)
	if _, ok := heard2()[1001]; ok {
		t.Errorf("the denied device 1001 was heard: %v", found)
	}

	// Allowed again, it answers.
	if err := c.SetDenied(rule, false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadProperty(ctx, devices[1001].Addr, deviceObject(1001), objects.PropertyIdObjectName); err != nil {
		t.Errorf("reading the device allowed again failed: %v", err)
	}
	if rules := c.Denied(); len(rules) != 0 {
		t.Errorf("rules in force: %v, want none", rules)
	}
}

// A routed network denied as a whole: every device on it, and not the
// router itself.
func TestDeniedByNetwork(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, func(cfg *client.Config) {
		cfg.Denied = []client.SourceRule{{Net: ptr(uint16(2001))}}
	})
	devices := awaitDevices(t, c, heard, 2000)
	time.Sleep(50 * time.Millisecond)
	if _, ok := heard()[2101]; ok {
		t.Error("a device on the denied network was heard")
	}
	ctx := context.Background()
	vav := client.Address{Net: 2001, MAC: []byte{5}}
	if _, err := c.ReadProperty(ctx, vav, deviceObject(2101), objects.PropertyIdObjectName); !errors.Is(err, client.ErrSourceDenied) {
		t.Errorf("reading a device on the denied network gave %v, want it denied", err)
	}
	if _, err := c.ReadProperty(ctx, devices[2000].Addr, deviceObject(2000), objects.PropertyIdObjectName); err != nil {
		t.Errorf("reading the router itself failed: %v", err)
	}
}

// A host denied by its address.
func TestDeniedByIP(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, nil)
	devices := awaitDevices(t, c, heard, 1001, 2000)
	host := netip.PrefixFrom(devices[1001].Addr.IP.Addr(), 32)
	if err := c.SetDenied(client.SourceRule{IP: host}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadProperty(context.Background(), devices[1001].Addr, deviceObject(1001), objects.PropertyIdObjectName); !errors.Is(err, client.ErrSourceDenied) {
		t.Errorf("reading a denied host gave %v, want it denied", err)
	}
	// A prefix covers every port on its hosts: with every simulated station
	// on 127.0.0.1, the rule denies them all.
	if _, err := c.ReadProperty(context.Background(), devices[2000].Addr, deviceObject(2000), objects.PropertyIdObjectName); !errors.Is(err, client.ErrSourceDenied) {
		t.Errorf("reading another station on the denied host gave %v, want it denied", err)
	}

	if err := c.SetDenied(client.SourceRule{}, true); err == nil {
		t.Error("a rule that sets nothing was accepted")
	}
	if err := c.SetDenied(client.SourceRule{IP: host, Net: ptr(uint16(1))}, true); err == nil {
		t.Error("a rule that sets two fields was accepted")
	}
}
