package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		channelUpgradeCmd(ctx),
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
			pathName := args[0]
			c, src, dst, err := ctx.Config.ChainsFromPath(pathName)
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
			} else if latestHeight, err := c[src].LatestHeight(context.TODO()); err != nil {
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
			} else if latestHeight, err := c[dst].LatestHeight(context.TODO()); err != nil {
				return fmt.Errorf("failed to get the latest height of dst chain: %v", err)
			} else {
				dstHeight = clienttypes.NewHeight(latestHeight.GetRevisionNumber(), height)
			}

			return core.CreateClients(pathName, c[src], c[dst], srcHeight, dstHeight)
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
			pathName := args[0]
			c, src, dst, err := ctx.Config.ChainsFromPath(pathName)
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

			return core.CreateConnection(pathName, c[src], c[dst], to)
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
			pathName := args[0]
			c, src, dst, err := ctx.Config.ChainsFromPath(pathName)
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

			return core.CreateChannel(pathName, c[src], c[dst], to)
		},
	}

	return timeoutFlag(cmd)
}

func channelUpgradeCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel-upgrade",
		Short: "execute operations related to IBC channel upgrade",
		Long:  "This command is meant to be used to upgrade a channel between two chains with a configured path in the config file",
		RunE:  noCommand,
	}

	cmd.AddCommand(
		channelUpgradeInitCmd(ctx),
		channelUpgradeExecuteCmd(ctx),
		channelUpgradeCancelCmd(ctx),
	)

	return cmd
}

func channelUpgradeInitCmd(ctx *config.Context) *cobra.Command {
	const (
		flagOrdering       = "ordering"
		flagConnectionHops = "connection-hops"
		flagVersion        = "version"
		flagUnsafe         = "unsafe"
	)

	cmd := cobra.Command{
		Use:   "init [path-name] [chain-id]",
		Short: "execute chanUpgradeInit",
		Long:  "This command is meant to be used to initialize an IBC channel upgrade on a configured chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathName := args[0]
			chainID := args[1]

			chains, srcID, dstID, err := ctx.Config.ChainsFromPath(pathName)
			if err != nil {
				return err
			}

			var chain, cp *core.ProvableChain
			switch chainID {
			case srcID:
				chain = chains[srcID]
				cp = chains[dstID]
			case dstID:
				chain = chains[dstID]
				cp = chains[srcID]
			default:
				return fmt.Errorf("unknown chain ID: %s", chainID)
			}

			// check cp state
			permitUnsafe, err := cmd.Flags().GetBool(flagUnsafe)
			if err != nil {
				return err
			}

			// get ordering from flags
			ordering, err := getOrderFromFlags(cmd.Flags(), flagOrdering)
			if err != nil {
				return err
			} else if ordering == chantypes.NONE {
				return errors.New("NONE is unacceptable channel ordering")
			}

			// get connection hops from flags
			connHops, err := cmd.Flags().GetStringSlice(flagConnectionHops)
			if err != nil {
				return err
			}

			// get version from flags
			version, err := cmd.Flags().GetString(flagVersion)
			if err != nil {
				return err
			}

			return core.InitChannelUpgrade(
				chain,
				cp,
				chantypes.UpgradeFields{
					Ordering:       ordering,
					ConnectionHops: connHops,
					Version:        version,
				},
				permitUnsafe,
			)
		},
	}

	cmd.Flags().String(flagOrdering, "", "channel ordering applied for the new channel")
	cmd.Flags().StringSlice(flagConnectionHops, nil, "connection hops applied for the new channel")
	cmd.Flags().String(flagVersion, "", "channel version applied for the new channel")
	cmd.Flags().Bool(flagUnsafe, false, "set true if you want to allow for initializing a new channel upgrade even though the counterparty chain is still flushing packets.")

	return &cmd
}

func channelUpgradeExecuteCmd(ctx *config.Context) *cobra.Command {
	const (
		flagInterval       = "interval"
		flagTargetSrcState = "target-src-state"
		flagTargetDstState = "target-dst-state"
	)

	const (
		defaultInterval    = time.Second
		defaultTargetState = "UNINIT"
	)

	cmd := cobra.Command{
		Use:   "execute [path-name]",
		Short: "execute channel upgrade handshake",
		Long:  "This command is meant to be used to execute an IBC channel upgrade handshake between two chains with a configured path in the config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathName := args[0]
			chains, srcChainID, dstChainID, err := ctx.Config.ChainsFromPath(pathName)
			if err != nil {
				return err
			}

			src, ok := chains[srcChainID]
			if !ok {
				panic("src chain not found")
			}
			dst, ok := chains[dstChainID]
			if !ok {
				panic("dst chain not found")
			}

			interval, err := cmd.Flags().GetDuration(flagInterval)
			if err != nil {
				return err
			}

			targetSrcState, err := getUpgradeStateFromFlags(cmd.Flags(), flagTargetSrcState)
			if err != nil {
				return err
			}

			targetDstState, err := getUpgradeStateFromFlags(cmd.Flags(), flagTargetDstState)
			if err != nil {
				return err
			}

			return core.ExecuteChannelUpgrade(pathName, src, dst, interval, targetSrcState, targetDstState)
		},
	}

	cmd.Flags().Duration(flagInterval, defaultInterval, "the interval between attempts to proceed the upgrade handshake")
	cmd.Flags().String(flagTargetSrcState, defaultTargetState, "the source channel's upgrade state to be reached")
	cmd.Flags().String(flagTargetDstState, defaultTargetState, "the destination channel's upgrade state to be reached")

	return &cmd
}

func channelUpgradeCancelCmd(ctx *config.Context) *cobra.Command {
	const (
		flagSettlementInterval = "settlement-interval"
	)

	cmd := cobra.Command{
		Use:   "cancel [path-name] [chain-id]",
		Short: "execute chanUpgradeCancel",
		Long:  "This command is meant to be used to cancel an IBC channel upgrade on a configured chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathName := args[0]
			chainID := args[1]

			settlementInterval, err := cmd.Flags().GetDuration(flagSettlementInterval)
			if err != nil {
				return err
			}

			_, srcChainID, dstChainID, err := ctx.Config.ChainsFromPath(pathName)
			if err != nil {
				return err
			}

			var cpChainID string
			switch chainID {
			case srcChainID:
				cpChainID = dstChainID
			case dstChainID:
				cpChainID = srcChainID
			default:
				return fmt.Errorf("invalid chain ID: %s or %s was expected, but %s was given", srcChainID, dstChainID, chainID)
			}

			chain, err := ctx.Config.GetChain(chainID)
			if err != nil {
				return err
			}
			cp, err := ctx.Config.GetChain(cpChainID)
			if err != nil {
				return err
			}

			return core.CancelChannelUpgrade(chain, cp, settlementInterval)
		},
	}

	cmd.Flags().Duration(flagSettlementInterval, 10*time.Second, "time interval between attemts to query for settled channel/upgrade states")

	return &cmd
}

func relayMsgsCmd(ctx *config.Context) *cobra.Command {
	const (
		flagDoRefresh = "do-refresh"
		flagSrcSeqs   = "src-seqs"
		flagDstSeqs   = "dst-seqs"
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

			sp, err := st.UnrelayedPackets(context.TODO(), c[src], c[dst], sh, false)
			if err != nil {
				return err
			}
			srcSeq := getUint64Slice(flagSrcSeqs)
			dstSeq := getUint64Slice(flagDstSeqs)
			if err = tryFilterRelayPackets(sp, srcSeq, dstSeq); err != nil {
				return err
			}

			msgs := core.NewRelayMsgs()

			doExecuteRelaySrc := len(sp.Dst) > 0
			doExecuteRelayDst := len(sp.Src) > 0
			doExecuteAckSrc := false
			doExecuteAckDst := false

			if m, err := st.UpdateClients(context.TODO(), c[src], c[dst], doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, sh, viper.GetBool(flagDoRefresh)); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			if m, err := st.RelayPackets(context.TODO(), c[src], c[dst], sp, sh, doExecuteRelaySrc, doExecuteRelayDst); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			st.Send(context.TODO(), c[src], c[dst], msgs)

			return nil
		},
	}
	cmd.Flags().Bool(flagDoRefresh, defaultDoRefresh, "execute light client refresh (updateClient) if required")
	cmd.Flags().IntSlice(flagSrcSeqs, nil, "packet filter for src chain")
	cmd.Flags().IntSlice(flagDstSeqs, nil, "packet filter for dst chain")
	// TODO add option support for strategy
	return cmd
}

func relayAcksCmd(ctx *config.Context) *cobra.Command {
	const (
		flagDoRefresh = "do-refresh"
		flagSrcSeqs   = "src-seqs"
		flagDstSeqs   = "dst-seqs"
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
			sp, err := st.UnrelayedAcknowledgements(context.TODO(), c[src], c[dst], sh, false)
			if err != nil {
				return err
			}
			srcSeq := getUint64Slice(flagSrcSeqs)
			dstSeq := getUint64Slice(flagDstSeqs)
			if err = tryFilterRelayPackets(sp, srcSeq, dstSeq); err != nil {
				return err
			}

			msgs := core.NewRelayMsgs()

			doExecuteRelaySrc := false
			doExecuteRelayDst := false
			doExecuteAckSrc := len(sp.Dst) > 0
			doExecuteAckDst := len(sp.Src) > 0

			if m, err := st.UpdateClients(context.TODO(), c[src], c[dst], doExecuteRelaySrc, doExecuteRelayDst, doExecuteAckSrc, doExecuteAckDst, sh, viper.GetBool(flagDoRefresh)); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			if m, err := st.RelayAcknowledgements(context.TODO(), c[src], c[dst], sp, sh, doExecuteAckSrc, doExecuteAckDst); err != nil {
				return err
			} else {
				msgs.Merge(m)
			}

			st.Send(context.TODO(), c[src], c[dst], msgs)

			return nil
		},
	}
	cmd.Flags().Bool(flagDoRefresh, defaultDoRefresh, "execute light client refresh (updateClient) if required")
	cmd.Flags().IntSlice(flagSrcSeqs, nil, "packet filter for src chain")
	cmd.Flags().IntSlice(flagDstSeqs, nil, "packet filter for dst chain")
	return cmd
}

func tryFilterRelayPackets(sp *core.RelayPackets, srcSeq []uint64, dstSeq []uint64) error {
	if len(srcSeq) > 0 {
		sp.Src = sp.Src.Filter(srcSeq)
		if len(sp.Src) != len(srcSeq) {
			return fmt.Errorf("src packet not found packetLength=%d selectedLength=%d", len(sp.Src), len(srcSeq))
		}
	}
	if len(dstSeq) > 0 {
		sp.Dst = sp.Dst.Filter(dstSeq)
		if len(sp.Dst) != len(dstSeq) {
			return fmt.Errorf("dst packet not found packetLength=%d selectedLength=%d", len(sp.Dst), len(dstSeq))
		}
	}
	return nil
}

func getUint64Slice(key string) []uint64 {
	org := viper.GetIntSlice(key)
	ret := make([]uint64, len(org))
	for i, e := range org {
		ret[i] = uint64(e)
	}
	return ret
}

func getUpgradeStateFromFlags(flags *pflag.FlagSet, flagName string) (core.UpgradeState, error) {
	s, err := flags.GetString(flagName)
	if err != nil {
		return 0, err
	}

	switch strings.ToUpper(s) {
	case "UNINIT":
		return core.UPGRADE_STATE_UNINIT, nil
	case "INIT":
		return core.UPGRADE_STATE_INIT, nil
	case "FLUSHING":
		return core.UPGRADE_STATE_FLUSHING, nil
	case "FLUSHCOMPLETE":
		return core.UPGRADE_STATE_FLUSHCOMPLETE, nil
	default:
		return 0, fmt.Errorf("invalid upgrade state specified: %s", s)
	}
}

func getOrderFromFlags(flags *pflag.FlagSet, flagName string) (chantypes.Order, error) {
	s, err := flags.GetString(flagName)
	if err != nil {
		return 0, err
	}

	s = "ORDER_" + strings.ToUpper(s)
	value, ok := chantypes.Order_value[s]
	if !ok {
		return 0, fmt.Errorf("invalid channel order specified: %s", s)
	}

	return chantypes.Order(value), nil
}
