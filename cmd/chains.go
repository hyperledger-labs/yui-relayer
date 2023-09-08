package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func chainsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chains",
		Short: "manage chain configurations",
		RunE:  noCommand,
	}

	cmd.AddCommand(
		chainsAddDirCmd(ctx),
	)

	return cmd
}

func chainsAddDirCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "add-dir [dir]",
		Args: cobra.ExactArgs(1),
		Short: `Add new chains to the configuration file from a directory 
		full of chain configuration, useful for adding testnet configurations`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := filesAdd(ctx, args[0]); err != nil {
				return err
			}
			return overWriteConfig(ctx, cmd)
		},
	}

	return cmd
}

func filesAdd(ctx *config.Context, dir string) error {
	dir = path.Clean(dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		pth := fmt.Sprintf("%s/%s", dir, f.Name())
		if f.IsDir() {
			fmt.Printf("directory at %s, skipping...\n", pth)
			continue
		}
		byt, err := os.ReadFile(pth)
		if err != nil {
			return fmt.Errorf("failed to read file %s, error: %v", pth, err)
		}
		var c core.ChainProverConfig
		if err := json.Unmarshal(byt, &c); err != nil {
			return fmt.Errorf("failed to unmarshal file %s, error: %v", pth, err)
		}
		if err := c.Init(ctx.Codec); err != nil {
			return fmt.Errorf("failed to init chain %s, error: %v", pth, err)
		}
		if err = ctx.Config.AddChain(ctx.Codec, c); err != nil {
			return fmt.Errorf("failed to add chain %s, error: %v", pth, err)
		}
		chain, err := c.Build()
		if err != nil {
			return err
		}
		fmt.Printf("added %s...\n", chain.ChainID())
	}
	return nil
}

func overWriteConfig(ctx *config.Context, cmd *cobra.Command) error {
	home, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}

	cfgPath := path.Join(home, "config", "config.yaml")
	if _, err = os.Stat(cfgPath); err == nil {
		viper.SetConfigFile(cfgPath)
		if err = viper.ReadInConfig(); err == nil {
			// ensure validateConfig runs properly
			err = config.InitChains(ctx, homePath, debug)
			if err != nil {
				return err
			}

			// marshal the new config
			out, err := config.MarshalJSON(*ctx.Config)
			if err != nil {
				return err
			}

			// overwrite the config file
			err = os.WriteFile(viper.ConfigFileUsed(), out, 0600)
			if err != nil {
				return err
			}
		}
	}
	return err
}
