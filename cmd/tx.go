package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// transactionCmd represents the tx command
func transactionCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "IBC Transaction Commands",
		Long: strings.TrimSpace(`Commands to create IBC transactions on configured chains. 
		Most of these commands take a '[path]' argument. Make sure:
	1. Chains are properly configured to relay over by using the 'rly chains list' command
	2. Path is properly configured to relay over by using the 'rly paths list' command`),
		RunE: noCommand,
	}

	cmd.AddCommand(
		xfersend(ctx),
		relayMsgsCmd(ctx),
		relayAcksCmd(ctx),
		flags.LineBreak,
		createClientsCmd(ctx),
		updateClientsCmd(ctx),
		createConnectionCmd(ctx),
		createChannelCmd(ctx),
	)

	return cmd
}

func createClientsCmd(ctx *config.Context) *cobra.Command {
	const (
		flagSrcHeight = "src-height"
		flagDstHeight = "dst-height"
	)
	const (
		defaultSrcHeight = 0
		defaultDstHeight = 0
	)
	cmd := &cobra.Command{
		Use:   "clients [path-name]",
		Short: "create a clients between two configured chains with a configured path",
		Long: "Creates a working ibc client for chain configured on each end of the" +
			" path by querying headers from each chain and then sending the corresponding create-client messages",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			// if the option "src-height" is not set or is set zero, the latest finalized height is used.
			var srcHeight exported.Height
			if height, err := cmd.Flags().GetUint64(flagSrcHeight); err != nil {
				return err
			} else if height == 0 {
				srcHeight = nil
			} else if latestHeight, err := c[src].LatestHeight(); err != nil {
				return fmt.Errorf("failed to get the latest height of src chain: %v", err)
			} else {
				srcHeight = clienttypes.NewHeight(latestHeight.GetRevisionNumber(), height)
			}

			// if the option "dst-height" is not set or is set zero, the latest finalized height is used.
			var dstHeight exported.Height
			if height, err := cmd.Flags().GetUint64(flagDstHeight); err != nil {
				return err
			} else if height == 0 {
				dstHeight = nil
			} else if latestHeight, err := c[dst].LatestHeight(); err != nil {
				return fmt.Errorf("failed to get the latest height of dst chain: %v", err)
			} else {
				dstHeight = clienttypes.NewHeight(latestHeight.GetRevisionNumber(), height)
			}

			return core.CreateClients(c[src], c[dst], srcHeight, dstHeight)
		},
	}
	cmd.Flags().Uint64(flagSrcHeight, defaultSrcHeight, "src header at this height is submitted to dst chain")
	cmd.Flags().Uint64(flagDstHeight, defaultDstHeight, "dst header at this height is submitted to src chain")
	return cmd
}

func updateClientsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-clients [path-name]",
		Short: "update the clients between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to updates 
			the clients with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.UpdateClients(c[src], c[dst])
		},
	}
	return cmd
}

func createConnectionCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection [path-name]",
		Short: "create a connection between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to repair or create 
		a connection between two chains with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			to, err := getTimeout(cmd)
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.CreateConnection(c[src], c[dst], to)
		},
	}

	return timeoutFlag(cmd)
}

func createChannelCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [path-name]",
		Short: "create a channel between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to repair or 
		create a channel between two chains with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			to, err := getTimeout(cmd)
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}
			return core.CreateChannel(c[src], c[dst], to)
		},
	}

	return timeoutFlag(cmd)
}

func relayMsgsCmd(ctx *config.Context) *cobra.Command {
	const (
		flagDoRefresh = "do-refresh"
	)
	const (
		defaultDoRefresh = false
	)
	cmd := &cobra.Command{
		Use:   "relay [path-name]",
		Short: "relay any packets that remain to be relayed on a given path, in both directions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			path, err := ctx.Config.Paths.Get(args[0])
			if err != nil {
				return err
			}
			sh, err := core.NewSyncHeaders(c[src], c[dst])
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

			sp, err := st.UnrelayedPackets(c[src], c[dst], sh, false)
			if err != nil {
				return err
			}

			msgs := core.NewRelayMsgs()

			if m, err := st.UpdateClients(c[src], c[dst], sp, &core.RelayPackets{}, sh, viper.GetBool(flagDoRefresh)); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			if m, err := st.RelayPackets(c[src], c[dst], sp, sh); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			st.Send(c[src], c[dst], msgs)

			return nil
		},
	}
	cmd.Flags().Bool(flagDoRefresh, defaultDoRefresh, "execute light client refresh (updateClient) if required")
	// TODO add option support for strategy
	return cmd
}

func relayAcksCmd(ctx *config.Context) *cobra.Command {
	const (
		flagDoRefresh = "do-refresh"
	)
	const (
		defaultDoRefresh = false
	)
	cmd := &cobra.Command{
		Use:     "relay-acknowledgements [path-name]",
		Aliases: []string{"acks"},
		Short:   "relay any acknowledgements that remain to be relayed on a given path, in both directions",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			path, err := ctx.Config.Paths.Get(args[0])
			if err != nil {
				return err
			}
			sh, err := core.NewSyncHeaders(c[src], c[dst])
			if err != nil {
				return err
			}
			st, err := core.GetStrategy(*path.Strategy)
			if err != nil {
				return err
			}

			// sp.Src contains all sequences acked on SRC but acknowledgement not processed on DST
			// sp.Dst contains all sequences acked on DST but acknowledgement not processed on SRC
			sp, err := st.UnrelayedAcknowledgements(c[src], c[dst], sh, false)
			if err != nil {
				return err
			}

			msgs := core.NewRelayMsgs()

			if m, err := st.UpdateClients(c[src], c[dst], &core.RelayPackets{}, sp, sh, viper.GetBool(flagDoRefresh)); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			if m, err := st.RelayAcknowledgements(c[src], c[dst], sp, sh); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			st.Send(c[src], c[dst], msgs)

			return nil
		},
	}
	cmd.Flags().Bool(flagDoRefresh, defaultDoRefresh, "execute light client refresh (updateClient) if required")
	return cmd
}
