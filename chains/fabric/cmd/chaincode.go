package cmd

import (
	"github.com/datachainlab/relayer/chains/fabric"
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"
)

const initChaincodeFunc = "initChaincode"

func chaincodeCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chaincode",
		Short: "manage IBC chaincode",
	}

	cmd.AddCommand(
		initChaincodeCmd(ctx),
	)

	return cmd
}

func initChaincodeCmd(ctx *config.Context) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "init [chain-id]",
		Short: "initialize the state of chaincode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}
			fc := c.(*fabric.Chain)
			if err = fc.Connect(); err != nil {
				return err
			}
			_, err = fc.Contract().SubmitTransaction(initChaincodeFunc, "{}")
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
