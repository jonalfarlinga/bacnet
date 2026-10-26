package client

import (
	"context"
	"net/netip"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Opt in with BACNET_TEST_DEVICE=IP[:port]. All requests are read-only;
// writes and subscriptions remain disabled even when the device supports them.
func TestLiveReadOnly(t *testing.T) {
	remote := os.Getenv("BACNET_TEST_DEVICE")
	if remote == "" {
		t.Skip("set BACNET_TEST_DEVICE to run against a live device")
	}
	ip, err := netip.ParseAddrPort(remote)
	if err != nil {
		addr, err := netip.ParseAddr(remote)
		if err != nil {
			t.Fatal(err)
		}
		ip = netip.AddrPortFrom(addr, DefaultPort)
	}
	var segments atomic.Int32
	c, err := Open(Config{Local: netip.AddrPortFrom(netip.IPv4Unspecified(), 0), ReadOnly: true, Timeout: 2 * time.Second, MaxRequestsPerSec: 5,
		OnPacket: func(p PacketEvent) {
			if !p.Inbound {
				return
			}
			if msg, err := bacnet.Parse(p.Data); err == nil {
				if ack, ok := msg.(*services.ComplexACK); ok && ack.APDU.IsSegmented() {
					segments.Add(1)
				}
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	to := Address{IP: ip}
	dev, err := c.Probe(ctx, to, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("device %d, vendor %d, MaxAPDU %d", dev.Instance, dev.VendorID, dev.MaxAPDU)
	ids, err := c.ReadObjectList(ctx, dev, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("object list: %d objects", len(ids))
	obj := ObjectID{Type: objects.ObjectTypeDevice, Instance: dev.Instance}
	t.Run("repeated and wildcard Device specs", func(t *testing.T) {
		specs := []ReadSpec{
			{Object: obj, Props: []uint16{objects.PropertyIdObjectName}},
			{Object: obj, Props: []uint16{objects.PropertyIdModelName}},
			{Object: ObjectID{Type: objects.ObjectTypeDevice, Instance: WildcardInstance}, Props: []uint16{objects.PropertyIdObjectName}},
		}
		results, err := c.ReadMany(ctx, dev, specs)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range results {
			if len(r.Props) != 1 || r.Props[0].ID != uint32(specs[i].Props[0]) || r.Props[0].Err != nil || len(r.Props[0].Values) != 1 {
				t.Fatalf("spec %d: %+v", i, r)
			}
		}
	})
	t.Run("batch names", func(t *testing.T) {
		var specs []ReadSpec
		for _, id := range ids {
			specs = append(specs, ReadSpec{Object: id, Props: []uint16{objects.PropertyIdObjectName}})
		}
		results, err := c.ReadMany(ctx, dev, specs)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range results {
			if len(r.Props) != 1 || r.Props[0].Err != nil {
				t.Fatalf("object %d: %+v", i, r)
			}
		}
	})
	t.Run("segmented object list", func(t *testing.T) {
		req := plumbing.NewAPDU(plumbing.ConfirmedReq, services.ServiceConfirmedReadProperty, services.ConfirmedReadPropertyObjects(obj.Type, obj.Instance, objects.PropertyIdObjectList))
		req.SetAPDUFlags(true, false, false)
		req.MaxSeg = 7
		req.MaxSize = 0 // force small reply segments
		key, ch := c.begin(to, req)
		if ch == nil {
			t.Fatal("no invoke ID available")
		}
		defer func() { c.mu.Lock(); delete(c.pending, key); c.mu.Unlock() }()
		if err := c.limiter.wait(ctx, c.closed); err != nil {
			t.Fatal(err)
		}
		if err := c.sendAPDU(plumbing.BVLCFuncUnicast, ip, npduTo(to, true), req); err != nil {
			t.Fatal(err)
		}
		tags, err := c.await(ctx, ch, to, req.InvokeID, req.Service)
		if err != nil {
			t.Fatal(err)
		}
		values, err := propertyValue(tags)
		if err != nil {
			t.Fatal(err)
		}
		got := objectIDs(values)
		if len(got) != len(ids) {
			t.Fatalf("segmented list has %d objects, expected %d", len(got), len(ids))
		}
		for i := range ids {
			if got[i] != ids[i] {
				t.Fatalf("segmented object %d differs", i)
			}
		}
		t.Logf("received %d reply segments", segments.Load())
	})
}
