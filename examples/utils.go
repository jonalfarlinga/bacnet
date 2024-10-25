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
		out = fmt.Sprintf(
			"%s\tTag %d:\n\t\tAppTag Type: %s\n\t\tValue: %v\n\t\tBinary Length: %d\n",
			out, i, objects.TagToString(t.TagNumber), t.Value, t.Length,
		)
	}
	log.Print(out)
}
