package cmd

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
)

// queryCmd represents the chain command
func queryCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "IBC Query Commands",
		Long:  "Commands to query IBC primatives, and other useful data on configured chains.",
	}

	cmd.AddCommand(
		querySequenceCmd(ctx),
	)

	return cmd
}

func querySequenceCmd(ctx *config.Context) *cobra.Command {
	c := &cobra.Command{
		Use:   "sequence [chain-id]",
		Short: "query sequence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}
			fc := c.ChainI.(*fabric.Chain)
			if err = fc.Connect(); err != nil {
				return err
			}
			seq, err := fc.QueryCurrentSequence()
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", seq.String())
			return nil
		},
	}

	return c
}
