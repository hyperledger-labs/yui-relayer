package cmd

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
	vsc "github.com/vsc-blockchain/core/types"
)

func TendermintCmd(m codec.Codec, ctx *config.Context) *cobra.Command {

	sdk.DefaultPowerReduction = vsc.PowerReduction

	cmd := &cobra.Command{
		Use:   "tendermint",
		Short: "manage tendermint configurations",
	}

	cmd.AddCommand(
		configCmd(m),
		keysCmd(ctx),
		lightCmd(ctx),
	)

	return cmd
}
