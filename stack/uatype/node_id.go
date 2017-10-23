//go:generate csv2code -o node_id_auto.go -csv ../../schemas/1.03/NodeIds.csv node_id_auto.go.tmpl
//go:generate opcua-xml2code -t=nodes -o node_id_info_auto.go ../../schemas/1.03/Opc.Ua.NodeSet.xml
//go:generate gofmt -s -w node_id_auto.go

package uatype

import (
	"fmt"
)

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

var nodeInfoMap = map[uint16]nodeInfo{}

type nodeInfo struct {
	class       enumNodeClass
	displayName string
	description string
}

// Class will return the node class of two-byte, four-byte or numeric node IDs.
// Other node ID types will always return NodeClassUnspecified.
func (nid NodeId) Class() enumNodeClass {
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
	case NodeIdTypeByteString:
		return string(nid.ByteString.Identifier)
	case NodeIdTypeGuid:
		return fmt.Sprintf("GUID:%s", nid.Guid.Identifier)
	default:
		// Ignore the error since a zero lookups should give the empty string.
		id := nid.Uint()
		if s := nodeInfoMap[id].displayName; s != "" {
			return s
		}
		return fmt.Sprintf("i:%d", id)
	}

}

// Description returns a description for two-byte, four-byte or numeric node
// IDs or the empty string.
func (nid NodeId) Description() string {
	// Ignore the error since a zero lookups should give the empty string.
	id := nid.Uint()
	return nodeInfoMap[id].description
}

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
