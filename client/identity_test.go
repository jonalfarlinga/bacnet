package client_test

import (
	"errors"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// listen opens a socket on the loopback interface that stands in for a
// device, or for the subnet's broadcast address.
func listen(t *testing.T) (*net.UDPConn, netip.AddrPort) {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, conn.LocalAddr().(*net.UDPAddr).AddrPort()
}

// openIdentity opens a client that is device 77 of vendor 260, with a change
// to the configuration if one is given, and returns it with the socket its
// broadcasts go to and the Who-Is it has heard.
func openIdentity(t *testing.T, change func(*client.Config)) (*client.Client, *net.UDPConn, func() []client.WhoIsRequest) {
	t.Helper()
	subnet, broadcast := listen(t)
	var mu sync.Mutex
	var asked []client.WhoIsRequest
	cfg := client.Config{
		Local:     netip.MustParseAddrPort("127.0.0.1:0"),
		Broadcast: broadcast,
		Timeout:   time.Second,
		Identity:  &client.Identity{Instance: 77, VendorID: 260},
		OnWhoIs: func(w client.WhoIsRequest) {
			mu.Lock()
			asked = append(asked, w)
			mu.Unlock()
		},
	}
	if change != nil {
		change(&cfg)
	}
	c, err := client.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c, subnet, func() []client.WhoIsRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]client.WhoIsRequest(nil), asked...)
	}
}

// sendWhoIs sends a Who-Is from a socket to the client.
func sendWhoIs(t *testing.T, from *net.UDPConn, to netip.AddrPort, function uint8, npdu *plumbing.NPDU, payload []objects.APDUPayload) {
	t.Helper()
	bvlc := plumbing.NewBVLC(function)
	whoIs := plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedWhoIs, payload)
	b := make([]byte, bvlc.MarshalLen()+npdu.MarshalLen()+whoIs.MarshalLen())
	bvlc.Length = uint16(len(b))
	bvlc.MarshalTo(b)
	npdu.MarshalTo(b[bvlc.MarshalLen():])
	if err := whoIs.MarshalTo(b[bvlc.MarshalLen()+npdu.MarshalLen():]); err != nil {
		t.Fatal(err)
	}
	if _, err := from.WriteToUDPAddrPort(b, to); err != nil {
		t.Fatal(err)
	}
}

// awaitIAm reads the I-Am that arrives at a socket, or nil when none does.
func awaitIAm(t *testing.T, at *net.UDPConn, wait time.Duration) *services.UnconfirmedIAm {
	t.Helper()
	at.SetReadDeadline(time.Now().Add(wait))
	buf := make([]byte, 2048)
	n, _, err := at.ReadFromUDPAddrPort(buf)
	if err != nil {
		return nil
	}
	msg, err := bacnet.Parse(buf[:n])
	if err != nil {
		t.Fatalf("the client sent what cannot be parsed: %v", err)
	}
	iAm, ok := msg.(*services.UnconfirmedIAm)
	if !ok {
		t.Fatalf("the client sent a %T, want an I-Am", msg)
	}
	return iAm
}

const (
	answer   = 2 * time.Second
	noAnswer = 150 * time.Millisecond
)

func plain() *plumbing.NPDU { return plumbing.NewNPDU(false, false, false, false) }

// A client with an identity answers the Who-Is that asks for it: a broadcast
// with a broadcast on the network it came from, a unicast with a unicast.
func TestWhoIsAnsweredForIdentity(t *testing.T) {
	c, subnet, asked := openIdentity(t, nil)
	asker, askerAddr := listen(t)

	sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, plain(), nil)
	iAm := awaitIAm(t, subnet, answer)
	if iAm == nil {
		t.Fatal("a broadcast Who-Is drew no I-Am on the subnet")
	}
	d, err := iAm.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if d.InstanceNum != 77 || d.DeviceType != objects.ObjectTypeDevice || d.VendorId != 260 ||
		d.MaxAPDULength != 1476 || d.SegmentationSupported != 3 {
		t.Errorf("the I-Am says %+v, want device 77 of vendor 260, 1476 bytes, no segmentation", d)
	}
	if iAm.BVLC.Function != plumbing.BVLCFuncBroadcast || iAm.NPDU.Control&0x20 != 0 {
		t.Errorf("the I-Am came as BVLC function %#x with NPDU control %#x, want a local broadcast",
			iAm.BVLC.Function, iAm.NPDU.Control)
	}
	if got := asked(); len(got) != 1 || got[0].Ranged || got[0].From.IP != askerAddr {
		t.Errorf("OnWhoIs was told %+v, want one Who-Is for every device from %v", got, askerAddr)
	}

	// Asked alone, it answers the one that asked.
	sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncUnicast, plain(), nil)
	if iAm := awaitIAm(t, asker, answer); iAm == nil || iAm.BVLC.Function != plumbing.BVLCFuncUnicast {
		t.Errorf("a unicast Who-Is drew %+v, want a unicast I-Am", iAm)
	}
	if awaitIAm(t, subnet, noAnswer) != nil {
		t.Error("a unicast Who-Is drew a broadcast as well")
	}

	// A Who-Is from behind a router is answered on the network it came from.
	routed := plumbing.NewNPDU(false, false, true, false)
	routed.SNET, routed.SLEN, routed.SADR = 5, 1, []byte{9}
	sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, routed, nil)
	iAm = awaitIAm(t, subnet, answer)
	if iAm == nil || iAm.NPDU.Control&0x20 == 0 || iAm.NPDU.DNET != 5 || iAm.NPDU.DLEN != 0 {
		t.Errorf("a Who-Is from network 5 drew %+v, want an I-Am broadcast on network 5", iAm)
	}

	// One addressed to every network is for the client too; one addressed to
	// a network is for that network's devices.
	global := plumbing.NewNPDU(false, true, false, false)
	global.DNET, global.Hop = 0xFFFF, 255
	sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, global, nil)
	if awaitIAm(t, subnet, answer) == nil {
		t.Error("a Who-Is to every network drew no I-Am")
	}
	remote := plumbing.NewNPDU(false, true, false, false)
	remote.DNET, remote.Hop = 7, 255
	sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, remote, nil)
	if awaitIAm(t, subnet, noAnswer) != nil {
		t.Error("a Who-Is for the devices of network 7 drew an I-Am")
	}
}

// A Who-Is for a range of instances is answered only when the identity is in
// it, and OnWhoIs hears of it either way.
func TestWhoIsRangeAnsweredForIdentity(t *testing.T) {
	c, subnet, asked := openIdentity(t, nil)
	asker, _ := listen(t)

	for _, tc := range []struct {
		low, high uint32
		want      bool
	}{
		{0, 76, false},
		{78, 4194303, false},
		{77, 77, true},
		{0, 4194303, true},
	} {
		payload, err := services.WhoIsObjects(tc.low, tc.high)
		if err != nil {
			t.Fatal(err)
		}
		sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, plain(), payload)
		wait := noAnswer
		if tc.want {
			wait = answer
		}
		if got := awaitIAm(t, subnet, wait) != nil; got != tc.want {
			t.Errorf("a Who-Is for %d to %d: answered %t, want %t", tc.low, tc.high, got, tc.want)
		}
	}
	got := asked()
	if len(got) != 4 || !got[0].Ranged || got[0].Low != 0 || got[0].High != 76 || got[0].Asks(77) || !got[2].Asks(77) {
		t.Errorf("OnWhoIs was told %+v, want the four ranges", got)
	}
}

// Without an identity the client hears a Who-Is and says nothing, and so
// does a read-only client that has one.
func TestWhoIsNotAnswered(t *testing.T) {
	for name, change := range map[string]func(*client.Config){
		"no identity": func(cfg *client.Config) { cfg.Identity = nil },
		"read-only":   func(cfg *client.Config) { cfg.ReadOnly = true },
	} {
		t.Run(name, func(t *testing.T) {
			c, subnet, asked := openIdentity(t, change)
			asker, _ := listen(t)
			sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncBroadcast, plain(), nil)
			sendWhoIs(t, asker, c.Local(), plumbing.BVLCFuncUnicast, plain(), nil)
			if awaitIAm(t, subnet, noAnswer) != nil || awaitIAm(t, asker, noAnswer) != nil {
				t.Error("the Who-Is drew an I-Am")
			}
			if got := asked(); len(got) != 2 {
				t.Errorf("OnWhoIs was told %+v, want both", got)
			}
			if err := c.IAm(); !errors.Is(err, client.ErrServiceNotAllowed) {
				t.Errorf("IAm gave %v, want it refused", err)
			}
			if awaitIAm(t, subnet, noAnswer) != nil {
				t.Error("IAm sent an I-Am")
			}
		})
	}
}

// IAm announces the identity to every network.
func TestIAm(t *testing.T) {
	c, subnet, _ := openIdentity(t, nil)
	if err := c.IAm(); err != nil {
		t.Fatal(err)
	}
	iAm := awaitIAm(t, subnet, answer)
	if iAm == nil {
		t.Fatal("IAm sent nothing")
	}
	d, err := iAm.Decode()
	if err != nil || d.InstanceNum != 77 {
		t.Errorf("the I-Am says %+v, %v, want device 77", d, err)
	}
	if iAm.BVLC.Function != plumbing.BVLCFuncBroadcast || iAm.NPDU.Control&0x20 == 0 || iAm.NPDU.DNET != 0xFFFF {
		t.Errorf("the I-Am came as BVLC function %#x to network %d, want a broadcast to every network",
			iAm.BVLC.Function, iAm.NPDU.DNET)
	}
}

// One client finds another that has an identity.
func TestClientFindsClientWithIdentity(t *testing.T) {
	device, _, _ := openIdentity(t, nil)
	var mu sync.Mutex
	var heard []client.Device
	finder, err := client.Open(client.Config{
		Local:   netip.MustParseAddrPort("127.0.0.1:0"),
		Timeout: time.Second,
		OnIAm: func(d client.Device) {
			mu.Lock()
			heard = append(heard, d)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { finder.Close() })

	if err := finder.WhoIsAt(device.Local()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(answer)
	for {
		mu.Lock()
		got := append([]client.Device(nil), heard...)
		mu.Unlock()
		if len(got) == 1 && got[0].Instance == 77 && got[0].Addr.IP == device.Local() && got[0].VendorID == 260 {
			return
		}
		if len(got) > 1 || time.Now().After(deadline) {
			t.Fatalf("the finder heard %+v, want device 77 at %v", got, device.Local())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// An identity has a device instance, not the wildcard.
func TestIdentityInstance(t *testing.T) {
	_, err := client.Open(client.Config{
		Local:    netip.MustParseAddrPort("127.0.0.1:0"),
		Identity: &client.Identity{Instance: 4194303},
	})
	if err == nil {
		t.Fatal("a client opened as the wildcard instance")
	}
}
