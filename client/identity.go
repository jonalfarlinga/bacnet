package client

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Identity is the BACnet device a program built on the client is. The client
// has none of its own: with an Identity it says I-Am for the program, on
// request and when a Who-Is asks, and without one it stays silent.
type Identity struct {
	// Instance is the device instance, from 0 to MaxIdentityInstance.
	Instance uint32
	// VendorID is the vendor identifier ASHRAE assigned.
	VendorID uint16
}

// MaxIdentityInstance is the highest instance a device may have: the one
// above it is the wildcard.
const MaxIdentityInstance = services.MaxInstance - 1

// What the I-Am says of the client itself, whatever the program is: the
// longest APDU it takes, and that it neither takes a segmented request nor
// sends a segmented reply.
const (
	identityMaxAPDU      = 1476
	identitySegmentation = 3 // BACnetSegmentation: none
)

// WhoIsRequest is a Who-Is the client heard.
type WhoIsRequest struct {
	// From is where it came from.
	From Address
	// Ranged says it asks only the devices with an instance from Low to
	// High; otherwise it asks every device.
	Ranged    bool
	Low, High uint32
}

// Asks reports whether the Who-Is asks the device with this instance.
func (w WhoIsRequest) Asks(instance uint32) bool {
	return !w.Ranged || (instance >= w.Low && instance <= w.High)
}

func (id *Identity) validate() error {
	if id.Instance > MaxIdentityInstance {
		return fmt.Errorf("identity: device instance %d is above %d", id.Instance, MaxIdentityInstance)
	}
	return nil
}

func (id *Identity) iAm() *plumbing.APDU {
	return plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedIAm,
		services.IAmObjects(id.Instance, identityMaxAPDU, identitySegmentation, id.VendorID))
}

// iAmAllowed reports whether an APDU is the I-Am of the identity the client
// was given: of that device, and of no other.
func iAmAllowed(a *plumbing.APDU, id *Identity) bool {
	if id == nil || a.Type != plumbing.UnConfirmedReq || a.Service != services.ServiceUnconfirmedIAm || len(a.Objects) != 4 {
		return false
	}
	t, ok := a.Objects[0].(*objects.Object)
	if !ok || t.TagClass || t.TagNumber != objects.TagBACnetObjectIdentifier || len(t.Data) != 4 {
		return false
	}
	return objectIDOf(t.Data) == ObjectID{Type: objects.ObjectTypeDevice, Instance: id.Instance}
}

// IAm broadcasts the I-Am of Config.Identity to every network: what a device
// does when it starts. Without an identity, or with Config.ReadOnly, it fails
// with ErrServiceNotAllowed. A Who-Is that asks for the identity is answered
// without it.
func (c *Client) IAm() error {
	if c.cfg.Identity == nil {
		return fmt.Errorf("I-Am without an identity: %w", ErrServiceNotAllowed)
	}
	if err := c.limiter.wait(context.Background(), c.closed); err != nil {
		return err
	}
	npdu := plumbing.NewNPDU(false, true, false, false)
	npdu.DNET = 0xFFFF
	npdu.Hop = 255
	to, function := c.broadcastTarget()
	return c.sendAPDU(function, to, npdu, c.cfg.Identity.iAm())
}

// heardWhoIs takes a Who-Is: it tells the program, and answers for the
// identity when the Who-Is asks for it.
func (c *Client) heardWhoIs(src netip.AddrPort, m *services.UnconfirmedWhoIs) {
	c.noteSource(src, m.NPDU)
	req := WhoIsRequest{From: sourceOf(src, m.NPDU)}
	switch len(m.APDU.Objects) {
	case 0:
	case 2:
		low, lowOK := m.APDU.Objects[0].(*objects.Object)
		high, highOK := m.APDU.Objects[1].(*objects.Object)
		if !lowOK || !highOK || !isContext(low, 0) || !isContext(high, 1) ||
			len(low.Data) == 0 || len(low.Data) > 4 || len(high.Data) == 0 || len(high.Data) > 4 {
			c.log.Debug("dropped a Who-Is that could not be decoded", "from", req.From)
			return
		}
		req.Ranged, req.Low, req.High = true, uint32(beUint(low.Data)), uint32(beUint(high.Data))
	default:
		c.log.Debug("dropped a Who-Is that could not be decoded", "from", req.From)
		return
	}
	if c.cfg.OnWhoIs != nil {
		c.cfg.OnWhoIs(req)
	}

	id := c.cfg.Identity
	if id == nil || c.cfg.ReadOnly || !req.Asks(id.Instance) {
		return
	}
	// A router takes the network off a message it delivers: one that still
	// names a network, other than all of them, is for the devices of that
	// network, and the client is on none it knows the number of.
	if m.NPDU.Control&0x20 != 0 && m.NPDU.DNET != 0xFFFF {
		return
	}
	c.log.Debug("answering a Who-Is", "from", req.From, "device", id.Instance)
	if m.BVLC.Function == plumbing.BVLCFuncUnicast {
		// Asked alone, it answers the one that asked: a broadcast may not
		// reach it.
		c.sendAPDU(plumbing.BVLCFuncUnicast, src, npduTo(req.From, false), id.iAm())
		return
	}
	// Otherwise a broadcast, on the network the Who-Is came from.
	npdu := plumbing.NewNPDU(false, false, false, false)
	if req.From.Routed() {
		npdu = npduTo(Address{Net: req.From.Net}, false)
	}
	to, function := c.broadcastTarget()
	c.sendAPDU(function, to, npdu, id.iAm())
}
