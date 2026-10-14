# BACnet Library Upgrade — Specification (v1 draft)

Companion to [explorer-spec.md](https://github.com/jonalfarlinga/bacnet-explorer/blob/main/docs/explorer-spec.md) §4, in the explorer's repository. This spec covers the library changes the Explorer needs. These changes are also useful to anyone else using the library.

## 1. Goals

1. A **`client` package**: one long-lived object that owns the UDP socket and runs confirmed transactions concurrently. It handles timeouts, retries, and segmentation, and returns typed results.
2. **Reach every device** on the local subnet, behind BACnet routers (DNET/DADR), and on remote subnets through a BBMD as a foreign device.
3. **Never crash on network input.** Malformed packets return errors, never panics or reads from stale buffer bytes.
4. **Typed values**: decode any property value into a Go value that the UI can render and the historian can store.
5. **Read-only is enforced structurally, per device**: every device is read-only unless it is explicitly switched to writable, and the client refuses to send non-read services to a read-only device.
6. **Talk to any source by default**: discovery and reads work against every device the client can reach. The user turns specific sources off with a deny list (§7.7).

Read-only over BACnet/IP on UDP is the first step, not the end state. §11 lists what follows.

## 2. Non-goals (this upgrade)

- Sending segmented *requests*. Read requests are small, and oversized RPMs are split instead (§6.4).
- Server-side TSM (answering confirmed requests) and an MS/TP data link. MS/TP devices are reached only through routers.
- Transports other than BACnet/IP on UDP. UDP is the starting point; other IP transports are on the roadmap (§11).
- COV subscriptions. On the roadmap (§11).
- A device identity. The client is not a BACnet device: it has no Device object, never sends I-Am, and never answers Who-Is.
- Client methods for write services. The low-level builders stay (§9). Write methods are on the roadmap (§11); they are additive and work only against a device switched to writable.

## 3. Current-state issues this spec fixes

| Area | Issue | Where |
|---|---|---|
| Tag codec | No bounds checks: a truncated tag panics or reads past `len(b)` into the receive buffer | `plumbing/apdu.go` `parseObjectTags`, `objects.Object.UnmarshalBinary` |
| Tag codec | Extended length handles only the 1-byte form. Lengths 254 and 255 (2- and 4-byte) are unsupported. `Object.Length` is `uint8` | `parseObjectTags`, `objects/objects.go` |
| Tag codec | Opening/closing tags are marked by `Length == 6/7` sentinels, which are ambiguous with real lengths after extended decoding | `objects`, `services/*Decode*` |
| Tag codec | Tag-number extension (tag 15 + next byte) is unsupported | `parseObjectTags` |
| APDU | Dead branch `a.Flags&0x2 == 1` in `MarshalTo` | `plumbing/apdu.go` |
| Builders | InvokeID hard-coded to `1`. No destination network. `NewWhois` is local-only (no DNET 0xFFFF), so routed devices never see it | `encoding.go` |
| Builders | `EncEnumerated(uint8)` truncates enums above 255 | `objects/primitive.go` |
| RPM | Request supports one object. `DecodeRPM` returns one object and drops property-access errors | `services/rp.go`, `services/cack-rpm.go` |
| Network layer | NPDUs with the network-message bit fail with `ErrNotImplemented` | `parsing.go` |
| BVLC | Missing Register-Foreign-Device, Distribute-Broadcast-To-Network, Read-BDT. Forwarded-NPDU origin address is not exposed | `plumbing/` |
| Library hygiene | 10 `log.Print*` calls inside library packages | `services/`, `objects/`, `plumbing/` |
| Reuse | Segment reassembly, correlation, and socket handling live in `examples/` | `examples/reassembly.go`, `orchestrator.go` |

## 4. Toolchain

- Raise `go.mod` to **go 1.25**. This brings `log/slog`, `net/netip`, and `testing/synctest` (GA in 1.25) for deterministic timeout and retry tests. The explorer-spec L9 minimum is superseded by this.
- The only new dependency allowed is `golang.org/x/time/rate`. Everything else uses the stdlib.

## 5. Codec layer changes (`objects`, `plumbing`, `services`, root)

### 5.1 Tag codec hardening
- Replace the sentinel lengths with an explicit kind:
  ```go
  type TagKind uint8
  const (TagPrimitive TagKind = iota; TagOpening; TagClosing)

  type Object struct {
      TagNumber uint8
      TagClass  bool     // true = context
      Kind      TagKind
      Length    uint32   // data length in bytes (widened from uint8)
      Data      []byte
      Value     interface{}
  }
  ```
- Decode and encode every length form: LVT 0–4, 5+1 byte, 5+254+2 bytes, and 5+255+4 bytes. Also handle tag-number extension.
- Every read is bounds-checked against `len(b)`. Errors wrap `common.ErrTooShortToParse` or `common.ErrInvalidData`.
- Decoded `Data` is **copied**, never aliased into the caller's receive buffer, because the client reuses its read buffer.
- Application booleans: the value lives in the LVT and there are no data bytes. Decode this the same way for every PDU type, and remove the `decodeBoolean` parameter.
- `EncEnumerated` takes `uint32`.
- Remove the dead branch in `APDU.MarshalTo`.

**Acceptance**:
- `FuzzParse` (root) and `FuzzObjectTags` (`plumbing`) each run for 10 minutes without panics. They are seeded from `traffcaps/` and existing test vectors.
- Round-trip tests cover each length form and tag-number extension.

### 5.2 Remove logging from library packages
Library code returns errors and never calls `log`. Where a warning is useful (e.g. a skipped unknown tag), return it as structured data or a wrapped error.

### 5.3 Typed values (`objects/value.go`)
```go
type Value interface{ Kind() ValueKind }

type (
    Null            struct{}
    Boolean         bool
    Unsigned        uint64
    Signed          int64
    Real            float32
    Double          float64
    OctetString     []byte
    CharacterString string        // UTF-8, UCS-2, ISO-8859-1 decoded; others see RawCharString
    RawCharString   struct{ Charset uint8; Data []byte }
    BitString       []bool
    Enumerated      uint32
    Date            struct{ Year, Month, Day, Weekday uint8 } // 0xFF = wildcard preserved
    Time            struct{ Hour, Minute, Second, Hundredths uint8 }
    ObjectID        struct{ Type uint16; Instance uint32 }
    ContextValue    struct{ Tag uint8; Data []byte }      // context-tagged primitive, type unknown
    Constructed     struct{ Tag uint8; Items []Value }    // opening/closing pair
    List            []Value                               // >1 top-level element
)

func DecodeValue(tags []APDUPayload) (Value, error)  // one property value (contents of [3] / [4])
func Numeric(v Value) (float64, bool)                // Real, Double, Unsigned, Signed, Enumerated, Boolean
func (d Date) Time(loc *time.Location) (time.Time, bool) // false if wildcarded
func DecodeStatusFlags(v Value) (StatusFlags, error) // exported; replaces services/cack-rr private helper
```
The existing `Dec*` functions stay. `DecodeValue` is built on top of them.

**Acceptance**: golden tests for every kind, including a `priority-array` (16 elements, with Nulls), a `Constructed` value, a wildcard date, and a UCS-2 string.

### 5.4 Names for display
Add lookup functions. They don't change existing constant types, so nothing breaks:
`objects.ObjectTypeName(uint16)`, `objects.PropertyName(uint16)`, `objects.UnitsName(uint16)`, `services.ErrorClassName`, `services.ErrorCodeName`, `services.AbortReasonName`, `services.RejectReasonName`. Unknown values render as `"proprietary(123)"` or `"unknown(123)"`.

### 5.5 ReadPropertyMultiple (multi-object)
```go
type PropertyRef struct { ID uint16; Index *uint32 }
type ReadAccessSpec struct { Object ObjectID; Properties []PropertyRef }

func ReadPropertyMultipleObjects(specs []ReadAccessSpec) []objects.APDUPayload

type PropertyResult struct { Ref PropertyRef; Value objects.Value; Err *PropertyError } // Err = [5] propertyAccessError
type ReadAccessResult struct { Object ObjectID; Results []PropertyResult }
func (c *ComplexACK) DecodeRPMResults() ([]ReadAccessResult, error)
```
The existing `DecodeRPM` is kept as a deprecated wrapper.

**Acceptance**: decode an RPM-ACK with 3 objects, a mix of values and property errors, and an array index. Also run it against a bacnet-stack capture.

### 5.6 ReadProperty decode
Add `(*ComplexACK) DecodeRP() (ObjectID, PropertyRef, objects.Value, error)` using `DecodeValue`.

## 6. Network, BVLC, and routing

### 6.1 NPDU destination and source
- `plumbing.NewNPDU` is unchanged. Add the following:
  ```go
  type NetAddr struct { Net uint16; MAC []byte }   // Net 0 = local; MAC empty with Net set = remote broadcast
  func NewNPDUTo(dst *NetAddr, expectingReply bool) *NPDU  // sets DNET/DLEN/DADR, Hop = 255
  ```
- Global broadcast is `NetAddr{Net: 0xFFFF}`.
- `NPDU.Source() *NetAddr` returns SNET/SADR when present.

### 6.2 Network-layer messages
When the NPDU control bit 7 is set, `Parse` returns a `*plumbing.NetworkMessage` instead of an error:
```go
type NetworkMessage struct { *BVLC; *NPDU; Type uint8; VendorID uint16; Data []byte }
```
Decode and encode helpers are required for:

| Type | Message | Direction |
|---|---|---|
| 0x00 | Who-Is-Router-To-Network (optional DNET) | send |
| 0x01 | I-Am-Router-To-Network (list of DNETs) | receive |
| 0x03 | Reject-Message-To-Network (reason, DNET) | receive |
| 0x04 / 0x05 | Router-Busy / Router-Available (DNET list) | receive |

Any other type parses into `NetworkMessage` with raw `Data`. None of them are errors.

### 6.3 BVLC additions
| Code | Function | Notes |
|---|---|---|
| 0x02 / 0x03 | Read-BDT / Read-BDT-Ack | Diagnostics only |
| 0x04 | Forwarded-NPDU | Expose `Origin netip.AddrPort` on the parsed message. `normalizeForwardedBVLC` must keep it. |
| 0x05 | Register-Foreign-Device (TTL uint16 seconds) | |
| 0x09 | Distribute-Broadcast-To-Network | Wraps an NPDU like Original-Broadcast |
| 0x00 | BVLC-Result | Named result codes: 0x0030 Register-FD NAK, 0x0060 Distribute-Broadcast NAK, and so on |

Write-BDT and Delete-FDT-Entry are **not** added, because they change BBMD state.

### 6.4 Builders
- A new internal builder path used by the client is `encodeConfirmed(dst Address, invokeID uint8, service uint8, objs) []byte`, plus the unconfirmed equivalent. It sets BVLC unicast, broadcast, or distribute; NPDU destination; `MaxSeg`/`MaxSize` from config; and the SA flag.
- Who-Is gets a range:
  ```go
  func WhoIsObjects(low, high uint32) []objects.APDUPayload // context [0], [1]; validates ≤ 4194303, low ≤ high
  ```
- The existing root `New*` functions keep their signatures. Add `NewWhoisGlobal()`, which uses DNET 0xFFFF. These functions remain for examples and tests.

## 7. The `client` package

### 7.1 Public API
```go
package client

type Address struct {
    IP  netip.AddrPort // peer or router B/IP address
    Net uint16         // 0 = directly reachable on B/IP
    MAC []byte         // DADR on Net
}

type Device struct {
    Instance     uint32
    Addr         Address
    MaxAPDU      uint16
    Segmentation uint8   // BACnetSegmentation enum
    VendorID     uint16
}

type ForeignDeviceConfig struct {
    BBMD netip.AddrPort
    TTL  time.Duration // default 300s
}

type Config struct {
    LocalAddr        netip.AddrPort // default 0.0.0.0:47808 (0.0.0.0 required to receive broadcasts on Windows)
    BroadcastAddr    netip.AddrPort // default derived from Interface
    Interface        string         // optional; picks addr + broadcast
    Conn             net.PacketConn // optional injection (tests); overrides LocalAddr
    APDUTimeout      time.Duration  // default 3s
    Retries          int            // default 3
    SegmentTimeout   time.Duration  // default 2s
    MaxAPDUAccepted  uint16         // default 1476
    MaxSegments      int            // default 64; reassembled cap = MaxSegments*MaxAPDUAccepted
    MaxInFlightDevice func(Address) int   // directly reachable device (Net == 0); default 4
    MaxInFlightNet    func(net uint16) int // all devices on one routed DNET together; default 1
    MaxRequestsPerSec float64       // global token bucket, default 50, 0 = unlimited
    Writable         []SourceRule   // devices switched out of read-only; default none, see §7.7
    Denied           []SourceRule   // sources turned off; default none, see §7.7
    ForeignDevice    *ForeignDeviceConfig
    Logger           *slog.Logger   // default discard
    OnPacket         func(PacketEvent) // diagnostics tap; must not block
}

// SourceRule matches a source. Set exactly one field.
type SourceRule struct {
    Instance *uint32       // one device, by device instance
    IP       netip.Prefix  // a B/IP host or subnet
    Net      *uint16       // every device on one routed DNET
}

func New(cfg Config) (*Client, error)
func (c *Client) Close() error

// Send guard (§7.7); both can change at runtime
func (c *Client) SetReadOnly(dev Device, readOnly bool) // per-device switch; devices start read-only
func (c *Client) SetDenied(rule SourceRule, denied bool)

// Discovery
func (c *Client) WhoIs(ctx context.Context, opts WhoIsOptions) ([]Device, error) // collects I-Ams until ctx done
func (c *Client) SubscribeIAm(buf int) (<-chan Device, func())                  // drops when full, counts drops
func (c *Client) WhoIsRouterToNetwork(ctx context.Context, dnet *uint16) ([]RouterInfo, error)

// Reads
func (c *Client) ReadProperty(ctx context.Context, dev Device, obj ObjectID, prop PropertyRef) (objects.Value, error)
func (c *Client) ReadPropertyMultiple(ctx context.Context, dev Device, specs []ReadAccessSpec) ([]ReadAccessResult, error)
func (c *Client) ReadRange(ctx context.Context, dev Device, req ReadRangeRequest) (ReadRangeResult, error)

// Helpers
func (c *Client) ReadObjectList(ctx context.Context, dev Device) ([]ObjectID, error) // whole → indexed fallback
func (c *Client) ReadMany(ctx context.Context, dev Device, specs []ReadAccessSpec) ([]ReadAccessResult, error)
    // splits to fit dev.MaxAPDU; falls back to RP if RPM unsupported

// BBMD / diagnostics
func (c *Client) ForeignDeviceStatus() FDStatus
func (c *Client) ReadFDT(ctx context.Context, bbmd netip.AddrPort) ([]FDTEntry, error)
func (c *Client) ReadBDT(ctx context.Context, bbmd netip.AddrPort) ([]BDTEntry, error)
func (c *Client) Stats() Stats // per-peer: sent, retries, timeouts, errors, latency EWMA/p95
```

`WhoIsOptions{Low, High *uint32; Target *Address}`. With no target, it does a global broadcast: DNET 0xFFFF to the local broadcast address, or Distribute-Broadcast to the BBMD when the client is registered as a foreign device.

### 7.2 Errors
```go
var (
    ErrTimeout, ErrClosed, ErrServiceNotAllowed, ErrSourceDenied, ErrNetworkUnreachable,
    ErrRouterBusy, ErrResponseTooLarge error
)
type BACnetError  struct{ Class, Code uint32 }        // Error-PDU
type RejectError  struct{ Reason uint8 }
type AbortError   struct{ Reason uint8; FromServer bool }
type PropertyError = BACnetError                      // RPM per-property
```
All of these support `errors.Is` and `errors.As`. `Error()` strings use the §5.4 names.

### 7.3 Transport and dispatch
- One socket and one read goroutine. For each datagram the client does the following:
  1. Drop it if the source is one of our own interface addresses *and* our port (local echo).
  2. Parse it with `ParseWithFallback`.
  3. Call `OnPacket`.
  4. Drop it if its source matches a deny rule (§7.7). The packet tap still sees it, marked as denied.
  5. Dispatch it by type:
     - Confirmed replies (SimpleAck, ComplexAck, Error, Reject, Abort, SegmentAck) go to the TSM, keyed on **peer key + invoke ID**.
     - I-Am goes to the device cache and subscribers.
     - Network messages go to the routing table.
     - BVLC-Result goes to the foreign-device state machine.
     - Incoming confirmed requests and Who-Is are ignored. The client is not a BACnet device and has no identity (§2).
- **Peer key**:
  - A routed reply is keyed on `(SNET, SADR)` from its NPDU source, because the router's IP is shared by many devices.
  - Otherwise the key is the B/IP source.
  - For a Forwarded-NPDU, the key is the **origin** address, not the BBMD's address.
- The read buffer is reused. Parsed data must not alias it (§5.1).
- Writes are serialized through one sender, which applies the global rate limiter.

### 7.4 Client TSM (confirmed requests)
- **Invoke IDs**: allocated per peer key, rotating sequentially to avoid reusing a recently timed-out ID. If all 256 are in use, the call blocks until one frees up or the context ends.
- **Concurrency**: a semaphore acquired before an invoke ID.
  - A directly reachable device has its own semaphore of size `MaxInFlightDevice(addr)`.
  - Routed devices share one semaphore per DNET of size `MaxInFlightNet(net)`. An MS/TP trunk carries one exchange at a time, so the limit counts requests to the whole network, not to each device on it.
- **Timeout and retry**: after `APDUTimeout`, the client retransmits identical bytes with the same invoke ID, up to `Retries` times, then returns `ErrTimeout`. Context cancellation ends the wait immediately and frees the ID. Late replies are dropped and counted.
- **Segmented ComplexAck** (moves `examples/reassembly.go` into `client/segment.go`):
  - Accept segment 0. Actual window = min(proposed, 64).
  - Send a Segment-ACK at the end of each window and on the final segment.
  - A duplicate segment gets a re-ack with the last in-order sequence number.
  - An out-of-order segment gets a NAK with the last in-order sequence number.
  - While segments are arriving, `SegmentTimeout` replaces `APDUTimeout`. If it expires, send Abort(tsm-timeout, 10) and return `ErrTimeout`.
  - If the reassembled size would exceed `MaxSegments*MaxAPDUAccepted`, send Abort(buffer-overflow, 1) and return `ErrResponseTooLarge`.
  - **Cap logging**: both caps are fixed, not per-device, so every time one is hit it must be visible.
    - A proposed window above 64 is clamped and logged at `Warn`, once per transaction: peer, device instance, invoke ID, service, proposed window, actual window.
    - A response above the size cap is logged at `Error`: the same fields plus segments received, bytes received, and the cap. `ErrResponseTooLarge` carries these values so the caller can report them even when `Logger` is the default discard.
    - `Stats` counts both per peer (`WindowClamped`, `ResponseTooLarge`).
- Requests advertise `MaxSize` and `MaxSeg` from config and always set SA.

### 7.5 Routing
- **Routing table**: DNET → router `netip.AddrPort`. It is learned from I-Am-Router-To-Network and from any received NPDU that carries SNET (the B/IP source is the router for that SNET).
- **Sending to** `Address{Net: n}`:
  - If `n` is in the table, unicast to that router.
  - Otherwise send Who-Is-Router-To-Network(n) and wait up to `APDUTimeout`. If that learns nothing, broadcast the request with the DNET set.
- **Reject-Message-To-Network(n)**: fail in-flight and queued requests to `n` with `ErrNetworkUnreachable`, and drop the route.
- **Router-Busy(n)**: hold new sends to `n` until Router-Available or 30 s pass. The requests already queued see this time as queue time, not retries.
- A device's `Address.IP` is updated from its latest I-Am or reply. This way a router IP change heals itself.

### 7.6 Foreign device
- **Registration**:
  - When `ForeignDevice` is set, send Register-Foreign-Device(TTL) at startup and re-register every TTL/2.
  - `FDStatus{State: Registering|Registered|Failed, LastResult, NextAttempt}`.
  - On a NAK or no response, retry with backoff (5 s → 5 min).
- **Broadcasts** become Distribute-Broadcast-To-Network to the BBMD. Unicasts go directly to the device's B/IP address, which is the Forwarded-NPDU origin.
- On `Close`, the client does not deregister (there is no BVLC function for it). The FDT entry expires after TTL plus 30 s.

### 7.7 Send guard: read-only switch and deny list
All outbound traffic goes through one `send` function, which applies two checks.

**Deny list.** The client talks to any source by default; there is no allow-list. The user turns specific sources off with `Denied` rules or `SetDenied`.
- A unicast send to a denied source returns `ErrSourceDenied`. This covers discovery follow-ups as well as reads.
- Inbound packets from a denied source are dropped before dispatch (§7.3), so its I-Am never reaches the device cache, `WhoIs` results, or subscribers.
- A broadcast Who-Is still reaches a denied device on the wire, because a broadcast cannot exclude one recipient. Its reply is dropped.
- Denying a source fails its in-flight and queued requests with `ErrSourceDenied`.

**Read-only switch.** Read-only is a per-device switch, and every device starts read-only. A device becomes writable only through a `Writable` rule or `SetReadOnly(dev, false)`.
- The switch is looked up by the destination of each send. Broadcasts are always treated as read-only.
- This upgrade adds no client write methods (§2), so the switch has nothing to unlock yet. It is in place so that the roadmap write methods (§11) need no change to the guard.
- For a read-only destination, `send` refuses any APDU whose (PDU type, service) pair is not on this list:

  | Allowed | |
  |---|---|
  | Unconfirmed | Who-Is |
  | Confirmed | ReadProperty, ReadPropertyMultiple, ReadRange |
  | PDU | Segment-ACK, Abort (client-originated) |
  | Network | Who-Is-Router-To-Network |
  | BVLC | Register-Foreign-Device, Distribute-Broadcast-To-Network, Read-FDT, Read-BDT |

- A refused send returns `ErrServiceNotAllowed` and is logged at `Warn`.
- A test asserts that every other service code in `services/consts.go` is refused for a read-only device, and that switching one device to writable does not change the result for any other device.

## 8. Testing

| Layer | Approach |
|---|---|
| Codec | Table-driven round trips with `go-cmp`, following the existing style. Fuzz `Parse` and `parseObjectTags`. |
| Captures | Extract frames from `traffcaps/*.pcapng` once, with `tshark`, into `testdata/*.hex`. There is no pcap dependency. Each frame must parse without error. |
| TSM | A fake `net.PacketConn` pair plus `testing/synctest` for virtual time. Scenarios: <br>• success <br>• timeout then retry success <br>• retries exhausted <br>• duplicate and late replies <br>• context cancel <br>• invoke-ID exhaustion <br>• segmented ack with window 1/4/16 <br>• duplicate, out-of-order, and missing segments <br>• oversized response <br>• Reject-Message-To-Network <br>• Router-Busy <br>• FD re-register and NAK <br>• read-only refusal, per device <br>• denied source: send refused, inbound dropped, in-flight failed |
| Integration | The Vagrant `bacnet-stack` server (`bacserv`) for RP, RPM, RR, and object-list. A routed scenario uses bacnet-stack's gateway app (`apps/gateway`, built with `make gateway`, run in place of `bacserv`): one process that presents a gateway device plus virtual devices on DNET 2709, each with a 3-byte MAC. It covers I-Am-Router-To-Network, Who-Is-Router-To-Network, RP/RPM/RR to a DNET, `(SNET, SADR)` peer keying with many devices behind one IP, and Reject-Message-To-Network for an unknown DNET. It never sends Router-Busy, which stays in the TSM fakes. It needs no Vagrant change. A foreign-device scenario runs the client on a different IP subnet from the BBMD and devices (§8.1). These run behind a build tag `integration`. |

### 8.1 Foreign-device test topology
The foreign-device code must be tested with the client truly foreign: on a subnet that the BACnet subnet's broadcasts do not reach. A same-subnet test is not sufficient, because the client would receive each broadcast directly as well as through the BBMD.

```
host (client)            foovm                         barvm
10.0.123.1  ── net A ──  10.0.123.2 | 10.0.124.2  ── net B ──  10.0.124.3
                         IP router + bacserv (BBMD)             bacserv (device)
                         BACnet bound to the net B side
```

- **Net A** (10.0.123.0/24) is the existing Vagrant private network. **Net B** (10.0.124.0/24) is a new VirtualBox internal network that the host is not attached to.
- `foovm` gets a second NIC on net B, forwards IP between the two networks, and runs `bacserv` as the BBMD bound to its net B interface.
- `barvm` is a new VM on net B only, running `bacserv` as an ordinary device, with its default route for net A through `foovm`.
- The host needs one static route: 10.0.124.0/24 via 10.0.123.2. With it, the client can unicast to the BBMD and to `barvm`, but never sees net B broadcasts.

**Acceptance**:
- Registration succeeds and Read-FDT lists the client's address. Re-registration happens at TTL/2.
- A Who-Is sent as Distribute-Broadcast-To-Network returns an I-Am from `barvm` as a Forwarded-NPDU, and the client records `barvm`'s address (the origin), not the BBMD's.
- ReadProperty then goes to `barvm` by unicast and succeeds.
- Without registration, the same Who-Is returns nothing. This proves the client is foreign.
- Stopping the BBMD moves `ForeignDeviceStatus` to `Failed` with backoff; restarting it recovers.

The Vagrantfile change and a README section for the host route are part of PR 8.

## 9. Compatibility

- **Unchanged**: root `Parse`, `ParseWithFallback`, and the `New*` signatures. `services` message types and their `Decode*` methods.
- **Breaking (tag the last commit before this work as `v0.1.0` and release this as `v0.2.0`)**:
  - `objects.Object.Length` changes `uint8 → uint32`, and a new `Kind` field is added.
  - Code that checks `Length == 6/7` must switch to `Kind`.
  - `EncEnumerated(uint32)`.
- **Deprecated**: `services.ComplexACKRPMDec` and `DecodeRPM`, replaced by `DecodeRPMResults`.
- **Examples**: `whois-client`, `rp-client`, `rpm-client`, and `rr-client` are rewritten on `client` as reference usage. `orchestrator` keeps its raw-socket design and imports the reassembler from `client`.

## 10. Delivery plan (PR sequence)

| PR | Content | Depends on |
|---|---|---|
| 1 | go 1.25 bump. Remove `log` from library code. Tag codec hardening (§5.1) and fuzz tests. Pcap fixtures. | — |
| 2 | Typed values, `DecodeValue`, `DecodeRP`, and name lookups (§5.3, 5.4, 5.6) | 1 |
| 3 | Multi-object RPM request and decode, with property errors (§5.5) | 2 |
| 4 | NPDU `NetAddr`, network messages, global and ranged Who-Is (§6.1, 6.2, 6.4) | 1 |
| 5 | BVLC additions and Forwarded-NPDU origin (§6.3) | 1 |
| 6 | `client`: transport, dispatch, unsegmented TSM, errors, read-only guard, stats (§7.1–7.4 minus segmentation, 7.7) | 2, 3 |
| 7 | `client`: segmentation. Move the reassembler (§7.4). | 6 |
| 8 | `client`: routing table and foreign device (§7.5, 7.6). Two-subnet Vagrant topology (§8.1). | 4, 5, 6 |
| 9 | Helpers (`ReadObjectList`, `ReadMany`), example migration, and a `probe` example command | 7, 8 |

PRs 4 and 5 can proceed in parallel with PRs 2 and 3.

### 10.1 Status, 2026-10-07

PRs 1 to 8 are delivered, with these differences from the text above, kept because they fit the explorer better:

- Typed values are one `client.Value` struct with a `Kind` field, not the `objects.Value` interface of §5.3, and the multi-object RPM decoder (§5.5) lives in `client`, not `services`.
- Discovery is `WhoIs` and `WhoIsRange` plus an `OnIAm` callback, not the collecting `WhoIs(ctx)` and `SubscribeIAm` of §7.1.
- The read-only guard is global rather than per device (§7.7). It has nothing to unlock until write methods exist, and the per-device switch is left for them.
- `Config.MaxRequestsPerSec` takes 0 as the default and a value below 0 as no limit. `MaxSegments` counts segments, not bytes, and a reply beyond it fails with `*ResponseTooLargeError`.
- Tests run against the simulator in `client/sim`, not `testing/synctest` fakes (§8).

The deny list, the in-flight limits, the rate limit, `Stats`, `ReadMany`, the ranged Who-Is and the cap logging of §7.4 are in.

COV is delivered, from the roadmap (§11), 2026-10-08: `SubscribeCOV` returns a `Subscription` that the client renews at half its lifetime, with a backoff after a failure, until `Cancel`; notifications reach a callback as `COVNotification`, and a confirmed one is answered with a SimpleACK. `CancelCOV` drops a subscription the client does not hold. The send guard takes SubscribeCOV and that SimpleACK, and nothing else new: a notification for no subscription of the client's gets no answer. `Config.ReadOnly` refuses both, which is how the explorer stays read-only. There is no subscription without a lifetime, no SubscribeCOVProperty, and `Close` does not cancel. The simulator takes subscriptions and notifies on `NotifyCOV`.

PR 9 is delivered: `ReadRange` with `LogRecords` for trend logs, the `whois`, `rpc`, `rpm` and `rrc` examples rewritten on `client`, a `probe` example, and the reassembler moved to `client.Reassembler` for the orchestrator. The name lookups of §5.4 live in `objects/names.go`. ReadRange by time is not sent, for want of date and time encoders; by position and by sequence number are.

## 11. Roadmap (after this upgrade)

This upgrade delivers read-only access over BACnet/IP on UDP. That is the first step. The items below are planned and are not designed here; the order is not fixed.

| Item | Notes |
|---|---|
| Other IP transports | UDP is the starting point. `Address`, the peer key, and `Config.Conn` are specific to B/IP on UDP in this upgrade and will need a transport boundary inside `client` when a second transport is added. |
| COV | Delivered, see §10.1. Left: SubscribeCOVProperty with a COV increment, and unsubscribed notifications. |
| Write services | WriteProperty with priority and relinquish, for devices switched to writable (§7.7). Writes must not be retransmitted blindly, so this needs a per-call retry override and an error that separates "sent, no answer" from "never sent". |
| Alarms | GetEventInformation (a read; add to the allow-list), then AcknowledgeAlarm as a write service. |

## 12. Open questions

Resolved: the window cap stays fixed at 64 with the cap logging in §7.4, the client has no device identity (§2), and the in-flight limit for routed devices is per DNET with a default of 1 (§7.4).

The routed test scenario uses bacnet-stack's gateway app (§8). This was chosen from reading its source at the pinned commit; it has not been built or run yet. The multi-port `apps/router` can be added later for real two-network forwarding, but it needs `libconfig-dev` and a second network or VM.

1. **BBMD test scenario**: the foreign-device test uses the two-subnet topology in §8.1. The Vagrantfile defines it and the README walks through it, but it has not been run yet. Still to verify: that `bacserv`'s BBMD mode accepts foreign-device registration and forwards Distribute-Broadcast-To-Network onto its subnet (its BDT must hold its own address for that). If it does not, the fallback is another BBMD implementation on `foovm`. Until then, the foreign-device code is covered by the simulated BBMD in `client/sim`.
2. **Other IP transports**: which transport comes after UDP (§11)?
