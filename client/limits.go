package client

import (
	"context"
	"errors"
	"fmt"
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

// MaxInFlight is the number of distinct invoke IDs available to one peer.
// Limits returned by Config's concurrency callbacks must not exceed it.
const MaxInFlight = 256

// ErrInFlightLimit is a limit from MaxInFlightDevice or MaxInFlightNet
// above MaxInFlight. The request fails with it, wrapped in ErrNotSent,
// before anything is sent.
var ErrInFlightLimit = errors.New("in-flight limit above the invoke IDs available")

// limitKey is what an in-flight limit applies to: one device reached
// directly, or one routed network as a whole.
type limitKey struct {
	ip  netip.AddrPort
	net uint16
}

// slots is the in-flight slots of one device or routed network.
type slots struct {
	sem chan struct{}
	// users is the requests that hold a slot or wait for one. The slots are
	// forgotten when the last one leaves, so that an address tried once and
	// never again leaves nothing behind.
	users int
}

// acquire takes a slot for a request to the device, waiting while the
// device, or the routed network it is on, has as many in flight as the
// configuration allows. It returns the function that gives the slot back.
func (c *Client) acquire(ctx context.Context, to Address) (func(), error) {
	if err := requestActive(ctx, c.closed); err != nil {
		return nil, err
	}
	key := limitKey{ip: to.IP}
	var size int
	if to.Routed() {
		key = limitKey{net: to.Net}
		size = c.cfg.MaxInFlightNet(to.Net)
	} else {
		size = c.cfg.MaxInFlightDevice(to)
	}
	if size > MaxInFlight {
		return nil, fmt.Errorf("%w: %d allowed, %d asked", ErrInFlightLimit, MaxInFlight, size)
	}
	size = max(1, size)

	c.mu.Lock()
	s := c.inflight[key]
	if s == nil {
		s = &slots{sem: make(chan struct{}, size)}
		c.inflight[key] = s
	}
	s.users++
	c.mu.Unlock()
	leave := func() {
		c.mu.Lock()
		if s.users--; s.users == 0 {
			delete(c.inflight, key)
		}
		c.mu.Unlock()
	}

	select {
	case s.sem <- struct{}{}:
		if err := requestActive(ctx, c.closed); err != nil {
			<-s.sem
			leave()
			return nil, err
		}
		return func() {
			<-s.sem
			leave()
		}, nil
	case <-ctx.Done():
		leave()
		return nil, ctx.Err()
	case <-c.closed:
		leave()
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
	for {
		// Checked before each try, so that a request that is cancelled or
		// closed does not go on with a token that happens to be free.
		if err := requestActive(ctx, closed); err != nil {
			return err
		}
		if b == nil {
			return nil
		}
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

func requestActive(ctx context.Context, closed <-chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-closed:
		return ErrClosed
	default:
		return nil
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
