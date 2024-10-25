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
	ReadRangeClientCmd.Flags().Uint16Var(&rrObjectType, "object-type", 0, "Object type to read.")
	ReadRangeClientCmd.Flags().Uint32Var(&rrInstanceId, "instance-id", 0, "Instance ID to read.")  // Analog-input
	ReadRangeClientCmd.Flags().Uint8Var(&rrPropertyId, "property-id", 131, "Property ID to read.") // Current-value
	ReadRangeClientCmd.Flags().Uint16Var(&rrRangeStart, "range-start", 1, "Range start index.")
	ReadRangeClientCmd.Flags().Int32Var(&rrLength, "length", 50, "Length of results.")
	ReadRangeClientCmd.Flags().IntVar(&rrPeriod, "period", 1, "Period, in seconds, between requests.")
	ReadRangeClientCmd.Flags().IntVar(&rrN, "messages", 1, "Number of messages to send, being 0 unlimited.")
}

var (
	rrObjectType uint16
	rrInstanceId uint32
	rrPropertyId uint8
	rrRangeStart uint16
	rrLength     int32
	rrPeriod     int
	rrN          int

	ReadRangeClientCmd = &cobra.Command{
		Use:   "rrc",
		Short: "Send ReadRange requests.",
		Long:  "There's not much more really. This command sends a configurable ReadProperty request.",
		Args:  argValidation,
		Run:   ReadRangeClientExample,
	}
)

func ReadRangeClientExample(cmd *cobra.Command, args []string) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	mReadRange, err := bacnet.NewReadRange(rrObjectType, rrInstanceId, rrPropertyId, rrRangeStart, rrLength)
	if err != nil {
		log.Fatalf("error generating initial ReadProperty: %v\n", err)
	}

	replyRaw := make([]byte, 2048)
	sentRequests := 0
	for {
		listenConn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := listenConn.WriteTo(mReadRange, remoteUDPAddr); err != nil {
			log.Fatalf("Failed to write the request: %s\n", err)
		}

		log.Printf("sent: %x", mReadRange)

		nBytes, remoteAddr, err := listenConn.ReadFrom(replyRaw)
		if err != nil {
			log.Fatalf("error reading incoming packet: %v\n", err)
		}

		log.Printf("read %d bytes from %s: %x\n", nBytes, remoteAddr, replyRaw[:nBytes])

		serviceMsg, t, err := bacnet.Parse(replyRaw[:nBytes])
		if err != nil {
			log.Fatalf("error parsing the received message: %v\n", err)
		}

		// switch between recieved message type
		switch t {
		case plumbing.ComplexAck:
			cACKEnc, ok := serviceMsg.(*services.ComplexACK)
			if !ok {
				log.Fatalf("we didn't receive a CACK reply...\n")
			}
			log.Printf("unmarshalled BVLC: %#v\n", cACKEnc.BVLC)
			log.Printf("unmarshalled NPDU: %#v\n", cACKEnc.NPDU)

			decodedCACK, err := cACKEnc.Decode()
			if err != nil {
				log.Fatalf("couldn't decode the CACK reply: %v\n", err)
			}
			printCACK(&decodedCACK)

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

		if sentRequests == rpN {
			break
		}

		time.Sleep(time.Duration(rpPeriod) * time.Second)
	}
}
