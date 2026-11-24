package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/objects"
)

// serveDevice makes a client with an identity readable: it answers
// ReadProperty and ReadPropertyMultiple for a Device object, the only object
// the example has. The client carries the requests and the replies; what the
// device is, is decided here.
func serveDevice(c *client.Client, instance uint32, vendor uint16, name string) {
	supported := make([]bool, 40)
	supported[12], supported[14] = true, true // readProperty, readPropertyMultiple
	supported[26], supported[34] = true, true // i-Am, who-Is
	types := make([]bool, 30)
	types[objects.ObjectTypeDevice] = true

	self := objects.EncObjectIdentifier(false, objects.TagBACnetObjectIdentifier, objects.ObjectTypeDevice, instance)
	one := func(t *objects.Object) []*objects.Object { return []*objects.Object{t} }
	props := map[uint32][]*objects.Object{
		uint32(objects.PropertyIdObjectIdentifier):             one(self),
		uint32(objects.PropertyIdObjectName):                   one(objects.EncString(name)),
		uint32(objects.PropertyIdObjectType):                   one(objects.EncEnumerated(uint32(objects.ObjectTypeDevice))),
		uint32(objects.PropertyIdSystemStatus):                 one(objects.EncEnumerated(0)), // operational
		uint32(objects.PropertyIdVendorName):                   one(objects.EncString("bacnet examples")),
		uint32(objects.PropertyIdVendorIdentifier):             one(objects.EncUnsignedInteger(uint(vendor))),
		uint32(objects.PropertyIdModelName):                    one(objects.EncString("bacnet-examples")),
		uint32(objects.PropertyIdFirmwareRevision):             one(objects.EncString("0")),
		uint32(objects.PropertyIdApplicationSoftwareVersion):   one(objects.EncString("0")),
		uint32(objects.PropertyIdProtocolVersion):              one(objects.EncUnsignedInteger(1)),
		uint32(objects.PropertyIdProtocolRevision):             one(objects.EncUnsignedInteger(14)),
		uint32(objects.PropertyIdProtocolServicesSupported):    one(objects.EncBitString(supported)),
		uint32(objects.PropertyIdProtocolObjectTypesSupported): one(objects.EncBitString(types)),
		uint32(objects.PropertyIdObjectList):                   one(self),
		uint32(objects.PropertyIdMaxApduLengthAccepted):        one(objects.EncUnsignedInteger(1476)),
		uint32(objects.PropertyIdSegmentationSupported):        one(objects.EncEnumerated(3)), // none
		uint32(objects.PropertyIdApduTimeout):                  one(objects.EncUnsignedInteger(3000)),
		uint32(objects.PropertyIdNumberOfApduRetries):          one(objects.EncUnsignedInteger(1)),
		uint32(objects.PropertyIdDeviceAddressBinding):         {},
		uint32(objects.PropertyIdDatabaseRevision):             one(objects.EncUnsignedInteger(0)),
	}

	read := func(_ context.Context, object client.ObjectID, property client.PropertyRef) ([]*objects.Object, error) {
		value, err := func() ([]*objects.Object, error) {
			if object.Type != objects.ObjectTypeDevice || object.Instance != instance {
				return nil, &client.BACnetError{Class: uint32(objects.ErrorClassObject), Code: client.ErrorCodeUnknownObject}
			}
			value, ok := props[property.ID]
			if !ok {
				return nil, &client.BACnetError{Class: uint32(objects.ErrorClassProperty), Code: client.ErrorCodeUnknownProperty}
			}
			if property.Index == nil {
				return value, nil
			}
			// Only the object list is an array: element 0 is its length.
			const propertyIsNotAnArray, invalidArrayIndex = 50, 42
			switch {
			case property.ID != uint32(objects.PropertyIdObjectList):
				return nil, &client.BACnetError{Class: uint32(objects.ErrorClassProperty), Code: propertyIsNotAnArray}
			case *property.Index == 0:
				return one(objects.EncUnsignedInteger(uint(len(value)))), nil
			case int(*property.Index) > len(value):
				return nil, &client.BACnetError{Class: uint32(objects.ErrorClassProperty), Code: invalidArrayIndex}
			}
			return value[*property.Index-1 : *property.Index], nil
		}()
		answer := "answered"
		if err != nil {
			answer = err.Error()
		}
		fmt.Printf("%s read of %s %d %s: %s\n", time.Now().Format("15:04:05.000"),
			objectTypeName(object.Type), object.Instance, propertyName(property.ID), answer)
		return value, err
	}
	if err := c.Handle(client.ServiceReadProperty, client.ServeReadProperty(read)); err != nil {
		log.Fatalf("serving ReadProperty: %v", err)
	}
	if err := c.Handle(client.ServiceReadPropertyMultiple, client.ServeReadPropertyMultiple(read)); err != nil {
		log.Fatalf("serving ReadPropertyMultiple: %v", err)
	}
}
