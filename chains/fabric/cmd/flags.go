package cmd

import "github.com/spf13/cobra"

const (
	flagFabClientCertPath       = "cert"
	flagFabClientPrivateKeyPath = "key"
	flagGenesisFile             = "genesis"
)

func populateWalletFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagFabClientCertPath, "", "", "a path of client cert file")
	cmd.Flags().StringP(flagFabClientPrivateKeyPath, "", "", "a path of client private key file")
	return cmd
}

func initChaincodeFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagGenesisFile, "", "", "a path of genesis file")
	return cmd
}
