package client_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
)

// notes collects the notifications of a subscription.
type notes struct {
	mu   sync.Mutex
	list []client.COVNotification
}

func (n *notes) add(note client.COVNotification) {
	n.mu.Lock()
	n.list = append(n.list, note)
	n.mu.Unlock()
}

func (n *notes) all() []client.COVNotification {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]client.COVNotification(nil), n.list...)
}

// subscribed opens a client on the simulated devices and returns it with
// the network and the devices that answered its Who-Is.
func subscribed(t *testing.T, change func(*client.Config)) (*client.Client, *sim.Network, map[uint32]client.Device) {
	t.Helper()
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, change)
	return c, network, awaitDevices(t, c, heard, 1001, 2101)
}

var supplyAir = client.ObjectID{Type: objects.ObjectTypeAnalogInput, Instance: 0}

func simObject(id client.ObjectID) objects.ObjectIdentifier {
	return objects.ObjectIdentifier{ObjectType: id.Type, InstanceNumber: id.Instance}
}

func TestSubscribeCOV(t *testing.T) {
	c, network, devices := subscribed(t, nil)
	ctx := context.Background()

	var got notes
	sub, err := c.SubscribeCOV(ctx, devices[1001].Addr, supplyAir, client.COVOptions{}, got.add)
	if err != nil {
		t.Fatal(err)
	}

	// The device tells a new subscriber the value as it is.
	waitFor(t, "the first notification", func() bool { return len(got.all()) == 1 })
	n := got.all()[0]
	if n.Device != 1001 || n.Object != supplyAir || n.ProcessID != client.DefaultCOVProcessID || n.Confirmed {
		t.Errorf("notification is %+v, want an unconfirmed one for the analog input of device 1001", n)
	}
	if n.Remaining < 4*time.Minute || n.Remaining > client.DefaultCOVLifetime {
		t.Errorf("%s remaining, want about %s", n.Remaining, client.DefaultCOVLifetime)
	}
	if len(n.Props) != 2 || n.Props[0].ID != uint32(objects.PropertyIdPresentValue) || n.Props[1].ID != uint32(objects.PropertyIdStatusFlags) {
		t.Fatalf("properties are %+v, want the present value and the status flags", n.Props)
	}
	if v := n.Props[0].Values; len(v) != 1 || v[0].Kind != client.KindReal || v[0].Float < 50 || v[0].Float > 60 {
		t.Errorf("present-value is %+v, want a real number near 55", v)
	}
	if v := n.Props[1].Values; len(v) != 1 || v[0].Kind != client.KindBits || len(v[0].Bits) != 4 {
		t.Errorf("status-flags is %+v, want 4 bits", v)
	}

	held := network.COVSubscriptions(1001)
	if len(held) != 1 || held[0].Subscriber != c.Local() || held[0].Confirmed || held[0].Lifetime != client.DefaultCOVLifetime {
		t.Fatalf("the device holds %+v, want one unconfirmed subscription of the client's for 5 minutes", held)
	}

	if sent := network.NotifyCOV(1001, simObject(supplyAir)); sent != 1 {
		t.Fatalf("the device notified %d subscribers, want 1", sent)
	}
	waitFor(t, "the second notification", func() bool { return len(got.all()) == 2 })
	if st := sub.Status(); st.Notifications != 2 || st.Last.IsZero() || st.Err != nil || !st.Expires.After(time.Now()) {
		t.Errorf("status is %+v, want 2 notifications and a subscription in force", st)
	}

	// The same object twice is one subscription to the device.
	if _, err := c.SubscribeCOV(ctx, devices[1001].Addr, supplyAir, client.COVOptions{}, got.add); !errors.Is(err, client.ErrSubscribed) {
		t.Errorf("subscribing again gave %v, want ErrSubscribed", err)
	}

	if err := sub.Cancel(ctx); err != nil {
		t.Fatal(err)
	}
	if held := network.COVSubscriptions(1001); len(held) != 0 {
		t.Errorf("the device still holds %+v after the cancellation", held)
	}
	if err := sub.Cancel(ctx); err != nil {
		t.Errorf("cancelling twice gave %v", err)
	}
	// And the object can be subscribed to anew.
	again, err := c.SubscribeCOV(ctx, devices[1001].Addr, supplyAir, client.COVOptions{}, got.add)
	if err != nil {
		t.Fatalf("subscribing after the cancellation: %v", err)
	}
	again.Cancel(ctx)
}

// A confirmed notification is acknowledged, from a device behind a router as
// from one with an address of its own.
func TestConfirmedCOVIsAcknowledged(t *testing.T) {
	c, network, devices := subscribed(t, nil)
	ctx := context.Background()

	for _, instance := range []uint32{1001, 2101} {
		var got notes
		sub, err := c.SubscribeCOV(ctx, devices[instance].Addr, supplyAir, client.COVOptions{Confirmed: true, ProcessID: 7}, got.add)
		if err != nil {
			t.Fatalf("device %d: %v", instance, err)
		}
		network.NotifyCOV(instance, simObject(supplyAir))
		waitFor(t, "both notifications to be acknowledged", func() bool {
			held := network.COVSubscriptions(instance)
			return len(held) == 1 && held[0].Sent == 2 && held[0].Acked == 2
		})
		for _, n := range got.all() {
			if !n.Confirmed || n.Device != instance || n.ProcessID != 7 {
				t.Errorf("device %d: notification is %+v, want a confirmed one for process 7", instance, n)
			}
		}
		if len(got.all()) != 2 {
			t.Errorf("device %d: %d notifications delivered, want 2", instance, len(got.all()))
		}
		if err := sub.Cancel(ctx); err != nil {
			t.Errorf("device %d: %v", instance, err)
		}
	}
}

// The client renews a subscription halfway through its lifetime, and keeps
// trying while the device does not answer.
func TestCOVIsRenewed(t *testing.T) {
	c, network, devices := subscribed(t, func(cfg *client.Config) {
		cfg.Timeout, cfg.Retries = 200*time.Millisecond, 0
	})
	ctx := context.Background()

	var got notes
	sub, err := c.SubscribeCOV(ctx, devices[1001].Addr, supplyAir, client.COVOptions{Lifetime: 2 * time.Second}, got.add)
	if err != nil {
		t.Fatal(err)
	}
	first := sub.Status().Expires
	waitFor(t, "the renewal", func() bool {
		held := network.COVSubscriptions(1001)
		return len(held) == 1 && held[0].Requests == 2
	})
	waitFor(t, "the renewal to be noted", func() bool { return sub.Status().Expires.After(first) })

	// The air handler is station 0.
	network.Silence(0, true)
	waitFor(t, "a renewal to fail", func() bool { return errors.Is(sub.Status().Err, client.ErrTimeout) })
	network.Silence(0, false)
	waitFor(t, "the subscription to be made again", func() bool {
		return sub.Status().Err == nil && len(network.COVSubscriptions(1001)) == 1
	})
	if err := sub.Cancel(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSubscribeCOVDeclined(t *testing.T) {
	c, network, devices := subscribed(t, nil)
	ctx := context.Background()
	nothing := func(client.COVNotification) {}

	missing := client.ObjectID{Type: objects.ObjectTypeAnalogInput, Instance: 99}
	for range 2 {
		// Twice: a subscription the device declined is not held.
		_, err := c.SubscribeCOV(ctx, devices[1001].Addr, missing, client.COVOptions{}, nothing)
		var be *client.BACnetError
		if !errors.As(err, &be) || be.Code != client.ErrorCodeUnknownObject {
			t.Errorf("subscribing to an object that is not there gave %v, want unknown object", err)
		}
	}
	if _, err := c.SubscribeCOV(ctx, devices[1001].Addr, supplyAir, client.COVOptions{}, nil); err == nil {
		t.Error("subscribed with nowhere to deliver the notifications")
	}
	if held := network.COVSubscriptions(1001); len(held) != 0 {
		t.Errorf("the device holds %+v, want nothing", held)
	}
}

// A read-only client does not subscribe.
func TestReadOnlyClientDoesNotSubscribe(t *testing.T) {
	c, network, devices := subscribed(t, func(cfg *client.Config) { cfg.ReadOnly = true })

	_, err := c.SubscribeCOV(context.Background(), devices[1001].Addr, supplyAir, client.COVOptions{}, func(client.COVNotification) {})
	if !errors.Is(err, client.ErrServiceNotAllowed) {
		t.Errorf("subscribing gave %v, want ErrServiceNotAllowed", err)
	}
	if err := c.CancelCOV(context.Background(), devices[1001].Addr, supplyAir, 0); !errors.Is(err, client.ErrServiceNotAllowed) {
		t.Errorf("cancelling gave %v, want ErrServiceNotAllowed", err)
	}
	if held := network.COVSubscriptions(1001); len(held) != 0 {
		t.Errorf("the device holds %+v, want nothing", held)
	}
}

// A notification for a subscription the client does not hold, such as one an
// earlier run left behind, gets no acknowledgement, and CancelCOV ends it.
func TestCOVLeftBehind(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	ctx := context.Background()

	earlier, heard := openOn(t, network, nil)
	dev := awaitDevices(t, earlier, heard, 1001)[1001]
	if _, err := earlier.SubscribeCOV(ctx, dev.Addr, supplyAir, client.COVOptions{Confirmed: true, ProcessID: 7}, func(client.COVNotification) {}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the first notification to be acknowledged", func() bool {
		held := network.COVSubscriptions(1001)
		return len(held) == 1 && held[0].Acked == 1
	})
	local := earlier.Local()
	earlier.Close()

	// A client at the same address, which knows nothing of the subscription.
	var sent atomic.Int32
	c, _ := openOn(t, network, func(cfg *client.Config) {
		cfg.Local = local
		cfg.OnPacket = func(e client.PacketEvent) {
			if !e.Inbound {
				sent.Add(1)
			}
		}
	})
	network.NotifyCOV(1001, simObject(supplyAir))
	// The answer to a read comes after the notification, so the client has
	// dealt with the notification by the time it has the answer.
	if _, err := c.ReadProperty(ctx, dev.Addr, supplyAir, objects.PropertyIdPresentValue); err != nil {
		t.Fatal(err)
	}
	if n := sent.Load(); n != 1 {
		t.Errorf("the client sent %d datagrams, want the read alone", n)
	}
	if held := network.COVSubscriptions(1001); len(held) != 1 || held[0].Sent != 2 || held[0].Acked != 1 {
		t.Errorf("the device holds %+v, want 2 notifications sent and only the first acknowledged", held)
	}

	if err := c.CancelCOV(ctx, dev.Addr, supplyAir, 7); err != nil {
		t.Fatal(err)
	}
	if held := network.COVSubscriptions(1001); len(held) != 0 {
		t.Errorf("the device still holds %+v after the cancellation", held)
	}
}
