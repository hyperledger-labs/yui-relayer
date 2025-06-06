package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
			if err := ctx.Config.CreateConfig(); err != nil {
				return err
			}
			return nil
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
			out, err := json.Marshal(*ctx.Config)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}
