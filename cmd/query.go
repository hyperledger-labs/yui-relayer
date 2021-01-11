package cmd

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
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
		queryClientCmd(ctx),
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
			fmt.Println(string(res.ClientState.Value))
			return nil
		},
	}

	return heightFlag(cmd)
}
