package core

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	AttributeKeyChainID        = attribute.Key("chain_id")
	AttributeKeyClientID       = attribute.Key("client_id")
	AttributeKeyConnectionID   = attribute.Key("connection_id")
	AttributeKeyChannelID      = attribute.Key("channel_id")
	AttributeKeyPortID         = attribute.Key("port_id")
	AttributeKeyDirection      = attribute.Key("direction")
	AttributeKeyPath           = attribute.Key("path")
	AttributeKeyRevisionNumber = attribute.Key("revision_number")
	AttributeKeyRevisionHeight = attribute.Key("revision_height")
	AttributeKeyPackage        = attribute.Key("package")
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
