package bacnet

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/plumbing"
)

// The BVLC functions a foreign device uses, and the table reads, parse from
// the bytes a BBMD exchanges and marshal back to them.
func TestParseBVLCOnlyMessages(t *testing.T) {
	testcases := []struct {
		description string
		serialized  string
		check       func(t *testing.T, msg plumbing.BACnet)
	}{
		{
			"Register-Foreign-Device for 300 seconds",
			"8105 0006 012c",
			func(t *testing.T, msg plumbing.BACnet) {
				r, ok := msg.(*plumbing.RegisterForeignDevice)
				if !ok {
					t.Fatalf("got %T, want *plumbing.RegisterForeignDevice", msg)
				}
				if r.TTL != 300 {
					t.Errorf("TTL is %d, want 300", r.TTL)
				}
			},
		},
		{
			"Read-Broadcast-Distribution-Table",
			"8102 0004",
			func(t *testing.T, msg plumbing.BACnet) {
				if _, ok := msg.(*plumbing.ReadBroadcastDistributionTable); !ok {
					t.Fatalf("got %T, want *plumbing.ReadBroadcastDistributionTable", msg)
				}
			},
		},
		{
			"Read-Broadcast-Distribution-Table-Ack with two entries",
			"8103 0018 0a007c02 bac0 ffffffff 0a007d02 bac0 ffffff00",
			func(t *testing.T, msg plumbing.BACnet) {
				a, ok := msg.(*plumbing.ReadBroadcastDistributionTableAck)
				if !ok {
					t.Fatalf("got %T, want *plumbing.ReadBroadcastDistributionTableAck", msg)
				}
				if len(a.Entries) != 2 {
					t.Fatalf("%d entries, want 2", len(a.Entries))
				}
				if got := a.Entries[0].AddrPort().String(); got != "10.0.124.2:47808" {
					t.Errorf("first entry is %s, want 10.0.124.2:47808", got)
				}
				if a.Entries[1].Mask != [4]byte{0xff, 0xff, 0xff, 0x00} {
					t.Errorf("second mask is %x, want ffffff00", a.Entries[1].Mask)
				}
			},
		},
		{
			"Register-Foreign-Device NAK",
			"8100 0006 0030",
			func(t *testing.T, msg plumbing.BACnet) {
				r, ok := msg.(*plumbing.BVLCResult)
				if !ok {
					t.Fatalf("got %T, want *plumbing.BVLCResult", msg)
				}
				if r.ResultCode != plumbing.BVLCResultRegisterForeignDeviceNAK {
					t.Errorf("result is %#x, want Register-Foreign-Device NAK", r.ResultCode)
				}
			},
		},
	}
	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			b := mustHex(t, c.serialized)
			msg, err := Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			c.check(t, msg)
			got, err := msg.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, got) {
				t.Errorf("differs: want %x, got %x", b, got)
			}
		})
	}

	t.Run("built", func(t *testing.T) {
		got, err := plumbing.NewRegisterForeignDevice(300).MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if want := mustHex(t, "8105 0006 012c"); !bytes.Equal(want, got) {
			t.Errorf("differs: want %x, got %x", want, got)
		}
		got, err = plumbing.NewReadBroadcastDistributionTableAck([]plumbing.BDTEntry{
			{IP: [4]byte{10, 0, 124, 2}, Port: 47808, Mask: [4]byte{255, 255, 255, 255}},
		}).MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if want := mustHex(t, "8103 000e 0a007c02 bac0 ffffffff"); !bytes.Equal(want, got) {
			t.Errorf("differs: want %x, got %x", want, got)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		for _, h := range []string{"8105 0006 01", "8103 000d 0a007c02 bac0 ffffff"} {
			_, err := Parse(mustHex(t, h))
			if !errors.Is(err, common.ErrTooShortToParse) && !errors.Is(err, common.ErrInvalidData) {
				t.Errorf("%s parsed with %v, want a too-short or invalid-data error", h, err)
			}
		}
	})
}

// A Distribute-Broadcast-To-Network wraps an NPDU like a broadcast does.
func TestParseDistributeBroadcast(t *testing.T) {
	b := mustHex(t, "8109 000c 0120 ffff 00 ff 1008")
	msg, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := msg.(interface{ GetService() uint8 }); !ok {
		t.Fatalf("got %T", msg)
	}
	got, err := msg.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, got) {
		t.Errorf("differs: want %x, got %x", b, got)
	}
}
