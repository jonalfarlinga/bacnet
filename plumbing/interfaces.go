// Copyright 2020 bacnet authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package plumbing

// BACnet is an interface defines BACnet messages.
type BACnet interface {
	MarshalBinary() ([]byte, error)
	MarshalTo([]byte) error
	UnmarshalBinary([]byte) error
	MarshalLen() int
	GetType() uint8
	GetService() uint8
}
