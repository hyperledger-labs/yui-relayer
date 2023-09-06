package cmd

import (
	"fmt"
	"sort"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/spf13/cobra"
)

func modulesCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modules",
		Short: "show an info about Relayer Module",
		RunE:  noCommand,
	}

	cmd.AddCommand(
		showModulesCmd(ctx),
	)

	return cmd
}

func showModulesCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Shows a list of modules included in the relayer",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := make([]string, len(ctx.Modules))
			for i, m := range ctx.Modules {
				names[i] = m.Name()
			}
			sort.Strings(names)
			for _, name := range names {
				fmt.Printf("%v\n", name)
			}
			return nil
		},
	}
	return cmd
}
