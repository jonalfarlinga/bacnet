package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/spf13/cobra"
)

func init() {
	probeCmd.Flags().Uint32Var(&probeDevice, "device", 0, "Device instance; 0 reads it from the device through the wildcard instance.")
}

var (
	probeDevice uint32

	probeCmd = &cobra.Command{
		Use:   "probe",
		Short: "Learn about a device that no Who-Is reaches.",
		Long: "Reads from the device at --remote-address what its I-Am would have said: its\n" +
			"instance, the largest APDU it accepts, whether it segments, and its vendor,\n" +
			"then its name and model. For a device on another IP subnet, or one that\n" +
			"does not announce itself.",
		Args: argValidation,
		Run:  probeExample,
	}
)

func probeExample(cmd *cobra.Command, args []string) {
	c := openClient(nil)
	defer c.Close()

	dev, err := c.Probe(context.Background(), target(), probeDevice)
	if err != nil {
		log.Fatalf("probing %v: %v", target(), err)
	}
	fmt.Printf("device %d at %v\n", dev.Instance, dev.Addr)
	fmt.Printf("  max APDU %d bytes, segmentation %d, vendor %d\n", dev.MaxAPDU, dev.Segmentation, dev.VendorID)

	results, err := c.ReadMany(context.Background(), dev, []client.ReadSpec{{
		Object: client.ObjectID{Type: objects.ObjectTypeDevice, Instance: dev.Instance},
		Props: []uint16{
			objects.PropertyIdObjectName, objects.PropertyIdModelName, objects.PropertyIdVendorName,
			objects.PropertyIdFirmwareRevision, objects.PropertyIdApplicationSoftwareVersion,
		},
	}})
	if err != nil {
		log.Fatalf("reading the device: %v", err)
	}
	printResults(results)

	st := c.Stats()
	for peer, s := range st.Peers {
		fmt.Printf("  %s: %d sent, %d retried, %d timed out, %d declined\n", peer, s.Sent, s.Retries, s.Timeouts, s.Errors)
	}
}
