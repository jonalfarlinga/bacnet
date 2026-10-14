// Package bacnet is a BACnet/IP implementation: the New* functions build the
// datagrams of the services, and Parse decodes a received datagram into the
// message it holds.
//
// The layers come from the packages under it: plumbing marshals the BVLC,
// NPDU and APDU, objects holds the tags and the primitive values they carry,
// and services puts them together as the service messages. The client
// package is a BACnet/IP client built on them, and client/sim simulates
// devices for tests. Only BACnet/IP over UDP is supported.
package bacnet
