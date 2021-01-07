package cmd

import (
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"
)

func walletCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "manage fabric client wallet",
	}

	cmd.AddCommand(
	// populateWalletCmd(ctx),
	)

	return cmd
}

// func populateWalletCmd(ctx *config.Context) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:     "populate [chain-id]",
// 		Aliases: []string{"p"},
// 		Short:   "populate wallet",
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			c, err := ctx.Config.GetChain(args[0])
// 			if err != nil {
// 				return err
// 			}

// 			return c.PopulateWallet(
// 				viper.GetString(flagFabClientCertPath),
// 				viper.GetString(flagFabClientPrivateKeyPath),
// 			)
// 		},
// 	}

// 	return populateWalletFlag(cmd)
// }
