package client_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
)

// Find the devices on an adapter's subnet and read each one's name. The
// client reaches a device directly, behind a router or through a BBMD, and
// the Address its I-Am gives says which.
func Example() {
	// Listen on an adapter, by name or IPv4 address, and broadcast to its subnet.
	ifc, err := client.FindInterface("Ethernet 3")
	if err != nil {
		log.Fatal(err)
	}
	cfg := ifc.Config(client.DefaultPort)
	cfg.ReadOnly = true // refuse SubscribeCOV as well: leave nothing on any device

	// I-Ams arrive on the client's read loop; the callback must not block.
	var mu sync.Mutex
	var devices []client.Device
	cfg.OnIAm = func(d client.Device) {
		mu.Lock()
		devices = append(devices, d)
		mu.Unlock()
	}

	c, err := client.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// One Who-Is, addressed to every network, then a moment for the answers.
	if err := c.WhoIs(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	mu.Lock()
	found := devices
	mu.Unlock()
	for _, d := range found {
		obj := client.ObjectID{Type: objects.ObjectTypeDevice, Instance: d.Instance}
		values, err := c.ReadProperty(ctx, d.Addr, obj, objects.PropertyIdObjectName)
		if err != nil || len(values) == 0 {
			fmt.Printf("device %d at %s: %v\n", d.Instance, d.Addr, err)
			continue
		}
		fmt.Printf("device %d at %s is %q\n", d.Instance, d.Addr, values[0].Str)
	}
}
