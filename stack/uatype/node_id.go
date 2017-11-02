//go:generate csv2code -o node_id_auto.go -csv ../../schemas/1.03/NodeIds.csv node_id_auto.go.tmpl
//go:generate opcua-xml2code -t=nodes -o node_id_info_auto.go ../../schemas/1.03/Opc.Ua.NodeSet.xml
//go:generate gofmt -s -w node_id_auto.go

package uatype

import (
	"fmt"
)

const (
	// DefaultNamespaceIndex is defined to always refer to the OPC UA name
	// space, which includes all types defined in the OPC UA Specification
	// itself.
	DefaultNamespaceIndex = 0

	// DefaultNamespaceURI defines the NodeID name space used to describe all
	// OPC UA types defined in the OPC UA specification.
	//
	// TODO: Verify from spec, e.g. with or without trailing slash? Is it
	// canonically defined at all?
	DefaultNamespaceURI = "http://opcfoundation.org/UA/"
)

// TODO: We probably want a custom implementation of NodeId, ExpandedNodeId
// and Variant, or alter the code-generation for these types to include Marshal
// Unmarshal methods along with and an interface-based switch value.

// NewTwoByteNodeID returns a NodeId of type TwoByte
func NewTwoByteNodeID(id uint16) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeTwoByte,
		TwoByte: TwoByteNodeId{
			Identifier: uint8(id),
		},
	}
}

// NewFourByteNodeID returns a NodeId of type FourByte
func NewFourByteNodeID(ns uint8, id uint16) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeFourByte,
		FourByte: FourByteNodeId{
			NamespaceIndex: ns,
			Identifier:     id,
		},
	}
}

// NewNumericNodeID returns a NodeId of type Numeric
func NewNumericNodeID(ns uint16, id uint32) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeNumeric,
		Numeric: NumericNodeId{
			NamespaceIndex: ns,
			Identifier:     id,
		},
	}
}

// NewStringNodeID returns a NodeId of type String
func NewStringNodeID(ns uint16, id string) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeString,
		String: StringNodeId{
			NamespaceIndex: ns,
			Identifier:     id,
		},
	}
}

// NewGuidNodeID returns a NodeId of type GUID
func NewGuidNodeID(ns uint16, id Guid) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeGuid,
		Guid: GuidNodeId{
			NamespaceIndex: ns,
			Identifier:     id,
		},
	}
}

// NewByteStringNodeID returns a NodeId of type ByteString
func NewByteStringNodeID(ns uint16, id ByteString) NodeId {
	return NodeId{
		NodeIdType: NodeIdTypeByteString,
		ByteString: ByteStringNodeId{
			NamespaceIndex: ns,
			Identifier:     id,
		},
	}
}

// Class will return the node class of two-byte, four-byte or numeric node IDs.
// Other node ID types will always return NodeClassUnspecified.
func (nid NodeId) Class() NodeClass {
	id := nid.Uint()
	if id == 0 {
		return NodeClassUnspecified
	}
	return nodeInfoMap[id].class
}

// DisplayName will return a human readable name for the node ID. String,
// byte-string and GUID node IDs will return a serialized version of the
// identifier. Will fallback to return a string representation of the identifier.
func (nid NodeId) DisplayName() string {
	switch nid.NodeIdType {
	case NodeIdTypeString:
		return nid.String.Identifier
	case NodeIdTypeGuid:
		return fmt.Sprintf("GUID:%s", nid.Guid.Identifier)
	case NodeIdTypeByteString:
		return string(nid.ByteString.Identifier)
	default:
		id := nid.Uint()
		if nid.NamespaceIndex() == 0 {
			if s := nodeInfoMap[id].displayName; s != "" && nid.NamespaceIndex() == 0 {
				return s
			}
		}
		return fmt.Sprintf("i:%d", id)
	}

}

// Description returns a description for two-byte, four-byte or numeric node
// IDs or the empty string.
func (nid NodeId) Description() string {
	if i := nid.NamespaceIndex(); i == 0 {
		// Ignore the error since a 0 lookups gives the empty string.
		return nodeInfoMap[nid.Uint()].description
	}
	// Not OPC UA namespace, a description lookup must be done against the
	// server.
	return ""
}

// Uint returns the identifier of two-byte, four-byte or numeric node IDs as
// uint16. Calling this method on other types of node IDs will return 0.
func (nid NodeId) Uint() uint16 {
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		return uint16(nid.TwoByte.Identifier)
	case NodeIdTypeFourByte:
		return uint16(nid.FourByte.Identifier)
	case NodeIdTypeNumeric:
		return uint16(nid.Numeric.Identifier)
	default:
		return 0
	}
}

// NamespaceIndex returns the namespace index of nid as uint16.
func (nid NodeId) NamespaceIndex() uint16 {
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		// The name-space index of TwoByte nodeIDs are defined as 0 according
		// to the spec.
		return 0
	case NodeIdTypeFourByte:
		return uint16(nid.FourByte.NamespaceIndex)
	case NodeIdTypeNumeric:
		return nid.Numeric.NamespaceIndex
	case NodeIdTypeString:
		return nid.String.NamespaceIndex
	case NodeIdTypeGuid:
		return nid.Guid.NamespaceIndex
	case NodeIdTypeByteString:
		return nid.ByteString.NamespaceIndex
	default:
		// Invalid NodeIdType ignored.
		return 0
	}
}

// Expanded returns nid as an ExpandedNodeId.
func (nid NodeId) Expanded() ExpandedNodeId {
	return ExpandedNodeId{
		NodeIdType: nid.NodeIdType,
		TwoByte:    nid.TwoByte,
		FourByte:   nid.FourByte,
		Numeric:    nid.Numeric,
		String:     nid.String,
		Guid:       nid.Guid,
		ByteString: nid.ByteString,
	}
}

// Uint returns the identifier of two-byte, four-byte or numeric node IDs as
// uint16. Calling this method on other types of node IDs will return 0.
func (nid ExpandedNodeId) Uint() uint16 {
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		return uint16(nid.TwoByte.Identifier)
	case NodeIdTypeFourByte:
		return uint16(nid.FourByte.Identifier)
	case NodeIdTypeNumeric:
		return uint16(nid.Numeric.Identifier)
	default:
		return 0
	}
}

// NamespaceIndex returns the namespace index of nid and true, or 0 and
// false if the namespace index should be ignored.
func (nid ExpandedNodeId) NamespaceIndex() (uint16, bool) {
	if nid.NamespaceURISpecified {
		// NamespaceIndex should be ignored if a namespace URI is given.
		return 0, false
	}
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		// The name-space index of TwoByte nodeIDs are defined as 0 according
		// to the spec.
		return 0, true
	case NodeIdTypeFourByte:
		return uint16(nid.FourByte.NamespaceIndex), true
	case NodeIdTypeNumeric:
		return nid.Numeric.NamespaceIndex, true
	case NodeIdTypeString:
		return nid.String.NamespaceIndex, true
	case NodeIdTypeGuid:
		return nid.Guid.NamespaceIndex, true
	case NodeIdTypeByteString:
		return nid.ByteString.NamespaceIndex, true
	default:
		// Invalid NodeIdType.
		return 0, false
	}
}

// DisplayName will return a human readable name for the node ID. String,
// byte-string and GUID node IDs will return a serialized version of the
// identifier. Will fallback to return a string representation of the identifier.
func (nid ExpandedNodeId) DisplayName() string {
	switch nid.NodeIdType {
	case NodeIdTypeString:
		return nid.String.Identifier
	case NodeIdTypeGuid:
		return fmt.Sprintf("GUID:%s", nid.Guid.Identifier)
	case NodeIdTypeByteString:
		return string(nid.ByteString.Identifier)
	default:
		// Ignore the error since a zero lookups should give the empty string.
		id := nid.Uint()
		if i, ok := nid.NamespaceIndex(); (ok && i == 0) || (!ok && nid.NamespaceURI == DefaultNamespaceURI) {
			if s := nodeInfoMap[id].displayName; s != "" {
				return s
			}
		}
		return fmt.Sprintf("i:%d", id)
	}

}

// Size returns the encoded size in bytes.
func (nid ExpandedNodeId) Size() int {
	var size int
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		size = 2
	case NodeIdTypeFourByte:
		size = 4
	case NodeIdTypeNumeric:
		size = 1 + 2 + 4
	case NodeIdTypeString:
		size = 1 + 2 + 4 + len([]byte(nid.String.Identifier))
	case NodeIdTypeGuid:
		size = 1 + 2 + 4 + len(nid.Guid.Identifier)
	case NodeIdTypeByteString:
		size = 1 + 2 + 4 + len([]byte(nid.ByteString.Identifier))
	}
	if nid.NamespaceURISpecified {
		size += 4 + len([]byte(nid.NamespaceURI))
	}
	if nid.ServerIndexSpecified {
		size += 4
	}
	return size

}

// Description returns a description for two-byte, four-byte or numeric node
// IDs or the empty string.
func (nid ExpandedNodeId) Description() string {
	if i, ok := nid.NamespaceIndex(); (ok && i == 0) || (!ok && nid.NamespaceURI == DefaultNamespaceURI) {
		// Ignore the error since a 0 lookups gives the empty string.
		return nodeInfoMap[nid.Uint()].description
	}
	// Not OPC UA namespace, a description lookup must be done against the
	// server.
	return ""
}

var nodeInfoMap = map[uint16]nodeInfo{}

type nodeInfo struct {
	class       NodeClass
	displayName string
	description string
}
