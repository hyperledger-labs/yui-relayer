package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homePath    string
	debug       bool
	defaultHome = os.ExpandEnv("$HOME/.urelayer")
)

// Execute adds all child commands to the root command and sets flags appropriately.
// It can support any chain by giving modules.
func Execute(modules ...config.ModuleI) error {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "uly",
		Short: "This application relays data between configured IBC enabled chains",
	}

	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true

	// Register top level flags --home and --debug
	rootCmd.PersistentFlags().StringVar(&homePath, flags.FlagHome, defaultHome, "set home directory")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output")
	if err := viper.BindPFlag(flags.FlagHome, rootCmd.Flags().Lookup(flags.FlagHome)); err != nil {
		return err
	}
	if err := viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug")); err != nil {
		return err
	}

	// Register interfaces

	codec := core.MakeCodec()
	for _, module := range modules {
		module.RegisterInterfaces(codec.InterfaceRegistry())
	}
	ctx := &config.Context{Config: &config.Config{}, Codec: codec}

	// Register subcommands

	rootCmd.AddCommand(
		configCmd(ctx),
		chainsCmd(ctx),
		transactionCmd(ctx),
		pathsCmd(ctx),
		queryCmd(ctx),
		serviceCmd(ctx),
		flags.LineBreak,
	)
	for _, module := range modules {
		if cmd := module.GetCmd(ctx); cmd != nil {
			rootCmd.AddCommand(cmd)
		}
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// reads `homeDir/config/config.yaml` into `var config *Config` before each command
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		return initConfig(ctx, rootCmd)
	}

	return rootCmd.Execute()
}

// readLineFromBuf reads one line from stdin.
func readStdin() (string, error) {
	str, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(str), err
}
