package client

import (
	"context"
	"net/netip"
	"sync"
	"time"
)

// Defaults for the limits in Config.
const (
	DefaultMaxInFlightDevice = 4
	DefaultMaxInFlightNet    = 1
	DefaultMaxRequestsPerSec = 50
)

// limitKey is what an in-flight limit applies to: one device reached
// directly, or one routed network as a whole.
type limitKey struct {
	ip  netip.AddrPort
	net uint16
}

// acquire takes a slot for a request to the device, waiting while the
// device, or the routed network it is on, has as many in flight as the
// configuration allows. It returns the function that gives the slot back.
func (c *Client) acquire(ctx context.Context, to Address) (func(), error) {
	key := limitKey{ip: to.IP}
	size := c.cfg.MaxInFlightDevice(to)
	if to.Routed() {
		key = limitKey{net: to.Net}
		size = c.cfg.MaxInFlightNet(to.Net)
	}
	size = max(1, size)

	c.mu.Lock()
	sem := c.inflight[key]
	if sem == nil {
		sem = make(chan struct{}, size)
		c.inflight[key] = sem
	}
	c.mu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, ErrClosed
	}
}

// bucket is a token bucket: it lets through a burst, then a steady rate.
type bucket struct {
	mu     sync.Mutex
	rate   float64 // tokens per second
	burst  float64
	tokens float64
	last   time.Time
}

func newBucket(perSecond float64) *bucket {
	burst := max(1, perSecond)
	return &bucket{rate: perSecond, burst: burst, tokens: burst, last: time.Now()}
}

// wait takes one token, waiting for it when none is there.
func (b *bucket) wait(ctx context.Context, closed <-chan struct{}) error {
	if b == nil {
		return nil
	}
	for {
		b.mu.Lock()
		now := time.Now()
		b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		wait := time.Duration((1 - b.tokens) / b.rate * float64(time.Second))
		b.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-closed:
			timer.Stop()
			return ErrClosed
		}
	}
}

// PeerStats counts what happened with one device.
type PeerStats struct {
	// Sent is the confirmed requests sent, retries included.
	Sent int
	// Retries is the requests sent again for want of a reply.
	Retries int
	// Timeouts is the requests that got no reply at all.
	Timeouts int
	// Errors is the requests the device answered with an Error, Reject or
	// Abort.
	Errors int
	// Denied is the requests refused because the device was on the deny
	// list.
	Denied int
	// WindowClamped is the segmented replies that proposed a window larger
	// than the client accepts.
	WindowClamped int
	// ResponseTooLarge is the segmented replies the client gave up on for
	// being longer than it reassembles.
	ResponseTooLarge int
}

// Stats is what the client counted, by device. The key is the device's
// address as Address.String gives it: a routed device by its network and
// MAC, the rest by their BACnet/IP address.
type Stats struct {
	Peers map[string]PeerStats
}

// Stats returns what the client has counted since it was opened.
func (c *Client) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := Stats{Peers: make(map[string]PeerStats, len(c.stats))}
	for key, st := range c.stats {
		s.Peers[key.address().String()] = *st
	}
	return s
}

// count updates the counters of one device.
func (c *Client) count(to Address, update func(*PeerStats)) {
	key := keyOf(to, 0)
	c.mu.Lock()
	st := c.stats[key]
	if st == nil {
		st = &PeerStats{}
		c.stats[key] = st
	}
	update(st)
	c.mu.Unlock()
}
