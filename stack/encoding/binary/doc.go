// Package binary provide functionality for encoding Go types into OPC UA
// binary representation as defined by
// [IEC-62541](https://webstore.iec.ch/webstore/webstore.nsf/mysearchajax?Openform&key=62541).
// Encode and decode functionality only aims to fully handle types that
// are defined in the uatype package, and can not be expected to handle all
// Go types. As an example, int and uint types without a bit size suffix are
// deliberately not handled.
package binary
