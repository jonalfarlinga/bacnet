//go:build ignore

// readonly_check runs read-only checks against one live BACnet/IP device and
// reports what this library could do with it.
//
//	go run scripts/readonly_check.go -remote 192.168.1.100
//	go run scripts/readonly_check.go -remote 10.0.0.1 -dnet 16604 -dadr 0c
//
// It never changes the device. The only things it sends are ReadProperty,
// ReadPropertyMultiple and ReadRange requests, the Segment-ACKs that a
// segmented reply needs, and, with -whois, one unicast Who-Is. Everything
// goes through one send function that refuses anything else. It does not
// subscribe, write, announce itself with an I-Am, or answer what the device
// sends to it.
//
// Requests go out one at a time with -delay between them. The exit status is
// 0 when no check failed, 1 when one did, and 2 when it could not start.
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

var (
	flagRemote  = flag.String("remote", "", "device, or the router in front of it: IP or IP:port (port 47808 if left out)")
	flagLocal   = flag.String("local", ":0", "local address to bind; give IP:47808 if the device only answers that port")
	flagDNET    = flag.Uint("dnet", 0, "network number of a routed device; 0 means the device is at -remote itself")
	flagDADR    = flag.String("dadr", "", "MAC address of a routed device on -dnet, in hex (for example 0c, or c0a80164bac0)")
	flagDevice  = flag.Int("device", -1, "device instance; found by reading the wildcard instance if left out")
	flagWhoIs   = flag.Bool("whois", false, "also send one unicast Who-Is (the I-Am it triggers may be broadcast on the device's network)")
	flagSegment = flag.Bool("segment-test", false, "also re-read the object list with a 50-byte max-APDU, to make the device segment its reply")
	flagMaxObj  = flag.Int("max-objects", 50, "how many objects of the object list to read")
	flagHistory = flag.Int("history", 10, "how many records to read from each of the first 3 trend logs; 0 skips ReadRange")
	flagTimeout = flag.Duration("timeout", 3*time.Second, "how long to wait for each reply")
	flagRetries = flag.Int("retries", 1, "how many times to resend a request that got no reply")
	flagDelay   = flag.Duration("delay", 50*time.Millisecond, "pause before each request")
	flagFrames  = flag.String("frames", "", "file to log every frame sent and received to, in hex")
	flagVerbose = flag.Bool("v", false, "print every object read")
)

// Property identifiers the objects package has no short name for here.
const (
	propDescription     = 28
	propFirmware        = 44
	propLocation        = 58
	propMaxAPDU         = 62
	propModelName       = 70
	propObjectID        = 75
	propObjectName      = 77
	propProtocolRev     = 139
	propRelinquishDflt  = 104
	propSegmentation    = 107
	propSystemStatus    = 112
	propVendorID        = 120
	propVendorName      = 121
	propRecordCount     = 141
	propTotalRecords    = 145
	propAppSoftware     = 12
	propServicesSupport = 97
	propObjectTypesSupp = 96

	wildcardInstance = 4194303
)

// Max-APDU codes of a confirmed request: the largest reply we accept.
const (
	maxAPDU50   = 0
	maxAPDU1476 = 5
)

type result struct{ area, check, status, detail string }

type checker struct {
	conn   net.PacketConn
	target *net.UDPAddr
	dnet   uint16
	dadr   []byte
	invoke uint8
	// maxAPDU is the max-APDU code put in requests.
	maxAPDU uint8
	frames  io.Writer

	results []result
	sent    map[string]int

	rxFrames, rxOther, rtMismatch int
	rtts                          []time.Duration
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Read-only checks against one BACnet/IP device.")
		fmt.Fprintln(os.Stderr, "usage: go run scripts/readonly_check.go -remote IP[:port] [flags]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *flagRemote == "" {
		flag.Usage()
		os.Exit(2)
	}

	c, err := newChecker()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot start:", err)
		os.Exit(2)
	}
	defer c.conn.Close()

	fmt.Printf("read-only check of %s from %s at %s\n", c.targetName(), c.conn.LocalAddr(), time.Now().Format(time.RFC3339))

	if *flagWhoIs {
		c.whoIs()
	}
	inst, ok := c.identify()
	if ok {
		supported := c.device(inst)
		objs := c.objectList(inst)
		c.points(objs, supported)
		c.priorityArray(objs)
		c.history(objs, supported)
		c.errorReply(inst)
		if *flagSegment {
			c.segmentTest(inst)
		}
		c.capabilities(supported, objs)
	}
	os.Exit(c.summary())
}

func newChecker() (*checker, error) {
	remote := *flagRemote
	if _, _, err := net.SplitHostPort(remote); err != nil {
		remote = net.JoinHostPort(remote, "47808")
	}
	target, err := net.ResolveUDPAddr("udp4", remote)
	if err != nil {
		return nil, err
	}
	if *flagDNET > 0xFFFE {
		return nil, fmt.Errorf("-dnet %d is not a network number", *flagDNET)
	}
	dadr, err := hex.DecodeString(strings.NewReplacer(":", "", " ", "").Replace(*flagDADR))
	if err != nil {
		return nil, fmt.Errorf("-dadr: %v", err)
	}
	if (*flagDNET == 0) != (len(dadr) == 0) {
		return nil, errors.New("-dnet and -dadr go together: a routed device needs both")
	}

	conn, err := net.ListenPacket("udp4", *flagLocal)
	if err != nil {
		return nil, err
	}
	c := &checker{
		conn:    conn,
		target:  target,
		dnet:    uint16(*flagDNET),
		dadr:    dadr,
		maxAPDU: maxAPDU1476,
		sent:    map[string]int{},
	}
	if *flagFrames != "" {
		f, err := os.Create(*flagFrames)
		if err != nil {
			conn.Close()
			return nil, err
		}
		c.frames = f
	}
	return c, nil
}

func (c *checker) targetName() string {
	if c.dnet == 0 {
		return c.target.String()
	}
	return fmt.Sprintf("network %d MAC %x via %s", c.dnet, c.dadr, c.target)
}

func (c *checker) rec(area, check, status, detail string) {
	c.results = append(c.results, result{area, check, status, detail})
	fmt.Printf("[%-4s] %-11s %-34s %s\n", status, area, check, detail)
}

// ---- sending ---------------------------------------------------------------

// allowed reports whether an APDU is one this program may send: a read
// request, the acknowledgement of a reply segment, or a Who-Is.
func allowed(a *plumbing.APDU) (name string, ok bool) {
	switch a.Type {
	case plumbing.ConfirmedReq:
		switch a.Service {
		case services.ServiceConfirmedReadProperty:
			return "ReadProperty", true
		case services.ServiceConfirmedReadPropMultiple:
			return "ReadPropertyMultiple", true
		case services.ServiceConfirmedReadRange:
			return "ReadRange", true
		}
	case plumbing.UnConfirmedReq:
		if a.Service == services.ServiceUnconfirmedWhoIs && len(a.Objects) == 0 {
			return "Who-Is", true
		}
	case plumbing.SegmentAck:
		return "Segment-ACK", true
	}
	return "", false
}

// send marshals and sends one APDU to the device. It is the only place this
// program writes to the network.
func (c *checker) send(a *plumbing.APDU) ([]byte, error) {
	name, ok := allowed(a)
	if !ok {
		return nil, fmt.Errorf("refusing to send PDU type %d service %d: not a read-only message", a.Type, a.Service)
	}

	bvlc := plumbing.NewBVLC(plumbing.BVLCFuncUnicast)
	npdu := plumbing.NewNPDU(false, c.dnet != 0, false, a.Type == plumbing.ConfirmedReq)
	if c.dnet != 0 {
		npdu.DNET = c.dnet
		npdu.DLEN = uint8(len(c.dadr))
		npdu.DADR = c.dadr
		npdu.Hop = 255
	}

	b := make([]byte, bvlc.MarshalLen()+npdu.MarshalLen()+a.MarshalLen())
	bvlc.Length = uint16(len(b))
	offset := 0
	if err := bvlc.MarshalTo(b); err != nil {
		return nil, err
	}
	offset += bvlc.MarshalLen()
	if err := npdu.MarshalTo(b[offset:]); err != nil {
		return nil, err
	}
	offset += npdu.MarshalLen()
	if err := a.MarshalTo(b[offset:]); err != nil {
		return nil, err
	}
	return b, c.write(name, b)
}

func (c *checker) write(name string, b []byte) error {
	c.sent[name]++
	c.logFrame("TX", c.target, b)
	_, err := c.conn.WriteTo(b, c.target)
	return err
}

func (c *checker) logFrame(dir string, addr net.Addr, b []byte) {
	if c.frames != nil {
		fmt.Fprintf(c.frames, "%s %s %s %x\n", time.Now().Format("15:04:05.000000"), dir, addr, b)
	}
}

// ---- receiving -------------------------------------------------------------

type parts struct {
	bvlc *plumbing.BVLC
	npdu *plumbing.NPDU
	apdu *plumbing.APDU
}

// partsOf returns the layers of a reply to a confirmed request, or false for
// any other message.
func partsOf(msg plumbing.BACnet) (parts, bool) {
	switch m := msg.(type) {
	case *services.ComplexACK:
		return parts{m.BVLC, m.NPDU, m.APDU}, true
	case *services.SimpleACK:
		return parts{m.BVLC, m.NPDU, m.APDU}, true
	case *services.Error:
		return parts{m.BVLC, m.NPDU, m.APDU}, true
	case *services.Reject:
		return parts{m.BVLC, m.NPDU, m.APDU}, true
	case *services.Abort:
		return parts{m.BVLC, m.NPDU, m.APDU}, true
	}
	return parts{}, false
}

// recv reads the next datagram from the device, parses it, and checks that
// marshalling the parsed message gives the received bytes back.
func (c *checker) recv(deadline time.Time) (plumbing.BACnet, []byte, error) {
	buf := make([]byte, 2048)
	for {
		c.conn.SetReadDeadline(deadline)
		n, addr, err := c.conn.ReadFrom(buf)
		if err != nil {
			return nil, nil, err
		}
		if ua, ok := addr.(*net.UDPAddr); !ok || !ua.IP.Equal(c.target.IP) {
			c.rxOther++
			continue
		}
		raw := append([]byte(nil), buf[:n]...)
		c.logFrame("RX", addr, raw)
		c.rxFrames++

		msg, err := bacnet.ParseWithFallback(raw)
		if err != nil {
			return nil, raw, fmt.Errorf("parsing %x: %w", raw, err)
		}
		if out, err := msg.MarshalBinary(); err != nil || !bytes.Equal(out, raw) {
			c.rtMismatch++
			if c.frames != nil {
				fmt.Fprintf(c.frames, "  ROUND-TRIP MISMATCH err=%v got=%x\n", err, out)
			}
		}
		return msg, raw, nil
	}
}

// fromTarget reports whether a reply came from the device being checked, as
// opposed to another device behind the same router.
func (c *checker) fromTarget(npdu *plumbing.NPDU) bool {
	if c.dnet == 0 {
		return npdu.Control&0x08 == 0
	}
	return npdu.Control&0x08 != 0 && npdu.SNET == c.dnet && bytes.Equal(npdu.SADR, c.dadr)
}

// pduError is a reply by which the device declined a request.
type pduError struct {
	kind        string // Error, Reject or Abort
	class, code uint8  // Error
	reason      uint8  // Reject and Abort
}

func (e *pduError) Error() string {
	switch e.kind {
	case "Error":
		return fmt.Sprintf("Error %s / %s", errorClassName(e.class), errorCodeName(e.code))
	case "Reject":
		return fmt.Sprintf("Reject: %s", nameOr(objects.RejectReasonMap, e.reason))
	}
	return fmt.Sprintf("Abort: %s", nameOr(objects.AbortReasonMap, e.reason))
}

type reply struct {
	cack     *services.ComplexACK
	rtt      time.Duration
	segments int
	window   uint8
	size     int
}

// confirmed sends one read request and returns the ComplexACK that answers
// it, reassembled if it came in segments. A request that gets no reply is
// resent with the same invoke ID.
func (c *checker) confirmed(service uint8, objs []objects.APDUPayload) (reply, error) {
	time.Sleep(*flagDelay)

	c.invoke++
	req := plumbing.NewAPDU(plumbing.ConfirmedReq, service, objs)
	req.SetAPDUFlags(true, false, false) // segmented reply accepted
	req.MaxSeg = 7                       // more than 64 segments
	req.MaxSize = c.maxAPDU
	req.InvokeID = c.invoke

	var lastErr error
	for attempt := 0; attempt <= *flagRetries; attempt++ {
		start := time.Now()
		if _, err := c.send(req); err != nil {
			return reply{}, err
		}
		r, err := c.await(req.InvokeID, start)
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			lastErr = fmt.Errorf("no reply within %s", *flagTimeout)
			continue
		}
		return r, err
	}
	return reply{}, lastErr
}

func (c *checker) await(invokeID uint8, start time.Time) (reply, error) {
	var (
		r        reply
		header   []byte
		data     []byte
		service  uint8
		nextSeq  uint8
		sinceAck uint8
	)
	deadline := start.Add(*flagTimeout)
	for {
		msg, raw, err := c.recv(deadline)
		if err != nil {
			return r, err
		}
		// A router that cannot reach the device's network says so instead
		// of forwarding the request.
		if nm, ok := msg.(*plumbing.NetworkMessage); ok && c.dnet != 0 {
			if reason, dnet, err := nm.RejectReason(); err == nil && dnet == c.dnet {
				return r, fmt.Errorf("the router rejected network %d: %s", dnet, routerRejectName(reason))
			}
			continue
		}
		p, ok := partsOf(msg)
		if !ok || p.apdu.InvokeID != invokeID || !c.fromTarget(p.npdu) {
			// Something the device sent on its own, or a late reply.
			continue
		}

		switch m := msg.(type) {
		case *services.Error:
			e := &pduError{kind: "Error"}
			if d, err := m.Decode(); err == nil {
				e.class, e.code = d.ErrorClass, d.ErrorCode
			}
			return r, e
		case *services.Reject:
			return r, &pduError{kind: "Reject", reason: p.apdu.Reason}
		case *services.Abort:
			return r, &pduError{kind: "Abort", reason: p.apdu.Reason}
		case *services.SimpleACK:
			return r, errors.New("SimpleACK in reply to a read request")
		case *services.ComplexACK:
			a := p.apdu
			if !a.IsSegmented() {
				r.cack, r.rtt, r.segments, r.size = m, time.Since(start), 1, len(raw)
				c.rtts = append(c.rtts, r.rtt)
				return r, nil
			}

			if a.SequenceNumber != nextSeq {
				// A duplicate or a gap: ask again from the last one in order.
				c.segmentAck(invokeID, nextSeq-1, a.ProposedWindowSize, true)
				continue
			}
			if nextSeq == 0 {
				header = raw[:p.bvlc.MarshalLen()+p.npdu.MarshalLen()]
				service = a.Service
				r.window = a.ProposedWindowSize
				r.rtt = time.Since(start)
				c.rtts = append(c.rtts, r.rtt)
			}
			data = append(data, a.Payload...)
			nextSeq++
			sinceAck++
			r.segments++

			// The first segment is acknowledged alone, then each full window,
			// then the last segment.
			more := a.HasMoreSegments()
			if a.SequenceNumber == 0 || sinceAck >= a.ProposedWindowSize || !more {
				c.segmentAck(invokeID, a.SequenceNumber, a.ProposedWindowSize, false)
				sinceAck = 0
			}
			deadline = time.Now().Add(*flagTimeout)
			if more {
				continue
			}

			whole := append([]byte(nil), header...)
			whole = append(whole, plumbing.ComplexAck<<4, invokeID, service)
			whole = append(whole, data...)
			r.size = len(whole)
			if len(whole) > 0xFFFF {
				return r, fmt.Errorf("reassembled reply of %d bytes is too long to parse", len(whole))
			}
			whole[2], whole[3] = byte(len(whole)>>8), byte(len(whole))
			re, err := bacnet.Parse(whole)
			if err != nil {
				return r, fmt.Errorf("parsing the reassembled reply: %w", err)
			}
			cack, ok := re.(*services.ComplexACK)
			if !ok {
				return r, fmt.Errorf("reassembled reply is a %T", re)
			}
			r.cack = cack
			return r, nil
		}
	}
}

func (c *checker) segmentAck(invokeID, sequence, window uint8, negative bool) {
	ack := &plumbing.APDU{
		Type:             plumbing.SegmentAck,
		InvokeID:         invokeID,
		SequenceNumber:   sequence,
		ActualWindowSize: window,
	}
	if negative {
		ack.Flags = 0x2
	}
	c.send(ack)
}

// ---- reads -----------------------------------------------------------------

func (c *checker) readProperty(ot uint16, inst uint32, prop uint16) ([]*objects.Object, reply, error) {
	return c.decodeRP(c.confirmed(services.ServiceConfirmedReadProperty,
		services.ConfirmedReadPropertyObjects(ot, inst, prop)))
}

func (c *checker) readPropertyIndexed(ot uint16, inst uint32, prop uint16, index uint32) ([]*objects.Object, reply, error) {
	return c.decodeRP(c.confirmed(services.ServiceConfirmedReadProperty,
		services.ConfirmedReadPropertyIndexedObjects(ot, inst, prop, index)))
}

func (c *checker) decodeRP(r reply, err error) ([]*objects.Object, reply, error) {
	if err != nil {
		return nil, r, err
	}
	dec, err := r.cack.Decode()
	if err != nil {
		return nil, r, fmt.Errorf("decoding the reply: %w", err)
	}
	return dec.Tags, r, nil
}

// declined reports whether err is the device saying no, which is an answer
// this library handled, as opposed to a timeout or a reply it could not read.
func declined(err error) bool {
	var e *pduError
	return errors.As(err, &e)
}

// ---- checks ----------------------------------------------------------------

func (c *checker) whoIs() {
	fmt.Println("\n== Who-Is")
	if _, err := c.send(plumbing.NewAPDU(plumbing.UnConfirmedReq, services.ServiceUnconfirmedWhoIs, nil)); err != nil {
		c.rec("discovery", "unicast Who-Is", "FAIL", err.Error())
		return
	}
	deadline := time.Now().Add(*flagTimeout)
	for {
		msg, _, err := c.recv(deadline)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				c.rec("discovery", "unicast Who-Is", "INFO",
					"no I-Am arrived here; the device may have broadcast it, which a socket on another port or subnet does not see")
				return
			}
			c.rec("discovery", "unicast Who-Is", "FAIL", err.Error())
			return
		}
		iam, ok := msg.(*services.UnconfirmedIAm)
		if !ok {
			continue
		}
		d, err := iam.Decode()
		if err != nil {
			c.rec("discovery", "unicast Who-Is", "FAIL", "decoding the I-Am: "+err.Error())
			return
		}
		detail := fmt.Sprintf("I-Am: device %d, vendor %d, max APDU %d, segmentation %s",
			d.InstanceNum, d.VendorId, d.MaxAPDULength, segmentationName(uint32(d.SegmentationSupported)))
		if iam.NPDU.Control&0x08 != 0 {
			detail += fmt.Sprintf(", from network %d MAC %x", iam.NPDU.SNET, iam.NPDU.SADR)
		}
		c.rec("discovery", "unicast Who-Is", "PASS", detail)
		return
	}
}

// identify finds the device instance by reading the Device object under its
// wildcard instance, which every device answers for itself.
func (c *checker) identify() (uint32, bool) {
	fmt.Println("\n== Device")
	if *flagDevice >= 0 {
		return uint32(*flagDevice), true
	}
	tags, _, err := c.readProperty(objects.ObjectTypeDevice, wildcardInstance, propObjectID)
	if err != nil {
		c.rec("device", "find device instance", "FAIL", err.Error()+"; give -device if the instance is known")
		return 0, false
	}
	if len(tags) == 1 {
		if oid, ok := tags[0].Value.(objects.ObjectIdentifier); ok && oid.ObjectType == objects.ObjectTypeDevice {
			c.rec("device", "find device instance", "PASS", fmt.Sprintf("device %d", oid.InstanceNumber))
			return oid.InstanceNumber, true
		}
	}
	c.rec("device", "find device instance", "FAIL", "unexpected value: "+fmtTags(tags))
	return 0, false
}

func (c *checker) device(inst uint32) []bool {
	props := []struct {
		id     uint16
		name   string
		render func([]*objects.Object) string
	}{
		{propObjectName, "object-name", nil},
		{propVendorName, "vendor-name", nil},
		{propVendorID, "vendor-identifier", nil},
		{propModelName, "model-name", nil},
		{propFirmware, "firmware-revision", nil},
		{propAppSoftware, "application-software-version", nil},
		{propDescription, "description", nil},
		{propLocation, "location", nil},
		{propProtocolRev, "protocol-revision", nil},
		{propSystemStatus, "system-status", enumRenderer(systemStatusNames)},
		{propMaxAPDU, "max-apdu-length-accepted", nil},
		{propSegmentation, "segmentation-supported", enumRenderer(segmentationNames)},
		{propObjectTypesSupp, "protocol-object-types-supported", nil},
		{propServicesSupport, "protocol-services-supported", nil},
	}
	var supported []bool
	for _, p := range props {
		tags, _, err := c.readProperty(objects.ObjectTypeDevice, inst, p.id)
		switch {
		case declined(err):
			c.rec("device", p.name, "INFO", err.Error())
		case err != nil:
			c.rec("device", p.name, "FAIL", err.Error())
		default:
			detail := fmtTags(tags)
			if p.render != nil {
				detail = p.render(tags)
			}
			c.rec("device", p.name, "PASS", detail)
			if p.id == propServicesSupport && len(tags) == 1 {
				supported, _ = tags[0].Value.([]bool)
			}
		}
	}
	if len(supported) > 0 {
		var names []string
		for i, on := range supported {
			if on && i < len(serviceNames) {
				names = append(names, serviceNames[i])
			}
		}
		c.rec("device", "services it executes", "INFO", strings.Join(names, ", "))
	}
	return supported
}

func (c *checker) objectList(inst uint32) []objects.ObjectIdentifier {
	fmt.Println("\n== Object list")

	// Element 0 of an array is its length.
	count := -1
	tags, _, err := c.readPropertyIndexed(objects.ObjectTypeDevice, inst, objects.PropertyIdObjectList, 0)
	switch {
	case err != nil && !declined(err):
		c.rec("object-list", "length (index 0)", "FAIL", err.Error())
	case err != nil:
		c.rec("object-list", "length (index 0)", "INFO", err.Error())
	case len(tags) == 1:
		if n, ok := tags[0].Value.(uint32); ok {
			count = int(n)
			c.rec("object-list", "length (index 0)", "PASS", fmt.Sprintf("%d objects", n))
		}
	}
	if count < 0 && err == nil {
		c.rec("object-list", "length (index 0)", "FAIL", "unexpected value: "+fmtTags(tags))
	}

	var whole []objects.ObjectIdentifier
	tags, r, err := c.readProperty(objects.ObjectTypeDevice, inst, objects.PropertyIdObjectList)
	switch {
	case declined(err):
		// Usual for a long list on a device that cannot segment.
		c.rec("object-list", "whole array", "INFO", err.Error()+"; reading it element by element instead")
	case err != nil:
		c.rec("object-list", "whole array", "FAIL", err.Error())
	default:
		whole = objectIDs(tags)
		status := "PASS"
		detail := fmt.Sprintf("%d objects in %d segment(s), %d bytes", len(whole), r.segments, r.size)
		if len(whole) != len(tags) || (count >= 0 && len(whole) != count) {
			status = "FAIL"
			detail += fmt.Sprintf("; %d values decoded, length property says %d", len(tags), count)
		}
		c.rec("object-list", "whole array", status, detail)
	}

	// Walk it by index as well: that works on every device, and comparing
	// the two shows up a reply that was put together wrongly.
	n := count
	if n < 0 {
		n = len(whole)
	}
	if n > *flagMaxObj {
		n = *flagMaxObj
	}
	var indexed []objects.ObjectIdentifier
	failed := 0
	for i := 1; i <= n; i++ {
		tags, _, err := c.readPropertyIndexed(objects.ObjectTypeDevice, inst, objects.PropertyIdObjectList, uint32(i))
		ids := objectIDs(tags)
		if err != nil || len(ids) != 1 {
			failed++
			continue
		}
		indexed = append(indexed, ids[0])
	}
	if n > 0 {
		status := "PASS"
		if failed > 0 {
			status = "FAIL"
		}
		c.rec("object-list", "element by element", status, fmt.Sprintf("%d of the first %d read", len(indexed), n))
	}
	if len(whole) > 0 && failed == 0 && len(indexed) > 0 {
		same := len(whole) >= len(indexed)
		for i := 0; same && i < len(indexed); i++ {
			same = whole[i] == indexed[i]
		}
		status := "PASS"
		if !same {
			status = "FAIL"
		}
		c.rec("object-list", "both readings agree", status, fmt.Sprintf("first %d elements compared", len(indexed)))
	}

	list := whole
	if len(list) == 0 {
		list = indexed
	}
	hist := map[uint16]int{}
	for _, o := range list {
		hist[o.ObjectType]++
	}
	types := make([]int, 0, len(hist))
	for t := range hist {
		types = append(types, int(t))
	}
	sort.Ints(types)
	parts := make([]string, 0, len(types))
	for _, t := range types {
		parts = append(parts, fmt.Sprintf("%s x%d", objectTypeName(uint16(t)), hist[uint16(t)]))
	}
	if len(parts) > 0 {
		c.rec("object-list", "object types", "INFO", strings.Join(parts, ", "))
	}
	return list
}

func objectIDs(tags []*objects.Object) []objects.ObjectIdentifier {
	var ids []objects.ObjectIdentifier
	for _, t := range tags {
		if oid, ok := t.Value.(objects.ObjectIdentifier); ok {
			ids = append(ids, oid)
		}
	}
	return ids
}

// hasPresentValue tells the object types whose Present_Value and
// Status_Flags are worth reading.
func hasPresentValue(t uint16) bool {
	switch t {
	case objects.ObjectTypeAnalogInput, objects.ObjectTypeAnalogOutput, objects.ObjectTypeAnalogValue,
		objects.ObjectTypeBinaryInput, objects.ObjectTypeBinaryOutput, objects.ObjectTypeBinaryValue,
		objects.ObjectTypeMultiStateInput, objects.ObjectTypeMultiStateOutput, objects.ObjectTypeMultiStateValue:
		return true
	}
	return false
}

func (c *checker) points(objs []objects.ObjectIdentifier, supported []bool) {
	fmt.Println("\n== Points")
	if len(objs) == 0 {
		c.rec("points", "read objects", "SKIP", "no object list")
		return
	}
	if len(objs) > *flagMaxObj {
		objs = objs[:*flagMaxObj]
	}
	useRPM := len(supported) > 14 && supported[14]
	if !useRPM {
		c.rec("points", "ReadPropertyMultiple", "SKIP", "the device does not list it as supported; using ReadProperty")
	}

	rpmOK, rpmDeclined, rpmFail, propErrors := 0, 0, 0, 0
	rpOK, rpDeclined, rpFail := 0, 0, 0
	var firstFail string
	fail := func(o objects.ObjectIdentifier, what string, err error) {
		if firstFail == "" {
			firstFail = fmt.Sprintf("%s %s: %v", objectName(o), what, err)
		}
	}

	for _, o := range objs {
		props := []uint16{objects.PropertyIdObjectName}
		if hasPresentValue(o.ObjectType) {
			props = append(props, objects.PropertyIdPresentValue, objects.PropertyIdStatusFlags)
		}
		if o.ObjectType <= objects.ObjectTypeAnalogValue {
			props = append(props, objects.PropertyIdUnits)
		}

		line := ""
		if useRPM {
			r, err := c.confirmed(services.ServiceConfirmedReadPropMultiple,
				services.ConfirmedReadPropertyMultipleObjects(o.ObjectType, o.InstanceNumber, props))
			switch {
			case declined(err):
				rpmDeclined++
				line = err.Error()
			case err != nil:
				rpmFail++
				fail(o, "ReadPropertyMultiple", err)
			default:
				dec, err := r.cack.DecodeRPM()
				if err != nil {
					rpmFail++
					fail(o, "ReadPropertyMultiple decode", err)
					break
				}
				rpmOK++
				var n int
				line, n = fmtRPM(dec.Tags, r.cack.APDU.Objects)
				propErrors += n
			}
		}

		// The same values one at a time, as a device without
		// ReadPropertyMultiple needs.
		if !useRPM || hasPresentValue(o.ObjectType) {
			single := props
			if useRPM {
				single = []uint16{objects.PropertyIdPresentValue}
			}
			var parts []string
			for _, prop := range single {
				tags, _, err := c.readProperty(o.ObjectType, o.InstanceNumber, prop)
				switch {
				case declined(err):
					rpDeclined++
				case err != nil:
					rpFail++
					fail(o, propertyName(prop), err)
				default:
					rpOK++
					parts = append(parts, propertyName(prop)+"="+fmtProperty(prop, tags))
				}
			}
			if !useRPM {
				line = strings.Join(parts, "  ")
			}
		}
		if *flagVerbose {
			fmt.Printf("       %-26s %s\n", objectName(o), line)
		}
	}

	if useRPM {
		status := "PASS"
		if rpmFail > 0 {
			status = "FAIL"
		}
		detail := fmt.Sprintf("%d objects read, %d declined by the device, %d failed", rpmOK, rpmDeclined, rpmFail)
		if propErrors > 0 {
			detail += fmt.Sprintf("; %d properties came back as errors", propErrors)
		}
		c.rec("points", "ReadPropertyMultiple", status, detail)
	}
	status := "PASS"
	if rpFail > 0 {
		status = "FAIL"
	}
	c.rec("points", "ReadProperty", status, fmt.Sprintf("%d values read, %d declined by the device, %d failed", rpOK, rpDeclined, rpFail))
	if firstFail != "" {
		c.rec("points", "first failure", "INFO", firstFail)
	}
}

// fmtRPM renders a decoded ReadPropertyMultiple ack. The decoder returns a
// property access error as two bare enumerated values, so the errors are
// counted from the raw tags, where context tag 5 brackets them.
func fmtRPM(tags []*objects.Object, raw []objects.APDUPayload) (string, int) {
	propErrors := 0
	for _, o := range raw {
		if t, ok := o.(*objects.Object); ok && t.Kind == objects.TagOpening && t.TagNumber == 5 {
			propErrors++
		}
	}

	var sb strings.Builder
	prop := uint16(0)
	for _, t := range tags {
		if t.TagClass && t.TagNumber == 2 {
			id, _ := t.Value.(uint32)
			prop = uint16(id)
			fmt.Fprintf(&sb, "  %s=", propertyName(prop))
			continue
		}
		sb.WriteString(fmtProperty(prop, []*objects.Object{t}) + " ")
	}
	return strings.TrimSpace(sb.String()), propErrors
}

func (c *checker) priorityArray(objs []objects.ObjectIdentifier) {
	fmt.Println("\n== Priority array")
	tried := 0
	for _, o := range objs {
		switch o.ObjectType {
		case objects.ObjectTypeAnalogOutput, objects.ObjectTypeBinaryOutput, objects.ObjectTypeMultiStateOutput,
			objects.ObjectTypeAnalogValue, objects.ObjectTypeBinaryValue, objects.ObjectTypeMultiStateValue:
		default:
			continue
		}
		if tried++; tried > 5 {
			break
		}
		tags, _, err := c.readProperty(o.ObjectType, o.InstanceNumber, objects.PropertyIdPriorityArray)
		if declined(err) {
			continue // not commandable
		}
		label := objectName(o) + " priority-array"
		if err != nil {
			c.rec("priority", label, "FAIL", err.Error())
			return
		}
		status := "PASS"
		if len(tags) != 16 {
			status = "FAIL"
		}
		c.rec("priority", label, status, fmt.Sprintf("%d slots: %s", len(tags), fmtTags(tags)))
		if tags, _, err := c.readProperty(o.ObjectType, o.InstanceNumber, propRelinquishDflt); err == nil {
			c.rec("priority", objectName(o)+" relinquish-default", "PASS", fmtTags(tags))
		}
		return
	}
	c.rec("priority", "commandable object", "SKIP", "none found among the first objects that could be")
}

func (c *checker) history(objs []objects.ObjectIdentifier, supported []bool) {
	fmt.Println("\n== History")
	if *flagHistory <= 0 {
		c.rec("history", "ReadRange", "SKIP", "-history 0")
		return
	}
	found := 0
	for _, o := range objs {
		if o.ObjectType != objects.ObjectTypeTrendLog {
			continue
		}
		if found++; found > 3 {
			break
		}
		label := objectName(o)
		for _, prop := range []uint16{propRecordCount, propTotalRecords} {
			tags, _, err := c.readProperty(o.ObjectType, o.InstanceNumber, prop)
			switch {
			case declined(err):
				c.rec("history", label+" "+propertyName(prop), "INFO", err.Error())
			case err != nil:
				c.rec("history", label+" "+propertyName(prop), "FAIL", err.Error())
			default:
				c.rec("history", label+" "+propertyName(prop), "PASS", fmtTags(tags))
			}
		}

		check := fmt.Sprintf("%s ReadRange, first %d records", label, *flagHistory)
		r, err := c.confirmed(services.ServiceConfirmedReadRange,
			services.ConfirmedReadRangeObjects(o.ObjectType, o.InstanceNumber, objects.PropertyIdLogBuffer, 1, int32(*flagHistory)))
		if declined(err) {
			c.rec("history", check, "INFO", err.Error())
			continue
		}
		if err != nil {
			c.rec("history", check, "FAIL", err.Error())
			continue
		}
		dec, err := r.cack.DecodeRR()
		if err != nil {
			c.rec("history", check, "FAIL", "decoding the reply: "+err.Error())
			continue
		}
		stamps, values := 0, 0
		var first, last string
		for i, t := range dec.Tags {
			if !t.TagClass && t.TagNumber == objects.TagDate && i+1 < len(dec.Tags) {
				stamps++
				stamp := fmtValue(t) + " " + fmtValue(dec.Tags[i+1])
				if first == "" {
					first = stamp
				}
				last = stamp
			}
			if _, ok := t.Value.(float32); ok && t.TagClass {
				values++
			}
		}
		status := "PASS"
		if stamps != int(dec.ItemCount) {
			status = "FAIL"
		}
		c.rec("history", check, status, fmt.Sprintf(
			"item count %d, %d timestamps, %d real values, more items %t, %d segment(s); %s to %s; %d tags the decoder skipped",
			dec.ItemCount, stamps, values, dec.MoreItems, r.segments, first, last, len(dec.Skipped)))
	}
	if found == 0 {
		adv := "does not list"
		if len(supported) > 35 && supported[35] {
			adv = "lists"
		}
		c.rec("history", "trend logs", "SKIP", "none among the objects read; the device "+adv+" ReadRange as supported")
	}
}

// errorReply asks for a property the Device object does not have, to see an
// Error reply decoded.
func (c *checker) errorReply(inst uint32) {
	fmt.Println("\n== Error reply")
	_, _, err := c.readProperty(objects.ObjectTypeDevice, inst, objects.PropertyIdLogBuffer)
	var e *pduError
	switch {
	case errors.As(err, &e):
		c.rec("errors", "unknown property", "PASS", e.Error())
	case err != nil:
		c.rec("errors", "unknown property", "FAIL", err.Error())
	default:
		c.rec("errors", "unknown property", "INFO", "the device answered with a value")
	}
}

// segmentTest reads the object list twice, the second time allowing only
// 50-byte replies, and compares.
func (c *checker) segmentTest(inst uint32) {
	fmt.Println("\n== Segmented reply")
	want, _, err := c.readProperty(objects.ObjectTypeDevice, inst, objects.PropertyIdObjectList)
	if err != nil {
		c.rec("segments", "object list, max-APDU 50", "SKIP", "the unsegmented read did not work: "+err.Error())
		return
	}
	c.maxAPDU = maxAPDU50
	got, r, err := c.readProperty(objects.ObjectTypeDevice, inst, objects.PropertyIdObjectList)
	c.maxAPDU = maxAPDU1476
	switch {
	case declined(err):
		c.rec("segments", "object list, max-APDU 50", "INFO", err.Error())
	case err != nil:
		c.rec("segments", "object list, max-APDU 50", "FAIL", fmt.Sprintf("%v, after %d segment(s)", err, r.segments))
	case fmt.Sprint(objectIDs(got)) != fmt.Sprint(objectIDs(want)):
		c.rec("segments", "object list, max-APDU 50", "FAIL", fmt.Sprintf(
			"%d objects in %d segments differ from the %d read unsegmented", len(got), r.segments, len(want)))
	default:
		c.rec("segments", "object list, max-APDU 50", "PASS", fmt.Sprintf(
			"%d objects in %d segment(s), window %d, same as the unsegmented read", len(got), r.segments, r.window))
	}
}

// capabilities reports what the device says it can do, for the services this
// run could not try because they change the device.
func (c *checker) capabilities(supported []bool, objs []objects.ObjectIdentifier) {
	fmt.Println("\n== Capabilities the device reports (not exercised)")
	if len(supported) == 0 {
		c.rec("reports", "protocol-services-supported", "SKIP", "not read")
		return
	}
	has := func(bit int) string {
		if bit < len(supported) && supported[bit] {
			return "yes"
		}
		return "no"
	}
	trendLogs := 0
	for _, o := range objs {
		if o.ObjectType == objects.ObjectTypeTrendLog {
			trendLogs++
		}
	}
	c.rec("reports", "change of value", "INFO", fmt.Sprintf("SubscribeCOV %s, SubscribeCOVProperty %s", has(5), has(38)))
	c.rec("reports", "batch reads", "INFO", "ReadPropertyMultiple "+has(14))
	c.rec("reports", "history", "INFO", fmt.Sprintf("ReadRange %s, %d trend log(s) among the objects read", has(35), trendLogs))
	c.rec("reports", "alarms", "INFO", fmt.Sprintf("GetEventInformation %s, GetAlarmSummary %s, AcknowledgeAlarm %s", has(39), has(3), has(0)))
	c.rec("reports", "writes", "INFO", fmt.Sprintf("WriteProperty %s, WritePropertyMultiple %s", has(15), has(16)))
}

func (c *checker) summary() int {
	fmt.Println("\n== Summary")
	counts := map[string]int{}
	for _, r := range c.results {
		counts[r.status]++
	}
	fmt.Printf("checks: %d passed, %d failed, %d informational, %d skipped\n",
		counts["PASS"], counts["FAIL"], counts["INFO"], counts["SKIP"])

	names := make([]string, 0, len(c.sent))
	for name := range c.sent {
		names = append(names, name)
	}
	sort.Strings(names)
	sent := make([]string, 0, len(names))
	for _, name := range names {
		sent = append(sent, fmt.Sprintf("%s x%d", name, c.sent[name]))
	}
	fmt.Printf("sent: %s\n", strings.Join(sent, ", "))
	fmt.Printf("received: %d frames from the device, %d marshalled back differently", c.rxFrames, c.rtMismatch)
	if c.rxOther > 0 {
		fmt.Printf(", %d datagrams from other addresses ignored", c.rxOther)
	}
	fmt.Println()
	if len(c.rtts) > 0 {
		sort.Slice(c.rtts, func(i, j int) bool { return c.rtts[i] < c.rtts[j] })
		fmt.Printf("reply time: median %s, slowest %s over %d replies\n",
			c.rtts[len(c.rtts)/2].Round(time.Millisecond), c.rtts[len(c.rtts)-1].Round(time.Millisecond), len(c.rtts))
	}

	status := 0
	if c.rtMismatch > 0 {
		fmt.Println("FAIL: frames that do not marshal back to what was received (see -frames)")
		status = 1
	}
	for _, r := range c.results {
		if r.status == "FAIL" {
			fmt.Printf("FAIL: %s / %s: %s\n", r.area, r.check, r.detail)
			status = 1
		}
	}
	return status
}

// ---- rendering -------------------------------------------------------------

func fmtValue(t *objects.Object) string {
	if t == nil {
		return "<nil>"
	}
	switch v := t.Value.(type) {
	case nil:
		if !t.TagClass && t.TagNumber == objects.TagNull {
			return "NULL"
		}
		return fmt.Sprintf("undecoded(%x)", t.Data)
	case string:
		return fmt.Sprintf("%q", v)
	case []bool:
		var sb strings.Builder
		for _, bit := range v {
			if bit {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
		return sb.String()
	case objects.ObjectIdentifier:
		return objectName(v)
	case time.Time:
		if t.TagNumber == objects.TagDate {
			return v.Format("2006-01-02")
		}
		return v.Format("15:04:05")
	case []byte:
		return fmt.Sprintf("%x", v)
	default:
		return fmt.Sprint(v)
	}
}

func fmtTags(tags []*objects.Object) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		parts = append(parts, fmtValue(t))
	}
	s := strings.Join(parts, ", ")
	if len(s) > 200 {
		s = fmt.Sprintf("%s... (%d values)", s[:200], len(tags))
	}
	return s
}

// fmtProperty renders a property value, naming the enumerations it knows.
func fmtProperty(prop uint16, tags []*objects.Object) string {
	if len(tags) == 1 {
		if v, ok := tags[0].Value.(uint32); ok && !tags[0].TagClass && tags[0].TagNumber == objects.TagEnumerated {
			switch prop {
			case objects.PropertyIdUnits:
				if name, ok := objects.UnitMap[uint16(v)]; ok {
					return name
				}
			}
		}
		if bits, ok := tags[0].Value.([]bool); ok && prop == objects.PropertyIdStatusFlags && len(bits) == 4 {
			var set []string
			for i, name := range []string{"in-alarm", "fault", "overridden", "out-of-service"} {
				if bits[i] {
					set = append(set, name)
				}
			}
			if len(set) == 0 {
				return "normal"
			}
			return strings.Join(set, "+")
		}
	}
	return fmtTags(tags)
}

func enumRenderer(names map[uint32]string) func([]*objects.Object) string {
	return func(tags []*objects.Object) string {
		if len(tags) == 1 {
			if v, ok := tags[0].Value.(uint32); ok {
				if name, ok := names[v]; ok {
					return fmt.Sprintf("%s (%d)", name, v)
				}
			}
		}
		return fmtTags(tags)
	}
}

var systemStatusNames = map[uint32]string{
	0: "operational", 1: "operational-read-only", 2: "download-required",
	3: "download-in-progress", 4: "non-operational", 5: "backup-in-progress",
}

var segmentationNames = map[uint32]string{
	0: "both", 1: "transmit only", 2: "receive only", 3: "none",
}

func segmentationName(v uint32) string {
	if name, ok := segmentationNames[v]; ok {
		return name
	}
	return fmt.Sprint(v)
}

var objectTypeNames = map[uint16]string{
	0: "analog-input", 1: "analog-output", 2: "analog-value", 3: "binary-input", 4: "binary-output",
	5: "binary-value", 6: "calendar", 7: "command", 8: "device", 9: "event-enrollment", 10: "file",
	11: "group", 12: "loop", 13: "multi-state-input", 14: "multi-state-output", 15: "notification-class",
	16: "program", 17: "schedule", 18: "averaging", 19: "multi-state-value", 20: "trend-log",
	21: "life-safety-point", 22: "life-safety-zone", 23: "accumulator", 24: "pulse-converter",
	25: "event-log", 27: "trend-log-multiple", 28: "load-control", 29: "structured-view",
	56: "network-port",
}

func objectTypeName(t uint16) string {
	if name, ok := objectTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("type-%d", t)
}

func objectName(o objects.ObjectIdentifier) string {
	return fmt.Sprintf("%s,%d", objectTypeName(o.ObjectType), o.InstanceNumber)
}

func propertyName(id uint16) string {
	if name, ok := objects.PropertyMap[id]; ok {
		return name
	}
	return fmt.Sprintf("property-%d", id)
}

func nameOr(names map[uint8]string, v uint8) string {
	if name, ok := names[v]; ok {
		return name
	}
	return fmt.Sprint(v)
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

func errorClassName(v uint8) string {
	names := []string{"device", "object", "property", "resources", "security", "services", "vt", "communication"}
	if int(v) < len(names) {
		return names[v]
	}
	return fmt.Sprintf("class-%d", v)
}

func errorCodeName(v uint8) string {
	names := map[uint8]string{
		0: "other", 2: "configuration-in-progress", 3: "device-busy", 9: "inconsistent-parameters",
		25: "operational-problem", 27: "read-access-denied", 29: "service-request-denied", 30: "timeout",
		31: "unknown-object", 32: "unknown-property", 37: "value-out-of-range", 40: "write-access-denied",
		42: "invalid-array-index", 44: "not-cov-property", 45: "optional-functionality-not-supported",
		50: "property-is-not-an-array",
	}
	if name, ok := names[v]; ok {
		return name
	}
	return fmt.Sprintf("code-%d", v)
}

// Bits of Protocol_Services_Supported, in order.
var serviceNames = []string{
	"acknowledgeAlarm", "confirmedCOVNotification", "confirmedEventNotification", "getAlarmSummary",
	"getEnrollmentSummary", "subscribeCOV", "atomicReadFile", "atomicWriteFile", "addListElement",
	"removeListElement", "createObject", "deleteObject", "readProperty", "readPropertyConditional",
	"readPropertyMultiple", "writeProperty", "writePropertyMultiple", "deviceCommunicationControl",
	"confirmedPrivateTransfer", "confirmedTextMessage", "reinitializeDevice", "vtOpen", "vtClose",
	"vtData", "authenticate", "requestKey", "i-Am", "i-Have", "unconfirmedCOVNotification",
	"unconfirmedEventNotification", "unconfirmedPrivateTransfer", "unconfirmedTextMessage",
	"timeSynchronization", "who-Has", "who-Is", "readRange", "utcTimeSynchronization",
	"lifeSafetyOperation", "subscribeCOVProperty", "getEventInformation", "writeGroup",
}
