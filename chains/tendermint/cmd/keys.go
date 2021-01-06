package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/datachainlab/relayer/chains/tendermint"
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"

	"github.com/cosmos/relayer/relayer"
)

// keysCmd represents the keys command
func keysCmd(cmgr config.ConfigManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Aliases: []string{"k"},
		Short:   "manage keys held by the relayer for each chain",
	}

	cmd.AddCommand(keysAddCmd(cmgr))
	cmd.AddCommand(keysRestoreCmd(cmgr))
	cmd.AddCommand(keysShowCmd(cmgr))

	return cmd
}

// keysAddCmd respresents the `keys add` command
func keysAddCmd(cmgr config.ConfigManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [chain-id] [[name]]",
		Aliases: []string{"a"},
		Short:   "adds a key to the keychain associated with a particular chain",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmgr.Get().GetChain(args[0])
			if err != nil {
				return err
			}
			chain := c.(*tendermint.Chain).Base()

			var keyName string
			if len(args) == 2 {
				keyName = args[1]
			} else {
				keyName = chain.Key
			}

			if chain.KeyExists(keyName) {
				return errKeyExists(keyName)
			}

			mnemonic, err := relayer.CreateMnemonic()
			if err != nil {
				return err
			}

			info, err := chain.Keybase.NewAccount(keyName, mnemonic, "", hd.CreateHDPath(118, 0, 0).String(), hd.Secp256k1)
			if err != nil {
				return err
			}

			ko := keyOutput{Mnemonic: mnemonic, Address: info.GetAddress().String()}

			out, err := json.Marshal(&ko)
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}

type keyOutput struct {
	Mnemonic string `json:"mnemonic" yaml:"mnemonic"`
	Address  string `json:"address" yaml:"address"`
}

// keysRestoreCmd respresents the `keys add` command
func keysRestoreCmd(cmgr config.ConfigManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restore [chain-id] [name] [mnemonic]",
		Aliases: []string{"r"},
		Short:   "restores a mnemonic to the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyName := args[1]
			c, err := cmgr.Get().GetChain(args[0])
			if err != nil {
				return err
			}
			chain := c.(*tendermint.Chain).Base()

			if chain.KeyExists(keyName) {
				return errKeyExists(keyName)
			}

			info, err := chain.Keybase.NewAccount(keyName, args[2], "", hd.CreateHDPath(118, 0, 0).String(), hd.Secp256k1)
			if err != nil {
				return err
			}

			defer chain.UseSDKContext()()
			fmt.Println(info.GetAddress().String())
			return nil
		},
	}

	return cmd
}

// keysShowCmd respresents the `keys show` command
func keysShowCmd(cmgr config.ConfigManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [chain-id] [[name]]",
		Aliases: []string{"s"},
		Short:   "shows a key from the keychain associated with a particular chain",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmgr.Get().GetChain(args[0])
			if err != nil {
				return err
			}
			chain := c.(*tendermint.Chain).Base()

			var keyName string
			if len(args) == 2 {
				keyName = args[1]
			} else {
				keyName = chain.Key
			}

			if !chain.KeyExists(keyName) {
				return errKeyDoesntExist(keyName)
			}

			info, err := chain.Keybase.Key(keyName)
			if err != nil {
				return err
			}

			fmt.Println(info.GetAddress().String())
			return nil
		},
	}

	return cmd
}
