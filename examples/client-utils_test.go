package main

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
)

func TestWaitForeignDeviceBeforeDiscovery(t *testing.T) {
	n, err := sim.StartDevices(sim.RemoteSite(false))
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	gate := make(chan struct{})
	heard := make(chan client.Device, 8)
	c, err := client.Open(client.Config{
		Local: netip.MustParseAddrPort("127.0.0.1:0"), Timeout: time.Second,
		ForeignDevice: &client.ForeignDeviceConfig{BBMD: n.Addr()},
		OnIAm:         func(d client.Device) { heard <- d },
		OnPacket: func(p client.PacketEvent) {
			if !p.Inbound && p.Data[1] == 5 {
				<-gate
			}
		},
	})
	if err != nil {
		close(gate)
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- waitForeignDevice(ctx, c) }()
	select {
	case err := <-done:
		t.Errorf("returned before registration: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(gate)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := c.WhoIs(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-heard:
	case <-ctx.Done():
		t.Fatal("no device answered after registration")
	}
}

func TestWaitForeignDeviceRefusal(t *testing.T) {
	n, err := sim.StartDevices(sim.RemoteSite(true))
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	c, err := client.Open(client.Config{Local: netip.MustParseAddrPort("127.0.0.1:0"), ForeignDevice: &client.ForeignDeviceConfig{BBMD: n.Addr()}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var refused *client.BVLCResultError
	if err := waitForeignDevice(ctx, c); !errors.As(err, &refused) {
		t.Fatalf("registration refusal: %v", err)
	}
}
