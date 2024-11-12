// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"log"
	"net"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/spf13/cobra"
)

func init() {
	ReadPropertyMultipleClientCmd.Flags().Uint16Var(&rmObjectType, "object-type", 0, "Object type to read.")
	ReadPropertyMultipleClientCmd.Flags().Uint32Var(&rmInstanceId, "instance-id", 0, "Instance ID to read.")  // Analog-input
	ReadPropertyMultipleClientCmd.Flags().Uint16Var(&rmPropertyId, "property-id", 85, "Property ID to read.") // Current-value
	ReadPropertyMultipleClientCmd.Flags().IntSliceVar(&rmProperties, "properties", []int{8}, "Properties to read.")
	ReadPropertyMultipleClientCmd.Flags().IntVar(&rmPeriod, "period", 1, "Period, in seconds, between requests.")
	ReadPropertyMultipleClientCmd.Flags().IntVar(&rmN, "messages", 1, "Number of messages to send, being 0 unlimited.")
}

var (
	rmObjectType uint16
	rmInstanceId uint32
	rmPropertyId uint16
	rmProperties []int
	rmPeriod     int
	rmN          int

	ReadPropertyMultipleClientCmd = &cobra.Command{
		Use:   "rpm",
		Short: "Send ReadProperty requests.",
		Long:  "There's not much more really. This command sends a configurable ReadProperty request.",
		Args:  argValidation,
		Run:   ReadPropertyMultipleClientExample,
	}
)

func ReadPropertyMultipleClientExample(cmd *cobra.Command, args []string) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	mReadProperty, err := bacnet.NewReadPropertyMultiple(rmObjectType, rmInstanceId, uint16slice(rmProperties))
	if err != nil {
		log.Fatalf("error generating initial ReadProperty: %v\n", err)
	}

	replyRaw := make([]byte, 1024)
	sentRequests := 0
	for {
		listenConn.SetDeadline(time.Now().Add(1 * time.Second))
		if _, err := listenConn.WriteTo(mReadProperty, remoteUDPAddr); err != nil {
			log.Fatalf("Failed to write the request: %s\n", err)
		}

		log.Printf("sent: %x", mReadProperty)

		nBytes, remoteAddr, err := listenConn.ReadFrom(replyRaw)
		if err != nil {
			log.Fatalf("error reading incoming packet: %v\n", err)
		}

		log.Printf("read %d bytes from %s: %x\n", nBytes, remoteAddr, replyRaw[:nBytes])

		serviceMsg, err := bacnet.Parse(replyRaw[:nBytes])
		if err != nil {
			log.Fatalf("error parsing the received message: %v\n", err)
		}

		// switch between recieved message type
		t := serviceMsg.GetType()
		switch t {
		case plumbing.ComplexAck:
			cACKEnc, ok := serviceMsg.(*services.ComplexACK)
			if !ok {
				log.Fatalf("we didn't receive a CACK reply...\n")
			}
			// multiPropCACK := services.NewComplexACKRPM(cACKEnc)
			log.Printf("unmarshalled BVLC: %#v\n", cACKEnc.BVLC)
			log.Printf("unmarshalled NPDU: %#v\n", cACKEnc.NPDU)

			decodedCACK, err := cACKEnc.DecodeRPM()
			if err != nil {
				log.Fatalf("couldn't decode the CACK reply: %v\n", err)
			}
			printPropM(&decodedCACK)

		case plumbing.Error:
			errEnc, ok := serviceMsg.(*services.Error)
			if !ok {
				log.Fatalf("we didn't receive an Error reply...\n")
			}
			log.Printf("unmarshalled BVLC: %#v\n", errEnc.BVLC)
			log.Printf("unmarshalled NPDU: %#v\n", errEnc.NPDU)

			decodedErr, err := errEnc.Decode()
			if err != nil {
				log.Fatalf("couldn't decode the Error reply: %v\n", err)
			}
			log.Printf("decoded Error reply:\n\tError Class: %d\n\tError Code: %d\n",
				decodedErr.ErrorClass, decodedErr.ErrorCode,
			)
		}

		sentRequests++

		if sentRequests == rmN {
			break
		}

		time.Sleep(time.Duration(rmPeriod) * time.Second)
	}
}

func uint16slice(s []int) []uint16 {
	u := make([]uint16, len(s))
	for i, v := range s {
		u[i] = uint16(v)
	}
	return u
}
