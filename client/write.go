package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// ErrNotSent is a request that never left the client. An error that wraps
// it wraps the reason as well: ErrServiceNotAllowed, the context's error,
// ErrClosed, or no route to the device. An error that does not wrap it,
// ErrTimeout among them, means the request was sent, so a write may have
// been done.
var ErrNotSent = errors.New("not sent")

// notSent marks an error from before a request left the client.
func notSent(err error) error { return fmt.Errorf("%w: %w", ErrNotSent, err) }

// WriteProperty writes one property of an object on a device the client has
// switched to writable (Config.Writable, SetWritable). value is the tags of
// the property value, a single Null to relinquish the priority; priority is
// 1 to 16, or 0 to send none, which a device takes as 16.
//
// A write is sent once and never resent, whatever Config.Retries says: a
// device that did it and lost the acknowledgement must not do it again.
// nil means the device acknowledged it. ErrTimeout means it was sent and
// nothing answered, so the device may or may not have done it. An error
// wrapping ErrNotSent means it never left the client: ErrServiceNotAllowed
// for a device that is not writable or a client opened ReadOnly, the
// context's error, or ErrClosed. A device's Error, Reject or Abort comes
// back as a BACnetError, RejectError or AbortError.
func (c *Client) WriteProperty(ctx context.Context, to Address, obj ObjectID, prop uint16, value []*objects.Object, priority uint8) error {
	if priority > 16 {
		return notSent(fmt.Errorf("priority %d is not 1 to 16", priority))
	}
	if len(value) == 0 {
		return notSent(errors.New("no value to write"))
	}
	payload := make([]objects.APDUPayload, 0, len(value)+5)
	payload = append(payload,
		objects.EncObjectIdentifier(true, 0, obj.Type, obj.Instance),
		objects.ContextTag(1, objects.EncUnsignedInteger(uint(prop))),
		objects.EncOpeningTag(3))
	for _, t := range value {
		payload = append(payload, t)
	}
	payload = append(payload, objects.EncClosingTag(3))
	if priority > 0 {
		payload = append(payload, objects.ContextTag(4, objects.EncUnsignedInteger(uint(priority))))
	}
	_, err := c.exchange(ctx, to, services.ServiceConfirmedWriteProperty, payload, sendOnce)
	return err
}

// ---- the writable switch ---------------------------------------------------

// SetWritable switches the devices a rule covers to writable, or back to
// read-only. Every device starts read-only, and only a writable device
// takes a WriteProperty; a client opened ReadOnly writes to none, whatever
// the rules say.
func (c *Client) SetWritable(rule SourceRule, writable bool) error {
	if err := rule.validate(); err != nil {
		return err
	}
	c.mu.Lock()
	kept := make([]SourceRule, 0, len(c.writable)+1)
	for _, r := range c.writable {
		if r.String() != rule.String() {
			kept = append(kept, r)
		}
	}
	if writable {
		kept = append(kept, rule)
	}
	c.writable = kept
	c.mu.Unlock()
	if writable {
		c.log.Info("source writable", "rule", rule)
	} else {
		c.log.Info("source read-only again", "rule", rule)
	}
	return nil
}

// Writable lists the rules in force.
func (c *Client) Writable() []SourceRule {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]SourceRule(nil), c.writable...)
}

// isWritable reports whether a device has been switched to writable.
func (c *Client) isWritable(addr Address) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.writable {
		if r.matches(addr, c.known) {
			return true
		}
	}
	return false
}
