// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/spf13/cobra"
)

func init() {
	ReadPropertyClientCmd.Flags().Uint16Var(&rpObjectType, "object-type", 0, "Object type to read.")
	ReadPropertyClientCmd.Flags().Uint32Var(&rpInstanceId, "instance-id", 0, "Instance ID to read.")  // Analog-input
	ReadPropertyClientCmd.Flags().Uint16Var(&rpPropertyId, "property-id", 85, "Property ID to read.") // Present-value
	ReadPropertyClientCmd.Flags().Int64Var(&rpIndex, "index", -1, "Array index to read; -1 reads the whole property.")
}

var (
	rpObjectType uint16
	rpInstanceId uint32
	rpPropertyId uint16
	rpIndex      int64

	ReadPropertyClientCmd = &cobra.Command{
		Use:   "rpc",
		Short: "Send a ReadProperty request.",
		Long: "Reads one property of one object at --remote-address, or of the device on\n" +
			"--dnet at --dadr behind the router there, and prints the value. The client\n" +
			"matches the reply to the request, retries it, and reassembles it when the\n" +
			"device segments it.",
		Args: argValidation,
		Run:  ReadPropertyClientExample,
	}
)

func ReadPropertyClientExample(cmd *cobra.Command, args []string) {
	c := openClient(nil)
	defer c.Close()

	obj := client.ObjectID{Type: rpObjectType, Instance: rpInstanceId}
	var (
		values []client.Value
		err    error
	)
	if rpIndex < 0 {
		values, err = c.ReadProperty(context.Background(), target(), obj, rpPropertyId)
	} else {
		values, err = c.ReadPropertyIndex(context.Background(), target(), obj, rpPropertyId, uint32(rpIndex))
	}
	if err != nil {
		log.Fatalf("reading %s of %s %d: %v", propertyName(uint32(rpPropertyId)), objectTypeName(obj.Type), obj.Instance, err)
	}
	fmt.Printf("%s %d %s: %s\n", objectTypeName(obj.Type), obj.Instance, propertyName(uint32(rpPropertyId)), formatValues(values))
}
