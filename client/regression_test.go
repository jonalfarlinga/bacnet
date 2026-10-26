package client

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

func TestExcessiveConcurrencyNotSent(t *testing.T) {
	for _, routed := range []bool{false, true} {
		t.Run(map[bool]string{false: "device", true: "network"}[routed], func(t *testing.T) {
			sent := 0
			c, network := openSim(t, func(cfg *Config) {
				cfg.MaxInFlightDevice = func(Address) int { return MaxInFlight + 1 }
				cfg.MaxInFlightNet = func(uint16) int { return MaxInFlight + 1 }
				cfg.OnPacket = func(p PacketEvent) {
					if !p.Inbound {
						sent++
					}
				}
			})
			to := Address{IP: network.Addr()}
			if routed {
				to = Address{Net: 2001, MAC: []byte{5}} // resolving would send discovery
			}
			_, err := c.ReadProperty(context.Background(), to, ObjectID{Type: 8, Instance: 1001}, 77)
			if !errors.Is(err, ErrNotSent) || !errors.Is(err, ErrInFlightLimit) || sent != 0 {
				t.Fatalf("got %v and %d sends for excessive concurrency", err, sent)
			}
		})
	}
}

func TestInvokeIDCapacity(t *testing.T) {
	c, _ := openSim(t, func(cfg *Config) {
		cfg.MaxInFlightDevice = func(Address) int { return MaxInFlight }
	})
	to := Address{IP: netip.MustParseAddrPort("127.0.0.1:47808")}
	defer func() { clear(c.pending) }()
	seen := map[txKey]chan inbound{}
	var release []func()
	for range MaxInFlight {
		free, err := c.acquire(context.Background(), to)
		if err != nil {
			t.Fatal(err)
		}
		release = append(release, free)
		key, ch := c.begin(to, &plumbing.APDU{})
		if ch == nil || seen[key] != nil {
			t.Fatal("invoke ID missing or reused before reaching capacity")
		}
		seen[key] = ch
	}
	defer func() {
		for _, free := range release {
			free()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := c.acquire(ctx, to); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("257th request was not held: %v", err)
	}
	if _, ch := c.begin(to, &plumbing.APDU{}); ch != nil {
		t.Fatal("exhaustion should not replace a pending request")
	}
	for key, ch := range seen {
		if c.pending[key] != ch {
			t.Fatal("a pending request was overwritten")
		}
	}
}

// The invoke ID of a request that got no reply is kept out of use for a
// timeout, so that a reply arriving late is not taken for the answer to the
// next request; the ID of a request that was answered is reused as before.
func TestTimedOutInvokeIDQuarantined(t *testing.T) {
	const timeout = 50 * time.Millisecond
	c, network := openSim(t, func(cfg *Config) { cfg.Timeout = timeout })
	to := Address{IP: network.Addr()}
	obj := ObjectID{Type: objects.ObjectTypeDevice, Instance: 1001}
	peer := keyOf(to, 0)
	nextWouldBe := func(id uint8) {
		c.mu.Lock()
		c.invokes[peer] = id - 1
		c.mu.Unlock()
	}
	beginOne := func() uint8 {
		req := plumbing.NewAPDU(plumbing.ConfirmedReq, services.ServiceConfirmedReadProperty, nil)
		key, ch := c.begin(to, req)
		if ch == nil {
			t.Fatal("no invoke ID available")
		}
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return req.InvokeID
	}

	network.Silence(0, true)
	if _, err := c.ReadProperty(context.Background(), to, obj, objects.PropertyIdObjectName); !errors.Is(err, ErrTimeout) {
		t.Fatalf("a read of a silent device gave %v, want a timeout", err)
	}
	network.Silence(0, false)
	c.mu.Lock()
	timedOut := c.invokes[peer]
	c.mu.Unlock()

	nextWouldBe(timedOut)
	if id := beginOne(); id == timedOut {
		t.Fatalf("the next request got invoke ID %d, the timed-out one, want it skipped", id)
	}
	time.Sleep(2 * timeout)
	nextWouldBe(timedOut)
	if id := beginOne(); id != timedOut {
		t.Errorf("after the timeout the next request got invoke ID %d, want %d back in use", id, timedOut)
	}

	if _, err := c.ReadProperty(context.Background(), to, obj, objects.PropertyIdObjectName); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	answered := c.invokes[peer]
	_, held := c.quarantine[keyOf(to, answered)]
	c.mu.Unlock()
	if held {
		t.Errorf("invoke ID %d of an answered request was kept out of use", answered)
	}
}

func TestReplyServiceMatching(t *testing.T) {
	for _, kind := range []uint8{plumbing.SimpleAck, plumbing.ComplexAck, plumbing.Error} {
		c := &Client{cfg: Config{Timeout: time.Second}, closed: make(chan struct{})}
		ch := make(chan inbound, 2)
		ch <- inbound{apdu: &plumbing.APDU{Type: kind, Service: services.ServiceConfirmedSubscribeCOV}}
		ch <- inbound{apdu: &plumbing.APDU{Type: plumbing.SimpleAck, Service: services.ServiceConfirmedWriteProperty}}
		if _, err := c.await(context.Background(), ch, Address{}, 1, services.ServiceConfirmedWriteProperty); err != nil || len(ch) != 0 {
			t.Fatalf("reply type %d: wrong-service reply completed transaction: %v", kind, err)
		}
	}
}

func TestReadManyFitsRequests(t *testing.T) {
	const budget = 50
	maxSize, requests := 0, 0
	c, network := openSim(t, func(cfg *Config) {
		cfg.MaxRequestsPerSec = -1
		cfg.OnPacket = func(p PacketEvent) {
			if p.Inbound {
				return
			}
			msg, err := bacnet.Parse(p.Data)
			if err != nil {
				t.Error(err)
				return
			}
			if rp, ok := msg.(*services.ConfirmedReadProperty); ok {
				requests++
				maxSize = max(maxSize, rp.APDU.MarshalLen())
			}
		}
	})
	dev := Device{Instance: 1001, Addr: Address{IP: network.Addr()}, MaxAPDU: budget, Segmentation: 1}
	var specs []ReadSpec
	for i := range 32 {
		props := []uint16{77, 85, 111, 79, 81}
		if i%2 == 0 {
			props = []uint16{77}
		}
		specs = append(specs, ReadSpec{Object: ObjectID{Type: 2, Instance: uint32(i)}, Props: props})
	}
	// One object's request exceeds the budget by itself, so it needs RP fallback.
	props := make([]uint16, 24)
	for i := range props {
		props[i] = uint16(200 + i)
	}
	specs = append(specs, ReadSpec{Object: ObjectID{Type: 8, Instance: 1001}, Props: props})
	results, err := c.ReadMany(context.Background(), dev, specs)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(specs) || maxSize > budget || requests <= 1 {
		t.Fatalf("%d results, %d requests, maximum APDU %d (budget %d)", len(results), requests, maxSize, budget)
	}
	if len(results[len(results)-1].Props) != len(props) {
		t.Fatal("oversized property list was not read completely")
	}
}

func TestReadManyMatchesRepeatedAndWildcardObjects(t *testing.T) {
	actual := ObjectID{Type: 8, Instance: 101}
	wild := ObjectID{Type: 8, Instance: WildcardInstance}
	specs := []ReadSpec{{Object: wild, Props: []uint16{77}}, {Object: actual, Props: []uint16{70}}, {Object: actual, Props: []uint16{77, 28}}}
	got := []ObjectResult{
		{Object: actual, Props: []PropertyResult{{ID: 77, Values: []Value{{Kind: KindString, Str: "first"}}}}},
		{Object: actual, Props: []PropertyResult{{ID: 70, Values: []Value{{Kind: KindString, Str: "model"}}}}},
		{Object: actual, Props: []PropertyResult{{ID: 77, Values: []Value{{Kind: KindString, Str: "second"}}}}},
	}
	out := make([]ObjectResult, len(specs))
	matchReadMany(101, specs, got, out)
	if out[0].Props[0].Values[0].Str != "first" || out[1].Props[0].ID != 70 || out[2].Props[0].Values[0].Str != "second" || !errors.Is(out[2].Props[1].Err, errNotInReply) {
		t.Fatalf("mismatched results: %+v", out)
	}
}

// With the device's instance not known, a wildcard Device spec takes the
// answer the device gives under its own instance.
func TestReadManyWildcardWithUnknownInstance(t *testing.T) {
	wild := ObjectID{Type: objects.ObjectTypeDevice, Instance: WildcardInstance}
	specs := []ReadSpec{{Object: wild, Props: []uint16{77}}}
	got := []ObjectResult{{
		Object: ObjectID{Type: objects.ObjectTypeDevice, Instance: 1001},
		Props:  []PropertyResult{{ID: 77, Values: []Value{{Kind: KindString, Str: "name"}}}},
	}}
	out := make([]ObjectResult, len(specs))
	matchReadMany(0, specs, got, out)
	if len(out[0].Props) != 1 || out[0].Props[0].Err != nil || out[0].Props[0].Values[0].Str != "name" {
		t.Fatalf("the wildcard spec got %+v, want the device's answer under its own instance", out[0].Props)
	}
}

// rpmSpecSize says what the request builder puts on the wire, for property
// identifiers of one byte and of two.
func TestRPMSpecSizeIsWireSize(t *testing.T) {
	for _, props := range [][]uint16{nil, {77}, {77, 85, 255, 256, 4095}} {
		spec := ReadSpec{Object: ObjectID{Type: 8, Instance: 4194302}, Props: props}
		want := 0
		for _, tag := range services.ConfirmedReadPropertyMultipleObjects(spec.Object.Type, spec.Object.Instance, spec.Props) {
			want += tag.MarshalLen()
		}
		if got := rpmSpecSize(spec); got != want {
			t.Errorf("properties %v: size %d, want the %d bytes on the wire", props, got, want)
		}
	}
}

// An object the device left out of the reply is marked not in the reply for
// every property asked, ALL included; one it answered for ALL gets the
// properties the device chose.
func TestReadManyAllOfMissingObject(t *testing.T) {
	present, missing := ObjectID{Type: 0, Instance: 1}, ObjectID{Type: 0, Instance: 2}
	specs := []ReadSpec{
		{Object: present, Props: []uint16{objects.PropertyIdAll}},
		{Object: missing, Props: []uint16{objects.PropertyIdAll}},
	}
	got := []ObjectResult{{Object: present, Props: []PropertyResult{{ID: 77}, {ID: 85}}}}
	out := make([]ObjectResult, len(specs))
	matchReadMany(1001, specs, got, out)
	if len(out[0].Props) != 2 || out[0].Props[1].ID != 85 {
		t.Errorf("the answered object has %+v, want the two properties the device chose", out[0].Props)
	}
	if len(out[1].Props) != 1 || out[1].Props[0].ID != uint32(objects.PropertyIdAll) || !errors.Is(out[1].Props[0].Err, errNotInReply) {
		t.Errorf("the missing object has %+v, want ALL marked not in the reply", out[1].Props)
	}
}

func TestReassemblerSequenceWrap(t *testing.T) {
	r := NewReassembler(time.Minute)
	bvlc, npdu := plumbing.NewBVLC(10), plumbing.NewNPDU(false, false, false, false)
	var result FeedResult
	for i := range 600 {
		result = r.Feed("peer", bvlc, npdu, segment(uint8(i), i < 599, []byte{0}))
		if result.Dropped != "" {
			t.Fatal(result.Dropped)
		}
		if i == 255 || i == 256 || i == 260 {
			dup := r.Feed("peer", bvlc, npdu, segment(uint8(i), true, []byte{0}))
			if !dup.Accepted || dup.Complete {
				t.Fatalf("duplicate %d: %+v", i, dup)
			}
		}
	}
	if !result.Complete || len(result.Raw) != 609 {
		t.Fatalf("600 segments produced %+v", result)
	}
	msg, err := bacnet.Parse(result.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.(*services.ComplexACK).APDU.Objects) != 600 {
		t.Fatal("segments were lost or duplicated")
	}
}

func TestProbeLearnsWritableInstance(t *testing.T) {
	instance := uint32(1001)
	c, n := openSim(t, func(cfg *Config) { cfg.Writable = []SourceRule{{Instance: &instance}} })
	dev, err := c.Probe(context.Background(), Address{IP: n.Addr()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !c.isWritable(dev.Addr) {
		t.Fatal("probed instance did not activate its writable rule")
	}
}

func TestCanceledBucketWithAvailableToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, b := range []*bucket{nil, newBucket(50)} {
		if err := b.wait(ctx, make(chan struct{})); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled wait: %v", err)
		}
	}
}
