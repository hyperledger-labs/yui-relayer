package cmd

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/datachainlab/relayer/config"
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
				height, err = c.QueryLatestHeight()
				if err != nil {
					return err
				}
			}
			res, err := c.QueryClientState(height, false)
			if err != nil {
				return err
			}
			var cs exported.ClientState
			if err := c.Marshaler().UnpackAny(res.ClientState, &cs); err != nil {
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

			height, err := c.QueryLatestHeight()
			if err != nil {
				return err
			}

			res, err := c.QueryConnection(height, false)
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

			height, err := c.QueryLatestHeight()
			if err != nil {
				return err
			}

			res, err := c.QueryChannel(height, false)
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

			coins, err := chain.QueryBalance(addr)
			if err != nil {
				return err
			}

			if showDenoms {
				fmt.Println(coins)
				return nil
			}

			h, err := chain.QueryLatestHeight()
			if err != nil {
				return err
			}

			dts, err := chain.QueryDenomTraces(0, 1000, h)
			if err != nil {
				return err
			}

			if len(dts.DenomTraces) > 0 {
				out := sdk.Coins{}
				for _, c := range coins {
					for _, d := range dts.DenomTraces {
						switch {
						case c.Amount.Equal(sdk.NewInt(0)):
						case c.Denom == d.IBCDenom():
							out = append(out, sdk.NewCoin(d.GetFullDenomPath(), c.Amount))
						default:
							out = append(out, c)
						}
					}
				}
				fmt.Println(out)
				return nil
			}

			fmt.Println(coins)
			return nil
		},
	}
	return ibcDenomFlags(cmd)
}
