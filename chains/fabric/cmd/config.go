package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
)

func configCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "manage configuration file",
	}

	cmd.AddCommand(
		generateChainConfigCmd(ctx),
	)

	return cmd
}

func generateChainConfigCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "generate",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// TODO make it configurable
			c := fabric.ChainConfig{
				ChainId:               args[0],
				WalletLabel:           "",
				Channel:               "",
				ChaincodeId:           "",
				ConnectionProfilePath: "",
			}
			p := fabric.ProverConfig{
				IbcPolicies:         []string{},
				EndorsementPolicies: []string{},
				MspConfigPaths:      []string{},
			}
			config, err := core.NewChainProverConfig(ctx.Codec, &c, &p)
			if err != nil {
				return err
			}
			bz, err := json.Marshal(config)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	return cmd
}
