package cmd

import (
	"context"
	"time"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func serviceCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "service",
	}
	cmd.AddCommand(
		startCmd(ctx),
	)
	return cmd
}

func startCmd(ctx *config.Context) *cobra.Command {
	const (
		flagRelayInterval = "relay-interval"
		flagRelayAttempts = "relay-attempts"
	)

	cmd := &cobra.Command{
		Use:  "start [path-name]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			path, err := ctx.Config.Paths.Get(args[0])
			if err != nil {
				return err
			}
			st, err := core.GetStrategy(*path.Strategy)
			if err != nil {
				return err
			}
			return core.StartService(context.Background(), st, c[src], c[dst], viper.GetDuration(flagRelayInterval), viper.GetUint(flagRelayAttempts))
		},
	}
	cmd.Flags().Duration(flagRelayInterval, 3*time.Second, "time interval to perform relays")
	cmd.Flags().Uint(flagRelayAttempts, 5, "count of retry")
	return cmd
}
