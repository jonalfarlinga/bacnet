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
			"\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %+v\n\t\tBinary Length: %d\n",
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
			"\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %+v\n\t\tData Length: %d\n",
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

func printPropM(d *services.ComplexACKRPMDec) {
	out := "Decoded CACK reply:\n"

	out += fmt.Sprintf(
		"\n\tObject Type: %d\n\tInstance Id: %d\n",
		d.ObjectType, d.InstanceId,
	)

	property := 0
	for _, t := range d.Tags {
		if t.TagClass && t.TagNumber == 2 {
			property++
			out += fmt.Sprintf("\n\t%d - ", property)
			propInt, ok := t.Value.(uint16)
			if ok {
				out += fmt.Sprintf("%s\n", objects.PropertyMap[propInt])
			}
		} else {
			out += fmt.Sprintf(
				"\n\t\tAppTag Type: %s\n\t\tValue: %+v\n\t\tBinary Length: %d\n",
				objects.TagToString(t), t.Value, t.Length,
			)
		}
	}
	log.Print(out)
}

func printCOVNot(d *services.UnconfirmedCOVNotificationDec) {
	out := "Decoded COV Notification:\n"

	out += fmt.Sprintf(
		"\n\tDevice Type: %d\n\tInstance Id: %d",
		d.DeviceType, d.DeviceId,
	)
	out += fmt.Sprintf(
		"\n\tMonitored Object Type: %d\n\tMonitored Instance Id: %d\n",
		d.ObjectType, d.ObjectID,
	)
	out += fmt.Sprintf(
		"\n\tProcess Id: %d\tLifetime: %d secs", d.ProcessId, d.Lifetime,
	)
	property := 0
	for _, t := range d.Tags {
		if t.TagClass && t.TagNumber == 0 {
			property++
			out += fmt.Sprintf("\n\t%d - ", property)
			propInt, ok := t.Value.(uint16)
			if ok {
				out += fmt.Sprintf("%s\n", objects.PropertyMap[propInt])
			}
		} else {
			out += fmt.Sprintf(
				"\n\t\tAppTag Type: %s\n\t\tValue: %+v\n\t\tBinary Length: %d\n",
				objects.TagToString(t), t.Value, t.Length,
			)
		}
	}
	log.Print(out)
}
