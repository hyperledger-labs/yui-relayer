package config

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/spf13/cobra"
)

// Module defines an interface of Module
type ModuleI interface {
	// Name returns the name of the module
	Name() string

	// RegisterInterfaces register the ibc transfer module interfaces to protobuf Any.
	RegisterInterfaces(registry codectypes.InterfaceRegistry)

	// GetCmd returns the command
	GetCmd(ctx *Context) *cobra.Command
}
