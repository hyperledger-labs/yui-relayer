package fabric

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/datachainlab/relayer/chains/corda"
	"github.com/datachainlab/relayer/core"
)

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*core.ChainConfigI)(nil),
		&ChainConfig{},
	)
}

func makeEncodingConfig() params.EncodingConfig {
	encodingConfig := core.MakeEncodingConfig()
	RegisterInterfaces(encodingConfig.InterfaceRegistry)
	corda.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
