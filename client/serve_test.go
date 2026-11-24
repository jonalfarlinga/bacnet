package client_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// program is what the device of these tests is: a Device object 77 with a
// name, a list of three numbers and a description too long for one APDU.
func program(_ context.Context, object client.ObjectID, property client.PropertyRef) ([]*objects.Object, error) {
	if object != deviceObject(77) {
		return nil, &client.BACnetError{Class: uint32(objects.ErrorClassObject), Code: client.ErrorCodeUnknownObject}
	}
	switch property.ID {
	case uint32(objects.PropertyIdObjectName):
		return []*objects.Object{objects.EncString("program")}, nil
	case uint32(objects.PropertyIdObjectList):
		list := []uint{10, 20, 30}
		if property.Index == nil {
			return []*objects.Object{objects.EncUnsignedInteger(10), objects.EncUnsignedInteger(20), objects.EncUnsignedInteger(30)}, nil
		}
		if *property.Index == 0 {
			return []*objects.Object{objects.EncUnsignedInteger(uint(len(list)))}, nil
		}
		return []*objects.Object{objects.EncUnsignedInteger(list[*property.Index-1])}, nil
	case uint32(objects.PropertyIdDescription):
		return []*objects.Object{objects.EncString(strings.Repeat("long ", 400))}, nil
	case uint32(objects.PropertyIdLocation):
		return nil, errors.New("the program broke")
	}
	return nil, &client.BACnetError{Class: uint32(objects.ErrorClassProperty), Code: client.ErrorCodeUnknownProperty}
}

// openServing opens a client that is device 77 and answers reads of it, and
// another that reads.
func openServing(t *testing.T) (device, reader *client.Client) {
	t.Helper()
	device, _, _ = openIdentity(t, nil)
	if err := device.Handle(client.ServiceReadProperty, client.ServeReadProperty(program)); err != nil {
		t.Fatal(err)
	}
	if err := device.Handle(client.ServiceReadPropertyMultiple, client.ServeReadPropertyMultiple(program)); err != nil {
		t.Fatal(err)
	}
	reader, err := client.Open(client.Config{Local: netip.MustParseAddrPort("127.0.0.1:0"), Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reader.Close() })
	return device, reader
}

// One client reads the Device object another serves.
func TestClientReadsClient(t *testing.T) {
	device, reader := openServing(t)
	ctx := context.Background()
	to := client.Address{IP: device.Local()}

	values, err := reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdObjectName)
	if err != nil || len(values) != 1 || values[0].Str != "program" {
		t.Fatalf("reading the name gave %+v, %v, want \"program\"", values, err)
	}
	values, err = reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdObjectList)
	if err != nil || len(values) != 3 || values[2].Uint != 30 {
		t.Errorf("reading the list gave %+v, %v, want 10, 20, 30", values, err)
	}
	values, err = reader.ReadPropertyIndex(ctx, to, deviceObject(77), objects.PropertyIdObjectList, 2)
	if err != nil || len(values) != 1 || values[0].Uint != 20 {
		t.Errorf("reading element 2 of the list gave %+v, %v, want 20", values, err)
	}

	// The wildcard instance is the device itself, and is answered as it.
	values, err = reader.ReadProperty(ctx, to, deviceObject(client.WildcardInstance), objects.PropertyIdObjectName)
	if err != nil || len(values) != 1 || values[0].Str != "program" {
		t.Errorf("reading the name of the wildcard device gave %+v, %v, want \"program\"", values, err)
	}
	wild, err := reader.ReadPropertyMultiple(ctx, to, []client.ReadSpec{
		{Object: deviceObject(client.WildcardInstance), Props: []uint16{objects.PropertyIdObjectName}}})
	if err != nil || len(wild) != 1 || wild[0].Object != deviceObject(77) || wild[0].Props[0].Err != nil {
		t.Errorf("reading the wildcard device gave %+v, %v, want the name of device 77", wild, err)
	}

	// What the program says no to, the requester is told.
	var be *client.BACnetError
	_, err = reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdModelName)
	if !errors.As(err, &be) || be.Code != client.ErrorCodeUnknownProperty {
		t.Errorf("reading a property the program has not gave %v, want unknown property", err)
	}
	_, err = reader.ReadProperty(ctx, to, deviceObject(78), objects.PropertyIdObjectName)
	if !errors.As(err, &be) || be.Code != client.ErrorCodeUnknownObject {
		t.Errorf("reading an object the program has not gave %v, want unknown object", err)
	}
	// A handler that fails is the device's problem, not a silence.
	_, err = reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdLocation)
	if !errors.As(err, &be) || be.Class != 0 || be.Code != 25 {
		t.Errorf("a handler that failed gave %v, want device: operational problem", err)
	}
	// The client does not segment: a reply longer than an APDU is aborted.
	var ae *client.AbortError
	_, err = reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdDescription)
	if !errors.As(err, &ae) || ae.Reason != objects.AbortReasonSegmentationNotSupported {
		t.Errorf("a reply of 2000 bytes gave %v, want an abort for want of segmentation", err)
	}

	// Several properties at once, the errors among the values.
	results, err := reader.ReadPropertyMultiple(ctx, to, []client.ReadSpec{
		{Object: deviceObject(77), Props: []uint16{objects.PropertyIdObjectName, objects.PropertyIdModelName, objects.PropertyIdObjectList}},
		{Object: deviceObject(78), Props: []uint16{objects.PropertyIdObjectName}},
	})
	if err != nil || len(results) != 2 || len(results[0].Props) != 3 || len(results[1].Props) != 1 {
		t.Fatalf("reading several properties gave %+v, %v, want two objects with 3 and 1", results, err)
	}
	p := results[0].Props
	if p[0].Err != nil || p[0].Values[0].Str != "program" || len(p[2].Values) != 3 {
		t.Errorf("the name and the list came as %+v and %+v", p[0], p[2])
	}
	if !errors.As(p[1].Err, &be) || be.Code != client.ErrorCodeUnknownProperty {
		t.Errorf("the property the program has not came as %+v, want unknown property", p[1])
	}
	if !errors.As(results[1].Props[0].Err, &be) || be.Code != client.ErrorCodeUnknownObject {
		t.Errorf("the object the program has not came as %+v, want unknown object", results[1].Props[0])
	}

	// A service the program has no handler for is rejected, not left to time
	// out.
	var re *client.RejectError
	_, err = reader.ReadRange(ctx, to, client.ReadRangeRequest{Object: deviceObject(77), Property: objects.PropertyIdObjectList})
	if !errors.As(err, &re) || re.Reason != objects.RejectReasonUnrecognizedService {
		t.Errorf("a service with no handler gave %v, want it rejected as unrecognized", err)
	}
	// And one whose handler is taken away is rejected again.
	if err := device.Handle(client.ServiceReadProperty, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = reader.ReadProperty(ctx, to, deviceObject(77), objects.PropertyIdObjectName); !errors.As(err, &re) {
		t.Errorf("a service whose handler was removed gave %v, want it rejected", err)
	}
}

// A handler that returns nothing acknowledges: a write is answered with a
// SimpleACK, and the handler is given what was written.
func TestClientWritesClient(t *testing.T) {
	device, _, _ := openIdentity(t, nil)
	got := make(chan client.Request, 1)
	err := device.Handle(services.ServiceConfirmedWriteProperty, func(_ context.Context, req client.Request) ([]*objects.Object, error) {
		got <- req
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := client.Open(client.Config{
		Local:    netip.MustParseAddrPort("127.0.0.1:0"),
		Timeout:  time.Second,
		Writable: []client.SourceRule{{IP: netip.PrefixFrom(device.Local().Addr(), 32)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { writer.Close() })

	err = writer.WriteProperty(context.Background(), client.Address{IP: device.Local()}, deviceObject(77),
		objects.PropertyIdDescription, []*objects.Object{objects.EncString("written")}, 0)
	if err != nil {
		t.Fatalf("the write gave %v, want it acknowledged", err)
	}
	req := <-got
	if req.Service != services.ServiceConfirmedWriteProperty || req.From.IP != writer.Local() || len(req.Tags) < 4 || req.MaxReply != 1476 {
		t.Errorf("the handler was given %+v, want the write from %v", req, writer.Local())
	}
}

// Without an identity, and with one on a read-only client, nothing is
// answered and no handler is taken.
func TestNoIdentityServesNothing(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(*client.Config)
		want   error
	}{
		"no identity": {func(cfg *client.Config) { cfg.Identity = nil }, client.ErrNoIdentity},
		"read-only":   {func(cfg *client.Config) { cfg.ReadOnly = true }, client.ErrServiceNotAllowed},
	} {
		t.Run(name, func(t *testing.T) {
			silent, _, _ := openIdentity(t, tc.change)
			if err := silent.Handle(client.ServiceReadProperty, client.ServeReadProperty(program)); !errors.Is(err, tc.want) {
				t.Errorf("Handle gave %v, want %v", err, tc.want)
			}
			reader, err := client.Open(client.Config{Local: netip.MustParseAddrPort("127.0.0.1:0"), Timeout: 200 * time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			_, err = reader.ReadProperty(context.Background(), client.Address{IP: silent.Local()}, deviceObject(77), objects.PropertyIdObjectName)
			if !errors.Is(err, client.ErrTimeout) {
				t.Errorf("reading it gave %v, want no answer", err)
			}
		})
	}
}

// sendRequest sends a confirmed request from a socket to the client.
func sendRequest(t *testing.T, from *net.UDPConn, to netip.AddrPort, function uint8, npdu *plumbing.NPDU, a *plumbing.APDU) {
	t.Helper()
	bvlc := plumbing.NewBVLC(function)
	b := make([]byte, bvlc.MarshalLen()+npdu.MarshalLen()+a.MarshalLen())
	bvlc.Length = uint16(len(b))
	bvlc.MarshalTo(b)
	npdu.MarshalTo(b[bvlc.MarshalLen():])
	if err := a.MarshalTo(b[bvlc.MarshalLen()+npdu.MarshalLen():]); err != nil {
		t.Fatal(err)
	}
	if _, err := from.WriteToUDPAddrPort(b, to); err != nil {
		t.Fatal(err)
	}
}

// awaitReply reads the reply that arrives at a socket, its NPDU and APDU, or
// nil when none does.
func awaitReply(t *testing.T, at *net.UDPConn, wait time.Duration) (*plumbing.NPDU, *plumbing.APDU) {
	t.Helper()
	at.SetReadDeadline(time.Now().Add(wait))
	buf := make([]byte, 2048)
	n, _, err := at.ReadFromUDPAddrPort(buf)
	if err != nil {
		return nil, nil
	}
	var bvlc plumbing.BVLC
	var npdu plumbing.NPDU
	var apdu plumbing.APDU
	if err := bvlc.UnmarshalBinary(buf[:n]); err != nil {
		t.Fatal(err)
	}
	if err := npdu.UnmarshalBinary(buf[bvlc.MarshalLen():n]); err != nil {
		t.Fatal(err)
	}
	if err := apdu.UnmarshalBinary(buf[bvlc.MarshalLen()+npdu.MarshalLen() : n]); err != nil {
		t.Fatal(err)
	}
	return &npdu, &apdu
}

func readNameRequest(invoke uint8) *plumbing.APDU {
	a := plumbing.NewAPDU(plumbing.ConfirmedReq, services.ServiceConfirmedReadProperty,
		services.ConfirmedReadPropertyObjects(objects.ObjectTypeDevice, 77, objects.PropertyIdObjectName))
	a.InvokeID, a.MaxSize = invoke, 5
	return a
}

// The reply echoes the invoke ID and goes back the way the request came:
// through the router, for a request from behind one. A request that is a
// broadcast, or for a device of another network, is not for the client.
func TestReplyGoesBackTheWayTheRequestCame(t *testing.T) {
	device, _ := openServing(t)
	asker, _ := listen(t)

	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), readNameRequest(41))
	npdu, reply := awaitReply(t, asker, answer)
	if reply == nil || reply.Type != plumbing.ComplexAck || reply.InvokeID != 41 || npdu.Control&0x20 != 0 {
		t.Fatalf("a read drew %+v, want a ComplexACK with invoke ID 41 and no network", reply)
	}

	routed := plumbing.NewNPDU(false, false, true, true)
	routed.SNET, routed.SLEN, routed.SADR = 5, 1, []byte{9}
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, routed, readNameRequest(42))
	npdu, reply = awaitReply(t, asker, answer)
	if reply == nil || reply.InvokeID != 42 || npdu.Control&0x20 == 0 || npdu.DNET != 5 || string(npdu.DADR) != "\x09" {
		t.Fatalf("a read from network 5 drew %+v in %+v, want a reply for MAC 9 of network 5", reply, npdu)
	}

	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncBroadcast, plain(), readNameRequest(43))
	elsewhere := plumbing.NewNPDU(false, true, false, true)
	elsewhere.DNET, elsewhere.DLEN, elsewhere.DADR, elsewhere.Hop = 7, 1, []byte{3}, 255
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, elsewhere, readNameRequest(44))
	if _, reply := awaitReply(t, asker, noAnswer); reply != nil {
		t.Errorf("a request that was not for the client drew %+v", reply)
	}
}

// A service the library has no type for still reaches the handler the
// program has for it, and is rejected when it has none. A request in
// segments is aborted.
func TestRequestsTheParserDoesNotKnow(t *testing.T) {
	device, _ := openServing(t)
	asker, _ := listen(t)
	file := func(invoke uint8) *plumbing.APDU {
		a := plumbing.NewAPDU(plumbing.ConfirmedReq, services.ServiceConfirmedAtomicReadFile,
			[]objects.APDUPayload{objects.EncUnsignedInteger(1)})
		a.InvokeID, a.MaxSize = invoke, 5
		return a
	}

	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), file(7))
	_, reply := awaitReply(t, asker, answer)
	if reply == nil || reply.Type != plumbing.Reject || reply.InvokeID != 7 || reply.Reason != objects.RejectReasonUnrecognizedService {
		t.Fatalf("a service with no handler drew %+v, want a Reject of invoke ID 7", reply)
	}

	err := device.Handle(services.ServiceConfirmedAtomicReadFile, func(_ context.Context, req client.Request) ([]*objects.Object, error) {
		return []*objects.Object{objects.EncUnsignedInteger(uint(len(req.Tags)))}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), file(8))
	_, reply = awaitReply(t, asker, answer)
	if reply == nil || reply.Type != plumbing.ComplexAck || reply.InvokeID != 8 || reply.Service != services.ServiceConfirmedAtomicReadFile {
		t.Fatalf("a service with a handler drew %+v, want its ComplexACK", reply)
	}

	segment := readNameRequest(9)
	segment.Flags |= 0x8 | 0x4 // segmented, more follow
	segment.Payload = []byte{0x0c}
	segment.Objects = nil
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), segment)
	_, reply = awaitReply(t, asker, answer)
	if reply == nil || reply.Type != plumbing.Abort || reply.Flags&0x1 == 0 || reply.InvokeID != 9 ||
		reply.Reason != objects.AbortReasonSegmentationNotSupported {
		t.Fatalf("a request in segments drew %+v, want a server's Abort of invoke ID 9", reply)
	}
}

// A request sent again while the handler has it is not handled twice, and a
// reply that the requester takes less of than it is long is aborted.
func TestRequestSentAgainAndReplyTooLong(t *testing.T) {
	device, _, _ := openIdentity(t, nil)
	asker, _ := listen(t)
	var calls atomic.Int32
	release := make(chan struct{})
	err := device.Handle(client.ServiceReadProperty, func(_ context.Context, _ client.Request) ([]*objects.Object, error) {
		calls.Add(1)
		<-release
		return []*objects.Object{objects.EncString(strings.Repeat("x", 100))}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), readNameRequest(3))
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), readNameRequest(3))
	time.Sleep(noAnswer)
	close(release)
	if _, reply := awaitReply(t, asker, answer); reply == nil || reply.Type != plumbing.ComplexAck || reply.InvokeID != 3 {
		t.Fatalf("the request drew %+v, want one ComplexACK", reply)
	}
	if _, reply := awaitReply(t, asker, noAnswer); reply != nil || calls.Load() != 1 {
		t.Errorf("the request sent twice was handled %d times and drew a second reply %+v", calls.Load(), reply)
	}

	// The same reply to a requester that takes 50 bytes.
	small := readNameRequest(4)
	small.MaxSize = 0
	sendRequest(t, asker, device.Local(), plumbing.BVLCFuncUnicast, plain(), small)
	_, reply := awaitReply(t, asker, answer)
	if reply == nil || reply.Type != plumbing.Abort || reply.Flags&0x1 == 0 || reply.InvokeID != 4 {
		t.Fatalf("a reply longer than the requester takes drew %+v, want a server's Abort", reply)
	}
}
