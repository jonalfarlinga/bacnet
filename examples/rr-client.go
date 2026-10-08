// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

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
	ReadRangeClientCmd.Flags().Uint16Var(&rrObjectType, "object-type", objects.ObjectTypeTrendLog, "Object type to read.")
	ReadRangeClientCmd.Flags().Uint32Var(&rrInstanceId, "instance-id", 0, "Instance ID to read.")
	ReadRangeClientCmd.Flags().Uint16Var(&rrPropertyId, "property-id", objects.PropertyIdLogBuffer, "Property ID to read.")
	ReadRangeClientCmd.Flags().Uint32Var(&rrRangeStart, "range-start", 1, "Position, or with --by-sequence the sequence number, to count from.")
	ReadRangeClientCmd.Flags().Int32Var(&rrLength, "length", 50, "How many items; negative counts backwards from --range-start. 0 asks for the whole list.")
	ReadRangeClientCmd.Flags().BoolVar(&rrBySequence, "by-sequence", false, "Count from a sequence number rather than a position.")
}

var (
	rrObjectType uint16
	rrInstanceId uint32
	rrPropertyId uint16
	rrRangeStart uint32
	rrLength     int32
	rrBySequence bool

	ReadRangeClientCmd = &cobra.Command{
		Use:   "rrc",
		Short: "Send a ReadRange request.",
		Long: "Reads part of a list property, a trend log's log-buffer unless told otherwise,\n" +
			"and prints the records: when each was logged, what, and its status flags.",
		Args: argValidation,
		Run:  ReadRangeClientExample,
	}
)

func ReadRangeClientExample(cmd *cobra.Command, args []string) {
	c := openClient(nil)
	defer c.Close()

	req := client.ReadRangeRequest{
		Object:   client.ObjectID{Type: rrObjectType, Instance: rrInstanceId},
		Property: rrPropertyId,
	}
	switch {
	case rrLength == 0:
	case rrBySequence:
		req.Range = client.BySequence(rrRangeStart, rrLength)
	default:
		req.Range = client.ByPosition(rrRangeStart, rrLength)
	}
	res, err := c.ReadRange(context.Background(), target(), req)
	if err != nil {
		log.Fatalf("reading: %v", err)
	}

	fmt.Printf("%s %d %s: %d item(s), first %t, last %t, more %t",
		objectTypeName(res.Object.Type), res.Object.Instance, propertyName(res.Property),
		res.ItemCount, res.FirstItem, res.LastItem, res.MoreItems)
	if res.FirstSequence != nil {
		fmt.Printf(", from sequence number %d", *res.FirstSequence)
	}
	fmt.Println()
	records := res.LogRecords()
	if len(records) == 0 {
		fmt.Printf("  items: %s\n", formatValues(res.Items))
		return
	}
	for _, r := range records {
		at := "wildcard time"
		if !r.At.IsZero() {
			at = r.At.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("  %s  %-24s %s\n", at, formatValue(r.Datum), formatValue(client.Value{Kind: client.KindBits, Bits: r.Status}))
	}
}
