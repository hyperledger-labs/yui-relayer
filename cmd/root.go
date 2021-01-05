package cmd

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/datachainlab/relayer/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homePath       string
	debug          bool
	configInstance *config.Config
	defaultHome    = os.ExpandEnv("$HOME/.urelayer")
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uly",
	Short: "This application relays data between configured IBC enabled chains",
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true

	// Register top level flags --home and --debug
	rootCmd.PersistentFlags().StringVar(&homePath, flags.FlagHome, defaultHome, "set home directory")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output")
	if err := viper.BindPFlag(flags.FlagHome, rootCmd.Flags().Lookup(flags.FlagHome)); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug")); err != nil {
		panic(err)
	}

	// Register subcommands
	rootCmd.AddCommand(
		configCmd(),
	)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		// reads `homeDir/config/config.yaml` into `var config *Config` before each command
		return initConfig(rootCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
