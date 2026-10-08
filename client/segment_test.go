package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
)

// A reply longer than the client reassembles is aborted, and the error
// says by how much.
func TestResponseTooLarge(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, func(cfg *client.Config) { cfg.MaxSegments = 1 })
	devices := awaitDevices(t, c, heard, 2000)
	dev := devices[2000]

	// Device 2000's object list does not fit its 480-byte APDU.
	_, err = c.ReadProperty(context.Background(), dev.Addr, deviceObject(2000), objects.PropertyIdObjectList)
	if !errors.Is(err, client.ErrResponseTooLarge) {
		t.Fatalf("reading a long reply with one segment allowed gave %v, want it too large", err)
	}
	var tooLarge *client.ResponseTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("%v does not carry the sizes", err)
	}
	if tooLarge.MaxSegments != 1 || tooLarge.Segments != 2 || tooLarge.Bytes == 0 || tooLarge.Peer.IP != dev.Addr.IP {
		t.Errorf("the error says %+v, want 2 segments where 1 is allowed, from %v", tooLarge, dev.Addr)
	}
	if st := c.Stats().Peers[dev.Addr.String()]; st.ResponseTooLarge != 1 {
		t.Errorf("device 2000 counts %+v, want one reply too large", st)
	}

	// The device dropped the rest of the reply, and answers the next read.
	if _, err := c.ReadProperty(context.Background(), dev.Addr, deviceObject(2000), objects.PropertyIdObjectName); err != nil {
		t.Errorf("the read after the aborted one failed: %v", err)
	}
}

// A window larger than the client accepts is clamped, counted, and the
// reply still arrives whole.
func TestWindowClamped(t *testing.T) {
	stations := sim.Default()
	stations[1][0].Window = 100
	network, err := sim.StartDevices(stations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, nil)
	devices := awaitDevices(t, c, heard, 2000)
	dev := devices[2000]

	values, err := c.ReadProperty(context.Background(), dev.Addr, deviceObject(2000), objects.PropertyIdObjectList)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 124 {
		t.Errorf("object-list has %d elements, want 124", len(values))
	}
	if st := c.Stats().Peers[dev.Addr.String()]; st.WindowClamped != 1 {
		t.Errorf("device 2000 counts %+v, want one window clamped", st)
	}
}
