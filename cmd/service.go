package cmd

import (
	"time"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func serviceCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Relay Service Commands",
		Long:  "Commands to manage the relay service",
		RunE:  noCommand,
	}
	cmd.AddCommand(
		startCmd(ctx),
	)
	return cmd
}

func startCmd(ctx *config.Context) *cobra.Command {
	const (
		flagRelayInterval            = "relay-interval"
		flagSrcRelayOptimizeInterval = "src-relay-optimize-interval"
		flagSrcRelayOptimizeCount    = "src-relay-optimize-count"
		flagDstRelayOptimizeInterval = "dst-relay-optimize-interval"
		flagDstRelayOptimizeCount    = "dst-relay-optimize-count"
	)
	const (
		defaultRelayInterval         = 3 * time.Second
		defaultRelayOptimizeInterval = 10 * time.Second
		defaultRelayOptimizeCount    = 5
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
			if err := st.SetupRelay(cmd.Context(), c[src], c[dst]); err != nil {
				return err
			}
			return core.StartService(
				cmd.Context(),
				st,
				c[src],
				c[dst],
				viper.GetDuration(flagRelayInterval),
				viper.GetDuration(flagSrcRelayOptimizeInterval),
				viper.GetUint64(flagSrcRelayOptimizeCount),
				viper.GetDuration(flagDstRelayOptimizeInterval),
				viper.GetUint64(flagDstRelayOptimizeCount),
			)
		},
	}
	cmd.Flags().Duration(flagRelayInterval, defaultRelayInterval, "time interval to perform relays")
	cmd.Flags().Duration(flagSrcRelayOptimizeInterval, defaultRelayOptimizeInterval, "maximum time interval to delay relays for optimization")
	cmd.Flags().Uint64(flagSrcRelayOptimizeCount, defaultRelayOptimizeCount, "maximum number of relays to delay for optimization")
	cmd.Flags().Duration(flagDstRelayOptimizeInterval, defaultRelayOptimizeInterval, "maximum time interval to delay relays for optimization")
	cmd.Flags().Uint64(flagDstRelayOptimizeCount, defaultRelayOptimizeCount, "maximum number of relays to delay for optimization")
	return cmd
}
