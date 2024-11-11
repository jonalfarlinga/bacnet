// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"log"
	"net"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/spf13/cobra"
)

func init() {
	whoIsCmdBroad.Flags().StringVar(&bacPort, "bacnet-port", ":47808", "broadcast port to bind to.")
	whoIsCmdBroad.Flags().IntVar(&nWhoB, "messages", 1, "Number of messages to send, being 0 unlimited.")
}

var (
	nWhoB         int
	bacPort         string
	whoIsCmdBroad = &cobra.Command{
		Use:   "whoisb",
		Short: "Send WhoIs requests.",
		Long: "There's not much more really. This command sends a configurable number of\n" +
			"WhoIs requests with a configurable period. That's pretty much it.",
		Args: argValidation,
		Run:  whoIsBroadExample,
	}
)

func whoIsBroadExample(cmd *cobra.Command, args []string) {
	localUDPAddr, err := net.ResolveUDPAddr("udp", bacPort)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	remoteUDPAddr, err := net.ResolveUDPAddr("udp", bAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("couldn't get interface information: %v\n", err)
	}

	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer conn.Close()

	log.Printf("Sending WhoIs requests on %s\n", bacPort)

	replyRaw := make([]byte, 1024)
	for {
		conn.SetDeadline(time.Now().Add(1 * time.Second))
		go broadcastMessage(conn, remoteUDPAddr)

		var nBytes int
		var remoteAddr net.Addr
		for {
			nBytes, remoteAddr, err = conn.ReadFrom(replyRaw)
			if err != nil {
				log.Fatalf("error reading incoming packet: %v\n", err)
			}
			if !common.IsLocalAddr(ifaceAddrs, remoteAddr) {
				break
			}
			log.Printf("got our own broadcast, back to listening...\n")
			continue
		}

		log.Printf("read %d bytes from %s: %x\n", nBytes, remoteAddr, replyRaw[:nBytes])

		serviceMsg, err := bacnet.Parse(replyRaw[:nBytes])
		if err != nil {
			log.Fatalf("error parsing the received message: %v\n", err)
		}

		// switch between recieved messages
		t := serviceMsg.GetType()
		switch t {

		}
		iAmMessage, ok := serviceMsg.(*services.UnconfirmedIAm)
		if !ok {
			log.Fatalf("we didn't receive an IAm reply...\n")
		}

		log.Printf("unmarshalled BVLC: %#v\n", iAmMessage.BVLC)
		log.Printf("unmarshalled NPDU: %#v\n", iAmMessage.NPDU)

		decodedIAm, err := iAmMessage.Decode()
		if err != nil {
			log.Fatalf("couldn't decode the IAm reply: %v\n", err)
		}

		printIAm(&decodedIAm)

		time.Sleep(time.Duration(wiPeriod) * time.Second)
	}
}

func broadcastMessage(conn *net.UDPConn, addr *net.UDPAddr) {
	mWhoIs, err := bacnet.NewWhois()
	if err != nil {
		log.Fatalf("error generating initial WhoIs: %v\n", err)
	}
	log.Println("Broadcasting on ", addr)
	_, err = conn.WriteToUDP(mWhoIs, addr)
	if err != nil {
		log.Fatalf("Failed to write the request: %s\n", err)
	}

	log.Printf("sent: %x\n", mWhoIs)
}
