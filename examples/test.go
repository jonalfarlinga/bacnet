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
	Testclient.Flags().Uint16Var(&testObjectType, "object-type", 0, "Object type to read.")
	Testclient.Flags().Uint32Var(&testInstanceId, "instance-id", 0, "Instance ID to read.")  // Analog-input
	Testclient.Flags().Uint16Var(&testPropertyId, "property-id", 85, "Property ID to read.") // Cutestent-value
	Testclient.Flags().IntVar(&testPeriod, "period", 1, "Period, in seconds, between requests.")
	Testclient.Flags().IntVar(&testN, "messages", 1, "Number of messages to send, being 0 unlimited.")
}

var (
	testObjectType uint16
	testInstanceId uint32
	testPropertyId uint16
	testPeriod     int
	testN          int

	Testclient = &cobra.Command{
		Use:   "testc",
		Short: "Send ReadRange requests.",
		Long:  "There's not much more really. This command sends a configurable ReadProperty request.",
		Args:  argValidation,
		Run:   TestClientExample,
	}
)

func TestClientExample(cmd *cobra.Command, args []string) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	// mReadRange, err := bacnet.NewReadRange(testObjectType, rrInstanceId, rrPropertyId, rrRangeStart, rrLength)
	// if err != nil {
	// 	log.Fatalf("error generating initial ReadProperty: %v\n", err)
	// }
	message := []byte{
		0x81, 0x0A, 0x00, 0x15, // BVLC length 21
		0x01, 0x04, // NPDU version 1, control 4
		0x00, 0x00, 0x01, 0x05, // APDU type 0, service 5
		0x09, 0x01, 0x1C, 0x00, 0x00, 0x00, 0x00, 0x29, 0x00, 0x39, 0xF0, // Subscription request processId 1 - objId 0, 0 - TRUE - life 4 min
	}

	replyRaw := make([]byte, 2048)
	sentRequests := 0
	for {
		listenConn.SetDeadline(time.Now().Add(5 * time.Second))
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

			serviceMsg, t, err := bacnet.Parse(replyRaw[:nBytes])
			if err != nil {
				log.Fatalf("error parsing the received message: %v\n", err)
			}

			// switch between recieved message type
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

			case plumbing.ComplexAck:
				cACK, ok := serviceMsg.(*services.ComplexACK)
				if !ok {
					log.Fatalf("we didn't receive a CACK reply...\n")
				}
				logBuffer := services.NewLogBufferCACK(cACK)
				log.Printf("unmarshalled BVLC: %#v\n", logBuffer.BVLC)
				log.Printf("unmarshalled NPDU: %#v\n", logBuffer.NPDU)

				decodedLogBuffer, err := logBuffer.Decode()
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

		if sentRequests == testN {
			break
		}

		time.Sleep(time.Duration(testPeriod) * time.Second)
	}
}
