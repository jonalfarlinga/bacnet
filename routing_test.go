package bacnet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// An NPDU with both a destination and a source carries the hop count after
// the source address. These are frames from a routed live system.
func TestParseRoutedNPDU(t *testing.T) {
	t.Run("I-Am", func(t *testing.T) {
		msg, err := Parse(mustHex(t,
			"81040022 aabbccddbbbb 0128 ffff00 40f40101 fd 1000 c402195849 2201e0 9103 2125"))
		if err != nil {
			t.Fatal(err)
		}
		iam, ok := msg.(*services.UnconfirmedIAm)
		if !ok {
			t.Fatalf("got %T, want *services.UnconfirmedIAm", msg)
		}
		want := &plumbing.NPDU{
			Version: 1, Control: 0x28,
			DNET: 0xFFFF,
			SNET: 16628, SLEN: 1, SADR: []byte{0x01},
			Hop: 0xfd,
		}
		if diff := cmp.Diff(want, iam.NPDU); diff != "" {
			t.Errorf("NPDU differs: (-want +got)\n%s", diff)
		}
		dec, err := iam.Decode()
		if err != nil {
			t.Fatal(err)
		}
		if dec.InstanceNum != 1661001 || dec.VendorId != 37 {
			t.Errorf("got device %d vendor %d, want device 1661001 vendor 37", dec.InstanceNum, dec.VendorId)
		}
		origin, ok := iam.BVLC.OriginAddrPort()
		if want := netip.MustParseAddrPort("170.187.204.221:48059"); !ok || origin != want {
			t.Errorf("got origin %v (%t), want %v", origin, ok, want)
		}
	})

	t.Run("Who-Is with a 6-byte source address", func(t *testing.T) {
		msg, err := Parse(mustHex(t,
			"81040023 aabbccddbbbb 0128 ffff00 40dc06c0a80064bac0 fd 1008 0b195530 1b195530"))
		if err != nil {
			t.Fatal(err)
		}
		whois, ok := msg.(*services.UnconfirmedWhoIs)
		if !ok {
			t.Fatalf("got %T, want *services.UnconfirmedWhoIs", msg)
		}
		want := &plumbing.NPDU{
			Version: 1, Control: 0x28,
			DNET: 0xFFFF,
			SNET: 16604, SLEN: 6, SADR: []byte{0xc0, 0xa8, 0x00, 0x64, 0xba, 0xc0},
			Hop: 0xfd,
		}
		if diff := cmp.Diff(want, whois.NPDU); diff != "" {
			t.Errorf("NPDU differs: (-want +got)\n%s", diff)
		}
	})
}

// Frames written by versions of this library that put the hop count ahead of
// the source still parse, through the fallback only.
func TestParseLegacyNPDULayout(t *testing.T) {
	// I-Am to the global broadcast, hop count 255, then source network 2709.
	b := mustHex(t, "810a001e 0128 ffff00 ff 0a9503010203 1000c4020000652205c491002107")

	if _, err := Parse(b); err == nil {
		t.Error("Parse accepted the legacy layout")
	}
	msg, err := ParseWithFallback(b)
	if err != nil {
		t.Fatal(err)
	}
	iam, ok := msg.(*services.UnconfirmedIAm)
	if !ok {
		t.Fatalf("got %T, want *services.UnconfirmedIAm", msg)
	}
	want := &plumbing.NPDU{
		Version: 1, Control: 0x28,
		DNET: 0xFFFF,
		SNET: 2709, SLEN: 3, SADR: []byte{1, 2, 3},
		Hop: 0xff,
	}
	if diff := cmp.Diff(want, iam.NPDU); diff != "" {
		t.Errorf("NPDU differs: (-want +got)\n%s", diff)
	}
}

func TestParseNetworkMessage(t *testing.T) {
	testcases := []struct {
		description string
		serialized  string
		msgType     uint8
		networks    []uint16
	}{
		{
			"Who-Is-Router-To-Network, forwarded",
			"8104000f aabbccddbbbb 0180 00 3eb8",
			plumbing.NetMsgWhoIsRouterToNetwork, []uint16{16056},
		},
		{
			"Who-Is-Router-To-Network for every network",
			"810b0007 0180 00",
			plumbing.NetMsgWhoIsRouterToNetwork, []uint16{},
		},
		{
			"I-Am-Router-To-Network, forwarded",
			"81040013 aabbccddbbbb 0180 01 40d9 40dc 40db",
			plumbing.NetMsgIAmRouterToNetwork, []uint16{16601, 16604, 16603},
		},
		{
			"Router-Busy-To-Network",
			"810b0009 0180 04 0a95",
			plumbing.NetMsgRouterBusyToNetwork, []uint16{2709},
		},
	}
	for _, c := range testcases {
		t.Run(c.description, func(t *testing.T) {
			b := mustHex(t, c.serialized)
			msg, err := Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			m, ok := msg.(*plumbing.NetworkMessage)
			if !ok {
				t.Fatalf("got %T, want *plumbing.NetworkMessage", msg)
			}
			if m.GetType() != plumbing.NetworkLayerMessage || m.GetService() != c.msgType {
				t.Errorf("got type %#x service %#x, want %#x and %#x",
					m.GetType(), m.GetService(), plumbing.NetworkLayerMessage, c.msgType)
			}
			nets, err := m.Networks()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(c.networks, nets); diff != "" {
				t.Errorf("networks differ: (-want +got)\n%s", diff)
			}
			got, err := m.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, got) {
				t.Errorf("differs: want %x, got %x", b, got)
			}
		})
	}

	t.Run("Reject-Message-To-Network", func(t *testing.T) {
		msg, err := Parse(mustHex(t, "810a000a 0180 03 01 0a95"))
		if err != nil {
			t.Fatal(err)
		}
		reason, dnet, err := msg.(*plumbing.NetworkMessage).RejectReason()
		if err != nil {
			t.Fatal(err)
		}
		if reason != 1 || dnet != 2709 {
			t.Errorf("got reason %d network %d, want reason 1 network 2709", reason, dnet)
		}
	})

	t.Run("proprietary type carries a vendor ID", func(t *testing.T) {
		b := mustHex(t, "810a000b 0180 80 0007 beef")
		msg, err := Parse(b)
		if err != nil {
			t.Fatal(err)
		}
		m := msg.(*plumbing.NetworkMessage)
		if m.VendorID != 7 || !bytes.Equal(m.Data, []byte{0xbe, 0xef}) {
			t.Errorf("got vendor %d data %x, want vendor 7 data beef", m.VendorID, m.Data)
		}
		if got, _ := m.MarshalBinary(); !bytes.Equal(b, got) {
			t.Errorf("differs: want %x, got %x", b, got)
		}
	})

	t.Run("built", func(t *testing.T) {
		m := plumbing.NewNetworkMessage(
			plumbing.NewBVLC(plumbing.BVLCFuncBroadcast),
			plumbing.NewNPDU(false, false, false, false),
			plumbing.NetMsgWhoIsRouterToNetwork,
			plumbing.NetworksData(2709),
		)
		got, err := m.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if want := mustHex(t, "810b0009 0180 00 0a95"); !bytes.Equal(want, got) {
			t.Errorf("differs: want %x, got %x", want, got)
		}
	})
}

// A frame that starts at the NPDU is not a BACnet/IP datagram: Parse refuses
// it and ParseNPDU takes it.
func TestParseNPDU(t *testing.T) {
	// Who-Is-Router-To-Network for network 16908, from network 7999.
	b := mustHex(t, "0188 1f3f06aabbccddbac0 00 420c")

	if _, err := Parse(b); !errors.Is(err, common.ErrInvalidData) {
		t.Errorf("Parse: got error %v, want %v", err, common.ErrInvalidData)
	}

	msg, err := ParseNPDU(b)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := msg.(*plumbing.NetworkMessage)
	if !ok {
		t.Fatalf("got %T, want *plumbing.NetworkMessage", msg)
	}
	if m.NPDU.SNET != 7999 || !bytes.Equal(m.NPDU.SADR, []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xba, 0xc0}) {
		t.Errorf("got source network %d address %x", m.NPDU.SNET, m.NPDU.SADR)
	}
	nets, err := m.Networks()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]uint16{16908}, nets); diff != "" {
		t.Errorf("networks differ: (-want +got)\n%s", diff)
	}
}

func TestWhoHas(t *testing.T) {
	t.Run("by name, from the live capture", func(t *testing.T) {
		msg, err := Parse(mustHex(t, "810b0019 0120ffff00ff 1007 0900 1b3fffff 3d0500444d4133"))
		if err != nil {
			t.Fatal(err)
		}
		whohas, ok := msg.(*services.UnconfirmedWhoHas)
		if !ok {
			t.Fatalf("got %T, want *services.UnconfirmedWhoHas", msg)
		}
		got, err := whohas.Decode()
		if err != nil {
			t.Fatal(err)
		}
		want := services.UnconfirmedWhoHasDec{HasLimits: true, LowLimit: 0, HighLimit: 4194303, ObjectName: "DMA3"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("differs: (-want +got)\n%s", diff)
		}
	})

	t.Run("by identifier, built", func(t *testing.T) {
		u := services.NewUnconfirmedWhoHas(
			plumbing.NewBVLC(plumbing.BVLCFuncBroadcast),
			plumbing.NewNPDU(false, false, false, false),
		)
		u.APDU.Objects = services.WhoHasObjects(objects.ObjectTypeAnalogInput, 7, "")
		u.SetLength()
		b, err := u.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if want := mustHex(t, "810b000d 0100 1007 2c00000007"); !bytes.Equal(want, b) {
			t.Fatalf("differs: want %x, got %x", want, b)
		}

		msg, err := Parse(b)
		if err != nil {
			t.Fatal(err)
		}
		got, err := msg.(*services.UnconfirmedWhoHas).Decode()
		if err != nil {
			t.Fatal(err)
		}
		want := services.UnconfirmedWhoHasDec{
			ObjectID: &objects.ObjectIdentifier{ObjectType: objects.ObjectTypeAnalogInput, InstanceNumber: 7},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("differs: (-want +got)\n%s", diff)
		}
	})

	t.Run("neither name nor identifier", func(t *testing.T) {
		msg, err := Parse(mustHex(t, "810b000e 0100 1007 0900 1b3fffff"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := msg.(*services.UnconfirmedWhoHas).Decode(); !errors.Is(err, common.ErrWrongStructure) {
			t.Errorf("got error %v, want %v", err, common.ErrWrongStructure)
		}
	})
}
