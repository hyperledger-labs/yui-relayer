package fabric

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/hyperledger-labs/yui-relayer/core"
)

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*core.ChainConfigI)(nil),
		&ChainConfig{},
	)
	registry.RegisterImplementations(
		(*core.ProverConfigI)(nil),
		&ProverConfig{},
	)
}

func makeEncodingConfig() params.EncodingConfig {
	encodingConfig := core.MakeEncodingConfig()
	RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
