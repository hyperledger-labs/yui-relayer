package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func pathsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "paths",
		Short: "manage path configurations",
		Long: `
A path represents the "full path" or "link" for communication between two chains. This includes the client, 
connection, and channel ids from both the source and destination chains as well as the strategy to use when relaying`,
	}

	cmd.AddCommand(
		pathsListCmd(ctx),
		pathsAddCmd(ctx),
	)

	return cmd
}

func pathsListCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "print out configured paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsn, _ := cmd.Flags().GetBool(flagJSON)
			yml, _ := cmd.Flags().GetBool(flagYAML)
			switch {
			case yml && jsn:
				return fmt.Errorf("can't pass both --json and --yaml, must pick one")
			case yml:
				out, err := yaml.Marshal(ctx.Config.Paths)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			default: // default format is json
				out, err := json.Marshal(ctx.Config.Paths)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}
		},
	}
	return yamlFlag(jsonFlag(cmd))
}

func pathsAddCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [path-name]",
		Short: "add a path to the list of paths",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := cmd.Flags().GetString(flagFile)
			if err != nil {
				return err
			}
			if err := fileInputPathAdd(ctx, file, args[0]); err != nil {
				return err
			}
			return overWriteConfig(ctx, cmd)
		},
	}
	cmd = fileFlag(cmd)
	cmd.MarkFlagRequired(flagFile)
	return cmd
}

func fileInputPathAdd(ctx *config.Context, file, name string) error {
	// If the user passes in a file, attempt to read the chain config from that file
	if _, err := os.Stat(file); err != nil {
		return err
	}

	byt, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	path, err := core.UnmarshalPath(ctx.Codec, byt)
	if err != nil {
		return err
	}

	if err = ctx.Config.Paths.Add(name, path); err != nil {
		return err
	}
	return nil
}
