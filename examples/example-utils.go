package main

import (
	"fmt"
	"log"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

func printCACK(d *services.ComplexACKDec) {
	out := "Decoded CACK reply:\n"

	out += fmt.Sprintf(
		"\n\tObject Type: %d\n\tInstance Id: %d\n\tProperty Id: %d\n",
		d.ObjectType, d.InstanceId, d.PropertyId,
	)
	for i, t := range d.Tags {
		out += fmt.Sprintf(
			"\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %v\n\t\tBinary Length: %d\n",
			i, objects.TagToString(t), t.Value, t.Length,
		)
	}
	log.Print(out)
}

func printLogBuffer(d *services.LogBufferCACKDec) {
	out := "Decoded CACK reply:\n"

	out += fmt.Sprintf(
		"\n\tObject Type: %d\n\tInstance Id: %d\n\tProperty Id: %d\n",
		d.ObjectType, d.InstanceId, d.PropertyId,
	)
	out += fmt.Sprintf(
		"\tFirst Item: %t\n\tLast Item: %t\n\tMore Items: %t\n",
		d.FirstItem, d.LastItem, d.MoreItems,
	)
	out += fmt.Sprintf(
		"\tItem Count: %d\n",
		d.ItemCount,
	)
	for i, t := range d.Tags {
		out += fmt.Sprintf(
			"\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %v\n\t\tData Length: %d\n",
			i, objects.TagToString(t), t.Value, t.Length,
		)
	}
	log.Print(out)
}

func printIAm(d *services.UnconfirmedIAmDec) {
	out := "Decoded IAm reply:\n"

	out += fmt.Sprintf(
		"\n\tObject Type: %d\n\tInstance Id: %d\n\tMax APDU Length: %d\n",
		d.DeviceType, d.InstanceNumber, d.MaxAPDULength,
	)
	out += fmt.Sprintf(
		"\tSegmentation Supported: %d\n\tVendor Id: %d\n",
		d.SegmentationSupported, d.VendorId,
	)
	log.Print(out)
}
