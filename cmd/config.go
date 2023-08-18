package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func configCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "manage configuration file",
	}

	cmd.AddCommand(
		configShowCmd(ctx),
		configInitCmd(),
		configEditCmd(ctx),
	)

	return cmd
}

// Command for inititalizing an empty config at the --home location
func configInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Creates a default home directory at path defined by --home",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cmd.Flags().GetString(flags.FlagHome)
			if err != nil {
				return err
			}

			cfgDir := path.Join(home, "config")
			cfgPath := path.Join(cfgDir, "config.yaml")

			// If the config doesn't exist...
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				// And the config folder doesn't exist...
				if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
					// And the home folder doesn't exist
					if _, err := os.Stat(home); os.IsNotExist(err) {
						// Create the home folder
						if err = os.Mkdir(home, os.ModePerm); err != nil {
							return err
						}
					}
					// Create the home config folder
					if err = os.Mkdir(cfgDir, os.ModePerm); err != nil {
						return err
					}
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
			home, err := cmd.Flags().GetString(flags.FlagHome)
			if err != nil {
				return err
			}

			cfgPath := path.Join(home, "config", "config.yaml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if _, err := os.Stat(home); os.IsNotExist(err) {
					return fmt.Errorf("home path does not exist: %s", home)
				}
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

func configEditCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit [path-name] [key] [value]",
		Aliases: []string{"e"},
		Short:   "Edit the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cmd.Flags().GetString(flags.FlagHome)
			if err != nil {
				return err
			}
			cfgPath := path.Join(home, "config", "config.yaml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if _, err := os.Stat(home); os.IsNotExist(err) {
					return fmt.Errorf("home path does not exist: %s", home)
				}
				return fmt.Errorf("config does not exist: %s", cfgPath)
			}
			pathName := args[0]
			key := args[1]
			value := args[2]
			configPath, err := ctx.Config.Paths.Get(pathName)
			if err != nil {
				return err
			}
			switch key {
			case "client-id":
				configPath.Src.ClientID = value
				configPath.Dst.ClientID = value
			case "channel-id":
				configPath.Src.ChannelID = value
				configPath.Dst.ChannelID = value
			case "connection-id":
				configPath.Src.ConnectionID = value
				configPath.Dst.ConnectionID = value
			case "port-id":
				configPath.Src.PortID = value
				configPath.Dst.PortID = value
			default:
				return fmt.Errorf("invalid key: %s. Valid keys are: client-id, channel-id, connection-id, port-id", key)
			}
			ctx.Config.Paths[pathName] = configPath
			out, err := config.MarshalJSON(*ctx.Config)
			if err != nil {
				return err
			}
			err = os.WriteFile(cfgPath, out, 0600)
			if err != nil {
				return err
			}
			fmt.Println("config file updated")
			return nil
		},
	}
	return cmd
}

func defaultConfig() []byte {
	bz, err := json.Marshal(config.DefaultConfig())
	if err != nil {
		panic(err)
	}
	return bz
}

// initConfig reads in config file and ENV variables if set.
func initConfig(ctx *config.Context, cmd *cobra.Command) error {
	home, err := cmd.PersistentFlags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}

	cfgPath := path.Join(home, "config", "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		viper.SetConfigFile(cfgPath)
		if err := viper.ReadInConfig(); err == nil {
			// read the config file bytes
			file, err := os.ReadFile(viper.ConfigFileUsed())
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}

			// unmarshall them into the struct
			err = config.UnmarshalJSON(ctx.Codec, file, ctx.Config)
			if err != nil {
				fmt.Println("Error unmarshalling config:", err)
				os.Exit(1)
			}

			// ensure config has []*relayer.Chain used for all chain operations
			err = config.InitChains(ctx, homePath, debug)
			if err != nil {
				fmt.Println("Error parsing chain config:", err)
				os.Exit(1)
			}
		}
	}
	return nil
}
