//go:generate csv2code -o node_id_auto.go -csv ../schemas/1.03/NodeIds.csv node_id_auto.go.tmpl
//go:generate opcua-xml2code -o node_id_info_auto.go ../schemas/1.03/Opc.Ua.NodeSet.xml
//go:generate gofmt -s -w node_id_auto.go

package uatype

import (
	"errors"
	"fmt"
)

var ErrNodeIdNotNumeric = errors.New("not a numeric identifier")

// Uint16Identifier returns the identifier of two-byte, four-byte or numeric
// node IDs as uint16. Calling this method on other types of node IDs will
// result in an ErrNodeIdNotNumeric.
func (nid NodeId) Uint16Identifier() (uint16, error) {
	switch nid.NodeIdType {
	case NodeIdTypeTwoByte:
		return uint16(nid.TwoByte.Identifier), nil
	case NodeIdTypeFourByte:
		return uint16(nid.FourByte.Identifier), nil
	case NodeIdTypeNumeric:
		return uint16(nid.Numeric.Identifier), nil
	default:
		return 0, ErrNodeIdNotNumeric
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
	id, err := nid.Uint16Identifier()
	if err != nil {
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
		id, _ := nid.Uint16Identifier()
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
	id, _ := nid.Uint16Identifier()
	return nodeInfoMap[id].description
}
