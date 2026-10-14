// Package objects holds BACnet tags and values. Object is one application or
// context tag with its data; the Dec* functions decode the primitive data
// types (Null, Boolean, Unsigned, Signed, Real, Double, OctetString,
// CharacterString, BitString, Enumerated, Date, Time and ObjectIdentifier),
// the Enc* functions encode all of them but dates and times, and the opening
// and closing tags of constructed values have encoders of their own. The
// names of object types, properties and engineering units are looked up by
// their numbers.
package objects
