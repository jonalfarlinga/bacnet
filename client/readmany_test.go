package client_test

import (
	"context"
	"net/netip"
	"sync"
	"testing"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func nameSpecs(ids []client.ObjectID) []client.ReadSpec {
	specs := make([]client.ReadSpec, len(ids))
	for i, id := range ids {
		specs[i] = client.ReadSpec{Object: id, Props: []uint16{objects.PropertyIdObjectName, objects.PropertyIdDescription}}
	}
	return specs
}

// checkNames wants a name for every object, in the order asked.
func checkNames(t *testing.T, ids []client.ObjectID, results []client.ObjectResult) {
	t.Helper()
	if len(results) != len(ids) {
		t.Fatalf("%d results for %d objects", len(results), len(ids))
	}
	for i, r := range results {
		if r.Object != ids[i] {
			t.Errorf("result %d is for %v, want %v", i, r.Object, ids[i])
		}
		named := false
		for _, p := range r.Props {
			if p.ID == uint32(objects.PropertyIdObjectName) && p.Err == nil && len(p.Values) == 1 && p.Values[0].Str != "" {
				named = true
			}
		}
		if !named {
			t.Errorf("object %v has no name in %+v", r.Object, r.Props)
		}
	}
}

func TestReadMany(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	// Count the ReadPropertyMultiples sent, by device.
	var mu sync.Mutex
	rpms := map[netip.AddrPort]int{}
	c, heard := openOn(t, network, func(cfg *client.Config) {
		cfg.OnPacket = func(ev client.PacketEvent) {
			if ev.Inbound {
				return
			}
			msg, err := bacnet.ParseWithFallback(ev.Data)
			if err == nil && msg.GetType() == plumbing.ConfirmedReq && msg.GetService() == services.ServiceConfirmedReadPropMultiple {
				mu.Lock()
				rpms[ev.Peer]++
				mu.Unlock()
			}
		}
	})
	rpmsTo := func(d client.Device) int {
		mu.Lock()
		defer mu.Unlock()
		return rpms[d.Addr.IP]
	}
	devices := awaitDevices(t, c, heard, 1001, 3000, 4000)
	ctx := context.Background()

	t.Run("segmented replies take a whole batch at once", func(t *testing.T) {
		dev := devices[1001]
		ids, err := c.ReadObjectList(ctx, dev, nil)
		if err != nil {
			t.Fatal(err)
		}
		results, err := c.ReadMany(ctx, dev, nameSpecs(ids))
		if err != nil {
			t.Fatal(err)
		}
		checkNames(t, ids, results)
		if st := c.Stats().Peers[dev.Addr.String()]; st.Sent != 2 {
			t.Errorf("device 1001 counts %+v, want one request for the list and one for the names", st)
		}
	})

	t.Run("a device that cannot segment is asked for what fits", func(t *testing.T) {
		dev := devices[3000]
		ids, err := c.ReadObjectList(ctx, dev, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 241 {
			t.Fatalf("%d objects, want 241", len(ids))
		}
		results, err := c.ReadMany(ctx, dev, nameSpecs(ids))
		if err != nil {
			t.Fatal(err)
		}
		checkNames(t, ids, results)
	})

	t.Run("a device without ReadPropertyMultiple is read one property at a time, and asked once", func(t *testing.T) {
		dev := devices[4000]
		ids, err := c.ReadObjectList(ctx, dev, nil)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			results, err := c.ReadMany(ctx, dev, nameSpecs(ids))
			if err != nil {
				t.Fatal(err)
			}
			checkNames(t, ids, results)
		}
		if n := rpmsTo(dev); n != 1 {
			t.Errorf("device 4000 was sent %d ReadPropertyMultiples, want one, which it rejected", n)
		}
	})

	t.Run("a property the device does not have is an error in its result", func(t *testing.T) {
		dev := devices[1001]
		results, err := c.ReadMany(ctx, dev, []client.ReadSpec{{
			Object: client.ObjectID{Type: objects.ObjectTypeAnalogInput, Instance: 0},
			Props:  []uint16{objects.PropertyIdObjectName, objects.PropertyIdLastRestoreTime},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || len(results[0].Props) != 2 {
			t.Fatalf("results %+v, want one object with two properties", results)
		}
		if p := results[0].Props[1]; p.Err == nil || !client.Declined(p.Err) {
			t.Errorf("a property the object lacks gave %+v, want the device's error", p)
		}
	})

	if _, err := c.ReadMany(ctx, devices[1001], nil); err != nil {
		t.Errorf("reading nothing gave %v", err)
	}
}
