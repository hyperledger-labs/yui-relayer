package cmd

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
)

func configCmd(m codec.Marshaler) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "manage configuration file",
	}

	cmd.AddCommand(
		generateChainConfigCmd(m),
	)

	return cmd
}

func generateChainConfigCmd(m codec.Marshaler) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "generate",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// TODO make it configurable
			c := tendermint.ChainConfig{
				Key:            "testkey",
				ChainId:        args[0],
				RpcAddr:        "http://localhost:26557",
				AccountPrefix:  "cosmos",
				GasAdjustment:  1.5,
				GasPrices:      "0.025stake",
				TrustingPeriod: "336h",
			}
			bz, err := config.MarshalJSONAny(m, &c)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	return cmd
}
