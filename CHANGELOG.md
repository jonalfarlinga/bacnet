# Changelog

Releases up to v0.3.3 had no changelog; the tags are their record.

## Unreleased

### Added

- `client`: `WriteProperty`, with a priority and a Null to relinquish, to a device switched to writable with
  `Config.Writable` or `SetWritable` (`Writable` lists the rules); every device starts read-only, a broadcast
  is never writable, and `ReadOnly` overrides the switch. A write is sent once and never resent: `ErrTimeout`
  means it was sent and unanswered, and an error wrapping the new `ErrNotSent` means it never left the
  client, with the reason wrapped as well. The guard test covers a writable device changing nothing for any
  other.
- `client/sim`: the devices take `WriteProperty` on the present value of their commandable objects (AO,
  AV, BO, BV and MSV), through a sixteen-slot priority array over the relinquish default, and on
  out-of-service and object-name, answering a SimpleACK or the error a device would, so that a program
  built on the client can be tried with writes. They record and acknowledge the confirmed services the
  parser has no type for that such a program sends only when told to, LifeSafetyOperation among them.
  `Writes` and `Services` list what reached them.

## v0.4.0 (2026-10-14)

The library is released on its own: the explorer developed in this repository lives in
[jonalfarlinga/bacnet-explorer](https://github.com/jonalfarlinga/bacnet-explorer) since 2026-10-14, with its
history. [docs/library-upgrade-spec.md](docs/library-upgrade-spec.md) is the design behind the changes
below, with the status of each part.

### Breaking

- Go 1.25 is the minimum; it was 1.19.
- `objects.Object.Length` is a `uint32`, the data length in bytes; it was a `uint8` that also stood for
  opening and closing tags. The new `Kind` field, a `TagKind`, says whether a tag is an application tag, a
  context tag, an opening tag or a closing tag: code that told them apart by `Length == 6` or `7` must
  switch to `Kind`.
- `objects.EncEnumerated` takes a `uint32`; it took a `uint8`.
- `test_utils` is `internal/testutil`: it was test support, not API.
- The examples are a module of their own, `examples/`, so cobra is no longer a requirement of the library.
- The codec packages no longer write to the standard logger.

### Added

- `client`: a BACnet/IP client on one UDP socket. Who-Is (`WhoIs`, `WhoIsRange`, `WhoIsAt`), with the
  I-Ams delivered to `Config.OnIAm`; `ReadProperty`, `ReadPropertyIndex`, `ReadPropertyMultiple`, and
  `ReadMany`, which splits a request to what the device accepts and falls back to ReadProperty for a device
  without ReadPropertyMultiple; `ReadObjectList`; `ReadRange`, with `LogRecord` for trend logs; `Probe` and
  `Knock`; `SubscribeCOV`, with a `Subscription` the client renews, and `CancelCOV`. Replies are matched to
  requests by invoke ID, retried on timeout, and reassembled when segmented, up to `Config.MaxSegments`.
  Values are decoded into `Value`, by `Kind`; the errors a device answers with are `BACnetError`,
  `RejectError`, `AbortError` and `BVLCResultError`. Devices behind BACnet routers are reached through the
  routing table the client learns (`WhoIsRouterToNetwork`, `Routes`), and devices on other IP subnets
  through a BBMD the client registers with as a foreign device (`Config.ForeignDevice`,
  `ForeignDeviceStatus`, `ReadBDT`, `ReadFDT`). One send function refuses every service but the reads, the
  Segment-ACKs and Aborts a segmented reply needs, Who-Is-Router-To-Network, the BBMD functions, and
  SubscribeCOV with the acknowledgement of a confirmed notification; `Config.ReadOnly` refuses those two as
  well. A deny list (`Config.Denied`, `SetDenied`), limits on what is in flight per device and per routed
  network, a rate limit, and `Stats`. `Config.OnPacket` sees every datagram, and `Config.Logger` what the
  client drops and learns. `Interfaces` and `FindInterface` pick an adapter, and `Interface.Config` gives
  the socket configuration that hears its broadcasts. `Reassembler` joins segments for a program on a
  socket of its own.
- `client/sim`: simulated BACnet/IP devices on the loopback interface, with a router and devices behind it,
  a BBMD, a device that cannot segment and one without ReadPropertyMultiple; they take COV subscriptions.
  The client's tests run against them.
- Codec: `Parse` returns an error, never panics, on a malformed datagram, with fuzz tests and the hex
  fixtures in `testdata/`, among them frames from a Siemens field panel. Segmented APDUs (`IsSegmented`,
  `HasMoreSegments`, the sequence number and window size), the `SegmentACK`, `Abort` and `Reject` messages,
  and `NewSegmentAck`. The NPDU's source and destination network addresses, and the network-layer messages
  (`plumbing.NetworkMessage`, `ParseNPDU`), Who-Is-Router-To-Network among them. BVLC: `BVLCResult`,
  Register-Foreign-Device, Read-Foreign-Device-Table and Read-Broadcast-Distribution-Table with their acks,
  and the origin of a Forwarded-NPDU (`OriginAddrPort`). Who-Has. ReadProperty with an array index
  (`NewReadPropertyIndexed`). Replies with a given invoke ID (`NewCACKReply`, `NewSACKReply`,
  `NewErrorReply`), for servers. `objects.DecodeTag`, and the names of object types, properties and units
  (`ObjectTypeName`, `PropertyName`, `UnitName`). `ParseWithFallback`.
- Examples: `whois`, `rpc`, `rpm`, `rrc`, `cov` and `probe` are rewritten on the client and are the
  reference for using it. `scripts/readonly_check.go` runs read-only checks against one live device.
- Package documentation for every package, a usage example for `client`, and a CI workflow that builds,
  vets and tests on Linux (under the race detector) and Windows.

### Notes

- `ComplexACK.DecodeRPM` and `ComplexACKRPMDec` stay, for a ReadPropertyMultiple reply received on a
  socket the program owns. The spec had planned to deprecate them for a typed decoder in `services`; that
  decoder lives in `client` instead, behind `ReadPropertyMultiple` and `ReadMany`.
