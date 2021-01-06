package cmd

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"
)

func TendermintCmd(m codec.Marshaler, cmgr config.ConfigManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tendermint",
		Short: "manage tendermint configurations",
	}

	cmd.AddCommand(
		configCmd(m),
		keysCmd(cmgr),
	)

	return cmd
}
