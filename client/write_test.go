package client_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// peer is a socket that stands in for a device: it reads what the client
// sends and answers as the test says.
type peer struct {
	t    *testing.T
	conn *net.UDPConn
}

func newPeer(t *testing.T) *peer {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &peer{t: t, conn: conn}
}

func (p *peer) addr() netip.AddrPort {
	ap := p.conn.LocalAddr().(*net.UDPAddr).AddrPort()
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}

// receive reads one WriteProperty, or returns nil when none comes within
// the wait.
func (p *peer) receive(wait time.Duration) (*services.ConfirmedWriteProperty, netip.AddrPort) {
	p.t.Helper()
	buf := make([]byte, 2048)
	p.conn.SetReadDeadline(time.Now().Add(wait))
	n, from, err := p.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		return nil, netip.AddrPort{}
	}
	msg, err := bacnet.Parse(buf[:n])
	if err != nil {
		p.t.Fatalf("the client sent %x: %v", buf[:n], err)
	}
	wp, ok := msg.(*services.ConfirmedWriteProperty)
	if !ok {
		p.t.Fatalf("the client sent a %T, want a WriteProperty", msg)
	}
	return wp, netip.AddrPortFrom(from.Addr().Unmap(), from.Port())
}

// reply answers the client with one APDU.
func (p *peer) reply(to netip.AddrPort, a *plumbing.APDU) {
	p.t.Helper()
	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, false, false, false)
	b := make([]byte, bvlc.MarshalLen()+npdu.MarshalLen()+a.MarshalLen())
	bvlc.Length = uint16(len(b))
	bvlc.MarshalTo(b)
	npdu.MarshalTo(b[bvlc.MarshalLen():])
	if err := a.MarshalTo(b[bvlc.MarshalLen()+npdu.MarshalLen():]); err != nil {
		p.t.Fatal(err)
	}
	if _, err := p.conn.WriteToUDPAddrPort(b, to); err != nil {
		p.t.Fatal(err)
	}
}

func (p *peer) ack(to netip.AddrPort, wp *services.ConfirmedWriteProperty) {
	p.reply(to, &plumbing.APDU{Type: plumbing.SimpleAck, InvokeID: wp.APDU.InvokeID, Service: services.ServiceConfirmedWriteProperty})
}

// tagsOf gives the APDU's payload as tags.
func tagsOf(t *testing.T, a *plumbing.APDU) []*objects.Object {
	t.Helper()
	var tags []*objects.Object
	for _, o := range a.Objects {
		tag, ok := o.(*objects.Object)
		if !ok {
			t.Fatalf("payload %T", o)
		}
		tags = append(tags, tag)
	}
	return tags
}

// openWriter opens a client that may write to the peer: no rate limit, and
// retries that a read would use and a write must not.
func openWriter(t *testing.T, p *peer, change func(*client.Config)) *client.Client {
	t.Helper()
	cfg := client.Config{
		Local: netip.MustParseAddrPort("127.0.0.1:0"), Broadcast: p.addr(),
		Timeout: 300 * time.Millisecond, Retries: 2, MaxRequestsPerSec: -1,
		Writable: []client.SourceRule{{IP: netip.PrefixFrom(p.addr().Addr(), 32)}},
	}
	if change != nil {
		change(&cfg)
	}
	c, err := client.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

var (
	setpoint = client.ObjectID{Type: objects.ObjectTypeAnalogValue, Instance: 3}
	oneReal  = []*objects.Object{objects.EncReal(1)}
)

// A write goes out once with the object, property, value and priority, and
// comes back as the device's answer: a SimpleACK, an Error, or silence
// after a single send.
func TestWriteProperty(t *testing.T) {
	p := newPeer(t)
	c := openWriter(t, p, nil)
	ctx := context.Background()
	to := client.Address{IP: p.addr()}
	done := make(chan error, 1)

	// Acknowledged, with the priority given.
	go func() {
		done <- c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, []*objects.Object{objects.EncReal(21.5)}, 13)
	}()
	wp, from := p.receive(time.Second)
	if wp == nil {
		t.Fatal("the client sent nothing")
	}
	tags := tagsOf(t, wp.APDU)
	if len(tags) != 6 {
		t.Fatalf("the write carries %d tags, want 6", len(tags))
	}
	obj, _ := objects.DecObjectIdentifier(tags[0])
	prop, _ := objects.DecUnsignedInteger(tags[1])
	value, _ := objects.DecReal(tags[3])
	priority, _ := objects.DecUnsignedInteger(tags[5])
	if !tags[0].TagClass || tags[0].TagNumber != 0 || obj.ObjectType != objects.ObjectTypeAnalogValue || obj.InstanceNumber != 3 ||
		!tags[1].TagClass || tags[1].TagNumber != 1 || prop != uint32(objects.PropertyIdPresentValue) ||
		tags[2].Kind != objects.TagOpening || tags[2].TagNumber != 3 || tags[3].TagClass || value != 21.5 ||
		tags[4].Kind != objects.TagClosing || tags[4].TagNumber != 3 ||
		!tags[5].TagClass || tags[5].TagNumber != 4 || priority != 13 {
		t.Errorf("the write is %+v", tags)
	}
	if wp.NPDU.Control&0x04 == 0 {
		t.Error("the write does not expect a reply")
	}
	p.ack(from, wp)
	if err := <-done; err != nil {
		t.Errorf("an acknowledged write gave %v", err)
	}

	// A relinquish with no priority: a Null, and no priority tag.
	go func() {
		done <- c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, []*objects.Object{objects.EncNull()}, 0)
	}()
	if wp, from = p.receive(time.Second); wp == nil {
		t.Fatal("the client sent nothing")
	}
	tags = tagsOf(t, wp.APDU)
	if len(tags) != 5 || tags[3].TagClass || tags[3].TagNumber != objects.EncNull().TagNumber {
		t.Errorf("the relinquish is %+v, want a Null and no priority", tags)
	}
	p.ack(from, wp)
	if err := <-done; err != nil {
		t.Errorf("an acknowledged relinquish gave %v", err)
	}

	// The device says no.
	go func() {
		done <- c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, oneReal, 16)
	}()
	if wp, from = p.receive(time.Second); wp == nil {
		t.Fatal("the client sent nothing")
	}
	e := plumbing.NewAPDU(plumbing.Error, services.ServiceConfirmedWriteProperty, services.ErrorObjects(objects.ErrorClassProperty, 40))
	e.InvokeID = wp.APDU.InvokeID
	p.reply(from, e)
	err := <-done
	var be *client.BACnetError
	if !errors.As(err, &be) || be.Class != uint32(objects.ErrorClassProperty) || be.Code != 40 || errors.Is(err, client.ErrNotSent) {
		t.Errorf("a refused write gave %v, want property: write access denied, sent", err)
	}

	// Silence: a timeout after exactly one send, where a read would have
	// been sent twice more.
	go func() {
		done <- c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, oneReal, 16)
	}()
	if wp, _ = p.receive(time.Second); wp == nil {
		t.Fatal("the client sent nothing")
	}
	if again, _ := p.receive(time.Second); again != nil {
		t.Error("the write was sent again")
	}
	if err := <-done; !errors.Is(err, client.ErrTimeout) || errors.Is(err, client.ErrNotSent) {
		t.Errorf("an unanswered write gave %v, want ErrTimeout, sent", err)
	}
	stats := c.Stats()
	if len(stats.Peers) != 1 {
		t.Fatalf("the stats cover %d peers, want 1", len(stats.Peers))
	}
	for _, s := range stats.Peers {
		if s.Sent != 4 || s.Retries != 0 || s.Timeouts != 1 || s.Errors != 1 {
			t.Errorf("the stats are %+v, want 4 sent, no retries, one timeout, one error", s)
		}
	}
}

// A write never leaves the client for a device that is not writable, for a
// client opened ReadOnly, or with nothing to send, and the error says so.
func TestWritePropertyNotSent(t *testing.T) {
	p := newPeer(t)
	ctx := context.Background()
	to := client.Address{IP: p.addr()}
	rule := client.SourceRule{IP: netip.PrefixFrom(p.addr().Addr(), 32)}

	for _, tc := range []struct {
		name   string
		change func(*client.Config)
		before func(*client.Client)
		want   error
	}{
		{"not writable", func(cfg *client.Config) { cfg.Writable = nil }, nil, client.ErrServiceNotAllowed},
		{"read-only", func(cfg *client.Config) { cfg.ReadOnly = true }, nil, client.ErrServiceNotAllowed},
		{"switched back", nil, func(c *client.Client) { c.SetWritable(rule, false) }, client.ErrServiceNotAllowed},
	} {
		c := openWriter(t, p, tc.change)
		if tc.before != nil {
			tc.before(c)
		}
		err := c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, oneReal, 16)
		if !errors.Is(err, client.ErrNotSent) || !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want ErrNotSent wrapping %v", tc.name, err, tc.want)
		}
		if wp, _ := p.receive(100 * time.Millisecond); wp != nil {
			t.Errorf("%s: the write was sent", tc.name)
		}
	}

	c := openWriter(t, p, nil)
	if err := c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, oneReal, 17); !errors.Is(err, client.ErrNotSent) {
		t.Errorf("priority 17: %v, want ErrNotSent", err)
	}
	if err := c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, nil, 16); !errors.Is(err, client.ErrNotSent) {
		t.Errorf("no value: %v, want ErrNotSent", err)
	}
	if wp, _ := p.receive(100 * time.Millisecond); wp != nil {
		t.Error("a write with nothing to send was sent")
	}

	// Switched back and forth, the device takes writes again, and Writable
	// lists the rule.
	if err := c.SetWritable(rule, false); err != nil {
		t.Fatal(err)
	}
	if err := c.SetWritable(rule, true); err != nil {
		t.Fatal(err)
	}
	if rules := c.Writable(); len(rules) != 1 || rules[0].String() != rule.String() {
		t.Errorf("Writable lists %v, want %v", rules, rule)
	}
	done := make(chan error, 1)
	go func() {
		done <- c.WriteProperty(ctx, to, setpoint, objects.PropertyIdPresentValue, oneReal, 16)
	}()
	wp, from := p.receive(time.Second)
	if wp == nil {
		t.Fatal("the client sent nothing after the device was switched back to writable")
	}
	p.ack(from, wp)
	if err := <-done; err != nil {
		t.Errorf("the write after the switch gave %v", err)
	}
}
