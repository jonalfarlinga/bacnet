package main

import (
	"fmt"
	"log"
	"net"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/spf13/cobra"
)

var (
	IAmCmd = &cobra.Command{
		Use:   "iam",
		Short: "Send IAm requests.",
		Long: "This example will wait until it receives a WhoIs request. Upon reception\n" +
			"it'll just reply with the configured IAm fields",
		Args: argValidation,
		Run:  IAmExample,
	}
)

func IAmExample(cmd *cobra.Command, args []string) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %s", err)
	}

	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("couldn't get interface information: %v\n", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	mIAm, err := bacnet.NewIAm(321, 31)
	if err != nil {
		log.Fatalf("error generating initial IAm: %v\n", err)
	}

	reqRaw := make([]byte, 1024)

	var nBytes int
	var remoteAddr net.Addr
	for {
		nBytes, remoteAddr, err = listenConn.ReadFrom(reqRaw)
		if err != nil {
			log.Fatalf("error reading incoming packet: %v\n", err)
		}

		if common.IsLocalAddr(ifaceAddrs, remoteAddr) {
			log.Printf("got our own broadcast, back to listening...\n")
			continue
		}

		log.Printf("read %d bytes from %s: %x\n", nBytes, remoteAddr, reqRaw[:nBytes])

		err = DecodeAndDispatch(reqRaw[:nBytes], MessageDispatcher{
			OnUnconfirmedWhoIs: func(whoIsMessage *services.UnconfirmedWhoIs) error {
				log.Printf("received a WhoIs request!\n")
				if _, err := whoIsMessage.Decode(); err != nil {
					return err
				}

				if _, err := listenConn.WriteTo(mIAm, remoteUDPAddr); err != nil {
					return err
				}
				return nil
			},
			OnConfirmedReadPropertyMultiple: func(readPropertyMessage *services.ConfirmedReadProperty) error {
				log.Printf("received a ReadPropertyMultiple request!\n")
				decodedReadPropertyMessage, err := readPropertyMessage.DecodeRPM()
				if err != nil {
					return err
				}
				out := "Decoded ReadPropertyMultiple message:\n"
				out += fmt.Sprintf(
					"\n\tObject Type: %d\n\tInstance Id: %d\n",
					decodedReadPropertyMessage.ObjectType, decodedReadPropertyMessage.InstanceNum,
				)
				for i, t := range decodedReadPropertyMessage.Tags {
					out += fmt.Sprintf(
						"\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %+v\n\t\tData Length: %d\n",
						i, objects.TagMap[t.TagNumber], t.Value, t.Length,
					)
				}
				log.Print(out)
				return nil
			},
			OnUnknown: func(msg plumbing.BACnet) error {
				log.Printf("ignoring message type=%d service=%d", msg.GetType(), msg.GetService())
				return nil
			},
		})
		if err != nil {
			log.Printf("failed to handle inbound packet from %s: %v", remoteAddr, err)
		}
	}
}
