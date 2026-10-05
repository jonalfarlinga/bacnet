package plumbing

import (
	"encoding/binary"
	"fmt"

	"github.com/jonalfarlinga/bacnet/common"
)

// NetworkLayerMessage is what NetworkMessage.GetType returns. It is the NPDU
// control bit that marks a network layer message, and no APDU type has it.
const NetworkLayerMessage uint8 = 0x80

// Network layer message types (ASHRAE 135 clause 6.2.4).
const (
	NetMsgWhoIsRouterToNetwork     uint8 = 0x00
	NetMsgIAmRouterToNetwork       uint8 = 0x01
	NetMsgICouldBeRouterToNetwork  uint8 = 0x02
	NetMsgRejectMessageToNetwork   uint8 = 0x03
	NetMsgRouterBusyToNetwork      uint8 = 0x04
	NetMsgRouterAvailableToNetwork uint8 = 0x05

	// netMsgProprietary is the first of the vendor proprietary message
	// types, which carry a vendor ID after the type.
	netMsgProprietary uint8 = 0x80
)

// NetworkMessage is a network layer message: an NPDU whose control octet has
// bit 7 set, followed by a message type and its data instead of an APDU.
type NetworkMessage struct {
	*BVLC
	*NPDU
	Type uint8
	// VendorID is on the wire only for proprietary types (0x80 and above).
	VendorID uint16
	// Data is everything after the type (and vendor ID). Networks and
	// RejectReason decode it for the types that the standard defines.
	Data []byte
}

// NewNetworkMessage creates a NetworkMessage. It sets the network layer
// message bit of npdu.
func NewNetworkMessage(bvlc *BVLC, npdu *NPDU, msgType uint8, data []byte) *NetworkMessage {
	npdu.Control |= NetworkLayerMessage
	m := &NetworkMessage{BVLC: bvlc, NPDU: npdu, Type: msgType, Data: data}
	m.SetLength()
	return m
}

// NetworksData encodes a list of network numbers, the data of
// Who-Is-Router-To-Network (none or one), I-Am-Router-To-Network,
// Router-Busy-To-Network and Router-Available-To-Network.
func NetworksData(nets ...uint16) []byte {
	if len(nets) == 0 {
		return nil
	}
	data := make([]byte, 2*len(nets))
	for i, n := range nets {
		binary.BigEndian.PutUint16(data[2*i:], n)
	}
	return data
}

// UnmarshalBinary sets the values retrieved from byte sequence in a
// NetworkMessage frame.
func (m *NetworkMessage) UnmarshalBinary(b []byte) error {
	bvlc := &BVLC{}
	if err := bvlc.UnmarshalBinary(b); err != nil {
		return fmt.Errorf("unmarshalling NetworkMessage: %w", err)
	}
	offset := bvlc.MarshalLen()

	npdu := &NPDU{}
	if err := npdu.UnmarshalBinary(b[offset:]); err != nil {
		return fmt.Errorf("unmarshalling NetworkMessage: %w", err)
	}
	offset += npdu.MarshalLen()

	if npdu.Control&NetworkLayerMessage == 0 {
		return fmt.Errorf(
			"unmarshalling NetworkMessage - NPDU control %#x carries an APDU: %w",
			npdu.Control, common.ErrInvalidData,
		)
	}
	if len(b) <= offset {
		return fmt.Errorf(
			"unmarshalling NetworkMessage - missing message type at offset %d: %w",
			offset, common.ErrTooShortToParse,
		)
	}
	msgType := b[offset]
	offset++

	var vendorID uint16
	if msgType >= netMsgProprietary {
		if len(b) < offset+2 {
			return fmt.Errorf(
				"unmarshalling NetworkMessage - missing vendor ID at offset %d: %w",
				offset, common.ErrTooShortToParse,
			)
		}
		vendorID = binary.BigEndian.Uint16(b[offset:])
		offset += 2
	}

	m.BVLC, m.NPDU = bvlc, npdu
	m.Type, m.VendorID = msgType, vendorID
	m.Data = append([]byte(nil), b[offset:]...)
	return nil
}

// MarshalBinary returns the byte sequence generated from a NetworkMessage
// instance.
func (m *NetworkMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, m.MarshalLen())
	if err := m.MarshalTo(b); err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %v", err)
	}
	return b, nil
}

// MarshalTo puts the byte sequence in the byte array given as b.
func (m *NetworkMessage) MarshalTo(b []byte) error {
	if len(b) < m.MarshalLen() {
		return fmt.Errorf(
			"failed to marshal NetworkMessage - marshal length %d binary length %d: %v",
			m.MarshalLen(), len(b),
			common.ErrTooShortToMarshalBinary,
		)
	}
	offset := 0
	if err := m.BVLC.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling NetworkMessage: %v", err)
	}
	offset += m.BVLC.MarshalLen()

	if err := m.NPDU.MarshalTo(b[offset:]); err != nil {
		return fmt.Errorf("marshalling NetworkMessage: %v", err)
	}
	offset += m.NPDU.MarshalLen()

	b[offset] = m.Type
	offset++
	if m.Type >= netMsgProprietary {
		binary.BigEndian.PutUint16(b[offset:], m.VendorID)
		offset += 2
	}
	copy(b[offset:], m.Data)
	return nil
}

// MarshalLen returns the serial length of NetworkMessage.
func (m *NetworkMessage) MarshalLen() int {
	l := m.BVLC.MarshalLen() + m.NPDU.MarshalLen() + 1 + len(m.Data)
	if m.Type >= netMsgProprietary {
		l += 2
	}
	return l
}

// SetLength sets the length in Length field.
func (m *NetworkMessage) SetLength() {
	m.BVLC.Length = uint16(m.MarshalLen())
}

// GetType returns NetworkLayerMessage: the message has no APDU type.
func (m *NetworkMessage) GetType() uint8 {
	return NetworkLayerMessage
}

// GetService returns the network layer message type.
func (m *NetworkMessage) GetService() uint8 {
	return m.Type
}

// Networks decodes Data as a list of network numbers. It applies to
// Who-Is-Router-To-Network (an empty list asks about every network),
// I-Am-Router-To-Network, Router-Busy-To-Network and
// Router-Available-To-Network.
func (m *NetworkMessage) Networks() ([]uint16, error) {
	switch m.Type {
	case NetMsgWhoIsRouterToNetwork:
		if len(m.Data) != 0 && len(m.Data) != 2 {
			return nil, fmt.Errorf(
				"Who-Is-Router-To-Network with %d data bytes: %w", len(m.Data), common.ErrInvalidData,
			)
		}
	case NetMsgIAmRouterToNetwork, NetMsgRouterBusyToNetwork, NetMsgRouterAvailableToNetwork:
		if len(m.Data)%2 != 0 {
			return nil, fmt.Errorf(
				"network list of %d bytes: %w", len(m.Data), common.ErrInvalidData,
			)
		}
	default:
		return nil, fmt.Errorf(
			"network layer message type %#x has no network list: %w", m.Type, common.ErrWrongStructure,
		)
	}

	nets := make([]uint16, len(m.Data)/2)
	for i := range nets {
		nets[i] = binary.BigEndian.Uint16(m.Data[2*i:])
	}
	return nets, nil
}

// RejectReason decodes the Data of a Reject-Message-To-Network: why the
// router rejected a message, and the network it was addressed to.
func (m *NetworkMessage) RejectReason() (reason uint8, dnet uint16, err error) {
	if m.Type != NetMsgRejectMessageToNetwork {
		return 0, 0, fmt.Errorf(
			"network layer message type %#x is not a reject: %w", m.Type, common.ErrWrongStructure,
		)
	}
	if len(m.Data) != 3 {
		return 0, 0, fmt.Errorf(
			"Reject-Message-To-Network with %d data bytes: %w", len(m.Data), common.ErrInvalidData,
		)
	}
	return m.Data[0], binary.BigEndian.Uint16(m.Data[1:]), nil
}
