package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonalfarlinga/bacnet/objects"
	"github.com/jonalfarlinga/bacnet/services"
)

// RangeKind says how a ReadRange names the items it wants.
type RangeKind uint8

const (
	// RangeAll asks for every item: a whole log buffer. Few devices allow
	// it for a long one.
	RangeAll RangeKind = iota
	// RangePosition counts items from a position in the list, the first
	// being 1.
	RangePosition
	// RangeSequence counts items from a sequence number, which a log
	// gives each record for its lifetime.
	RangeSequence
)

// Range is the part of a list a ReadRange asks for: Count items from
// Start, forwards when Count is positive and backwards when negative.
type Range struct {
	Kind  RangeKind
	Start uint32
	Count int32
}

// ByPosition is a Range of count items from a position, the first being 1.
func ByPosition(start uint32, count int32) Range {
	return Range{Kind: RangePosition, Start: start, Count: count}
}

// BySequence is a Range of count items from a sequence number.
func BySequence(start uint32, count int32) Range {
	return Range{Kind: RangeSequence, Start: start, Count: count}
}

// ReadRangeRequest asks for part of a list property, such as a trend log's
// log-buffer.
type ReadRangeRequest struct {
	Object   ObjectID
	Property uint16
	Index    *uint32 // array index, for a list that is one element of an array
	Range    Range
}

// ReadRangeResult is what a ReadRange returned.
type ReadRangeResult struct {
	Object   ObjectID
	Property uint32
	Index    *uint32
	// FirstItem and LastItem say whether the first and the last item of
	// the whole list are among the items; MoreItems that the device cut the
	// reply short of what was asked.
	FirstItem, LastItem, MoreItems bool
	ItemCount                      uint32
	// Items is the item data as sent: for a log-buffer, three values a
	// record, which LogRecords groups.
	Items []Value
	// FirstSequence is the sequence number of the first item, when the
	// device gives it.
	FirstSequence *uint32
}

// LogRecord is one record of a trend log's log-buffer.
type LogRecord struct {
	// At is when the record was logged; zero when the timestamp has a
	// wildcard in it.
	At time.Time
	// Datum is what was logged: a value of the usual kinds; KindBits with
	// Tag 0 for a log-status; KindConstructed for a failure (Tag 8) or a
	// value of any type (Tag 10).
	Datum Value
	// Status is the status flags logged with the value, nil when absent.
	Status []bool
}

// ReadRange reads part of a list property.
func (c *Client) ReadRange(ctx context.Context, to Address, req ReadRangeRequest) (ReadRangeResult, error) {
	payload := []objects.APDUPayload{
		objects.EncObjectIdentifier(true, 0, req.Object.Type, req.Object.Instance),
		objects.ContextTag(1, objects.EncUnsignedInteger(uint(req.Property))),
	}
	if req.Index != nil {
		payload = append(payload, objects.ContextTag(2, objects.EncUnsignedInteger(uint(*req.Index))))
	}
	var tag uint8
	switch req.Range.Kind {
	case RangeAll:
	case RangePosition:
		tag = 3
	case RangeSequence:
		tag = 6
	default:
		return ReadRangeResult{}, fmt.Errorf("range kind %d is not one the client sends", req.Range.Kind)
	}
	if tag != 0 {
		if req.Range.Count == 0 {
			return ReadRangeResult{}, errors.New("a range counts at least one item")
		}
		payload = append(payload,
			objects.EncOpeningTag(tag),
			objects.EncUnsignedInteger(uint(req.Range.Start)),
			objects.EncSignedInteger(int(req.Range.Count)),
			objects.EncClosingTag(tag),
		)
	}

	tags, err := c.confirmed(ctx, to, services.ServiceConfirmedReadRange, payload)
	if err != nil {
		return ReadRangeResult{}, err
	}
	res, err := decodeReadRange(tags)
	if err != nil {
		return ReadRangeResult{}, fmt.Errorf("decoding the reply: %w", err)
	}
	return res, nil
}

// decodeReadRange decodes a ReadRange ack: object [0], property [1], perhaps
// an index [2], result flags [3], item count [4], the items between tags 5,
// and perhaps the first sequence number [6].
func decodeReadRange(tags []*objects.Object) (ReadRangeResult, error) {
	var res ReadRangeResult
	if len(tags) < 4 || !isContext(tags[0], 0) || len(tags[0].Data) != 4 {
		return res, errors.New("expected an object identifier first")
	}
	res.Object = objectIDOf(tags[0].Data)
	tags = tags[1:]
	items := false
	for len(tags) > 0 {
		t := tags[0]
		switch {
		case isContext(t, 1):
			res.Property = uint32(beUint(t.Data))
		case isContext(t, 2):
			index := uint32(beUint(t.Data))
			res.Index = &index
		case isContext(t, 3):
			if len(t.Data) < 2 {
				return res, errors.New("the result flags are too short")
			}
			res.FirstItem = t.Data[1]&0x80 != 0
			res.LastItem = t.Data[1]&0x40 != 0
			res.MoreItems = t.Data[1]&0x20 != 0
		case isContext(t, 4):
			res.ItemCount = uint32(beUint(t.Data))
		case isOpening(t, 5):
			values, rest, err := decodeUntilClose(tags[1:])
			if err != nil {
				return res, err
			}
			if len(rest) == 0 || rest[0].TagNumber != 5 {
				return res, errors.New("the items are not closed")
			}
			res.Items = values
			items = true
			tags = rest
		case isContext(t, 6):
			seq := uint32(beUint(t.Data))
			res.FirstSequence = &seq
		default:
			return res, fmt.Errorf("unexpected tag %d in the reply", t.TagNumber)
		}
		tags = tags[1:]
	}
	if !items {
		return res, errors.New("no items in the reply")
	}
	return res, nil
}

// LogRecords groups the items of a log-buffer into records: a timestamp
// [0], the datum [1] and perhaps status flags [2] each.
func (r ReadRangeResult) LogRecords() []LogRecord {
	var records []LogRecord
	for _, v := range r.Items {
		switch {
		case v.Kind == KindConstructed && v.Tag == 0:
			rec := LogRecord{}
			if len(v.Items) == 2 {
				rec.At, _ = dateTimeOf(v.Items[0], v.Items[1])
			}
			records = append(records, rec)
		case len(records) == 0:
			continue // items before any timestamp are not a record
		case v.Kind == KindConstructed && v.Tag == 1:
			if len(v.Items) == 1 {
				records[len(records)-1].Datum = logDatum(v.Items[0])
			}
		case v.Kind == KindContext && v.Tag == 2:
			records[len(records)-1].Status = bitsOf(v.Bytes)
		}
	}
	return records
}

// logDatum gives a log record's datum, a choice by context tag, as the
// value it holds.
func logDatum(v Value) Value {
	if v.Kind != KindContext {
		return v
	}
	appTag := map[uint8]uint8{
		0: objects.TagBitString, 1: objects.TagBoolean, 2: objects.TagReal, 3: objects.TagEnumerated,
		4: objects.TagUnsignedInteger, 5: objects.TagSignedInteger, 6: objects.TagBitString,
		7: objects.TagNull, 9: objects.TagReal,
	}
	tag, ok := appTag[v.Tag]
	if !ok {
		return v
	}
	if tag == objects.TagBoolean {
		return Value{Kind: KindBoolean, Bool: len(v.Bytes) == 1 && v.Bytes[0] != 0}
	}
	out := decodePrimitive(&objects.Object{TagNumber: tag, Data: v.Bytes, Length: uint32(len(v.Bytes))})
	if v.Tag == 0 {
		out.Tag = 0 // a log-status, told apart from a bit string by its tag
	}
	return out
}

// bitsOf decodes a bit string: the unused bits of the last byte, then the
// bits.
func bitsOf(data []byte) []bool {
	if len(data) < 1 {
		return nil
	}
	unused := int(data[0])
	n := (len(data)-1)*8 - unused
	if n < 0 {
		return nil
	}
	bits := make([]bool, n)
	for i := range bits {
		bits[i] = data[1+i/8]&(0x80>>(i%8)) != 0
	}
	return bits
}

// dateTimeOf makes a time of a Date and a Time value, in the local zone.
// It is false when either has a wildcard in it.
func dateTimeOf(date, clock Value) (time.Time, bool) {
	if date.Kind != KindDate || clock.Kind != KindTime || len(date.Bytes) != 4 || len(clock.Bytes) != 4 {
		return time.Time{}, false
	}
	for _, b := range date.Bytes[:3] {
		if b == 0xFF {
			return time.Time{}, false
		}
	}
	for _, b := range clock.Bytes {
		if b == 0xFF {
			return time.Time{}, false
		}
	}
	d, c := date.Bytes, clock.Bytes
	return time.Date(int(d[0])+1900, time.Month(d[1]), int(d[2]),
		int(c[0]), int(c[1]), int(c[2]), int(c[3])*10_000_000, time.Local), true
}
