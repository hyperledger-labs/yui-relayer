package cmd

import (
	"fmt"

	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	"github.com/hyperledger-labs/yui-relayer/config"
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
				MspId:                 "",
				Channel:               "",
				ChaincodeId:           "",
				ConnectionProfilePath: "",
				IbcPolicies:           []string{},
				EndorsementPolicies:   []string{},
				MspConfigPaths:        []string{},
			}
			bz, err := config.MarshalJSONAny(ctx.Marshaler, &c)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	return cmd
}
