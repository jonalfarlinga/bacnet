# BACnet
Forked from [@ulbios's](https://github.com/ulbios) repository, implementing a Go BACnet protocol.

BACnet implementation in pure Golang. This work was initially based on [@kazukiigeta's](https://github.com/kazukiigeta)
work available on the [kazukiigeta/bacnet](https://github.com/kazukiigeta/bacnet) repository.

Please bear in mind we **only** support IP as the BACnet transport layer for now.

We began working with the marshalling and unmarshalling routines defined in the original project and added
a set of new messages. These are exposed through `New*()` functions defined on `encoding.go` and are
parsed with the `Parse()` function defined on `parsing.go`.

In order to make adding new messages easier, we restructured the project and broke everything up in several directories:

1. `plumbing/`: Everything related to BVLV, NPDU and APDU marshalling and unmarshalling.
2. `objects/`: Definition of different BACnet objects so that they can be reused.
3. `services/`: Implementation of several BACnet services such as *ReadProperty* and *WriteProperty*.
4. `common/`: Utilities and definitions used across all the above.

On top of the BACnet implementation, we also offer a CLI-based program offering a way to test every
available service. All the sources are contained on `examples/`. The binary can be generated with:

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
binary with no arguments to get an idea of what it can do. You can also take a look at its implementation: it's a
good example of how to use the library on your own project!

## Explorer
`cmd/explorer` is a web UI for browsing a BACnet network, in the manner of YABE. On load it broadcasts a
Who-Is and lists the devices that answer. Picking a device reads its object list and hangs the objects
under it, with their names, present values and status flags; devices behind a router hang off the router,
under their network number. Picking an object reads all of its properties.

It is read-only: it sends Who-Is, ReadProperty and ReadPropertyMultiple requests, the Segment-ACKs that
segmented replies need, Who-Is-Router-To-Network to find the router for a network, and with `-bbmd`, the
foreign-device registration and the table reads of the BBMD, and refuses to send anything else. It does not
write, subscribe, send an I-Am or answer what devices send to it.

The UI is a React app in `web/` that is compiled into the binary, so it has to be built first (a current Node
LTS):

    # Build the UI into web/dist.
    $ cd web
    $ npm install
    $ npm run build
    $ cd ..

    # Browse the network on an adapter, given by name or by IPv4 address.
    $ go run ./cmd/explorer -interface "Ethernet 3"

    # Or try it on simulated devices, with no network involved.
    $ go run ./cmd/explorer -sim

Then open http://127.0.0.1:8080. Without `-interface` the UI asks which adapter to use, unless there is only
one, and remembers the answer. The explorer needs UDP port 47808 on that adapter to hear the I-Ams, so it
cannot share an adapter with another BACnet program such as YABE. `-h` lists the other flags.

Devices behind BACnet routers are found through the Who-Is, which is addressed to every network; the
explorer learns which router serves each network from their replies, and asks with Who-Is-Router-To-Network
when it has to. A BACnet network on another IP subnet, which broadcasts do not reach, is reached through its
BBMD: with `-bbmd 10.0.124.2` the explorer registers with it as a foreign device, sends its broadcasts to it as
Distribute-Broadcast-To-Network, and gets the subnet's broadcasts back forwarded. The registration lasts
`-bbmd-ttl` (5 minutes unless set) and is renewed halfway through; if the BBMD refuses it or stops answering,
the explorer tries again with a growing backoff, from 5 seconds to 5 minutes.

To work on the UI, run the explorer and then `npm run dev` in `web/`: Vite serves the UI with hot reload and
passes `/api` on to the explorer.

| Directory | What is in it |
|---|---|
| `cmd/explorer` | The program: flags and wiring |
| `internal/client` | One UDP socket, request and reply matching, segmented replies, the routing table, foreign-device registration, the read-only guard |
| `internal/discovery` | The devices and objects found so far, and the reads that fill them in |
| `internal/api` | The HTTP API and the event stream the UI listens to |
| `internal/sim` | The simulated devices behind `-sim`, which the tests also use |
| `web` | The UI |

This is the browsing part of [docs/explorer-spec.md](docs/explorer-spec.md). Historizing, charts and the
packet log in that spec are not built yet.

## Running the examples
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

    # The interface name on this machine's different!
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

From the host, the explorer registers with the BBMD and finds `barvm` through it:

    $ go run ./cmd/explorer -interface 10.0.123.1 -bbmd 10.0.124.2

What to check: the explorer prints that it is a foreign device of the BBMD; device 125 appears, at
10.0.124.3 and not at the BBMD's address; its objects read; `scripts/readonly_check.go -remote 10.0.124.2`
lists the host in the BBMD's foreign device table; and without `-bbmd`, the same Who-Is finds nothing, which
proves the client is foreign. If the BBMD does not forward the Who-Is onto net B, read its broadcast
distribution table: it must contain its own net B address for it to broadcast there.

The same scenario runs without any of this on the simulated devices: the `internal/client` tests register
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
