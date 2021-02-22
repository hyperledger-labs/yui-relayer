package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagFabClientCertPath       = "cert"
	flagFabClientPrivateKeyPath = "key"
	flagFile                    = "file"
)

func populateWalletFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagFabClientCertPath, "", "", "a path of client cert file")
	cmd.Flags().StringP(flagFabClientPrivateKeyPath, "", "", "a path of client private key file")
	return cmd
}

func fileFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagFile, "f", "", "fetch json data from specified file")
	if err := viper.BindPFlag(flagFile, cmd.Flags().Lookup(flagFile)); err != nil {
		panic(err)
	}
	return cmd
}
