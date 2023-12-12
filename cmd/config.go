package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
)

func configCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "manage configuration file",
		RunE:    noCommand,
	}

	cmd.AddCommand(
		configShowCmd(ctx),
		configInitCmd(ctx),
	)

	return cmd
}

// Command for inititalizing an empty config at the --home location
func configInitCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Creates a default home directory at path defined by --home",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := ctx.Config.ConfigPath
			// If the config doesn't exist...
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				dirPath := filepath.Dir(cfgPath)
				if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
					return err
				}
				// Then create the file...
				f, err := os.Create(cfgPath)
				if err != nil {
					return err
				}
				defer f.Close()

				// And write the default config to that location...
				if _, err = f.Write(defaultConfig()); err != nil {
					return err
				}

				// And return no error...
				return nil
			}

			// Otherwise, the config file exists, and an error is returned...
			return fmt.Errorf("config already exists: %s", cfgPath)
		},
	}
	return cmd
}

// Command for printing current configuration
func configShowCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"s", "list", "l"},
		Short:   "Prints current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := ctx.Config.ConfigPath
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				return fmt.Errorf("config does not exist: %s", cfgPath)
			}

			out, err := config.MarshalJSON(*ctx.Config)
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}

func defaultConfig() []byte {
	bz, err := json.Marshal(config.DefaultConfig(fmt.Sprintf("%s/%s", homePath, configPath)))
	if err != nil {
		panic(err)
	}
	return bz
}

// initConfig reads in config file and ENV variables if set.
func initConfig(ctx *config.Context, cmd *cobra.Command) error {
	cfgPath := ctx.Config.ConfigPath
	if _, err := os.Stat(cfgPath); err == nil {
		file, err := os.ReadFile(cfgPath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			os.Exit(1)
		}

		// unmarshall them into the struct
		if err = config.UnmarshalJSON(ctx.Codec, file, ctx.Config); err != nil {
			fmt.Println("Error unmarshalling config:", err)
			os.Exit(1)
		}

		// ensure config has []*relayer.Chain used for all chain operations
		if err = config.InitChains(ctx, homePath, debug); err != nil {
			fmt.Println("Error parsing chain config:", err)
			os.Exit(1)
		}
	} else {
		defConfig := config.DefaultConfig(fmt.Sprintf("%s/%s", homePath, configPath))
		ctx.Config = &defConfig
	}
	ctx.Config.InitCoreConfig()
	return nil
}
