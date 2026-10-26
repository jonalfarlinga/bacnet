package client

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// Knock asks whether a BACnet device is at an address, and which: one read
// of the Device object's identifier through the wildcard instance, sent once
// and not again. It is for trying addresses where there may be nobody. An
// address that does not answer fails with ErrTimeout, and the client keeps
// nothing about it. A device that answers and will not say which it is, as
// one older than the wildcard instance does, fails with an error that
// Declined reports.
func (c *Client) Knock(ctx context.Context, to netip.AddrPort) (uint32, error) {
	addr := Address{IP: to}
	tags, err := c.exchange(ctx, addr, services.ServiceConfirmedReadProperty,
		services.ConfirmedReadPropertyObjects(objects.ObjectTypeDevice, WildcardInstance, objects.PropertyIdObjectIdentifier), sendProbe)
	if err != nil {
		return 0, err
	}
	values, err := propertyValue(tags)
	if err != nil {
		return 0, err
	}
	if len(values) != 1 || values[0].Kind != KindObjectID || values[0].Object.Type != objects.ObjectTypeDevice {
		return 0, fmt.Errorf("the device's identifier is %+v, not a Device object", values)
	}
	instance := values[0].Object.Instance
	c.learnInstance(instance, addr)
	return instance, nil
}

// Probe learns what an I-Am would have said about the device at an address,
// by reading it: for a device that a Who-Is cannot reach, on another IP
// subnet, or one that does not announce itself. With instance 0 the device's
// own instance is read first, through the wildcard instance.
func (c *Client) Probe(ctx context.Context, to Address, instance uint32) (Device, error) {
	dev := Device{Instance: instance, Addr: to}
	if instance == 0 || instance == WildcardInstance {
		values, err := c.ReadProperty(ctx, to, ObjectID{Type: objects.ObjectTypeDevice, Instance: WildcardInstance}, objects.PropertyIdObjectIdentifier)
		if err != nil {
			return dev, fmt.Errorf("reading the device's identifier: %w", err)
		}
		if len(values) != 1 || values[0].Kind != KindObjectID || values[0].Object.Type != objects.ObjectTypeDevice {
			return dev, fmt.Errorf("the device's identifier is %+v, not a Device object", values)
		}
		dev.Instance = values[0].Object.Instance
	}

	props := []uint16{
		objects.PropertyIdMaxApduLengthAccepted, objects.PropertyIdSegmentationSupported, objects.PropertyIdVendorIdentifier,
	}
	obj := ObjectID{Type: objects.ObjectTypeDevice, Instance: dev.Instance}
	got := map[uint16]uint64{}

	results, err := c.ReadPropertyMultiple(ctx, to, []ReadSpec{{Object: obj, Props: props}})
	switch {
	case err == nil:
		for _, r := range results {
			for _, p := range r.Props {
				if p.Err == nil && len(p.Values) == 1 && (p.Values[0].Kind == KindUnsigned || p.Values[0].Kind == KindEnumerated) {
					got[uint16(p.ID)] = p.Values[0].Uint
				}
			}
		}
	case Declined(err):
		// No ReadPropertyMultiple: one request each.
		for _, prop := range props {
			values, err := c.ReadProperty(ctx, to, obj, prop)
			if err != nil && !Declined(err) {
				return dev, err
			}
			if err == nil && len(values) == 1 && (values[0].Kind == KindUnsigned || values[0].Kind == KindEnumerated) {
				got[prop] = values[0].Uint
			}
		}
	default:
		return dev, err
	}

	// What the device would not say is left at what an I-Am could not say
	// either: the smallest APDU and no segmentation, which every read then
	// fits.
	dev.MaxAPDU = 50
	dev.Segmentation = 3
	if v, ok := got[objects.PropertyIdMaxApduLengthAccepted]; ok && v > 0 && v <= 0xFFFF {
		dev.MaxAPDU = uint16(v)
	}
	if v, ok := got[objects.PropertyIdSegmentationSupported]; ok && v <= 3 {
		dev.Segmentation = uint8(v)
	}
	if v, ok := got[objects.PropertyIdVendorIdentifier]; ok && v <= 0xFFFF {
		dev.VendorID = uint16(v)
	}
	if len(got) == 0 {
		return dev, errors.New("the device answered none of the reads that describe it")
	}
	c.learnInstance(dev.Instance, to)
	return dev, nil
}
