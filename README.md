# BACnet

A BACnet/IP implementation in pure Go: a codec for the protocol's layers and services, and a client that
reads devices on the local subnet, behind BACnet routers, and on other IP subnets through a BBMD.

Forked from [@ulbios's](https://github.com/ulbios) repository. That work was initially based on
[@kazukiigeta's](https://github.com/kazukiigeta) work available on the
[kazukiigeta/bacnet](https://github.com/kazukiigeta/bacnet) repository.

Please bear in mind we **only** support IP as the BACnet transport layer for now.

    go get github.com/jonalfarlinga/bacnet

## Packages

The datagrams of the services are built with the `New*()` functions defined on `encoding.go` and parsed
with the `Parse()` function defined on `parsing.go`. The pieces they are built from are broken up in
several directories:

| Package | What is in it |
|---|---|
| `plumbing/` | BVLC, NPDU and APDU marshalling and unmarshalling: the BACnet/IP functions, a BBMD's among them, the routing fields and the network-layer messages, and the segmentation fields |
| `objects/` | The tags and the primitive values they carry, and the names of object types, properties and engineering units |
| `services/` | The services as messages: Who-Is and I-Am, Who-Has, ReadProperty, ReadPropertyMultiple, ReadRange, WriteProperty, SubscribeCOV and its notifications, and the replies: SimpleACK, ComplexACK, SegmentACK, Error, Reject and Abort |
| `common/` | Errors and utilities used across all the above |
| `client/` | A BACnet/IP client: one UDP socket, request and reply matching with timeouts and retries, segmented replies, the routing table, foreign-device registration with a BBMD, COV subscriptions (`SubscribeCOV`: renewed halfway through their lifetime, confirmed notifications acknowledged), `WriteProperty` to devices switched to writable, and the guard on what it sends, which `ReadOnly` narrows to reads |
| `client/sim/` | Simulated BACnet/IP devices on the loopback interface, which the client's tests run against, and which a program built on the client can be tried on without a network |

[docs/library-upgrade-spec.md](docs/library-upgrade-spec.md) is the design of the client and of the codec
changes made for it, with the status of each part. [CHANGELOG.md](CHANGELOG.md) lists what each release
changed.

## Using the client

The client reaches a device directly, behind a router, or through a BBMD, and the `Address` an I-Am gives
says which. This finds the devices on an adapter's subnet and reads each one's name:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
)

func main() {
	// Listen on an adapter, by name or IPv4 address, and broadcast to its subnet.
	ifc, err := client.FindInterface("Ethernet 3")
	if err != nil {
		log.Fatal(err)
	}
	cfg := ifc.Config(client.DefaultPort)
	cfg.ReadOnly = true // refuse SubscribeCOV as well: leave nothing on any device

	// I-Ams arrive on the client's read loop; the callback must not block.
	var mu sync.Mutex
	var devices []client.Device
	cfg.OnIAm = func(d client.Device) {
		mu.Lock()
		devices = append(devices, d)
		mu.Unlock()
	}

	c, err := client.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// One Who-Is, addressed to every network, then a moment for the answers.
	if err := c.WhoIs(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	mu.Lock()
	found := devices
	mu.Unlock()
	for _, d := range found {
		obj := client.ObjectID{Type: objects.ObjectTypeDevice, Instance: d.Instance}
		values, err := c.ReadProperty(ctx, d.Addr, obj, objects.PropertyIdObjectName)
		if err != nil || len(values) == 0 {
			fmt.Printf("device %d at %s: %v\n", d.Instance, d.Addr, err)
			continue
		}
		fmt.Printf("device %d at %s is %q\n", d.Instance, d.Addr, values[0].Str)
	}
}
```

`ReadMany` reads several properties of several objects in one ReadPropertyMultiple, split to what the device
accepts, and falls back to ReadProperty for a device without it; `ReadObjectList` reads a device's object
list, one entry at a time when the device will not segment; `ReadRange` reads a trend log's records; and
`SubscribeCOV` asks a device to report an object's changes. The errors a device answers with are typed:
`BACnetError`, `RejectError`, `AbortError`, and `ResponseTooLargeError` for a reply with more segments than
`Config.MaxSegments` allows.

The client writes only where it is told to. Everything it sends goes through one function that refuses any
service but the reads, the Segment-ACKs and Aborts a segmented reply needs, Who-Is-Router-To-Network, the
BBMD functions, SubscribeCOV with the acknowledgement of a confirmed notification, and `WriteProperty` to a
device switched to writable with `Config.Writable` or `SetWritable`; every device starts read-only, and
`ReadOnly` refuses the subscriptions and the writes as well. A write is sent once and never resent: it
returns `ErrTimeout` when it was sent and unanswered, and an error wrapping `ErrNotSent` when it never left
the client. It has no device identity: it never sends an I-Am and answers nothing but the notifications of
its own subscriptions. The builders for the other write services, such as `NewWriteProperty`, stay in the
root package for programs that send them on a socket of their own.

## The examples

On top of the BACnet implementation, we also offer a CLI-based program offering a way to test every
available service. All the sources are contained on `examples/`, which is its own Go module, so that its
command-line dependency is not one of the library's: it builds against the library in the parent directory.
The binary can be generated with:

    # Getting to the directory.
    $ cd examples

    # Time to build it!
    $ go build

As always, you can alter the generated binary with environment flags and the like. We also provide a `Taskfile`
you can leverage with `task`:

    # Getting to the directory.
    $ cd examples/

    # Compile for Linux-based distros.
    $ task

    # Compile for macOS.
    $ task build-mac

You can find more information on Task on its [official site](https://taskfile.dev). Once compiled, just invoke the
binary with no arguments to get an idea of what it can do. The `whois`, `rpc`, `rpm`, `rrc`, `cov` and
`probe` commands are built on the `client` package and are the reference for using it; the servers and the
`orchestrator` work on a raw socket with the root package's builders and `Parse()`.

`scripts/readonly_check.go` runs read-only checks against one live device and reports what the library can
do with it, from the Device object's properties to segmented replies, routers and BBMD tables:

    $ go run scripts/readonly_check.go -remote 192.168.1.100

## The explorer

The web UI for browsing a BACnet network that was developed in this repository lives in
[jonalfarlinga/bacnet-explorer](https://github.com/jonalfarlinga/bacnet-explorer) since 2026-10-14. It is
built on `client` and `client/sim`, and is the client's first user.

## Running the examples against bacnet-stack
In order to check our implementation is compliant, we interact with the server implementation offered by the
[`bacnet-stack/bacnet-stack`](https://github.com/bacnet-stack/bacnet-stack/tree/master) project. We have added
it as a submodule to this repo so that you can easily get a copy to then compile the server.

In order to initialise the submodule you can either clone this repository with the `--recursive` flag or you
can run `git submodule update --init --recursive` after you clone it: up to you!

Compiling the example applications is a matter of navigating to the correct directory and then running `make`:

    # Navigate to the apps directory.
    $ cd bacnet-stack/apps

    # Time to compile!
    $ make

If you do not have `make` installed, please refer to your system's package manager: it should be fairly easy to
get it up and running. Once that's done, we can just run the examples.

### Configuration
The `bacnet-stack` project's been written in pure C. In a fairly common C style, some of the configuration is
done through environment variables. Please refer to the documentation at `bacnet-stack/doc/README.*` so that
you know what to expect: for instance `bacnet-stack/doc/README..utils` contains information on how to configure
the network interface the `bacnet-stack` tools will bind to.

The following examples have been executed leveraging the Vagrant escenario defined in the `Vagrantfile`. We'll
make it clear from where (i.e. which machine) we're running things. Also, bear in mind you'll need to find the
name of the bridge interface providing Vagrant's network backbone. On macOS that can be accomplished with
`ifconfig -a` and on Linux the same's possible with `ip a`. You can also use [WireShark](https://www.wireshark.org)
to check what interfaces have traffic... At any rate, the names in our examples may (and very possibly will) differ
from yours. Another easy way to identify the interface is to check its associated IPv4 address belongs to the
`10.0.123.0/24` subnet, which is defined within the `Vagrantfile`. Bear in mind you **must** change this setting
if it conflicts with existing subnets on your machine.

The purpose of running these examples is both making sure our tools work **and** capturing traffic to check
it's compliant with WireShark's BACnet dissector.

#### Testing bacnet-stack server
From macOS we just need to start the server with Device ID `123` and Device Name `foo` with:

    # Getting to the server directory.
    $ cd bacnet-stack/apps/server

    # Exporting the interface we want the server to bind to.
    $ export BACNET_IFACE=bridge100

    # Running it! You can use `--help` for more information.
    $ ./bacserv 123 foo

From the VM we just need to run some clients:

    # Getting to the apps directory.
    $ cd bacnet-stack/apps

    # The interface name on this machine's different!
    $ export BACNET_IFACE=enp0s8

    # Writing a property to Device ID 123. The MAC IPv4 address is the one assigned to the
    # host machine on the Vagrant subnet and the port is BACnet's default one.
    ./writeprop/bacwp 123 analog-output 0 present-value 16 -1 4 1.1 --mac 10.0.123.1:47808

    # Reading a property from Device ID 123. In this case, just like before, you can use the
    # --help flag to get some more information on what you can use and what's configurable.
    ./readprop/bacrp 123 analog-output 1 present-value --mac 10.0.123.1:47808

    # Making a WhoIs request.
    ./bacwi --mac 10.0.123.1:47808

Do bear in mind you can find the predefined devices and mock data on `bacnet-stack`'s server on
`bacnet-stack/apps/server/epics_vts3.tpi`.

#### Testing as a foreign device
The foreign-device code has to be tested with the client truly foreign: on a subnet whose broadcasts do not
reach the devices, so that everything arrives through the BBMD. The `Vagrantfile` defines two subnets for
that:

    host (client)            foovm                         barvm
    10.0.123.1  ── net A ──  10.0.123.2 | 10.0.124.2  ── net B ──  10.0.124.3
                             IP router + bacserv (BBMD)             bacserv (device)
                             BACnet bound to the net B side

Net A is the private network above. Net B is a VirtualBox internal network the host is not attached to:
`foovm` has a second adapter on it and forwards IP between the two, and `barvm` is on net B alone, with its
route to net A through `foovm`. The host needs one static route for net B, which is the only manual step:

    # Windows (as administrator); on Linux: ip route add 10.0.124.0/24 via 10.0.123.2
    $ route add 10.0.124.0 mask 255.255.255.0 10.0.123.2

Then on `foovm`, run `bacserv` as the BBMD, bound to its net B side (the Linux port of `bacnet-stack` is
built with BBMD support), and on `barvm`, run it as an ordinary device:

    # foovm: the BBMD, bound to the net B adapter.
    $ export BACNET_IFACE=enp0s9
    $ ./bacserv 124 bbmd

    # barvm: a device on net B.
    $ export BACNET_IFACE=enp0s8
    $ ./bacserv 125 bar

From the host, the `whois` example registers with the BBMD and finds `barvm` through it:

    $ cd examples
    $ go run . whois --interface 10.0.123.1 --bbmd 10.0.124.2

What to check: the client logs its registration with the BBMD; device 125 is listed, at 10.0.124.3 and not
at the BBMD's address; `go run scripts/readonly_check.go -remote 10.0.124.2 -bbmd -only` lists the host in
the BBMD's foreign device table; and without `--bbmd`, the same Who-Is finds nothing, which proves the
client is foreign. If the BBMD does not forward the Who-Is onto net B, look at its broadcast distribution
table in the same output: it must contain its own net B address for it to broadcast there.

The same `-bbmd -only` query tells whether any BACnet/IP device is a BBMD: one answers with its tables,
another refuses or stays silent. Run it against each panel on a subnet to find its BBMD.

The same scenario runs without any of this on the simulated devices: the `client` tests register
with a simulated BBMD, discover and read the devices behind it, and are refused by one that NAKs.

#### Testing our ReadProperty server
From macOS we can just run:

    # Go to the examples directory
    $ cd examples

    # Run the example ReadProperty server
    $ ./bin/bexamples.osx.ex --remote-address 10.0.123.255:47808 rp

From the VM we can just query for some data:

    # navigate to the ReadProperty directory
    $ cd bacnet-stack/apps/readprop

    # Query Device ID 321 for analog-output 2 and point it to the host's
    # IPv4 address (i.e. 10.0.123.1).
    $ ./bacrp 321 analog-output 2 present-value --mac 10.0.123.1:47808

Our ReadProperty server will answer requests for `analog-output`s `0` and `1` and will reply
with an `Error` otherwise.

With that, you can get the gist of what's going on: happy tweaking!

## Traffic captures
In order to develop parsing and encoding functions we have captured traffic generated by the
`bacnet-stack` implementation with WireShark. We have stored those capture files on `traffcaps`
so that you can also leverage them. You should be able to open them up with WireShark without
too much trouble. We will also store traffic captures generated with out examples to showcase
how we are also compliant with the dissector.
