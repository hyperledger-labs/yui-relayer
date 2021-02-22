package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cosmos/cosmos-sdk/simapp"
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
			// TODO we should get a genesis state from a given json file
			file, err := cmd.Flags().GetString(flagFile)
			if err != nil {
				return err
			}
			if _, err := os.Stat(file); err != nil {
				return err
			}
			bz, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}
			if !isGenesisState(bz) {
				return fmt.Errorf("input file must be a GenesisState")
			}
			_, err = fc.Contract().SubmitTransaction(initChaincodeFunc, string(bz))
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd = fileFlag(cmd)
	cmd.MarkFlagRequired(flagFile)
	return cmd
}

func isGenesisState(bz []byte) bool {
	var gs simapp.GenesisState
	return json.Unmarshal(bz, &gs) == nil
}
