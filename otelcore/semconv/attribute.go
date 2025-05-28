package semconv

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	// ChainIDKey represents the chain ID.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "ibc0"
	ChainIDKey = attribute.Key("chain_id")

	// ClientIDKey represents the client ID.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "07-tendermint-0"
	ClientIDKey = attribute.Key("client_id")

	// ConnectionIDKey represents the connection ID.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "connection-0"
	ConnectionIDKey = attribute.Key("connection_id")

	// ChannelIDKey represents the channel ID.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "channel-0"
	ChannelIDKey = attribute.Key("channel_id")

	// PortIDKey represents the port ID.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "mockapp"
	PortIDKey = attribute.Key("port_id")

	// DirectionKey represents the direction.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "src", "dst"
	DirectionKey = attribute.Key("direction")

	// PathKey represents the path.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "commitments/ports/mockapp/channels/channel-0/sequences/3"
	PathKey = attribute.Key("path")

	// HeightRevisionNumberKey represents the revision number of the height.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "0"
	HeightRevisionNumberKey = attribute.Key("height.revision_number")

	// HeightRevisionHeightKey represents the revision height of the height.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "123"
	HeightRevisionHeightKey = attribute.Key("height.revision_height")

	// TxHashKey represents the transaction hash.
	//
	// Type: string
	// RequirementLevel: Recommended
	// Stability: Development
	// Examples: "A829D898DF92C54346408380BA7D0CDE557B2425593F29579D87388DAA64430A"
	TxHashKey = attribute.Key("tx_hash")
)

// AttributeGroup prefixes the given key to all attributes.
//
// For example, if the key is "foo" and the key of an attribute is "bar", the new key will be "foo.bar".
func AttributeGroup(key string, attributes ...attribute.KeyValue) []attribute.KeyValue {
	newAttrs := make([]attribute.KeyValue, 0, len(attributes))
	for _, attr := range attributes {
		newAttrs = append(newAttrs, attribute.KeyValue{
			Key:   attribute.Key(key + "." + string(attr.Key)),
			Value: attr.Value,
		})

	}
	return newAttrs
}
