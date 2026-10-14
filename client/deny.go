package client

import (
	"errors"
	"fmt"
	"net/netip"
)

// SourceRule matches devices, for the deny list and the writable switch.
// Exactly one field is set.
type SourceRule struct {
	// Instance matches one device, by its device instance. The client
	// learns which address that is from the device's I-Am.
	Instance *uint32
	// IP matches the devices at the BACnet/IP addresses in a prefix: a host
	// or a subnet. It does not match the devices routed through a router at
	// one of them; a routed network is denied with Net.
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

// matches reports whether the rule covers the device at addr, whose instance
// is given when known and 0 otherwise. known maps the instances heard in
// I-Ams to their addresses.
func (r SourceRule) matches(addr Address, instance uint32, known map[uint32]Address) bool {
	switch {
	case r.Instance != nil:
		if instance != 0 {
			return instance == *r.Instance
		}
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

// SetDenied turns a source off, or back on. A denied source gets no
// unicast from the client: a request to it fails with ErrSourceDenied, as do
// the requests to it that are in flight. What it sends is dropped before it
// reaches OnIAm or a waiting request. A broadcast still reaches it on the
// wire, since a broadcast cannot leave one recipient out, but its reply is
// dropped.
func (c *Client) SetDenied(rule SourceRule, denied bool) error {
	if err := rule.validate(); err != nil {
		return err
	}
	c.mu.Lock()
	kept := make([]SourceRule, 0, len(c.denied)+1)
	for _, r := range c.denied {
		if r.String() != rule.String() {
			kept = append(kept, r)
		}
	}
	if denied {
		kept = append(kept, rule)
	}
	c.denied = kept
	c.mu.Unlock()

	if !denied {
		c.log.Info("source allowed again", "rule", rule)
		return nil
	}
	c.log.Info("source denied", "rule", rule)
	c.failWhere(func(key txKey) bool {
		return rule.matches(key.address(), 0, c.known)
	}, fmt.Errorf("%w: %s", ErrSourceDenied, rule))
	return nil
}

// Denied lists the rules in force.
func (c *Client) Denied() []SourceRule {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]SourceRule(nil), c.denied...)
}

// isDenied reports whether a device is on the deny list, and by which rule.
// instance is 0 when unknown.
func (c *Client) isDenied(addr Address, instance uint32) (SourceRule, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.denied {
		if r.matches(addr, instance, c.known) {
			return r, true
		}
	}
	return SourceRule{}, false
}

// learnInstance remembers which address a device instance answers from, so
// that a rule by instance can be applied to the address.
func (c *Client) learnInstance(instance uint32, addr Address) {
	c.mu.Lock()
	c.known[instance] = addr
	c.mu.Unlock()
}

// failWhere ends every request in flight whose peer the predicate picks.
// The predicate runs with the client's lock held.
func (c *Client) failWhere(pick func(txKey) bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, ch := range c.pending {
		if pick(key) {
			select {
			case ch <- inbound{err: err}:
			default:
			}
		}
	}
}
