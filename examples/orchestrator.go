package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/common"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
	"github.com/spf13/cobra"
)

func init() {
	OrchestratorCmd.Flags().StringSliceVar(&orchSend, "send", []string{}, "Message kinds to send every cycle. Supported: whois, iam, rp, rpm, rr, wp, cov, sack, cack, err, rfdt")
	OrchestratorCmd.Flags().IntVar(&orchPeriod, "period", 30, "Seconds between outbound send cycles.")
	OrchestratorCmd.Flags().IntVar(&orchN, "messages", 0, "Number of send cycles. 0 means unlimited.")
	OrchestratorCmd.Flags().BoolVar(&orchStayAlive, "stay-alive", true, "Keep listening and handling inbound traffic after sending cycles complete.")
	OrchestratorCmd.Flags().DurationVar(&orchShutdownAfter, "shutdown-after", 0, "Automatically stop the orchestrator after this duration (e.g. 30s, 5m). 0 disables auto-shutdown.")
	OrchestratorCmd.Flags().BoolVar(&orchIgnoreLocal, "ignore-local", true, "Ignore packets originated from local interfaces.")

	OrchestratorCmd.Flags().Uint32Var(&orchDeviceInst, "device-instance", 321, "Device instance used when replying to WhoIs and building IAm.")
	OrchestratorCmd.Flags().Uint16Var(&orchVendorID, "vendor-id", 31, "Vendor ID used when replying to WhoIs and building IAm.")

	OrchestratorCmd.Flags().Uint16Var(&orchObjectType, "object-type", objects.ObjectTypeAnalogOutput, "Object type used for RP/RPM/RR/WP/COV messages.")
	OrchestratorCmd.Flags().Uint32Var(&orchInstanceID, "instance-id", 0, "Instance ID used for RP/RPM/RR/WP/COV messages.")
	OrchestratorCmd.Flags().Uint16Var(&orchPropertyID, "property-id", objects.PropertyIdPresentValue, "Property ID used for RP/RR/WP messages.")
	OrchestratorCmd.Flags().IntSliceVar(&orchProperties, "properties", []int{int(objects.PropertyIdPresentValue)}, "Property IDs used for RPM messages.")

	OrchestratorCmd.Flags().Uint16Var(&orchRangeStart, "range-start", 1, "ReadRange start index.")
	OrchestratorCmd.Flags().Int32Var(&orchRangeLength, "range-length", 10, "ReadRange item count.")

	OrchestratorCmd.Flags().StringVar(&orchWPKind, "wp-kind", "real", "WriteProperty payload kind: real or string.")
	OrchestratorCmd.Flags().Float32Var(&orchWPReal, "wp-real", 1.1, "WriteProperty real payload.")
	OrchestratorCmd.Flags().StringVar(&orchWPString, "wp-string", "hello", "WriteProperty string payload.")

	OrchestratorCmd.Flags().UintVar(&orchCOVProcessID, "cov-process-id", 1, "SubscribeCOV process ID.")
	OrchestratorCmd.Flags().UintVar(&orchCOVLifetime, "cov-lifetime", 240, "SubscribeCOV lifetime in seconds.")
	OrchestratorCmd.Flags().BoolVar(&orchCOVExpectConfirmed, "cov-expect-confirmed", true, "SubscribeCOV expects confirmed notifications.")
	OrchestratorCmd.Flags().BoolVar(&orchCOVCancel, "cov-cancel", false, "SubscribeCOV cancellation request.")

	OrchestratorCmd.Flags().Float32Var(&orchDefaultValue, "default-value", 1.1, "Default present-value used in RP responses.")
	OrchestratorCmd.Flags().Uint16Var(&orchStateInstances, "state-instances", 2, "Number of object instances preloaded in local state map.")

	OrchestratorCmd.Flags().StringVar(&orchDumpDir, "dump-reassembled", "", "If set, write each fully reassembled segmented message's raw bytes (hex-encoded) to this directory.")
}

var (
	orchSend               []string
	orchPeriod             int
	orchN                  int
	orchStayAlive          bool
	orchShutdownAfter      time.Duration
	orchIgnoreLocal        bool
	orchDeviceInst         uint32
	orchVendorID           uint16
	orchObjectType         uint16
	orchInstanceID         uint32
	orchPropertyID         uint16
	orchProperties         []int
	orchRangeStart         uint16
	orchRangeLength        int32
	orchWPKind             string
	orchWPReal             float32
	orchWPString           string
	orchCOVProcessID       uint
	orchCOVLifetime        uint
	orchCOVExpectConfirmed bool
	orchCOVCancel          bool
	orchDefaultValue       float32
	orchStateInstances     uint16
	orchDumpDir            string

	OrchestratorCmd = &cobra.Command{
		Use:   "orchestrator",
		Short: "Run a hub-like BACnet message orchestrator.",
		Long: "This command can periodically emit any implemented BACnet messages while also\n" +
			"processing every incoming packet and replying to confirmed requests when possible.",
		Args: argValidation,
		Run:  OrchestratorExample,
	}
)

type orchState struct {
	values      map[uint32]float32
	reassembler *SegmentReassembler
}

type outboundBuilder struct {
	kind  string
	build func() ([]byte, error)
}

func OrchestratorExample(cmd *cobra.Command, args []string) {
	if orchPeriod < 1 {
		log.Fatalf("invalid --period %d: must be >= 1", orchPeriod)
	}

	if orchShutdownAfter < 0 {
		log.Fatalf("invalid --shutdown-after %s: must be >= 0", orchShutdownAfter)
	}

	remoteUDPAddr, err := net.ResolveUDPAddr("udp", rAddr)
	if err != nil {
		log.Fatalf("failed to resolve remote UDP address: %v", err)
	}

	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("couldn't get interface information: %v", err)
	}

	listenConn, err := net.ListenPacket("udp", bAddr)
	if err != nil {
		log.Fatalf("failed to begin listening for packets: %v", err)
	}
	defer listenConn.Close()

	state := newOrchestratorState()
	builders, err := buildOutboundPlan(orchSend)
	if err != nil {
		log.Fatalf("invalid --send plan: %v", err)
	}

	log.Printf("orchestrator listening on %s and targeting %s", bAddr, remoteUDPAddr)
	log.Printf("IAm configured with: \n\tInstance: %d \n\tVendor: %d \n\tObjectType: %d", orchDeviceInst, orchVendorID, orchObjectType)
	if len(builders) == 0 {
		log.Print("no outbound plan configured; running in listen/respond mode")
	}

	replyRaw := make([]byte, 4096)
	ticker := time.NewTicker(time.Duration(orchPeriod) * time.Second)
	defer ticker.Stop()

	var shutdownC <-chan time.Time
	if orchShutdownAfter > 0 {
		shutdownTimer := time.NewTimer(orchShutdownAfter)
		defer shutdownTimer.Stop()
		shutdownC = shutdownTimer.C
		log.Printf("auto-shutdown configured for %s", orchShutdownAfter)
	}

	sentCycles := 0
	for {
		if len(builders) > 0 && (orchN == 0 || sentCycles < orchN) {
			sendOutboundCycle(listenConn, remoteUDPAddr, builders)
			sentCycles++
		}

		if len(builders) > 0 && orchN > 0 && sentCycles >= orchN && !orchStayAlive {
			log.Printf("send cycles complete (%d); exiting", sentCycles)
			return
		}

		handleInboundWindow(listenConn, ifaceAddrs, state, replyRaw)

		if shutdownC != nil {
			select {
			case <-shutdownC:
				log.Printf("shutdown timer reached (%s); exiting", orchShutdownAfter)
				return
			default:
			}
		}

		if len(builders) == 0 || (orchN > 0 && sentCycles >= orchN) {
			continue
		}

	waitForNextCycle:
		for {
			select {
			case <-ticker.C:
				break waitForNextCycle
			case <-shutdownC:
				log.Printf("shutdown timer reached (%s); exiting", orchShutdownAfter)
				return
			default:
			}
			handleInboundWindow(listenConn, ifaceAddrs, state, replyRaw)
		}
	}
}

func newOrchestratorState() *orchState {
	values := make(map[uint32]float32, int(orchStateInstances))
	for i := uint16(0); i < orchStateInstances; i++ {
		values[uint32(i)] = orchDefaultValue * float32(i+1)
	}
	return &orchState{values: values, reassembler: NewSegmentReassembler(segmentReassemblyTTL)}
}

func buildOutboundPlan(kinds []string) ([]outboundBuilder, error) {
	builders := make([]outboundBuilder, 0, len(kinds))
	for _, kindRaw := range kinds {
		kind := strings.ToLower(strings.TrimSpace(kindRaw))
		if kind == "" {
			continue
		}

		builder, err := buildOutbound(kind)
		if err != nil {
			return nil, err
		}
		builders = append(builders, builder)
	}
	return builders, nil
}

func buildOutbound(kind string) (outboundBuilder, error) {
	switch kind {
	case "whois":
		return outboundBuilder{kind: kind, build: bacnet.NewWhois}, nil
	case "iam":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewIAm(orchDeviceInst, orchVendorID)
		}}, nil
	case "rp":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewReadProperty(orchObjectType, orchInstanceID, orchPropertyID)
		}}, nil
	case "rpm":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewReadPropertyMultiple(orchObjectType, orchInstanceID, uint16slice(orchProperties))
		}}, nil
	case "rr":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewReadRange(orchObjectType, orchInstanceID, orchPropertyID, orchRangeStart, orchRangeLength)
		}}, nil
	case "wp":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewWriteProperty(orchObjectType, orchInstanceID, orchPropertyID, wpPayload())
		}}, nil
	case "cov":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewSubscribeCOV(
				orchObjectType,
				orchInstanceID,
				orchCOVProcessID,
				orchCOVLifetime,
				orchCOVExpectConfirmed,
				orchCOVCancel,
			)
		}}, nil
	case "sack":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewSACK(services.ServiceConfirmedWriteProperty)
		}}, nil
	case "cack":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewCACK(
				services.ServiceConfirmedReadProperty,
				orchObjectType,
				orchInstanceID,
				orchPropertyID,
				orchDefaultValue,
			)
		}}, nil
	case "err":
		return outboundBuilder{kind: kind, build: func() ([]byte, error) {
			return bacnet.NewError(
				services.ServiceConfirmedReadProperty,
				objects.ErrorClassServices,
				objects.ErrorCodeServiceRequestDenied,
			)
		}}, nil
	case "rfdt":
		return outboundBuilder{kind: kind, build: bacnet.NewReadForeignDeviceTable}, nil
	default:
		return outboundBuilder{}, fmt.Errorf("unsupported outbound kind %q", kind)
	}
}

func wpPayload() interface{} {
	switch strings.ToLower(strings.TrimSpace(orchWPKind)) {
	case "string":
		return orchWPString
	default:
		return orchWPReal
	}
}

func sendOutboundCycle(conn net.PacketConn, target net.Addr, builders []outboundBuilder) {
	for _, b := range builders {
		payload, err := b.build()
		if err != nil {
			log.Printf("send[%s] build failed: %v", b.kind, err)
			continue
		}
		if _, err := conn.WriteTo(payload, target); err != nil {
			log.Printf("send[%s] write failed: %v", b.kind, err)
			continue
		}
		log.Printf("sent[%s]: %x", b.kind, payload)
	}
}

func handleInboundWindow(conn net.PacketConn, ifaceAddrs []net.Addr, state *orchState, buffer []byte) {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(350 * time.Millisecond)); err != nil {
			log.Printf("could not set deadline: %v", err)
			return
		}

		nBytes, remoteAddr, err := conn.ReadFrom(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return
			}
			log.Printf("read error: %v", err)
			return
		}

		if orchIgnoreLocal && common.IsLocalAddr(ifaceAddrs, remoteAddr) {
			continue
		}

		log.Printf("read %d bytes from %s: %x", nBytes, remoteAddr, buffer[:nBytes])
		dumpRawPacket(orchDumpDir, remoteAddr, buffer[:nBytes])
		handleIncoming(conn, remoteAddr, state, buffer[:nBytes])
	}
}

func handleIncoming(conn net.PacketConn, remoteAddr net.Addr, state *orchState, payload []byte) {
	err := DecodeAndDispatch(payload, MessageDispatcher{
		OnParseError: func(raw []byte, parseErr error) error {
			logUnsupportedPacket(remoteAddr, raw, parseErr)
			return nil
		},
		OnParsed: func(msg plumbing.BACnet) error {
			log.Printf("incoming type=%d service=%d from=%s", msg.GetType(), msg.GetService(), remoteAddr)
			return nil
		},
		OnUnconfirmedWhoIs: func(msg *services.UnconfirmedWhoIs) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode WhoIs failed: %v", err)
				return nil
			}

			if !shouldReplyToWhoIs(decoded, orchDeviceInst) {
				log.Printf("ignoring WhoIs for non-matching device range (our instance=%d)", orchDeviceInst)
				return nil
			}
			sendIAmReply(conn, remoteAddr)
			return nil
		},
		OnUnconfirmedIAm: func(msg *services.UnconfirmedIAm) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode IAm failed: %v", err)
				return nil
			}
			printIAm(&decoded)
			return nil
		},
		OnConfirmedReadProperty: func(msg *services.ConfirmedReadProperty) error {
			handleConfirmedReadProperty(conn, remoteAddr, state, msg)
			return nil
		},
		OnConfirmedReadPropertyMultiple: func(msg *services.ConfirmedReadProperty) error {
			decoded, err := msg.DecodeRPM()
			if err != nil {
				log.Printf("decode ReadPropertyMultiple failed: %v", err)
				sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadPropMultiple, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
				return nil
			}
			log.Printf("received ReadPropertyMultiple request objectType=%d instanceID=%d with %d tags", decoded.ObjectType, decoded.InstanceNum, len(decoded.Tags))
			sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadPropMultiple, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
			return nil
		},
		OnConfirmedWriteProperty: func(msg *services.ConfirmedWriteProperty) error {
			handleConfirmedWriteProperty(conn, remoteAddr, state, msg)
			return nil
		},
		OnConfirmedReadRange: func(msg *services.ConfirmedReadRange) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode ReadRange failed: %v", err)
			}
			log.Printf("received ReadRange request objectType=%d instanceID=%d propertyID=%d", decoded.ObjectType, decoded.InstanceNum, decoded.PropertyId)
			sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadRange, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
			return nil
		},
		OnConfirmedSubscribeCOV: func(msg *services.ConfirmedCOV) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode SubscribeCOV failed: %v", err)
			}
			log.Printf(
				"received SubscribeCOV processID=%d objectType=%d instanceID=%d expectConfirmed=%t lifetime=%d",
				decoded.ProcessId,
				decoded.MonitoredObjType,
				decoded.MonitoredInstNum,
				decoded.ExpectConfirmed,
				decoded.Lifetime,
			)
			sendSimpleAckReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedSubscribeCOV)
			return nil
		},
		OnCOVNotification: func(msg *services.COVNotification) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode COV notification failed: %v", err)
				return nil
			}
			printCOVNot(&decoded)
			if msg.GetType() == plumbing.ConfirmedReq {
				sendSimpleAckReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedCOVNotification)
			}
			return nil
		},
		OnComplexACK: func(msg *services.ComplexACK) error {
			if msg.APDU.IsSegmented() {
				handleSegment(conn, remoteAddr, state, msg.BVLC, msg.NPDU, msg.APDU)
				return nil
			}
			handleComplexAck(msg)
			return nil
		},
		OnSegmentedAPDU: func(msg *services.SegmentedAPDU) error {
			handleSegment(conn, remoteAddr, state, msg.BVLC, msg.NPDU, msg.APDU)
			return nil
		},
		OnSimpleACK: func(msg *services.SimpleACK) error {
			log.Printf("SimpleACK service=%d invokeID=%d", msg.APDU.Service, msg.APDU.InvokeID)
			return nil
		},
		OnError: func(msg *services.Error) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode Error failed: %v", err)
				return nil
			}
			log.Printf("Error service=%d class=%d code=%d", msg.APDU.Service, decoded.ErrorClass, decoded.ErrorCode)
			return nil
		},
		OnReject: func(msg *services.Reject) error {
			printReject(msg)
			return nil
		},
		OnAbort: func(msg *services.Abort) error {
			printAbort(msg)
			return nil
		},
		OnSegmentAck: func(msg *services.SegmentACK) error {
			log.Printf(
				"received SegmentAck invokeID=%d seq=%d windowSize=%d negativeAck=%t server=%t",
				msg.APDU.InvokeID, msg.APDU.SequenceNumber, msg.APDU.ActualWindowSize,
				msg.APDU.IsNegativeAck(), msg.APDU.IsServerAck(),
			)
			return nil
		},
		OnReadForeignDeviceTableAck: func(msg *plumbing.ReadForeignDeviceTableAck) error {
			printFDT(msg)
			return nil
		},
		OnBVLCResult: func(msg *plumbing.BVLCResult) error {
			printBVLCResult(msg)
			return nil
		},
		OnUnconfirmedWhoHas: func(msg *services.UnconfirmedWhoHas) error {
			decoded, err := msg.Decode()
			if err != nil {
				log.Printf("decode WhoHas failed: %v", err)
				return nil
			}
			if decoded.ObjectID != nil {
				log.Printf("received WhoHas for objectType=%d instanceID=%d", decoded.ObjectID.ObjectType, decoded.ObjectID.InstanceNumber)
			} else {
				log.Printf("received WhoHas for object name %q", decoded.ObjectName)
			}
			return nil
		},
		OnNetworkMessage: func(msg *plumbing.NetworkMessage) error {
			if nets, err := msg.Networks(); err == nil {
				log.Printf("received network-layer message type=0x%02x networks=%v from=%s", msg.Type, nets, remoteAddr)
				return nil
			}
			log.Printf("received network-layer message type=0x%02x data=%x from=%s", msg.Type, msg.Data, remoteAddr)
			return nil
		},
		OnUnknown: func(msg plumbing.BACnet) error {
			log.Printf("received parsed message with unhandled concrete type %T", msg)
			return nil
		},
	})
	if err != nil {
		log.Printf("dispatch error from %s: %v", remoteAddr, err)
	}
}

func logUnsupportedPacket(remoteAddr net.Addr, payload []byte, parseErr error) {
	if len(payload) < 4 {
		log.Printf("dropped short packet from %s (%d bytes): %v", remoteAddr, len(payload), parseErr)
		return
	}

	bvlcFunc := payload[1]
	bvlcLen := int(payload[2])<<8 | int(payload[3])
	npduOffset := 4
	if bvlcFunc == plumbing.BVLCFuncForwardedNPDU {
		npduOffset += 6
	}

	if len(payload) < npduOffset+2 {
		log.Printf("dropped packet from %s (bvlcFunc=0x%02x, bvlcLen=%d): incomplete NPDU, parse error: %v", remoteAddr, bvlcFunc, bvlcLen, parseErr)
		return
	}

	npduControl := payload[npduOffset+1]
	apduOffset, npduErr := inferAPDUOffsetLenient(payload, npduOffset)
	if npduErr != nil {
		log.Printf("ignored malformed NPDU from %s (bvlcFunc=0x%02x, npduControl=0x%02x): %v (parse error: %v)", remoteAddr, bvlcFunc, npduControl, npduErr, parseErr)
		return
	}

	networkLayer := (npduControl & 0x80) != 0

	if networkLayer {
		if len(payload) > apduOffset {
			log.Printf("ignored network-layer message from %s (bvlcFunc=0x%02x, npduControl=0x%02x, nlType=0x%02x)", remoteAddr, bvlcFunc, npduControl, payload[apduOffset])
			return
		}
		log.Printf("ignored network-layer message from %s (bvlcFunc=0x%02x, npduControl=0x%02x)", remoteAddr, bvlcFunc, npduControl)
		return
	}

	if len(payload) <= apduOffset {
		log.Printf("dropped packet from %s (bvlcFunc=0x%02x, npduControl=0x%02x): missing APDU, parse error: %v", remoteAddr, bvlcFunc, npduControl, parseErr)
		return
	}

	pduType := payload[apduOffset] >> 4
	service := uint8(0)
	knownService := false
	switch pduType {
	case plumbing.UnConfirmedReq:
		if len(payload) > apduOffset+1 {
			service = payload[apduOffset+1]
			knownService = true
		}
	case plumbing.ConfirmedReq:
		if len(payload) > apduOffset+3 {
			service = payload[apduOffset+3]
			knownService = true
		}
	case plumbing.SimpleAck, plumbing.ComplexAck, plumbing.Error:
		if len(payload) > apduOffset+2 {
			service = payload[apduOffset+2]
			knownService = true
		}
	}

	if knownService {
		log.Printf("ignored unsupported APDU from %s (bvlcFunc=0x%02x, pduType=%d, service=%d): %v", remoteAddr, bvlcFunc, pduType, service, parseErr)
		return
	}

	log.Printf("ignored unsupported APDU from %s (bvlcFunc=0x%02x, pduType=%d): %v", remoteAddr, bvlcFunc, pduType, parseErr)
}

func inferAPDUOffsetLenient(payload []byte, npduOffset int) (int, error) {
	if len(payload) < npduOffset+2 {
		return 0, fmt.Errorf("NPDU header too short")
	}

	control := payload[npduOffset+1]
	offset := npduOffset + 2

	if (control & 0x20) != 0 {
		if len(payload) < offset+3 {
			return 0, fmt.Errorf("NPDU destination fields too short")
		}
		offset += 2 // DNET
		dlen := int(payload[offset])
		offset++
		if len(payload) < offset+dlen {
			return 0, fmt.Errorf("NPDU DADR too short")
		}
		offset += dlen // DADR
	}

	if (control & 0x08) != 0 {
		if len(payload) < offset+3 {
			return 0, fmt.Errorf("NPDU source fields too short")
		}
		offset += 2 // SNET
		slen := int(payload[offset])
		offset++
		if len(payload) < offset+slen {
			return 0, fmt.Errorf("NPDU SADR too short")
		}
		offset += slen // SADR
	}

	// The hop count follows the addresses, and only a destination brings one.
	if (control & 0x20) != 0 {
		if len(payload) <= offset {
			return 0, fmt.Errorf("NPDU hop too short")
		}
		offset++ // Hop
	}

	return offset, nil
}

func handleSegment(conn net.PacketConn, remoteAddr net.Addr, state *orchState, bvlc *plumbing.BVLC, npdu *plumbing.NPDU, apdu *plumbing.APDU) {
	dumpSegment(orchDumpDir, remoteAddr, apdu)

	raw, complete, accepted := state.reassembler.Feed(remoteAddr, bvlc, npdu, apdu)
	if accepted {
		sendSegmentAck(conn, remoteAddr, apdu)
	}
	if !complete {
		return
	}
	log.Printf(
		"reassembled %d-byte message from %d segment(s) (invokeID=%d) from %s",
		len(raw), apdu.SequenceNumber+1, apdu.InvokeID, remoteAddr,
	)
	dumpReassembled(orchDumpDir, raw, apdu.InvokeID)
	handleIncoming(conn, remoteAddr, state, raw)
}

// dumpRawPacket writes the complete, unparsed bytes of one incoming UDP
// datagram to a hex-encoded text file, exactly as read from the socket -
// before any BVLC/NPDU/APDU parsing touches them. This is the ground truth
// for verifying our own header-offset assumptions independently of our own
// parser, which can't validate itself. A no-op if dir is empty.
func dumpRawPacket(dir string, remoteAddr net.Addr, raw []byte) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("dump raw packet: mkdir %s failed: %v", dir, err)
		return
	}

	name := fmt.Sprintf("raw-%s.hex", time.Now().Format("20060102-150405.000000"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(hex.EncodeToString(raw)), 0o644); err != nil {
		log.Printf("dump raw packet: write %s failed: %v", path, err)
		return
	}
	log.Printf("dumped raw packet (%d bytes) from %s to %s", len(raw), remoteAddr, path)
}

// dumpSegment writes one arriving segment's raw Payload to a hex-encoded
// text file, named with a microsecond timestamp plus invoke/sequence
// numbers so every delivery - including retransmitted duplicates - gets its
// own file in strict arrival order. Lets duplicate deliveries of the "same"
// segment be diffed byte-for-byte instead of guessed at. A no-op if dir is
// empty.
func dumpSegment(dir string, remoteAddr net.Addr, apdu *plumbing.APDU) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("dump segment: mkdir %s failed: %v", dir, err)
		return
	}

	name := fmt.Sprintf(
		"segment-%s-invoke%d-seq%d-more%t.hex",
		time.Now().Format("20060102-150405.000000"), apdu.InvokeID, apdu.SequenceNumber, apdu.HasMoreSegments(),
	)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(hex.EncodeToString(apdu.Payload)), 0o644); err != nil {
		log.Printf("dump segment: write %s failed: %v", path, err)
		return
	}
	log.Printf(
		"dumped segment seq=%d (%d bytes payload, more=%t) from %s to %s",
		apdu.SequenceNumber, len(apdu.Payload), apdu.HasMoreSegments(), remoteAddr, path,
	)
}

// dumpReassembled writes the raw bytes of a fully reassembled message to a
// hex-encoded text file in dir, named with a timestamp and invoke ID so
// repeated runs don't collide. A no-op if dir is empty.
func dumpReassembled(dir string, raw []byte, invokeID uint8) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("dump reassembled: mkdir %s failed: %v", dir, err)
		return
	}

	name := fmt.Sprintf("reassembled-%s-invoke%d.hex", time.Now().Format("20060102-150405.000000"), invokeID)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(hex.EncodeToString(raw)), 0o644); err != nil {
		log.Printf("dump reassembled: write %s failed: %v", path, err)
		return
	}
	log.Printf("dumped reassembled message (%d bytes) to %s", len(raw), path)
}

// sendSegmentAck acknowledges one received segment so the sender continues
// with its next window. server reflects the role of the ack-sender (us),
// not of whoever sent the segment: acking a ComplexAck means we're the
// client receiving a reply to our own request (server=false); acking a
// ConfirmedReq means we're the server receiving a request (server=true).
func sendSegmentAck(conn net.PacketConn, target net.Addr, apdu *plumbing.APDU) {
	server := apdu.Type == plumbing.ConfirmedReq
	reply, err := bacnet.NewSegmentAck(apdu.InvokeID, apdu.SequenceNumber, apdu.ProposedWindowSize, false, server)
	if err != nil {
		log.Printf("build SegmentAck failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(reply, target); err != nil {
		log.Printf("send SegmentAck failed: %v", err)
		return
	}
	log.Printf(
		"sent SegmentAck invokeID=%d seq=%d windowSize=%d to %s",
		apdu.InvokeID, apdu.SequenceNumber, apdu.ProposedWindowSize, target,
	)
}

func handleConfirmedReadProperty(conn net.PacketConn, remoteAddr net.Addr, state *orchState, msg *services.ConfirmedReadProperty) {
	decoded, err := msg.Decode()
	if err != nil {
		log.Printf("decode ReadProperty failed: %v", err)
		sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadProperty, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
		return
	}

	value, ok := state.values[decoded.InstanceNum]
	if !ok {
		sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadProperty, objects.ErrorClassObject, objects.ErrorCodeUnknownObject)
		return
	}

	reply, err := bacnet.NewCACKReply(
		msg.APDU.InvokeID,
		services.ServiceConfirmedReadProperty,
		decoded.ObjectType,
		decoded.InstanceNum,
		decoded.PropertyId,
		value,
	)
	if err != nil {
		log.Printf("build CACK failed: %v", err)
		sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedReadProperty, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
		return
	}

	if _, err := conn.WriteTo(reply, remoteAddr); err != nil {
		log.Printf("sending CACK failed: %v", err)
		return
	}
	log.Printf("replied with CACK for instance %d value=%f", decoded.InstanceNum, value)
}

func handleConfirmedWriteProperty(conn net.PacketConn, remoteAddr net.Addr, state *orchState, msg *services.ConfirmedWriteProperty) {
	decoded, err := msg.Decode()
	if err != nil {
		log.Printf("decode WriteProperty failed: %v", err)
		sendErrorReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedWriteProperty, objects.ErrorClassServices, objects.ErrorCodeServiceRequestDenied)
		return
	}

	if !extractAndStoreValue(state, decoded) {
		log.Printf("WriteProperty payload had no storable real value, preserving previous state for instance %d", decoded.InstanceNum)
	}

	sendSimpleAckReply(conn, remoteAddr, msg.APDU.InvokeID, services.ServiceConfirmedWriteProperty)
}

func extractAndStoreValue(state *orchState, decoded services.ConfirmedWritePropertyDec) bool {
	for _, tag := range decoded.Tags {
		if tag.TagNumber != objects.TagReal {
			continue
		}

		value, ok := tag.Value.(float32)
		if !ok {
			continue
		}

		state.values[decoded.InstanceNum] = value
		log.Printf("stored instance=%d value=%f", decoded.InstanceNum, value)
		return true
	}
	return false
}

func handleComplexAck(msg *services.ComplexACK) {
	switch msg.APDU.Service {
	case services.ServiceConfirmedReadPropMultiple:
		decoded, err := msg.DecodeRPM()
		if err != nil {
			log.Printf("decode RPM CACK failed: %v", err)
			return
		}
		printPropM(&decoded)
	case services.ServiceConfirmedReadRange:
		decoded, err := msg.DecodeRR()
		if err != nil {
			log.Printf("decode RR CACK failed: %v", err)
			return
		}
		printLogBuffer(&decoded)
	default:
		decoded, err := msg.Decode()
		if err != nil {
			log.Printf("decode CACK failed: %v", err)
			return
		}
		printCACK(&decoded)
	}
}

func sendIAmReply(conn net.PacketConn, target net.Addr) {
	reply, err := bacnet.NewIAm(orchDeviceInst, orchVendorID)
	if err != nil {
		log.Printf("build IAm failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(reply, target); err != nil {
		log.Printf("send IAm failed: %v", err)
		return
	}
	log.Printf("replied with IAm to %s", target)
}

func sendSimpleAckReply(conn net.PacketConn, target net.Addr, invokeID uint8, service uint8) {
	reply, err := bacnet.NewSACKReply(invokeID, service)
	if err != nil {
		log.Printf("build SACK failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(reply, target); err != nil {
		log.Printf("send SACK failed: %v", err)
		return
	}
	log.Printf("replied with SACK service=%d invokeID=%d to %s", service, invokeID, target)
}

func sendErrorReply(conn net.PacketConn, target net.Addr, invokeID uint8, service uint8, class uint8, code uint8) {
	reply, err := bacnet.NewErrorReply(invokeID, service, class, code)
	if err != nil {
		log.Printf("build Error failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(reply, target); err != nil {
		log.Printf("send Error failed: %v", err)
		return
	}
	log.Printf("replied with Error service=%d invokeID=%d class=%d code=%d to %s", service, invokeID, class, code, target)
}

func shouldReplyToWhoIs(decoded services.UnconfirmedWhoIsDec, deviceInstance uint32) bool {
	if len(decoded.Tags) == 0 {
		return true
	}

	var (
		hasLow  bool
		hasHigh bool
		low     uint32
		high    uint32
	)

	for _, tag := range decoded.Tags {
		switch tag.TagNumber {
		case 0:
			v, ok := coerceUint32(tag.Value)
			if !ok {
				return false
			}
			low = v
			hasLow = true
		case 1:
			v, ok := coerceUint32(tag.Value)
			if !ok {
				return false
			}
			high = v
			hasHigh = true
		}
	}

	if !hasLow && !hasHigh {
		return true
	}

	if !hasLow || !hasHigh {
		return false
	}

	if low > high {
		return false
	}

	return deviceInstance >= low && deviceInstance <= high
}

func coerceUint32(v interface{}) (uint32, bool) {
	switch n := v.(type) {
	case uint8:
		return uint32(n), true
	case uint16:
		return uint32(n), true
	case uint32:
		return n, true
	case uint64:
		if n > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(n), true
	case int:
		if n < 0 {
			return 0, false
		}
		return uint32(n), true
	case int32:
		if n < 0 {
			return 0, false
		}
		return uint32(n), true
	case int64:
		if n < 0 || n > int64(^uint32(0)) {
			return 0, false
		}
		return uint32(n), true
	default:
		return 0, false
	}
}
