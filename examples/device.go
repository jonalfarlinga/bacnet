package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/spf13/cobra"
)

func init() {
	deviceCmd.Flags().Uint32Var(&devInstance, "instance", 0, "Device instance to answer as.")
	deviceCmd.Flags().Uint16Var(&devVendor, "vendor", 0, "Vendor identifier to state in the I-Am.")
	deviceCmd.Flags().BoolVar(&devAnnounce, "announce", false, "Broadcast one I-Am on starting, without waiting to be asked.")
	deviceCmd.Flags().StringVar(&devName, "name", "bacnet-examples", "Object name of the Device object.")
	deviceCmd.MarkFlagRequired("instance")
}

var (
	devInstance uint32
	devVendor   uint16
	devAnnounce bool
	devName     string

	deviceCmd = &cobra.Command{
		Use:   "device",
		Short: "Listen for Who-Is and answer as a device, until interrupted.",
		Long: "Gives the client an identity, device --instance, and lists every Who-Is it\n" +
			"hears. One that asks for the instance is answered with an I-Am: a broadcast\n" +
			"with a broadcast on --interface's subnet, a unicast with a unicast. Reads of\n" +
			"its Device object, the only object it has, are answered and listed too; any\n" +
			"other service is rejected.",
		Args: argValidation,
		Run:  deviceExample,
	}
)

func deviceExample(cmd *cobra.Command, args []string) {
	c := openClientWith(nil, func(cfg *client.Config) {
		cfg.Identity = &client.Identity{Instance: devInstance, VendorID: devVendor}
		cfg.OnWhoIs = func(w client.WhoIsRequest) {
			asked := "every device"
			if w.Ranged {
				asked = fmt.Sprintf("devices %d to %d", w.Low, w.High)
			}
			answer := "not answered"
			if w.Asks(devInstance) {
				answer = fmt.Sprintf("answering as device %d", devInstance)
			}
			fmt.Printf("%s  Who-Is from %v for %s: %s\n", time.Now().Format("15:04:05.000"), w.From, asked, answer)
		}
	})
	defer c.Close()
	serveDevice(c, devInstance, devVendor, devName)

	fmt.Printf("listening on %v as device %d; Ctrl-C to stop\n", c.Local(), devInstance)
	if devAnnounce {
		if err := c.IAm(); err != nil {
			log.Fatalf("sending the I-Am: %v", err)
		}
		fmt.Println("I-Am broadcast")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
