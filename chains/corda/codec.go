package corda

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	cordatypes "github.com/datachainlab/corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/datachainlab/relayer/core"
)

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	cordatypes.RegisterInterfaces(registry)
	registry.RegisterImplementations(
		(*core.ChainConfigI)(nil),
		&ChainConfig{},
	)
}

func makeEncodingConfig() params.EncodingConfig {
	encodingConfig := core.MakeEncodingConfig()
	RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
