// Package client is a BACnet/IP client: one UDP socket, a read loop that
// matches replies to requests, the read services, and change-of-value
// subscriptions. It is the client of docs/library-upgrade-spec.md §7. The
// explorer, github.com/jonalfarlinga/bacnet-explorer, is its first user.
//
// It reaches devices on the local subnet, behind BACnet routers (it learns
// which router serves each network), and on remote IP subnets through a BBMD
// that it registers with as a foreign device.
//
// It writes only where it is told to. Everything it sends goes through one
// function that refuses any service except Who-Is, ReadProperty,
// ReadPropertyMultiple, ReadRange, the Segment-ACKs and Aborts a segmented
// reply needs, Who-Is-Router-To-Network, the BBMD functions
// Register-Foreign-Device, Read-Foreign-Device-Table and
// Read-Broadcast-Distribution-Table, for subscriptions SubscribeCOV and the
// SimpleACK a confirmed COV notification waits for, and WriteProperty to a
// device switched to writable (Config.Writable, SetWritable); every device
// starts read-only. With Config.ReadOnly it refuses the subscriptions and
// the writes as well, and is then read-only: it leaves nothing behind on a
// device. It has no device identity: it never sends an I-Am, and answers
// nothing but the notifications of its own subscriptions.
//
// It talks to every device it can reach, except those on its deny list
// (Config.Denied, SetDenied). It limits what it has in flight to each device
// and to each routed network, and how many requests it sends in a second.
package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// Address is where a device is reached.
type Address struct {
	// IP is the device's BACnet/IP address, or that of the router in front
	// of it. For a routed device it may be left unset: the client then uses
	// the router it has learned for Net, or finds one.
	IP netip.AddrPort
	// Net is the device's network number, 0 for a device at IP itself.
	Net uint16
	// MAC is the device's address on Net.
	MAC []byte
}

// Routed reports whether the device is behind a router.
func (a Address) Routed() bool { return a.Net != 0 }

func (a Address) String() string {
	if !a.Routed() {
		return a.IP.String()
	}
	if !a.IP.IsValid() {
		return fmt.Sprintf("network %d MAC %x", a.Net, a.MAC)
	}
	return fmt.Sprintf("network %d MAC %x via %s", a.Net, a.MAC, a.IP)
}

// Device is what an I-Am says about a device.
type Device struct {
	Instance     uint32
	Addr         Address
	MaxAPDU      uint16
	Segmentation uint8 // BACnetSegmentation: 0 both, 1 transmit, 2 receive, 3 none
	VendorID     uint16
}

// ObjectID identifies an object in a device.
type ObjectID struct {
	Type     uint16
	Instance uint32
}

// Config says where the client listens and where its broadcasts go.
type Config struct {
	// Local is this machine's address on the BACnet network. Datagrams from
	// it are the client's own broadcasts coming back, and are dropped.
	Local netip.AddrPort
	// BindAny binds the wildcard address on Local's port instead of Local.
	// A socket bound to one address does not receive broadcasts on Linux or
	// macOS; on Windows it does, and binding it keeps the other adapters out.
	BindAny bool
	// Broadcast is where Who-Is goes.
	Broadcast netip.AddrPort
	// Exclude drops datagrams from these subnets: those of the machine's
	// other adapters, which a socket bound with BindAny hears as well.
	// Anything else is accepted, since a device on another IP subnet may
	// answer directly, or through a BBMD.
	Exclude []netip.Prefix

	Timeout time.Duration // wait for each reply; 3s if not set
	Retries int           // resends of a request that got no reply

	// Denied is the sources the client leaves alone: it sends them nothing
	// but broadcasts and drops what they send. SetDenied changes the list
	// while the client runs.
	Denied []SourceRule
	// Writable is the devices the client may write to with WriteProperty;
	// every other device is read-only, as is a broadcast. SetWritable
	// changes the list while the client runs. ReadOnly overrides it.
	Writable []SourceRule

	// MaxInFlightDevice says how many requests may be in flight at once to
	// a device reached directly, by its BACnet/IP address. nil allows
	// DefaultMaxInFlightDevice.
	MaxInFlightDevice func(Address) int
	// MaxInFlightNet says how many requests may be in flight at once to all
	// the devices of one routed network together: a serial trunk behind a
	// router carries one exchange at a time. nil allows DefaultMaxInFlightNet.
	MaxInFlightNet func(net uint16) int
	// MaxRequestsPerSec limits the requests the client sends, to all devices
	// together: a token bucket that allows a burst of a second's worth. 0 is
	// DefaultMaxRequestsPerSec; below 0 is no limit.
	MaxRequestsPerSec float64

	// ReadOnly makes the client refuse what leaves state on a device or
	// answers one: SubscribeCOV and WriteProperty fail with
	// ErrServiceNotAllowed, whatever Writable says, and no notification is
	// acknowledged.
	ReadOnly bool

	// MaxSegments is the most segments the client reassembles into one
	// reply; a longer reply is aborted and fails with ErrResponseTooLarge.
	// 0 is DefaultMaxSegments.
	MaxSegments int

	// OnPacket is called from the read loop for every datagram received,
	// and from the sender for every one sent, before anything is done with
	// it. It must not block. Data is the datagram, which the receiver may
	// keep.
	OnPacket func(PacketEvent)

	// ForeignDevice registers the client with a BBMD on another IP subnet.
	// Broadcasts then go to the BBMD as Distribute-Broadcast-To-Network,
	// which it broadcasts on its subnet, and the broadcasts of that subnet
	// come back forwarded by it.
	ForeignDevice *ForeignDeviceConfig

	// Logger gets what the client refuses to send, the routes it learns and
	// drops, and the foreign-device registration. nil discards it all.
	Logger *slog.Logger

	// OnIAm is called from the read loop for every I-Am heard. It must not
	// block.
	OnIAm func(Device)
}

// PacketEvent is one datagram sent or received.
type PacketEvent struct {
	Time time.Time
	// Inbound is true for a datagram received, from Peer; false for one
	// sent, to Peer.
	Inbound bool
	Peer    netip.AddrPort
	Data    []byte
}

var (
	ErrTimeout = errors.New("no reply from the device")
	ErrClosed  = errors.New("client closed")
	// ErrServiceNotAllowed is a message the client does not send: one that
	// is not on its list, or with Config.ReadOnly, one that is not a read.
	ErrServiceNotAllowed = errors.New("not a service the client sends")
	// ErrNetworkUnreachable is a router's Reject-Message-To-Network: it
	// does not reach the network the request was for.
	ErrNetworkUnreachable = errors.New("no router reaches the network")
	// ErrSourceDenied is a request to a device on the deny list.
	ErrSourceDenied = errors.New("source denied")
	// ErrResponseTooLarge is a segmented reply longer than the client
	// reassembles. The error carries the sizes, as a *ResponseTooLargeError.
	ErrResponseTooLarge = errors.New("the reply is longer than the client reassembles")
)

// ResponseTooLargeError says which reply was too long, and by how much.
type ResponseTooLargeError struct {
	Peer        Address
	InvokeID    uint8
	Service     uint8
	Segments    int // received before the client gave up
	Bytes       int // received before the client gave up
	MaxSegments int // what the client reassembles at most
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("%v: %d bytes in %d segments from %v, more than the %d segments the client reassembles",
		ErrResponseTooLarge, e.Bytes, e.Segments, e.Peer, e.MaxSegments)
}

func (e *ResponseTooLargeError) Unwrap() error { return ErrResponseTooLarge }

// DefaultMaxSegments is the most segments a reply may have: at up to 1476
// bytes each, about 94 kB.
const DefaultMaxSegments = 64

// The client takes windows of up to 64 segments, the most a device may
// propose.
const maxWindow = 64

// Client owns the socket.
type Client struct {
	cfg  Config
	conn *net.UDPConn
	log  *slog.Logger

	mu      sync.Mutex
	pending map[txKey]chan inbound
	// invokes is the last invoke ID given to a request to each device.
	invokes map[txKey]uint8
	// inflight is the slots for requests to each device reached directly
	// and to each routed network.
	inflight map[limitKey]*slots
	// limiter paces the requests to all devices together.
	limiter *bucket
	stats   map[txKey]*PeerStats
	// noRPM marks the devices that have turned ReadPropertyMultiple down.
	noRPM map[txKey]bool
	// denied is the deny list, and known the address each device instance
	// was last heard from, which rules by instance are applied to.
	denied []SourceRule
	known  map[uint32]Address
	// writable is the devices switched out of read-only (write.go).
	writable []SourceRule
	// routes is the router for each network number, learned from
	// I-Am-Router-To-Network and from the source of routed messages.
	routes map[uint16]netip.AddrPort
	// busy holds, for each network a router has said it is busy for, when
	// the hold ends on its own.
	busy map[uint16]time.Time
	// available is closed, and replaced, when a router says it is available
	// again, to wake what waits on busy.
	available chan struct{}
	// routerWatch is the Who-Is-Router-To-Network queries waiting for
	// I-Am-Router-To-Network.
	routerWatch map[*routerWatcher]struct{}
	// bvlcWait is the BBMD requests waiting for a BVLC-only reply.
	bvlcWait map[bvlcKey]chan plumbing.BACnet
	// subs is the COV subscriptions whose notifications are delivered.
	subs map[covKey]*Subscription

	fd *foreign // nil unless ForeignDevice is configured

	closed chan struct{}
	done   chan struct{}
}

// txKey identifies a confirmed request by its peer and invoke ID. A routed
// device is known by its network and MAC rather than the router's IP, so
// that its reply matches whichever router carries it.
type txKey struct {
	ip     netip.AddrPort
	net    uint16
	mac    string
	invoke uint8
}

func keyOf(a Address, invoke uint8) txKey {
	if a.Routed() {
		return txKey{net: a.Net, mac: string(a.MAC), invoke: invoke}
	}
	return txKey{ip: a.IP, invoke: invoke}
}

// address is the device the key names.
func (k txKey) address() Address {
	if k.net != 0 {
		return Address{Net: k.net, MAC: []byte(k.mac)}
	}
	return Address{IP: k.ip}
}

// inbound is a reply PDU on its way to the request that waits for it, or the
// reason there will be none.
type inbound struct {
	apdu *plumbing.APDU
	err  error
}

// Open binds the socket and starts listening.
func Open(cfg Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}
	if cfg.MaxInFlightDevice == nil {
		cfg.MaxInFlightDevice = func(Address) int { return DefaultMaxInFlightDevice }
	}
	if cfg.MaxInFlightNet == nil {
		cfg.MaxInFlightNet = func(uint16) int { return DefaultMaxInFlightNet }
	}
	if cfg.MaxRequestsPerSec == 0 {
		cfg.MaxRequestsPerSec = DefaultMaxRequestsPerSec
	}
	if cfg.MaxSegments <= 0 {
		cfg.MaxSegments = DefaultMaxSegments
	}
	for _, r := range cfg.Denied {
		if err := r.validate(); err != nil {
			return nil, err
		}
	}
	for _, r := range cfg.Writable {
		if err := r.validate(); err != nil {
			return nil, err
		}
	}
	if cfg.ForeignDevice != nil {
		if !cfg.ForeignDevice.BBMD.IsValid() || cfg.ForeignDevice.BBMD.Port() == 0 {
			return nil, fmt.Errorf("foreign device: BBMD address %s is not an address and port", cfg.ForeignDevice.BBMD)
		}
		if cfg.ForeignDevice.TTL <= 0 {
			cfg.ForeignDevice.TTL = DefaultTTL
		}
	}
	bind := cfg.Local
	if cfg.BindAny {
		bind = netip.AddrPortFrom(netip.IPv4Unspecified(), cfg.Local.Port())
	}
	conn, err := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(bind))
	if err != nil {
		return nil, err
	}
	if !cfg.Local.IsValid() || cfg.Local.Port() == 0 {
		// An ephemeral port: learn which one.
		if la, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			cfg.Local = netip.AddrPortFrom(la.AddrPort().Addr().Unmap(), la.AddrPort().Port())
		}
	}
	c := &Client{
		cfg:         cfg,
		conn:        conn,
		log:         cfg.Logger,
		pending:     map[txKey]chan inbound{},
		invokes:     map[txKey]uint8{},
		inflight:    map[limitKey]*slots{},
		stats:       map[txKey]*PeerStats{},
		noRPM:       map[txKey]bool{},
		denied:      append([]SourceRule(nil), cfg.Denied...),
		known:       map[uint32]Address{},
		writable:    append([]SourceRule(nil), cfg.Writable...),
		routes:      map[uint16]netip.AddrPort{},
		busy:        map[uint16]time.Time{},
		available:   make(chan struct{}),
		routerWatch: map[*routerWatcher]struct{}{},
		bvlcWait:    map[bvlcKey]chan plumbing.BACnet{},
		subs:        map[covKey]*Subscription{},
		closed:      make(chan struct{}),
		done:        make(chan struct{}),
	}
	if cfg.MaxRequestsPerSec > 0 {
		c.limiter = newBucket(cfg.MaxRequestsPerSec)
	}
	if cfg.ForeignDevice != nil {
		c.fd = newForeign(*cfg.ForeignDevice)
	}
	go c.readLoop()
	if c.fd != nil {
		go c.registerLoop()
	}
	return c, nil
}

// Local returns the address the client sends from.
func (c *Client) Local() netip.AddrPort { return c.cfg.Local }

// Close stops the client. Requests in flight fail with ErrClosed.
func (c *Client) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
	}
	close(c.closed)
	err := c.conn.Close()
	<-c.done
	return err
}

// ---- sending ---------------------------------------------------------------

// frame is an NPDU with an APDU in a BVLC envelope: the unit in which
// application services are sent.
type frame struct {
	bvlc *plumbing.BVLC
	npdu *plumbing.NPDU
	apdu *plumbing.APDU
}

func (f *frame) MarshalLen() int {
	return f.bvlc.MarshalLen() + f.npdu.MarshalLen() + f.apdu.MarshalLen()
}

func (f *frame) MarshalTo(b []byte) error {
	total := f.MarshalLen()
	if total > 0xFFFF {
		return fmt.Errorf("frame of %d bytes is too long", total)
	}
	f.bvlc.Length = uint16(total)
	if err := f.bvlc.MarshalTo(b); err != nil {
		return err
	}
	offset := f.bvlc.MarshalLen()
	if err := f.npdu.MarshalTo(b[offset:]); err != nil {
		return err
	}
	offset += f.npdu.MarshalLen()
	return f.apdu.MarshalTo(b[offset:])
}

func (f *frame) MarshalBinary() ([]byte, error) {
	b := make([]byte, f.MarshalLen())
	if err := f.MarshalTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (f *frame) UnmarshalBinary([]byte) error { return errors.New("a frame is only sent") }
func (f *frame) GetType() uint8               { return f.apdu.Type }
func (f *frame) GetService() uint8            { return f.apdu.Service }

// allowed reports whether a message is one this client may send: a read
// service or a Segment-ACK in a unicast, broadcast or distribute envelope, a
// Who-Is-Router-To-Network, or one of the BBMD functions that change nothing
// but the client's own registration; unless readOnly, what a COV
// subscription takes; and when the destination is writable, a
// WriteProperty.
func allowed(msg plumbing.BACnet, readOnly, writable bool) error {
	switch m := msg.(type) {
	case *frame:
		if err := envelopeAllowed(m.bvlc.Function); err != nil {
			return err
		}
		if !apduAllowed(m.apdu) && (readOnly || !covAllowed(m.apdu)) && (!writable || !writeAllowed(m.apdu)) {
			return fmt.Errorf("PDU type %d service %d: %w", m.apdu.Type, m.apdu.Service, ErrServiceNotAllowed)
		}
		return nil
	case *plumbing.NetworkMessage:
		if err := envelopeAllowed(m.BVLC.Function); err != nil {
			return err
		}
		if m.Type != plumbing.NetMsgWhoIsRouterToNetwork {
			return fmt.Errorf("network layer message %#x: %w", m.Type, ErrServiceNotAllowed)
		}
		return nil
	case *plumbing.RegisterForeignDevice, *plumbing.ReadForeignDeviceTable, *plumbing.ReadBroadcastDistributionTable:
		return nil
	}
	return fmt.Errorf("%T: %w", msg, ErrServiceNotAllowed)
}

func envelopeAllowed(function uint8) error {
	switch function {
	case plumbing.BVLCFuncUnicast, plumbing.BVLCFuncBroadcast, plumbing.BVLCFuncDistributeBroadcastToNetwork:
		return nil
	}
	return fmt.Errorf("BVLC function %#x: %w", function, ErrServiceNotAllowed)
}

func apduAllowed(a *plumbing.APDU) bool {
	switch a.Type {
	case plumbing.ConfirmedReq:
		return a.Service == services.ServiceConfirmedReadProperty ||
			a.Service == services.ServiceConfirmedReadPropMultiple ||
			a.Service == services.ServiceConfirmedReadRange
	case plumbing.UnConfirmedReq:
		return a.Service == services.ServiceUnconfirmedWhoIs && whoIsRangeOnly(a.Objects)
	case plumbing.SegmentAck:
		return true
	case plumbing.Abort:
		// The client's own abort of a reply it cannot take, never a
		// server's.
		return a.Flags&0x1 == 0
	}
	return false
}

// covAllowed reports whether an APDU is one a subscription takes: the
// SubscribeCOV that makes, renews or cancels it, or the SimpleACK of a
// confirmed notification.
func covAllowed(a *plumbing.APDU) bool {
	switch a.Type {
	case plumbing.ConfirmedReq:
		return a.Service == services.ServiceConfirmedSubscribeCOV
	case plumbing.SimpleAck:
		return a.Service == services.ServiceConfirmedCOVNotification
	}
	return false
}

// writeAllowed reports whether an APDU is a WriteProperty, which goes to a
// device switched to writable and nowhere else.
func writeAllowed(a *plumbing.APDU) bool {
	return a.Type == plumbing.ConfirmedReq && a.Service == services.ServiceConfirmedWriteProperty
}

// whoIsRangeOnly reports whether a Who-Is carries nothing, or just the range
// of instances asked for: context tags 0 and 1.
func whoIsRangeOnly(payload []objects.APDUPayload) bool {
	if len(payload) == 0 {
		return true
	}
	if len(payload) != 2 {
		return false
	}
	for i, p := range payload {
		t, ok := p.(*objects.Object)
		if !ok || !t.TagClass || t.Kind != objects.TagPrimitive || int(t.TagNumber) != i {
			return false
		}
	}
	return true
}

// destination says which device a message is for, and whether it is a
// unicast: a broadcast cannot leave one device out, so the deny list is
// applied to unicasts only.
func destination(to netip.AddrPort, msg plumbing.BACnet) (Address, bool) {
	dst := Address{IP: to}
	f, ok := msg.(*frame)
	if !ok {
		return dst, true
	}
	if f.bvlc.Function != plumbing.BVLCFuncUnicast {
		return dst, false
	}
	if f.npdu.Control&0x20 != 0 {
		if f.npdu.DNET == 0xFFFF || len(f.npdu.DADR) == 0 {
			return dst, false // a broadcast on a remote network
		}
		dst.Net, dst.MAC = f.npdu.DNET, f.npdu.DADR
	}
	return dst, true
}

// send marshals one message and sends it. It is the only place the client
// writes to the network.
func (c *Client) send(to netip.AddrPort, msg plumbing.BACnet) error {
	dst, unicast := destination(to, msg)
	writable := unicast && !c.cfg.ReadOnly && c.isWritable(dst)
	if err := allowed(msg, c.cfg.ReadOnly, writable); err != nil {
		c.log.Warn("refused to send", "to", to, "error", err)
		return err
	}
	if unicast {
		if rule, denied := c.isDenied(dst, 0); denied {
			c.count(dst, func(s *PeerStats) { s.Denied++ })
			c.log.Debug("refused to send to a denied source", "to", dst, "rule", rule)
			return fmt.Errorf("%w: %s", ErrSourceDenied, rule)
		}
	}
	b, err := msg.MarshalBinary()
	if err != nil {
		return err
	}
	if c.cfg.OnPacket != nil {
		c.cfg.OnPacket(PacketEvent{Time: time.Now(), Peer: to, Data: b})
	}
	_, err = c.conn.WriteToUDPAddrPort(b, to)
	return err
}

// sendAPDU sends one APDU in the given envelope.
func (c *Client) sendAPDU(bvlcFunc uint8, to netip.AddrPort, npdu *plumbing.NPDU, a *plumbing.APDU) error {
	return c.send(to, &frame{bvlc: plumbing.NewBVLC(bvlcFunc), npdu: npdu, apdu: a})
}

// npduTo builds the NPDU of a message for one device.
func npduTo(to Address, expectingReply bool) *plumbing.NPDU {
	npdu := plumbing.NewNPDU(false, to.Routed(), false, expectingReply)
	if to.Routed() {
		npdu.DNET = to.Net
		npdu.DLEN = uint8(len(to.MAC))
		npdu.DADR = to.MAC
		npdu.Hop = 255
	}
	return npdu
}

// broadcastTarget is where a broadcast goes: the subnet's broadcast
// address, or the BBMD as a Distribute-Broadcast-To-Network when the client
// is a foreign device.
func (c *Client) broadcastTarget() (netip.AddrPort, uint8) {
	if c.fd != nil {
		return c.fd.cfg.BBMD, plumbing.BVLCFuncDistributeBroadcastToNetwork
	}
	return c.cfg.Broadcast, plumbing.BVLCFuncBroadcast
}

// WhoIs broadcasts one Who-Is addressed to every network, so that devices
// behind routers answer as well. The I-Ams it draws arrive through OnIAm.
func (c *Client) WhoIs() error {
	return c.whoIs(nil)
}

// WhoIsRange is a WhoIs that only the devices with an instance from low to
// high answer: on a large network, discovery in slices rather than a storm
// of I-Ams at once.
func (c *Client) WhoIsRange(low, high uint32) error {
	payload, err := services.WhoIsObjects(low, high)
	if err != nil {
		return err
	}
	return c.whoIs(payload)
}

// WhoIsAt sends a Who-Is to one address instead of broadcasting it, for a
// device that is known to be there and not which it is. Its I-Am arrives
// through OnIAm if it reaches the client: many devices broadcast it on their
// own subnet, whatever the Who-Is came as.
func (c *Client) WhoIsAt(to netip.AddrPort) error {
	if err := c.limiter.wait(context.Background(), c.closed); err != nil {
		return err
	}
	whoIs := plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedWhoIs, nil)
	return c.sendAPDU(plumbing.BVLCFuncUnicast, to, plumbing.NewNPDU(false, false, false, false), whoIs)
}

func (c *Client) whoIs(payload []objects.APDUPayload) error {
	if err := c.limiter.wait(context.Background(), c.closed); err != nil {
		return err
	}
	npdu := plumbing.NewNPDU(false, true, false, false)
	npdu.DNET = 0xFFFF
	npdu.Hop = 255
	whoIs := plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedWhoIs, payload)
	to, function := c.broadcastTarget()
	return c.sendAPDU(function, to, npdu, whoIs)
}

// ---- receiving -------------------------------------------------------------

func (c *Client) readLoop() {
	defer close(c.done)
	defer c.failAll(ErrClosed)

	// Nothing parsed from buf aliases it, so it is reused.
	buf := make([]byte, 2048)
	failures := 0
	for {
		n, src, err := c.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			select {
			case <-c.closed:
				return
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// Windows reports an ICMP port-unreachable for an earlier send as
			// a read error; the socket is still good. An error that keeps
			// coming back is not that, and must not spin.
			if failures++; failures > 100 {
				time.Sleep(50 * time.Millisecond)
			}
			continue
		}
		failures = 0
		src = netip.AddrPortFrom(src.Addr().Unmap(), src.Port())
		if c.cfg.OnPacket != nil {
			c.cfg.OnPacket(PacketEvent{Time: time.Now(), Inbound: true, Peer: src, Data: append([]byte(nil), buf[:n]...)})
		}

		if src == c.cfg.Local {
			continue // the client's own broadcast coming back
		}
		if c.excluded(src.Addr()) {
			c.log.Debug("dropped a datagram from another adapter's subnet", "from", src)
			continue
		}

		// A message a BBMD forwarded comes from the device at its origin;
		// the BBMD is only the carrier.
		var hdr plumbing.BVLC
		if err := hdr.UnmarshalBinary(buf[:n]); err != nil {
			c.log.Debug("dropped a datagram that is not BACnet/IP", "from", src, "error", err)
			continue
		}
		if origin, ok := hdr.OriginAddrPort(); ok {
			if origin == c.cfg.Local {
				continue // the client's own broadcast, back from the BBMD
			}
			src = origin
		}
		msg, err := bacnet.ParseWithFallback(buf[:n])
		if err != nil {
			c.log.Debug("dropped a datagram that could not be parsed", "from", src, "error", err, "bytes", fmt.Sprintf("%x", buf[:n]))
			continue
		}
		c.dispatch(src, msg)
	}
}

func (c *Client) excluded(addr netip.Addr) bool {
	for _, p := range c.cfg.Exclude {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (c *Client) dispatch(src netip.AddrPort, msg plumbing.BACnet) {
	switch m := msg.(type) {
	case *services.UnconfirmedIAm:
		c.noteSource(src, m.NPDU)
		d, err := m.Decode()
		if err != nil {
			c.log.Warn("dropped an I-Am that could not be decoded", "from", sourceOf(src, m.NPDU), "error", err)
			return
		}
		if d.DeviceType != objects.ObjectTypeDevice {
			c.log.Warn("dropped an I-Am for an object that is not a device", "from", sourceOf(src, m.NPDU), "type", d.DeviceType, "instance", d.InstanceNum)
			return
		}
		from := sourceOf(src, m.NPDU)
		c.learnInstance(d.InstanceNum, from)
		if c.dropDenied(from, d.InstanceNum) {
			return
		}
		c.log.Debug("I-Am", "device", d.InstanceNum, "from", from)
		if c.cfg.OnIAm == nil {
			return
		}
		c.cfg.OnIAm(Device{
			Instance:     d.InstanceNum,
			Addr:         from,
			MaxAPDU:      d.MaxAPDULength,
			Segmentation: d.SegmentationSupported,
			VendorID:     d.VendorId,
		})
	case *services.ComplexACK:
		c.deliver(src, m.NPDU, m.APDU)
	case *services.Error:
		c.deliver(src, m.NPDU, m.APDU)
	case *services.Reject:
		c.deliver(src, m.NPDU, m.APDU)
	case *services.Abort:
		c.deliver(src, m.NPDU, m.APDU)
	case *services.SimpleACK:
		c.deliver(src, m.NPDU, m.APDU)
	case *services.COVNotification:
		c.covNotification(src, m.NPDU, m.APDU)
	case *plumbing.NetworkMessage:
		c.networkMessage(src, m)
	case *plumbing.BVLCResult:
		c.bvlcResult(src, m)
	case *plumbing.ReadForeignDeviceTableAck:
		c.bvlcReply(src, waitFDT, m)
	case *plumbing.ReadBroadcastDistributionTableAck:
		c.bvlcReply(src, waitBDT, m)
	default:
		c.log.Debug("ignored a message", "from", src, "type", fmt.Sprintf("%T", msg), "service", msg.GetService())
	}
}

// sourceOf gives the address of the device that sent a message.
func sourceOf(src netip.AddrPort, npdu *plumbing.NPDU) Address {
	addr := Address{IP: src}
	if npdu.Control&0x08 != 0 {
		addr.Net = npdu.SNET
		addr.MAC = npdu.SADR
	}
	return addr
}

// noteSource learns from a message with a source network that the BACnet/IP
// address it came from is the router for that network.
func (c *Client) noteSource(src netip.AddrPort, npdu *plumbing.NPDU) {
	if npdu.Control&0x08 != 0 && npdu.SNET != 0 && npdu.SNET != 0xFFFF {
		c.learnRoute(npdu.SNET, src)
	}
}

// dropDenied reports whether a message from a device is to be dropped for
// the device being on the deny list, and logs it.
func (c *Client) dropDenied(from Address, instance uint32) bool {
	rule, denied := c.isDenied(from, instance)
	if denied {
		c.log.Debug("dropped a message from a denied source", "from", from, "rule", rule)
	}
	return denied
}

func (c *Client) deliver(src netip.AddrPort, npdu *plumbing.NPDU, a *plumbing.APDU) {
	c.noteSource(src, npdu)
	from := sourceOf(src, npdu)
	if c.dropDenied(from, 0) {
		return
	}
	key := keyOf(from, a.InvokeID)
	c.mu.Lock()
	ch := c.pending[key]
	c.mu.Unlock()
	if ch == nil {
		return // a late reply, or something the device sent on its own
	}
	select {
	case ch <- inbound{apdu: a}:
	default:
	}
}

// failNet ends every request in flight to the devices of one network.
func (c *Client) failNet(dnet uint16, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, ch := range c.pending {
		if key.net == dnet {
			select {
			case ch <- inbound{err: err}:
			default:
			}
		}
	}
}

func (c *Client) failAll(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range c.pending {
		select {
		case ch <- inbound{err: err}:
		default:
		}
	}
}

// ---- confirmed requests ----------------------------------------------------

// errNoReply is a timeout before any part of a reply arrived, when resending
// the request is safe.
var errNoReply = errors.New("no reply yet")

// sendMode says how exchange sends a request.
type sendMode uint8

const (
	// sendRetrying resends a request that got no reply, Config.Retries
	// times, with the same invoke ID: a read.
	sendRetrying sendMode = iota
	// sendOnce sends a request once and never again: a write, which a
	// device that did it and lost the acknowledgement must not do twice.
	sendOnce
	// sendProbe sends once to an address where there may be nobody, and
	// when nothing answers at an address the client has not talked to
	// before, keeps nothing about it, no counters and no invoke ID.
	sendProbe
)

// confirmed sends one request and returns the tags of the ComplexACK that
// answers it, reassembled if it came in segments, or none for a service that
// is answered with a SimpleACK. A request that gets no reply is resent with
// the same invoke ID.
func (c *Client) confirmed(ctx context.Context, to Address, service uint8, payload []objects.APDUPayload) ([]*objects.Object, error) {
	return c.exchange(ctx, to, service, payload, sendRetrying)
}

// exchange is confirmed, sending as the mode says. An error from before the
// request left the client wraps ErrNotSent.
func (c *Client) exchange(ctx context.Context, to Address, service uint8, payload []objects.APDUPayload, mode sendMode) ([]*objects.Object, error) {
	if rule, denied := c.isDenied(to, 0); denied {
		if mode != sendProbe {
			c.count(to, func(s *PeerStats) { s.Denied++ })
		}
		return nil, notSent(fmt.Errorf("%w: %s", ErrSourceDenied, rule))
	}
	via, function, err := c.resolve(ctx, to)
	if err != nil {
		return nil, notSent(err)
	}

	release, err := c.acquire(ctx, to)
	if err != nil {
		return nil, notSent(err)
	}
	defer release()

	retries := c.cfg.Retries
	if mode != sendRetrying {
		retries = 0
	}
	stranger, heard := false, false
	if mode == sendProbe {
		who := keyOf(to, 0)
		c.mu.Lock()
		_, talked := c.invokes[who]
		c.mu.Unlock()
		stranger = !talked
		defer func() {
			if stranger && !heard {
				c.mu.Lock()
				delete(c.invokes, who)
				delete(c.stats, who)
				c.mu.Unlock()
			}
		}()
	}

	req := plumbing.NewAPDU(plumbing.ConfirmedReq, service, payload)
	req.SetAPDUFlags(true, false, false) // segmented reply accepted
	req.MaxSeg = 7                       // more than 64 segments
	req.MaxSize = 5                      // up to 1476 bytes
	key, ch := c.begin(to, req)
	defer func() {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
	}()

	// Segment-ACKs go back the way the request went.
	peer := to
	to.IP = via
	for attempt := 0; attempt <= retries; attempt++ {
		// Before the first send, an error means the request never left.
		unsent := func(err error) error {
			if attempt == 0 {
				return notSent(err)
			}
			return err
		}
		if to.Routed() {
			if err := c.waitAvailable(ctx, to.Net, ch); err != nil {
				return nil, unsent(err)
			}
		}
		if err := c.limiter.wait(ctx, c.closed); err != nil {
			return nil, unsent(err)
		}
		if err := c.sendAPDU(function, via, npduTo(to, true), req); err != nil {
			return nil, unsent(err)
		}
		c.count(peer, func(s *PeerStats) {
			s.Sent++
			if attempt > 0 {
				s.Retries++
			}
		})
		tags, err := c.await(ctx, ch, to, req.InvokeID, service)
		if !errors.Is(err, errNoReply) {
			if Declined(err) {
				c.count(peer, func(s *PeerStats) { s.Errors++ })
			}
			heard = err == nil || Declined(err)
			return tags, err
		}
	}
	if stranger {
		return nil, ErrTimeout
	}
	c.count(peer, func(s *PeerStats) { s.Timeouts++ })
	if to.Routed() && function == plumbing.BVLCFuncUnicast {
		// The router may have moved: find it again next time.
		c.dropRoute(to.Net, via)
	}
	return nil, ErrTimeout
}

// begin gives a request the next invoke ID for its device, one that no
// request in flight to the device has, and registers where its reply goes.
func (c *Client) begin(to Address, req *plumbing.APDU) (txKey, chan inbound) {
	ch := make(chan inbound, 64)
	peer := keyOf(to, 0)
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.invokes[peer]
	for range 256 {
		id++
		if _, used := c.pending[keyOf(to, id)]; !used {
			break
		}
	}
	c.invokes[peer] = id
	req.InvokeID = id
	key := keyOf(to, id)
	c.pending[key] = ch
	return key, ch
}

// resolve says where a request for to goes: the device's own address, the
// router learned for its network, or failing both, every router at once.
func (c *Client) resolve(ctx context.Context, to Address) (netip.AddrPort, uint8, error) {
	if !to.Routed() {
		if !to.IP.IsValid() {
			return netip.AddrPort{}, 0, errors.New("no address to send to")
		}
		return to.IP, plumbing.BVLCFuncUnicast, nil
	}
	if router, ok := c.route(to.Net); ok {
		return router, plumbing.BVLCFuncUnicast, nil
	}
	if to.IP.IsValid() {
		return to.IP, plumbing.BVLCFuncUnicast, nil
	}
	routers, err := c.whoIsRouter(ctx, &to.Net, true)
	if err != nil {
		return netip.AddrPort{}, 0, err
	}
	if len(routers) > 0 {
		return routers[0].Router, plumbing.BVLCFuncUnicast, nil
	}
	// No router has answered for the network. Broadcast the request with
	// the network set, and learn the router from the reply.
	bcast, function := c.broadcastTarget()
	return bcast, function, nil
}

func (c *Client) await(ctx context.Context, ch chan inbound, to Address, invokeID, service uint8) ([]*objects.Object, error) {
	var (
		data     []byte
		nextSeq  uint8
		sinceAck uint8
		segments int
		// window is the segments the device sends between acknowledgements,
		// settled from its proposal on the first segment.
		window uint8
	)
	timer := time.NewTimer(c.cfg.Timeout)
	defer timer.Stop()
	for {
		var in inbound
		select {
		case <-ctx.Done():
			if segments > 0 {
				c.abort(to, invokeID, objects.AbortReasonOther)
			}
			return nil, ctx.Err()
		case <-c.closed:
			return nil, ErrClosed
		case <-timer.C:
			if segments > 0 {
				c.abort(to, invokeID, objects.AbortReasonTSMTimeout)
				return nil, fmt.Errorf("%w: the reply stopped after %d segments", ErrTimeout, segments)
			}
			return nil, errNoReply
		case in = <-ch:
		}
		if in.err != nil {
			return nil, in.err
		}

		a := in.apdu
		switch a.Type {
		case plumbing.Error:
			return nil, decodeError(a.Objects)
		case plumbing.Reject:
			return nil, &RejectError{Reason: a.Reason}
		case plumbing.Abort:
			return nil, &AbortError{Reason: a.Reason}
		case plumbing.SimpleAck:
			if service == services.ServiceConfirmedSubscribeCOV || service == services.ServiceConfirmedWriteProperty {
				return nil, nil
			}
			return nil, errors.New("SimpleACK in reply to a read request")
		case plumbing.ComplexAck:
		default:
			continue
		}

		if !a.IsSegmented() {
			return toTags(a.Objects)
		}

		if window == 0 {
			window = max(1, a.ProposedWindowSize)
			if window > maxWindow {
				c.log.Warn("segment window clamped", "peer", to, "invoke", invokeID, "service", service,
					"proposed", window, "actual", maxWindow)
				c.count(to, func(s *PeerStats) { s.WindowClamped++ })
				window = maxWindow
			}
		}
		if a.SequenceNumber != nextSeq {
			// A duplicate or a gap: ask again from the last one in order.
			c.segmentAck(to, invokeID, nextSeq-1, window, true)
			continue
		}
		data = append(data, a.Payload...)
		segments++
		if segments > c.cfg.MaxSegments {
			c.abort(to, invokeID, objects.AbortReasonBufferOverflow)
			c.log.Error("reply too large", "peer", to, "invoke", invokeID, "service", service,
				"segments", segments, "bytes", len(data), "max_segments", c.cfg.MaxSegments)
			c.count(to, func(s *PeerStats) { s.ResponseTooLarge++ })
			return nil, &ResponseTooLargeError{
				Peer: to, InvokeID: invokeID, Service: service,
				Segments: segments, Bytes: len(data), MaxSegments: c.cfg.MaxSegments,
			}
		}
		nextSeq++
		sinceAck++

		// The first segment is acknowledged alone, then each full window,
		// then the last segment.
		more := a.HasMoreSegments()
		if a.SequenceNumber == 0 || sinceAck >= window || !more {
			c.segmentAck(to, invokeID, a.SequenceNumber, window, false)
			sinceAck = 0
		}
		if more {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(c.cfg.Timeout)
			continue
		}

		var tags []*objects.Object
		for offset := 0; offset < len(data); {
			tag, n, err := objects.DecodeTag(data[offset:])
			if err != nil {
				return nil, fmt.Errorf("decoding the reassembled reply: %w", err)
			}
			tags = append(tags, tag)
			offset += n
		}
		return tags, nil
	}
}

func (c *Client) segmentAck(to Address, invokeID, sequence, window uint8, negative bool) {
	ack := &plumbing.APDU{
		Type:             plumbing.SegmentAck,
		InvokeID:         invokeID,
		SequenceNumber:   sequence,
		ActualWindowSize: window,
	}
	if negative {
		ack.Flags = 0x2
	}
	c.sendAPDU(plumbing.BVLCFuncUnicast, to.IP, npduTo(to, false), ack)
}

// abort tells a device to stop a segmented reply the client will not take.
func (c *Client) abort(to Address, invokeID, reason uint8) {
	c.sendAPDU(plumbing.BVLCFuncUnicast, to.IP, npduTo(to, false),
		&plumbing.APDU{Type: plumbing.Abort, InvokeID: invokeID, Reason: reason})
}

func toTags(payload []objects.APDUPayload) ([]*objects.Object, error) {
	tags := make([]*objects.Object, 0, len(payload))
	for _, p := range payload {
		tag, ok := p.(*objects.Object)
		if !ok {
			return nil, fmt.Errorf("unexpected payload %T in a reply", p)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// ---- errors ----------------------------------------------------------------

// BACnetError is a device's Error reply, or the error it gave for one
// property of a ReadPropertyMultiple.
type BACnetError struct{ Class, Code uint32 }

func (e *BACnetError) Error() string {
	return fmt.Sprintf("%s: %s", ErrorClassName(e.Class), ErrorCodeName(e.Code))
}

// RejectError is a device's Reject reply.
type RejectError struct{ Reason uint8 }

func (e *RejectError) Error() string {
	return "rejected: " + reasonName(objects.RejectReasonMap, e.Reason)
}

// AbortError is a device's Abort reply.
type AbortError struct{ Reason uint8 }

func (e *AbortError) Error() string {
	return "aborted: " + reasonName(objects.AbortReasonMap, e.Reason)
}

// Declined reports whether err is the device saying no, as opposed to a
// timeout or a reply that could not be read.
func Declined(err error) bool {
	var (
		be *BACnetError
		re *RejectError
		ae *AbortError
	)
	return errors.As(err, &be) || errors.As(err, &re) || errors.As(err, &ae)
}

func decodeError(payload []objects.APDUPayload) error {
	e := &BACnetError{}
	if len(payload) >= 2 {
		if class, err := objects.DecEnumerated(payload[0]); err == nil {
			e.Class = class
		}
		if code, err := objects.DecEnumerated(payload[1]); err == nil {
			e.Code = code
		}
	}
	return e
}

func reasonName(names map[uint8]string, v uint8) string {
	if name, ok := names[v]; ok {
		return name
	}
	return fmt.Sprintf("reason %d", v)
}

func routerRejectName(reason uint8) string {
	names := []string{
		"other error", "not directly connected to it and no route known", "router busy",
		"unknown network layer message", "message too long", "security error", "addressing error",
	}
	if int(reason) < len(names) {
		return names[reason]
	}
	return fmt.Sprintf("reason %d", reason)
}

// MACString renders a MAC address the way it is usually written: an IP
// address and port when it is a BACnet/IP one, a station number when it is
// one byte, hex otherwise.
func MACString(mac []byte) string {
	switch {
	case len(mac) == 1:
		return fmt.Sprint(mac[0])
	case len(mac) == 6 && mac[4] == 0xBA && mac[5]&0xF0 == 0xC0:
		return fmt.Sprintf("%s:%d", net.IP(mac[:4]), uint16(mac[4])<<8|uint16(mac[5]))
	}
	return hex.EncodeToString(mac)
}
