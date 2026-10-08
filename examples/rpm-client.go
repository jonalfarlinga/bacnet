// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/spf13/cobra"
)

func init() {
	ReadPropertyMultipleClientCmd.Flags().Uint16Var(&rmObjectType, "object-type", 0, "Object type to read.")
	ReadPropertyMultipleClientCmd.Flags().Uint32Var(&rmInstanceId, "instance-id", 0, "Instance ID to read.")
	ReadPropertyMultipleClientCmd.Flags().IntSliceVar(&rmProperties, "properties", []int{77, 85}, "Properties to read.")
}

var (
	rmObjectType uint16
	rmInstanceId uint32
	rmProperties []int

	ReadPropertyMultipleClientCmd = &cobra.Command{
		Use:   "rpm",
		Short: "Send a ReadPropertyMultiple request.",
		Long: "Reads several properties of one object in one request and prints each value,\n" +
			"or the error the device gave for it. A device without ReadPropertyMultiple is\n" +
			"read one property at a time instead.",
		Args: argValidation,
		Run:  ReadPropertyMultipleClientExample,
	}
)

func ReadPropertyMultipleClientExample(cmd *cobra.Command, args []string) {
	c := openClient(nil)
	defer c.Close()

	// ReadMany sizes the request to the device and falls back to
	// ReadProperty where it has to, so it wants to know the device.
	dev, err := c.Probe(context.Background(), target(), 0)
	if err != nil {
		log.Fatalf("learning about the device: %v", err)
	}
	results, err := c.ReadMany(context.Background(), dev, []client.ReadSpec{{
		Object: client.ObjectID{Type: rmObjectType, Instance: rmInstanceId},
		Props:  uint16slice(rmProperties),
	}})
	if err != nil {
		log.Fatalf("reading: %v", err)
	}
	printResults(results)
}
