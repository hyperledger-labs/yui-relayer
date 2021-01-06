package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	tendermintcmd "github.com/datachainlab/relayer/chains/tendermint/cmd"
	"github.com/datachainlab/relayer/config"
	"github.com/datachainlab/relayer/encoding"
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

	ec := encoding.MakeEncodingConfig()

	// Register subcommands
	rootCmd.AddCommand(
		configCmd(),
		chainsCmd(ec.Marshaler),
		transactionCmd(),
		pathsCmd(),
		flags.LineBreak,
		tendermintcmd.TendermintCmd(ec.Marshaler, makeConfigManager()),
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

func makeConfigManager() config.ConfigManager {
	return configManager{}
}

type configManager struct{}

func (configManager) Get() *config.Config {
	return configInstance
}

func (configManager) Set(config config.Config) {
	configInstance = &config
}

// readLineFromBuf reads one line from stdin.
func readStdin() (string, error) {
	str, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(str), err
}
