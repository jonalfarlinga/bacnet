// Package plumbing marshals and unmarshals the layers under a BACnet service:
// the BVLC header of BACnet/IP, with its result codes and the functions a
// foreign device and a BBMD exchange (the registration, the foreign device
// and broadcast distribution tables, and the origin of a forwarded NPDU); the
// NPDU, with the source and destination network addresses of a routed
// message and the network-layer messages such as Who-Is-Router-To-Network;
// and the APDU, with the invoke ID, the service choice and the segmentation
// fields.
package plumbing
