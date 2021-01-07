package cmd

import (
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"
)

func FabricCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fabric",
		Short: "manage fabric configurations",
	}

	cmd.AddCommand(
		configCmd(ctx),
		walletCmd(ctx),
		chaincodeCmd(ctx),
		queryCmd(ctx),
	)

	return cmd
}
