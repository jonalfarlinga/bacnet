package main

import (
	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/plumbing"
	"github.com/jonalfarlinga/bacnet/services"
)

// MessageDispatcher defines callbacks for parsed BACnet messages.
// Any callback may be nil when the caller does not care about that message class.
type MessageDispatcher struct {
	OnParseError                    func(payload []byte, parseErr error) error
	OnParsed                        func(msg plumbing.BACnet) error
	OnUnconfirmedWhoIs              func(msg *services.UnconfirmedWhoIs) error
	OnUnconfirmedIAm                func(msg *services.UnconfirmedIAm) error
	OnConfirmedReadProperty         func(msg *services.ConfirmedReadProperty) error
	OnConfirmedReadPropertyMultiple func(msg *services.ConfirmedReadProperty) error
	OnConfirmedWriteProperty        func(msg *services.ConfirmedWriteProperty) error
	OnConfirmedReadRange            func(msg *services.ConfirmedReadRange) error
	OnConfirmedSubscribeCOV         func(msg *services.ConfirmedCOV) error
	OnCOVNotification               func(msg *services.COVNotification) error
	OnComplexACK                    func(msg *services.ComplexACK) error
	OnSimpleACK                     func(msg *services.SimpleACK) error
	OnError                         func(msg *services.Error) error
	OnReject                        func(msg *services.Reject) error
	OnAbort                         func(msg *services.Abort) error
	OnSegmentAck                    func(msg *services.SegmentACK) error
	OnReadForeignDeviceTableAck     func(msg *plumbing.ReadForeignDeviceTableAck) error
	OnBVLCResult                    func(msg *plumbing.BVLCResult) error
	OnUnconfirmedWhoHas             func(msg *services.UnconfirmedWhoHas) error
	OnNetworkMessage                func(msg *plumbing.NetworkMessage) error
	OnSegmentedAPDU                 func(msg *services.SegmentedAPDU) error
	OnUnknown                       func(msg plumbing.BACnet) error
}

// DecodeAndDispatch parses raw BACnet bytes and dispatches the resulting message.
func DecodeAndDispatch(payload []byte, dispatcher MessageDispatcher) error {
	msg, err := bacnet.ParseWithFallback(payload)
	if err != nil {
		if dispatcher.OnParseError != nil {
			return dispatcher.OnParseError(payload, err)
		}
		return err
	}

	return DispatchMessage(msg, dispatcher)
}

// DispatchMessage routes an already parsed BACnet message to the appropriate callback.
func DispatchMessage(msg plumbing.BACnet, dispatcher MessageDispatcher) error {
	if dispatcher.OnParsed != nil {
		if err := dispatcher.OnParsed(msg); err != nil {
			return err
		}
	}

	switch m := msg.(type) {
	case *services.UnconfirmedWhoIs:
		if dispatcher.OnUnconfirmedWhoIs != nil {
			return dispatcher.OnUnconfirmedWhoIs(m)
		}
	case *services.UnconfirmedIAm:
		if dispatcher.OnUnconfirmedIAm != nil {
			return dispatcher.OnUnconfirmedIAm(m)
		}
	case *services.ConfirmedReadProperty:
		switch m.APDU.Service {
		case services.ServiceConfirmedReadProperty:
			if dispatcher.OnConfirmedReadProperty != nil {
				return dispatcher.OnConfirmedReadProperty(m)
			}
		case services.ServiceConfirmedReadPropMultiple:
			if dispatcher.OnConfirmedReadPropertyMultiple != nil {
				return dispatcher.OnConfirmedReadPropertyMultiple(m)
			}
		}
	case *services.ConfirmedWriteProperty:
		if dispatcher.OnConfirmedWriteProperty != nil {
			return dispatcher.OnConfirmedWriteProperty(m)
		}
	case *services.ConfirmedReadRange:
		if dispatcher.OnConfirmedReadRange != nil {
			return dispatcher.OnConfirmedReadRange(m)
		}
	case *services.ConfirmedCOV:
		if dispatcher.OnConfirmedSubscribeCOV != nil {
			return dispatcher.OnConfirmedSubscribeCOV(m)
		}
	case *services.COVNotification:
		if dispatcher.OnCOVNotification != nil {
			return dispatcher.OnCOVNotification(m)
		}
	case *services.ComplexACK:
		if dispatcher.OnComplexACK != nil {
			return dispatcher.OnComplexACK(m)
		}
	case *services.SimpleACK:
		if dispatcher.OnSimpleACK != nil {
			return dispatcher.OnSimpleACK(m)
		}
	case *services.Error:
		if dispatcher.OnError != nil {
			return dispatcher.OnError(m)
		}
	case *services.Reject:
		if dispatcher.OnReject != nil {
			return dispatcher.OnReject(m)
		}
	case *services.Abort:
		if dispatcher.OnAbort != nil {
			return dispatcher.OnAbort(m)
		}
	case *services.SegmentACK:
		if dispatcher.OnSegmentAck != nil {
			return dispatcher.OnSegmentAck(m)
		}
	case *plumbing.ReadForeignDeviceTableAck:
		if dispatcher.OnReadForeignDeviceTableAck != nil {
			return dispatcher.OnReadForeignDeviceTableAck(m)
		}
	case *plumbing.BVLCResult:
		if dispatcher.OnBVLCResult != nil {
			return dispatcher.OnBVLCResult(m)
		}
	case *services.UnconfirmedWhoHas:
		if dispatcher.OnUnconfirmedWhoHas != nil {
			return dispatcher.OnUnconfirmedWhoHas(m)
		}
	case *plumbing.NetworkMessage:
		if dispatcher.OnNetworkMessage != nil {
			return dispatcher.OnNetworkMessage(m)
		}
	case *services.SegmentedAPDU:
		if dispatcher.OnSegmentedAPDU != nil {
			return dispatcher.OnSegmentedAPDU(m)
		}
	}

	if dispatcher.OnUnknown != nil {
		return dispatcher.OnUnknown(msg)
	}

	return nil
}
