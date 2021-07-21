package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
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

			height, err := cmd.Flags().GetInt64(flags.FlagHeight)
			if err != nil {
				return err
			}

			if height == 0 {
				height, err = c.GetLatestHeight()
				if err != nil {
					return err
				}
			}
			res, err := c.QueryClientState(height)
			if err != nil {
				return err
			}
			var cs exported.ClientState
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

			height, err := c.GetLatestHeight()
			if err != nil {
				return err
			}

			res, err := c.QueryConnection(height)
			if err != nil {
				return err
			}
			fmt.Println(res.Connection.String())
			return nil
		},
	}

	return cmd
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

			height, err := c.GetLatestHeight()
			if err != nil {
				return err
			}

			res, err := c.QueryChannel(height)
			if err != nil {
				return err
			}
			fmt.Println(res.Channel.String())
			return nil
		},
	}

	return cmd
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

			coins, err := helpers.QueryBalance(chain, addr, showDenoms)
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
			sp, err := st.UnrelayedSequences(c[src], c[dst], sh)
			if err != nil {
				return err
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

			sp, err := st.UnrelayedAcknowledgements(c[src], c[dst], sh)
			if err != nil {
				return err
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
