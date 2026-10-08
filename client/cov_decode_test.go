package client

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet"
	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// A confirmed COV notification as the bench panel sent it: process 7107,
// analog value 3 of device 101, 300 seconds left, a present value of 0 and
// status flags with nothing set.
func TestDecodeCOV(t *testing.T) {
	raw, err := hex.DecodeString("810a002c01040205f8010a1bc31c020000652c008000033a012c" +
		"4e09552e44000000002f096f2e8204002f4f")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := bacnet.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	tags, err := toTags(msg.(*services.COVNotification).APDU.Objects)
	if err != nil {
		t.Fatal(err)
	}
	n, err := decodeCOV(tags)
	if err != nil {
		t.Fatal(err)
	}
	if n.ProcessID != 7107 || n.Device != 101 || n.Remaining != 300*time.Second ||
		n.Object != (ObjectID{Type: objects.ObjectTypeAnalogValue, Instance: 3}) {
		t.Errorf("decoded %+v, want process 7107, device 101, analog value 3 and 300 seconds", n)
	}
	if len(n.Props) != 2 {
		t.Fatalf("%d properties, want 2", len(n.Props))
	}
	if p := n.Props[0]; p.ID != uint32(objects.PropertyIdPresentValue) || len(p.Values) != 1 || p.Values[0].Kind != KindReal || p.Values[0].Float != 0 {
		t.Errorf("first property is %+v, want a present value of 0", p)
	}
	if p := n.Props[1]; p.ID != uint32(objects.PropertyIdStatusFlags) || len(p.Values) != 1 || p.Values[0].Kind != KindBits || len(p.Values[0].Bits) != 4 {
		t.Errorf("second property is %+v, want status flags of 4 bits", p)
	}

	for name, cut := range map[string][]*objects.Object{
		"no values":        tags[:4],
		"unclosed values":  tags[:len(tags)-1],
		"unclosed value":   tags[:8],
		"no property name": tags[6:],
	} {
		if _, err := decodeCOV(cut); err == nil {
			t.Errorf("%s: decoded without an error", name)
		}
	}
}
