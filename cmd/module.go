package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"slices"
	"sort"
	"strings"

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
			modules := make([]string, len(ctx.Modules))
			bi, ok := debug.ReadBuildInfo()
			if !ok {
				return fmt.Errorf("could not read build info")
			}

			for i, m := range ctx.Modules {
				info, err := retrieveModuleInfo(bi, m)
				if err != nil {
					return err
				}

				modules[i] = m.Name() + " " + info
			}
			sort.Strings(modules)
			for _, module := range modules {
				fmt.Printf("%v\n", module)
			}
			return nil
		},
	}
	return cmd
}

func retrieveModuleInfo(info *debug.BuildInfo, m config.ModuleI) (string, error) {
	if info == nil {
		return "", errors.New("build info is unavailable")
	}

	pkgPath := reflect.TypeOf(m).PkgPath()
	if strings.HasPrefix(pkgPath, info.Main.Path) {
		return info.Main.Path + " " + info.Main.Version, nil
	}

	i := slices.IndexFunc(info.Deps, func(dm *debug.Module) bool {
		return strings.HasPrefix(pkgPath, dm.Path)
	})
	if i == -1 {
		return "", fmt.Errorf("could not find module info for %s", m.Name())
	}

	return info.Deps[i].Path + " " + info.Deps[i].Version, nil
}
