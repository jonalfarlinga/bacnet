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
	COVClient.Flags().Uint16Var(&covObjectType, "object-type", 0, "Object type to read.")
	COVClient.Flags().Uint32Var(&covInstanceId, "instance-id", 0, "Instance ID to read.") // Analog-input
	COVClient.Flags().UintVar(&covProcessId, "process-id", 85, "Property ID to read.")    // Cucovent-value
	COVClient.Flags().UintVar(&covLifetime, "lifetime", 240, "Lifetime of subscription in minutes.")
	COVClient.Flags().BoolVar(&covExpectConf, "expect-confirmed", true, "Expect a confirmed notification.")
	COVClient.Flags().IntVar(&covPeriod, "period", 1, "Period, in seconds, between requests.")
	COVClient.Flags().IntVar(&covN, "messages", 1, "Number of messages to send, being 0 unlimited.")
	COVClient.Flags().BoolVar(&covCancellation, "cancel", false, "Cancel the subscription.")
}

var (
	covObjectType   uint16
	covInstanceId   uint32
	covProcessId    uint
	covPeriod       int
	covN            int
	covCancellation bool
	covLifetime     uint
	covExpectConf   bool

	COVClient = &cobra.Command{
		Use:   "cov",
		Short: "Send ReadRange requests.",
		Long:  "There's not much more really. This command sends a configurable ReadProperty request.",
		Args:  argValidation,
		Run:   COVClientExample,
	}
)

func COVClientExample(cmd *cobra.Command, args []string) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	message, err := bacnet.NewSubscribeCOV(covObjectType, covInstanceId, covProcessId, covLifetime, covExpectConf, covCancellation)
	if err != nil {
		log.Fatalf("error generating initial ReadProperty: %v\n", err)
	}

	replyRaw := make([]byte, 2048)
	sentRequests := 0
	for {
		if _, err := listenConn.WriteTo(message, remoteUDPAddr); err != nil {
			log.Fatalf("Failed to write the request: %s\n", err)
		}

		log.Printf("sent: %x", message)
		always := true
		for always {
			listenConn.SetDeadline(time.Now().Add(1 * time.Second))
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
			case plumbing.UnConfirmedReq:
				unConf, ok := serviceMsg.(*services.UnconfirmedCOVNotification)
				if !ok {
					log.Fatalf("we didn't receive an UnconfirmedCOVNotification reply...\n")
				}
				log.Printf("unmarshalled BVLC: %#v\n", unConf.BVLC)
				log.Printf("unmarshalled NPDU: %#v\n", unConf.NPDU)

				decodedUnConf, err := unConf.Decode()
				if err != nil {
					log.Fatalf("couldn't decode the UnconfirmedCOVNotification reply: %v\n", err)
				}
				printCOVNot(&decodedUnConf)
			case plumbing.SimpleAck:
				sACK, ok := serviceMsg.(*services.SimpleACK)
				if !ok {
					log.Fatalf("we didn't receive a SACK reply...\n")
				}
				log.Printf("unmarshalled BVLC: %#v\n", sACK.BVLC)
				log.Printf("unmarshalled NPDU: %#v\n", sACK.NPDU)
				log.Printf("decoded SimpleACK reply:\n\n\tService Choice: %d\n\tInvoke ID: %d\n\n", sACK.APDU.Service, sACK.APDU.InvokeID)
			case plumbing.ComplexAck:
				cACK, ok := serviceMsg.(*services.ComplexACK)
				if !ok {
					log.Fatalf("we didn't receive a CACK reply...\n")
				}
				// logBuffer := services.NewLogBufferCACK(cACK)
				log.Printf("unmarshalled BVLC: %#v\n", cACK.BVLC)
				log.Printf("unmarshalled NPDU: %#v\n", cACK.NPDU)

				decodedLogBuffer, err := cACK.DecodeRR()
				if err != nil {
					log.Fatalf("couldn't decode the LogBuffer reply: %v\n", err)
				}
				printLogBuffer(&decodedLogBuffer)
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
		}

		sentRequests++

		if sentRequests == covN {
			break
		}

		time.Sleep(time.Duration(covPeriod) * time.Second)
	}
}
