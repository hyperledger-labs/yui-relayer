package cmd

import (
	"bufio"
	"os"
	"strings"

	// "github.com/hyperledger-labs/yui-relayer/chains/corda"
	"github.com/hyperledger-labs/yui-relayer/chains/fabric"
	fabriccmd "github.com/hyperledger-labs/yui-relayer/chains/fabric/cmd"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	tendermintcmd "github.com/hyperledger-labs/yui-relayer/chains/tendermint/cmd"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homePath string
	debug    bool
	// configInstance *config.Config
	defaultHome = os.ExpandEnv("$HOME/.urelayer")
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

	ec := core.MakeEncodingConfig()
	tendermint.RegisterInterfaces(ec.InterfaceRegistry)
	fabric.RegisterInterfaces(ec.InterfaceRegistry)
	// corda.RegisterInterfaces(ec.InterfaceRegistry)
	ctx := &config.Context{Config: &config.Config{}, Marshaler: ec.Marshaler}

	// Register subcommands
	rootCmd.AddCommand(
		configCmd(ctx),
		chainsCmd(ctx),
		transactionCmd(ctx),
		pathsCmd(ctx),
		queryCmd(ctx),
		serviceCmd(ctx),
		flags.LineBreak,
		tendermintcmd.TendermintCmd(ec.Marshaler, ctx),
		fabriccmd.FabricCmd(ctx),
	)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// reads `homeDir/config/config.yaml` into `var config *Config` before each command
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		return initConfig(ctx, rootCmd)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// readLineFromBuf reads one line from stdin.
func readStdin() (string, error) {
	str, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(str), err
}
