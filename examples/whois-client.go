// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/spf13/cobra"
)

func init() {
	whoIsCmd.Flags().Uint32Var(&wiLow, "low", 0, "Lowest device instance to ask for.")
	whoIsCmd.Flags().Uint32Var(&wiHigh, "high", 0, "Highest device instance to ask for; 0 with --low 0 asks every device.")
	whoIsCmd.Flags().DurationVar(&wiWait, "wait", 3*time.Second, "How long to collect I-Ams.")
}

var (
	wiLow  uint32
	wiHigh uint32
	wiWait time.Duration

	whoIsCmd = &cobra.Command{
		Use:   "whois",
		Short: "Send a Who-Is and list the devices that answer.",
		Long: "Sends one Who-Is, addressed to every network so that devices behind routers\n" +
			"answer too, and lists the devices by instance as their I-Ams arrive. With\n" +
			"--interface the Who-Is goes to the adapter's subnet broadcast; without it, to\n" +
			"--remote-address. With --bbmd it goes through the BBMD, as a foreign device.",
		Args: argValidation,
		Run:  whoIsExample,
	}
)

func whoIsExample(cmd *cobra.Command, args []string) {
	var (
		mu    sync.Mutex
		found = map[uint32]client.Device{}
	)
	c := openClient(func(d client.Device) {
		mu.Lock()
		found[d.Instance] = d
		mu.Unlock()
	})
	defer c.Close()

	var err error
	if wiLow == 0 && wiHigh == 0 {
		err = c.WhoIs()
	} else {
		err = c.WhoIsRange(wiLow, wiHigh)
	}
	if err != nil {
		log.Fatalf("sending the Who-Is: %v", err)
	}
	time.Sleep(wiWait)

	mu.Lock()
	defer mu.Unlock()
	instances := make([]uint32, 0, len(found))
	for i := range found {
		instances = append(instances, i)
	}
	sort.Slice(instances, func(i, j int) bool { return instances[i] < instances[j] })
	fmt.Printf("%d device(s) answered in %s\n", len(found), wiWait)
	for _, i := range instances {
		d := found[i]
		fmt.Printf("  device %-8d at %-40v max APDU %-5d segmentation %d vendor %d\n",
			d.Instance, d.Addr, d.MaxAPDU, d.Segmentation, d.VendorID)
	}
}
