package client

import (
	"errors"
	"fmt"
	"net/netip"
)

// SourceRule names devices, for the writable switch. Exactly one field is
// set.
type SourceRule struct {
	// Instance matches one device, by its device instance. The client
	// learns which address that is from the device's I-Am, or from Knock.
	Instance *uint32
	// IP matches the devices at the BACnet/IP addresses in a prefix: a host
	// or a subnet. It does not match the devices routed through a router at
	// one of them; a routed network is named with Net.
	IP netip.Prefix
	// Net matches every device on one routed network.
	Net *uint16
}

func (r SourceRule) String() string {
	switch {
	case r.Instance != nil:
		return fmt.Sprintf("device %d", *r.Instance)
	case r.Net != nil:
		return fmt.Sprintf("network %d", *r.Net)
	case r.IP.IsValid():
		return r.IP.String()
	}
	return "nothing"
}

func (r SourceRule) validate() error {
	set := 0
	if r.Instance != nil {
		set++
	}
	if r.Net != nil {
		set++
	}
	if r.IP.IsValid() {
		set++
	}
	if set != 1 {
		return errors.New("a source rule sets exactly one of Instance, IP and Net")
	}
	return nil
}

// matches reports whether the rule covers the device at addr. known maps
// the device instances the client has heard to their addresses.
func (r SourceRule) matches(addr Address, known map[uint32]Address) bool {
	switch {
	case r.Instance != nil:
		at, ok := known[*r.Instance]
		return ok && sameDevice(at, addr)
	case r.Net != nil:
		return addr.Net == *r.Net
	case r.IP.IsValid():
		return !addr.Routed() && addr.IP.IsValid() && r.IP.Contains(addr.IP.Addr())
	}
	return false
}

// sameDevice reports whether two addresses name the same device: the same
// network and MAC for a routed one, the same BACnet/IP address otherwise.
func sameDevice(a, b Address) bool {
	if a.Routed() || b.Routed() {
		return a.Net == b.Net && string(a.MAC) == string(b.MAC)
	}
	return a.IP == b.IP
}

// learnInstance remembers which address a device instance answers from, so
// that a rule by instance can be applied to the address.
func (c *Client) learnInstance(instance uint32, addr Address) {
	c.mu.Lock()
	c.known[instance] = addr
	c.mu.Unlock()
}
