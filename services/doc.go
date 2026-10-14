// Package services encodes and decodes BACnet services as messages, each a
// BVLC, NPDU and APDU with the service's parameters as tags: Who-Is and I-Am,
// Who-Has, ReadProperty, ReadPropertyMultiple, ReadRange, WriteProperty,
// SubscribeCOV and the COV notifications, and the replies: SimpleACK,
// ComplexACK, SegmentACK, Error, Reject and Abort. The *Objects functions
// build a service's tags for the New* functions of the root package, and the
// Decode* methods take the tags of a parsed message apart.
package services
