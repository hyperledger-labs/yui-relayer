package corda

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/core"
	"github.com/datachainlab/relayer/jp_datachain_corda_ibc_grpc"
)

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*core.ChainConfigI)(nil),
		&ChainConfig{},
	)
	registry.RegisterCustomTypeURL(
		(*exported.ClientState)(nil),
		"type.googleapis.com/jp.datachain.corda.ibc.grpc.ClientState",
		&jp_datachain_corda_ibc_grpc.ClientState{},
	)
}

func makeEncodingConfig() params.EncodingConfig {
	encodingConfig := core.MakeEncodingConfig()
	RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
