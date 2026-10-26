package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/spf13/cobra"
)

// The flags the examples built on the client share: where it listens, how
// it reaches the device, and how long it waits.
var (
	exInterface string
	exBBMD      string
	exDNET      uint
	exDADR      string
	exTimeout   time.Duration
	exDebug     bool
)

func clientFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringVar(&exInterface, "interface", "", "Adapter to use, by name or IPv4 address; its subnet is broadcast to. Without it, the client listens on an ephemeral port and broadcasts to --remote-address.")
	f.StringVar(&exBBMD, "bbmd", "", "BBMD to register with as a foreign device, IP[:port], when the devices are on another IP subnet.")
	f.UintVar(&exDNET, "dnet", 0, "Network number of a device behind a BACnet router at --remote-address; 0 for a device at the address itself.")
	f.StringVar(&exDADR, "dadr", "", "MAC address of the device on --dnet, in hex (for example 05, or c0a8016fbac0).")
	f.DurationVar(&exTimeout, "timeout", 3*time.Second, "How long to wait for each reply.")
	f.BoolVar(&exDebug, "debug", false, "Log what the client hears and leaves unanswered as well.")
}

// openClient opens a client as the flags say, logging what it does to
// stderr. onIAm, if given, hears every I-Am.
func openClient(onIAm func(client.Device)) *client.Client {
	var cfg client.Config
	if exInterface != "" {
		ifc, err := client.FindInterface(exInterface)
		if err != nil {
			log.Fatal(err)
		}
		cfg = ifc.Config(client.DefaultPort)
	} else {
		cfg.Local = netip.AddrPortFrom(netip.IPv4Unspecified(), 0)
		cfg.Broadcast = remoteAddrPort()
	}
	if exBBMD != "" {
		cfg.ForeignDevice = &client.ForeignDeviceConfig{BBMD: addrPort(exBBMD)}
	}
	cfg.Timeout = exTimeout
	cfg.Retries = 1
	cfg.OnIAm = onIAm
	level := slog.LevelInfo
	if exDebug {
		level = slog.LevelDebug
	}
	cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	c, err := client.Open(cfg)
	if err != nil {
		log.Fatalf("opening the client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), exTimeout)
	defer cancel()
	if err := waitForeignDevice(ctx, c); err != nil {
		c.Close()
		log.Fatalf("registering with the BBMD: %v", err)
	}
	return c
}

// Open starts registration asynchronously. The examples must wait before
// their first broadcast, otherwise a BBMD may discard their only Who-Is.
func waitForeignDevice(ctx context.Context, c *client.Client) error {
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		status, configured := c.ForeignDeviceStatus()
		if !configured || status.State == client.FDRegistered {
			return nil
		}
		if status.State == client.FDFailed {
			return status.LastResult
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

// target is the device the flags name: at --remote-address, or behind the
// router there on --dnet at --dadr.
func target() client.Address {
	to := client.Address{IP: remoteAddrPort()}
	if exDNET != 0 {
		mac, err := hex.DecodeString(exDADR)
		if err != nil || len(mac) == 0 {
			log.Fatalf("--dadr %q is not a MAC address in hex", exDADR)
		}
		to.Net, to.MAC = uint16(exDNET), mac
	}
	return to
}

func remoteAddrPort() netip.AddrPort {
	return addrPort(rAddr)
}

// addrPort parses IP[:port], with the BACnet port when none is given.
func addrPort(s string) netip.AddrPort {
	if !strings.Contains(s, ":") {
		s += fmt.Sprintf(":%d", client.DefaultPort)
	}
	ap, err := netip.ParseAddrPort(s)
	if err != nil {
		log.Fatalf("%q is not an IP address and port: %v", s, err)
	}
	return ap
}

// formatValue renders one decoded value for the terminal.
func formatValue(v client.Value) string {
	switch v.Kind {
	case client.KindNull:
		return "null"
	case client.KindBoolean:
		return fmt.Sprint(v.Bool)
	case client.KindUnsigned:
		return fmt.Sprint(v.Uint)
	case client.KindSigned:
		return fmt.Sprint(v.Int)
	case client.KindReal, client.KindDouble:
		return fmt.Sprintf("%g", v.Float)
	case client.KindString:
		return fmt.Sprintf("%q", v.Str)
	case client.KindOctets:
		return "0x" + hex.EncodeToString(v.Bytes)
	case client.KindBits:
		bits := make([]byte, len(v.Bits))
		for i, b := range v.Bits {
			bits[i] = '0'
			if b {
				bits[i] = '1'
			}
		}
		return "bits " + string(bits)
	case client.KindEnumerated:
		return fmt.Sprintf("enumerated %d", v.Uint)
	case client.KindDate:
		if len(v.Bytes) == 4 {
			return fmt.Sprintf("date %d-%02d-%02d", int(v.Bytes[0])+1900, v.Bytes[1], v.Bytes[2])
		}
	case client.KindTime:
		if len(v.Bytes) == 4 {
			return fmt.Sprintf("time %02d:%02d:%02d.%02d", v.Bytes[0], v.Bytes[1], v.Bytes[2], v.Bytes[3])
		}
	case client.KindObjectID:
		return fmt.Sprintf("%s %d", objectTypeName(v.Object.Type), v.Object.Instance)
	case client.KindContext:
		return fmt.Sprintf("[%d] 0x%s", v.Tag, hex.EncodeToString(v.Bytes))
	case client.KindConstructed:
		return fmt.Sprintf("[%d]{%s}", v.Tag, formatValues(v.Items))
	}
	return fmt.Sprintf("%+v", v)
}

func formatValues(values []client.Value) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = formatValue(v)
	}
	return strings.Join(parts, ", ")
}

func objectTypeName(t uint16) string { return objects.ObjectTypeName(t) }
func propertyName(id uint32) string  { return objects.PropertyName(id) }

// printResults prints what a ReadPropertyMultiple or ReadMany returned.
func printResults(results []client.ObjectResult) {
	for _, r := range results {
		fmt.Printf("%s %d\n", objectTypeName(r.Object.Type), r.Object.Instance)
		for _, p := range r.Props {
			name := propertyName(p.ID)
			if p.Index != nil {
				name += fmt.Sprintf("[%d]", *p.Index)
			}
			if p.Err != nil {
				fmt.Printf("  %-28s error: %v\n", name, p.Err)
				continue
			}
			fmt.Printf("  %-28s %s\n", name, formatValues(p.Values))
		}
	}
}

func uint16slice(s []int) []uint16 {
	u := make([]uint16, len(s))
	for i, v := range s {
		u[i] = uint16(v)
	}
	return u
}
