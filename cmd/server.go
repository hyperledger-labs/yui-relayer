package cmd

import (
	"github.com/datachainlab/relayer/config"
	"github.com/datachainlab/relayer/server"
	"github.com/spf13/cobra"
)

func serverCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "API service for IBC",
	}
	cmd.AddCommand(
		serverStart(ctx),
	)
	return cmd
}

func serverStart(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "start",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := cmd.Flags().GetString(flagAddr)
			if err != nil {
				return err
			}
			m := server.NewAPIServer(ctx)
			return m.Start(addr)
		},
	}
	return serverFlags(cmd)
}
