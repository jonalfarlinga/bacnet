package client_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
)

// TestReadObjectListRefusesAnImpossibleLength is a device that cannot send
// its object list whole and then says, in element 0, that it has 4294967295
// objects. ReadObjectList used to allocate for that many before reading one,
// a 64 GiB slice; it must fail instead, having sent no element read.
func TestReadObjectListRefusesAnImpossibleLength(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	requests := make(chan int, 16)
	go func() {
		buf := make([]byte, 1500)
		for n := 0; ; n++ {
			size, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			requests <- n
			// BVLC (4) and a plain NPDU (2), then a confirmed request:
			// type, max segments and APDU, invoke id.
			if size < 9 {
				continue
			}
			invoke := buf[8]
			var apdu []byte
			if n == 0 {
				// The whole list: segmentation not supported.
				apdu = []byte{0x71, invoke, 4}
			} else {
				// Element 0: device 1's object-list[0] = 4294967295.
				apdu = []byte{0x30, invoke, 0x0C, 0x0C, 0x02, 0x00, 0x00, 0x01, 0x19, 0x4C, 0x29, 0x00,
					0x3E, 0x24, 0xFF, 0xFF, 0xFF, 0xFF, 0x3F}
			}
			msg := append([]byte{0x81, 0x0A, 0, byte(6 + len(apdu)), 0x01, 0x00}, apdu...)
			conn.WriteToUDP(msg, from)
		}
	}()

	c, err := client.Open(client.Config{Local: netip.MustParseAddrPort("127.0.0.1:0"), Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	dev := client.Device{
		Instance:     1,
		Addr:         client.Address{IP: conn.LocalAddr().(*net.UDPAddr).AddrPort()},
		MaxAPDU:      1476,
		Segmentation: 3,
	}
	_, err = c.ReadObjectList(context.Background(), dev, nil)
	if !errors.Is(err, client.ErrObjectListTooLong) {
		t.Fatalf("ReadObjectList = %v, want ErrObjectListTooLong", err)
	}
	if n := len(requests); n != 2 {
		t.Errorf("%d requests sent, want 2: the whole list, then its length", n)
	}
}
