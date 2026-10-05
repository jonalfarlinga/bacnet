package main

import (
	"log"
	"net"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/spf13/cobra"
)

var (
	ReadPropertyServerCmd = &cobra.Command{
		Use:   "rps",
		Short: "Reply ReadProperty requests with Complex ACKs.",
		Long: "This example will wait until it receives a ReadProperty request. Upon reception\n" +
			"it'll just reply with the configured Complex ACK fields",
		Args: argValidation,
		Run:  ReadPropertyServerExample,
	}
)

func ReadPropertyServerExample(cmd *cobra.Command, args []string) {
	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v\n", err)
	}
	defer listenConn.Close()

	// Present values of analog-output 0 and 1.
	values := []float32{1.1, 2.2}

	reqRaw := make([]byte, 1024)

	var nBytes int
	var remoteAddr net.Addr
	for {
		nBytes, remoteAddr, err = listenConn.ReadFrom(reqRaw)
		if err != nil {
			log.Fatalf("error reading incoming packet: %v\n", err)
		}

		log.Printf("read %d bytes from %s: %x\n", nBytes, remoteAddr, reqRaw[:nBytes])

		err = DecodeAndDispatch(reqRaw[:nBytes], MessageDispatcher{
			OnConfirmedReadProperty: func(readPropertyMessage *services.ConfirmedReadProperty) error {
				log.Printf("received a ReadProperty request!\n")

				decodedReadPropertyMessage, err := readPropertyMessage.Decode()
				if err != nil {
					return err
				}

				log.Printf("decoded ReadProperty message:\n\tObjectType: %d\n\tInstance ID: %d\n\tProperty ID: %d\n",
					decodedReadPropertyMessage.ObjectType, decodedReadPropertyMessage.InstanceNum,
					decodedReadPropertyMessage.PropertyId)

				// A reply echoes the invoke ID of the request it answers.
				invokeID := readPropertyMessage.APDU.InvokeID

				if decodedReadPropertyMessage.InstanceNum >= uint32(len(values)) {
					bErr, err := bacnet.NewErrorReply(invokeID, services.ServiceConfirmedReadProperty, objects.ErrorClassObject, objects.ErrorCodeUnknownObject)
					if err != nil {
						return err
					}
					if _, err := listenConn.WriteTo(bErr, remoteAddr); err != nil {
						return err
					}
					log.Printf("we were asked for a wrong instance ID!\n")
					return nil
				}

				mCACK, err := bacnet.NewCACKReply(
					invokeID,
					services.ServiceConfirmedReadProperty,
					objects.ObjectTypeAnalogOutput,
					decodedReadPropertyMessage.InstanceNum,
					objects.PropertyIdPresentValue,
					values[decodedReadPropertyMessage.InstanceNum],
				)
				if err != nil {
					return err
				}
				if _, err := listenConn.WriteTo(mCACK, remoteAddr); err != nil {
					return err
				}

				log.Printf("replied with our CACK!\n")
				return nil
			},
			OnUnknown: func(msg plumbing.BACnet) error {
				log.Printf("ignoring non-ReadProperty message type=%d service=%d", msg.GetType(), msg.GetService())
				return nil
			},
		})
		if err != nil {
			log.Printf("failed to handle inbound packet from %s: %v", remoteAddr, err)
		}
	}
}
