package corda

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cordatypes "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

// RegisterInterfaces register the module interfaces to protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	cordatypes.RegisterInterfaces(registry)

	registry.RegisterImplementations(
		(*core.ChainConfigI)(nil),
		&ChainConfig{},
	)
	registry.RegisterImplementations(
		(*core.ProverConfigI)(nil),
		&ProverConfig{},
	)
}
