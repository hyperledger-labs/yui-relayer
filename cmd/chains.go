package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/datachainlab/relayer/config"
	"github.com/datachainlab/relayer/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func chainsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chains",
		Short: "manage chain configurations",
	}

	cmd.AddCommand(
		chainsAddDirCmd(ctx),
		chainsEditCmd(ctx),
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
			return overWriteConfig(cmd, ctx.Config)
		},
	}

	return cmd
}

func chainsEditCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [chain-id] [key] [value]",
		Short: "Returns chain configuration data",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := ctx.Config.GetChain(args[0])
			if err != nil {
				return err
			}

			c, err := chain.Update(args[1], args[2])
			if err != nil {
				return err
			}

			if err = ctx.Config.DeleteChain(args[0]).AddChain(ctx.Marshaler, c); err != nil {
				return err
			}

			return overWriteConfig(cmd, ctx.Config)
		},
	}
	return cmd
}

func filesAdd(ctx *config.Context, dir string) error {
	dir = path.Clean(dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		pth := fmt.Sprintf("%s/%s", dir, f.Name())
		if f.IsDir() {
			fmt.Printf("directory at %s, skipping...\n", pth)
			continue
		}
		byt, err := ioutil.ReadFile(pth)
		if err != nil {
			fmt.Printf("failed to read file %s, skipping...\n", pth)
			continue
		}
		var c core.ChainConfigI
		if err = config.UnmarshalJSONAny(ctx.Marshaler, &c, byt); err != nil {
			fmt.Printf("failed to unmarshal file %s, skipping...\n", pth)
			continue
		}
		if err = ctx.Config.AddChain(ctx.Marshaler, c); err != nil {
			fmt.Printf("%s: %s\n", pth, err.Error())
			continue
		}
		fmt.Printf("added %s...\n", c.GetChain().ChainID())
	}
	return nil
}

func overWriteConfig(cmd *cobra.Command, cfg *config.Config) error {
	home, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}

	cfgPath := path.Join(home, "config", "config.yaml")
	if _, err = os.Stat(cfgPath); err == nil {
		viper.SetConfigFile(cfgPath)
		if err = viper.ReadInConfig(); err == nil {
			// ensure validateConfig runs properly
			err = config.InitChains(cfg, homePath, debug)
			if err != nil {
				return err
			}

			// marshal the new config
			out, err := config.MarshalJSON(*cfg)
			if err != nil {
				return err
			}

			// overwrite the config file
			err = ioutil.WriteFile(viper.ConfigFileUsed(), out, 0600)
			if err != nil {
				return err
			}
		}
	}
	return err
}
