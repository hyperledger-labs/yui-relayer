package cmd

import (
	"encoding/json"

	"github.com/hyperledger-labs/yui-fabric-ibc/example"
	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	"github.com/hyperledger-labs/yui-relayer/config"
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
			fc := c.ChainI.(*fabric.Chain)
			if err = fc.Connect(); err != nil {
				return err
			}
			// TODO we should get a genesis state from a given json file
			genesisState := example.NewDefaultGenesisState()
			bz, err := json.Marshal(genesisState)
			if err != nil {
				return err
			}
			_, err = fc.Contract().SubmitTransaction(initChaincodeFunc, string(bz))
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
