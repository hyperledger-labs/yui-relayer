package cmd

import (
	"fmt"

	fabricauthtypes "github.com/hyperledger-labs/yui-fabric-ibc/x/auth/types"
	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func walletCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "manage fabric client wallet",
	}

	cmd.AddCommand(
		populateWalletCmd(ctx),
		showAddressCmd(ctx),
	)

	return cmd
}

func populateWalletCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "populate [chain-id]",
		Short: "populate wallet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}
			fc := c.ChainI.(*fabric.Chain)
			return fc.PopulateWallet(
				viper.GetString(flagFabClientCertPath),
				viper.GetString(flagFabClientPrivateKeyPath),
			)
		},
	}

	return populateWalletFlag(cmd)
}

func showAddressCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address [chain-id]",
		Short: "show an address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}
			fc := c.ChainI.(*fabric.Chain)
			if err := fc.Connect(); err != nil {
				return err
			}
			sid, err := fc.GetSerializedIdentity(fc.Config().WalletLabel)
			if err != nil {
				return err
			}
			addr, err := fabricauthtypes.MakeCreatorAddressWithSerializedIdentity(sid)
			if err != nil {
				return err
			}
			fmt.Println(addr.String())
			return nil
		},
	}

	return cmd
}
