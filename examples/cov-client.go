// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/spf13/cobra"
)

func init() {
	COVClient.Flags().Uint16Var(&covObjectType, "object-type", 0, "Object type to subscribe to.")
	COVClient.Flags().Uint32Var(&covInstanceId, "instance-id", 0, "Instance ID to subscribe to.")
	COVClient.Flags().Uint32Var(&covProcessId, "process-id", 0, "Subscriber process identifier; 0 is the client's default.")
	COVClient.Flags().DurationVar(&covLifetime, "lifetime", client.DefaultCOVLifetime, "How long the device keeps the subscription without a renewal; the client renews it at half of that.")
	COVClient.Flags().BoolVar(&covConfirmed, "confirmed", false, "Ask for confirmed notifications, which the client acknowledges.")
	COVClient.Flags().DurationVar(&covListen, "listen", 0, "How long to listen before cancelling; 0 listens until interrupted.")
}

var (
	covObjectType uint16
	covInstanceId uint32
	covProcessId  uint32
	covLifetime   time.Duration
	covConfirmed  bool
	covListen     time.Duration

	COVClient = &cobra.Command{
		Use:   "cov",
		Short: "Subscribe to an object's changes of value.",
		Long: "Subscribes to one object at --remote-address, or of the device on --dnet at\n" +
			"--dadr behind the router there, and prints the notifications the device\n" +
			"sends. The client renews the subscription halfway through its lifetime and\n" +
			"acknowledges confirmed notifications. After --listen, or when interrupted,\n" +
			"it cancels the subscription.",
		Args: argValidation,
		Run:  COVClientExample,
	}
)

func COVClientExample(cmd *cobra.Command, args []string) {
	c := openClient(nil)
	defer c.Close()

	obj := client.ObjectID{Type: covObjectType, Instance: covInstanceId}
	name := fmt.Sprintf("%s %d", objectTypeName(obj.Type), obj.Instance)
	stamp := func() string { return time.Now().Format("15:04:05.000") }

	sub, err := c.SubscribeCOV(context.Background(), target(), obj,
		client.COVOptions{Confirmed: covConfirmed, Lifetime: covLifetime, ProcessID: covProcessId},
		func(n client.COVNotification) {
			kind := "unconfirmed"
			if n.Confirmed {
				kind = "confirmed"
			}
			fmt.Printf("%s %s notification from device %d, process %d, %s remaining\n",
				stamp(), kind, n.Device, n.ProcessID, n.Remaining)
			for _, p := range n.Props {
				fmt.Printf("  %-28s %s\n", propertyName(p.ID), formatValues(p.Values))
			}
		})
	if err != nil {
		log.Fatalf("subscribing to %s: %v", name, err)
	}
	fmt.Printf("%s subscribed to %s of %v until %s\n", stamp(), name, target(), sub.Status().Expires.Format("15:04:05"))

	// Listen, saying so whenever the client has renewed the subscription.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if covListen > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, covListen)
		defer cancel()
	}
	renewed := sub.Status().Renewed
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for listening := true; listening; {
		select {
		case <-ctx.Done():
			listening = false
		case <-tick.C:
			if st := sub.Status(); st.Renewed.After(renewed) {
				renewed = st.Renewed
				fmt.Printf("%s renewed until %s\n", renewed.Format("15:04:05.000"), st.Expires.Format("15:04:05"))
			}
		}
	}

	st := sub.Status()
	if err := sub.Cancel(context.Background()); err != nil {
		log.Fatalf("cancelling the subscription to %s: %v", name, err)
	}
	fmt.Printf("%s cancelled after %d notifications", stamp(), st.Notifications)
	if st.Err != nil {
		fmt.Printf("; the last renewal failed: %v", st.Err)
	}
	fmt.Println()
}
