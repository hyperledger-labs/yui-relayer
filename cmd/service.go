package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/metrics"
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
		flagPrometheusAddr           = "prometheus-addr"
		flagSrcRelayOptimizeInterval = "src-relay-optimize-interval"
		flagSrcRelayOptimizeCount    = "src-relay-optimize-count"
		flagDstRelayOptimizeInterval = "dst-relay-optimize-interval"
		flagDstRelayOptimizeCount    = "dst-relay-optimize-count"
	)
	const (
		defaultRelayInterval         = 3 * time.Second
		defaultPrometheusAddr        = "localhost:2223"
		defaultRelayOptimizeInterval = 10 * time.Second
		defaultRelayOptimizeCount    = 5
	)

	cmd := &cobra.Command{
		Use:  "start [path-name]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := metrics.ShutdownMetrics(cmd.Context()); err != nil {
				return fmt.Errorf("failed to shutdown the metrics subsystem with null exporter: %v", err)
			}
			if err := metrics.InitializeMetrics(metrics.ExporterProm{Addr: viper.GetString(flagPrometheusAddr)}); err != nil {
				return fmt.Errorf("failed to re-initialize the metrics subsystem with prometheus exporter: %v", err)
			}
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
			if err := st.SetupRelay(context.TODO(), c[src], c[dst]); err != nil {
				return err
			}
			return core.StartService(
				context.Background(),
				st,
				c[src],
				c[dst],
				viper.GetDuration(flagRelayInterval),
				viper.GetDuration(flagSrcRelayOptimizeInterval),
				viper.GetInt64(flagSrcRelayOptimizeCount),
				viper.GetDuration(flagDstRelayOptimizeInterval),
				viper.GetInt64(flagDstRelayOptimizeCount),
			)
		},
	}
	cmd.Flags().Duration(flagRelayInterval, defaultRelayInterval, "time interval to perform relays")
	cmd.Flags().String(flagPrometheusAddr, defaultPrometheusAddr, "host address to which the prometheus exporter listens")
	cmd.Flags().Duration(flagSrcRelayOptimizeInterval, defaultRelayOptimizeInterval, "time interval to perform relays optimization on source chain")
	cmd.Flags().Int64(flagSrcRelayOptimizeCount, defaultRelayOptimizeCount, "number of packets to relays optimization on source chain")
	cmd.Flags().Duration(flagDstRelayOptimizeInterval, defaultRelayOptimizeInterval, "time interval to perform relays optimization on destination chain")
	cmd.Flags().Int64(flagDstRelayOptimizeCount, defaultRelayOptimizeCount, "number of packets to relays optimization on destination chain")
	return cmd
}
