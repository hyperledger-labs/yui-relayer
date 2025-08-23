package module

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/hyperledger-labs/yui-relayer/config"
	debugprover "github.com/hyperledger-labs/yui-relayer/provers/debug"
	"github.com/spf13/cobra"
)

type Module struct{}

var _ config.ModuleI = (*Module)(nil)

// Name returns the name of the module
func (Module) Name() string {
	return "debug.prover"
}

// RegisterInterfaces register the module interfaces to protobuf Any.
func (Module) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	debugprover.RegisterInterfaces(registry)
}

// GetCmd returns the command
func (Module) GetCmd(ctx *config.Context) *cobra.Command {
	return nil
}
