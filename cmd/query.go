package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/helpers"
	"github.com/spf13/cobra"
)

// queryCmd represents the chain command
func queryCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "IBC Query Commands",
		Long:  "Commands to query IBC primitives, and other useful data on configured chains.",
		RunE:  noCommand,
	}

	cmd.AddCommand(
		queryBalanceCmd(ctx),
		queryUnrelayedPackets(ctx),
		queryUnrelayedAcknowledgements(ctx),
		flags.LineBreak,
		queryClientCmd(ctx),
		queryConnection(ctx),
		queryChannel(ctx),
	)

	return cmd
}

func queryClientCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client [path-name] [chain-id]",
		Short: "Query the state of a client in a given path",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chains, _, _, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			c := chains[args[1]]

			height, err := cmd.Flags().GetUint64(flags.FlagHeight)
			if err != nil {
				return err
			}
			latestHeight, err := c.LatestHeight()
			if err != nil {
				return err
			}
			queryHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), uint64(height))
			res, err := c.QueryClientState(core.NewQueryContext(context.TODO(), queryHeight))
			if err != nil {
				return err
			}
			var cs ibcexported.ClientState
			if err := c.Codec().UnpackAny(res.ClientState, &cs); err != nil {
				return err
			}
			fmt.Println(cs)
			return nil
		},
	}

	return heightFlag(cmd)
}

func queryConnection(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection [path-name] [chain-id]",
		Short: "Query the connection state for the given connection id",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chains, _, _, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			c := chains[args[1]]

			height, err := cmd.Flags().GetUint64(flags.FlagHeight)
			if err != nil {
				return err
			}
			latestHeight, err := c.LatestHeight()
			if err != nil {
				return err
			}
			queryHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), uint64(height))
			res, err := c.QueryConnection(core.NewQueryContext(context.TODO(), queryHeight), c.Path().ConnectionID)
			if err != nil {
				return err
			}
			fmt.Println(res.Connection.String())
			return nil
		},
	}

	return heightFlag(cmd)
}

func queryChannel(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [path-name] [chain-id]",
		Short: "Query the connection state for the given connection id",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chains, _, _, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			c := chains[args[1]]

			height, err := cmd.Flags().GetUint64(flags.FlagHeight)
			if err != nil {
				return err
			}
			latestHeight, err := c.LatestHeight()
			if err != nil {
				return err
			}
			queryHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), uint64(height))
			res, err := c.QueryChannel(core.NewQueryContext(context.TODO(), queryHeight))
			if err != nil {
				return err
			}
			fmt.Println(res.Channel.String())
			return nil
		},
	}

	return heightFlag(cmd)
}

func queryBalanceCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [chain-id] [address]",
		Short: "Query the account balances",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}

			showDenoms, err := cmd.Flags().GetBool(flagIBCDenoms)
			if err != nil {
				return err
			}

			addr, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			h, err := chain.LatestHeight()
			if err != nil {
				return err
			}

			coins, err := helpers.QueryBalance(chain, h, addr, showDenoms)
			if err != nil {
				return err
			}

			fmt.Println(coins)
			return nil
		},
	}
	return ibcDenomFlags(cmd)
}

func queryUnrelayedPackets(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unrelayed-packets [path]",
		Short: "Query for the packet sequence numbers that remain to be relayed on a given path",
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
			sp, err := st.UnrelayedPackets(c[src], c[dst], sh, true)
			if err != nil {
				return err
			}

			// Some use cases need `{"src":[],"dst":[]}` instead of `{"src":null,"dst":null}`
			if sp.Src == nil {
				sp.Src = []*core.PacketInfo{}
			}
			if sp.Dst == nil {
				sp.Dst = []*core.PacketInfo{}
			}

			out, err := json.Marshal(sp)
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}

func queryUnrelayedAcknowledgements(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unrelayed-acknowledgements [path]",
		Short: "Query for the packet sequence numbers that remain to be relayed on a given path",
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

			sp, err := st.UnrelayedAcknowledgements(c[src], c[dst], sh, true)
			if err != nil {
				return err
			}

			// Some use cases need `{"src":[],"dst":[]}` instead of `{"src":null,"dst":null}`
			if sp.Src == nil {
				sp.Src = []*core.PacketInfo{}
			}
			if sp.Dst == nil {
				sp.Dst = []*core.PacketInfo{}
			}

			out, err := json.Marshal(sp)
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}
