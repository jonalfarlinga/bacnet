package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonalfarlinga/bacnet/client"
	"github.com/jonalfarlinga/bacnet/client/sim"
	"github.com/jonalfarlinga/bacnet/objects"
)

func TestReadRange(t *testing.T) {
	network, err := sim.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(network.Close)
	c, heard := openOn(t, network, nil)
	devices := awaitDevices(t, c, heard, 1001)
	ahu := devices[1001]
	ctx := context.Background()
	trendLog := client.ObjectID{Type: objects.ObjectTypeTrendLog, Instance: 0}

	// The first ten records, by position.
	res, err := c.ReadRange(ctx, ahu.Addr, client.ReadRangeRequest{
		Object: trendLog, Property: objects.PropertyIdLogBuffer, Range: client.ByPosition(1, 10),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Object != trendLog || res.Property != uint32(objects.PropertyIdLogBuffer) {
		t.Errorf("the reply is for %v property %d", res.Object, res.Property)
	}
	// MoreItems would say the device sent fewer than asked; it sent them all.
	if res.ItemCount != 10 || !res.FirstItem || res.LastItem || res.MoreItems {
		t.Errorf("count %d, first %t, last %t, more %t; want 10 items including the first, not the last, none held back",
			res.ItemCount, res.FirstItem, res.LastItem, res.MoreItems)
	}
	records := res.LogRecords()
	if len(records) != 10 {
		t.Fatalf("%d records, want 10", len(records))
	}
	for i, r := range records {
		if r.Datum.Kind != client.KindReal {
			t.Errorf("record %d logged %+v, want a real", i, r.Datum)
		}
		if len(r.Status) != 4 {
			t.Errorf("record %d has status flags %v, want 4", i, r.Status)
		}
		if i > 0 && !r.At.After(records[i-1].At) {
			t.Errorf("record %d at %v is not after record %d at %v", i, r.At, i-1, records[i-1].At)
		}
	}
	if gap := records[1].At.Sub(records[0].At); gap != 5*time.Minute {
		t.Errorf("records are %s apart, want 5 minutes", gap)
	}

	// The last five, counting backwards from the end, by sequence number.
	last, err := c.ReadRange(ctx, ahu.Addr, client.ReadRangeRequest{
		Object: trendLog, Property: objects.PropertyIdLogBuffer, Range: client.BySequence(sim.LogRecords, -5),
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.ItemCount != 5 || !last.LastItem || last.FirstItem || last.MoreItems {
		t.Errorf("the last five: %+v", last)
	}
	if last.FirstSequence == nil || *last.FirstSequence != sim.LogRecords-4 {
		t.Errorf("first sequence number %v, want %d", last.FirstSequence, sim.LogRecords-4)
	}
	if recs := last.LogRecords(); len(recs) != 5 || !recs[4].At.After(records[9].At) {
		t.Errorf("the last records are %+v", recs)
	}

	// The whole buffer at once.
	all, err := c.ReadRange(ctx, ahu.Addr, client.ReadRangeRequest{Object: trendLog, Property: objects.PropertyIdLogBuffer})
	if err != nil {
		t.Fatal(err)
	}
	if all.ItemCount != sim.LogRecords || !all.FirstItem || !all.LastItem || all.MoreItems {
		t.Errorf("the whole buffer: count %d flags %t %t %t", all.ItemCount, all.FirstItem, all.LastItem, all.MoreItems)
	}

	// A property that is not a list is the device's error.
	_, err = c.ReadRange(ctx, ahu.Addr, client.ReadRangeRequest{
		Object: trendLog, Property: objects.PropertyIdObjectName, Range: client.ByPosition(1, 1),
	})
	var be *client.BACnetError
	if !errors.As(err, &be) {
		t.Errorf("a ReadRange of a name gave %v, want the device's error", err)
	}
	if _, err := c.ReadRange(ctx, ahu.Addr, client.ReadRangeRequest{Object: trendLog, Range: client.ByPosition(1, 0)}); err == nil {
		t.Error("a range of no items was sent")
	}
}
