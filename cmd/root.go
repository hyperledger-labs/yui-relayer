package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/metrics"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homePath         string
	debug            bool
	timeout          time.Duration
	logLevel         string
	logFormat        string
	logOutput        string
	defaultHome      = os.ExpandEnv("$HOME/.yui-relayer")
	defaultTimeout   = 10 * time.Second
	defaultLogLevel  = "DEBUG"
	defaultLogFormat = "json"
	defaultLogOutput = "stderr"
	configPath       = "config/config.json"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// It can support any chain by giving modules.
func Execute(modules ...config.ModuleI) error {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "yrly",
		Short: "This application relays data between configured IBC enabled chains",
		RunE:  noCommand,
	}

	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Register top level flags --home and --debug
	rootCmd.PersistentFlags().StringVar(&homePath, flags.FlagHome, defaultHome, "set home directory")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", defaultTimeout, "rpc timeout duration")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", defaultLogLevel, "set the log level")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", defaultLogFormat, "set the log format")
	rootCmd.PersistentFlags().StringVar(&logOutput, "log-output", defaultLogOutput, "set the log output")
	if err := viper.BindPFlag(flags.FlagHome, rootCmd.PersistentFlags().Lookup(flags.FlagHome)); err != nil {
		return err
	}
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		return err
	}

	// Register interfaces

	codec := core.MakeCodec()
	for _, module := range modules {
		module.RegisterInterfaces(codec.InterfaceRegistry())
	}
	ctx := &config.Context{Modules: modules, Config: &config.Config{}, Codec: codec}

	// Register subcommands

	rootCmd.AddCommand(
		configCmd(ctx),
		chainsCmd(ctx),
		transactionCmd(ctx),
		pathsCmd(ctx),
		queryCmd(ctx),
		modulesCmd(ctx),
		serviceCmd(ctx),
		flags.LineBreak,
	)
	for _, module := range modules {
		if cmd := module.GetCmd(ctx); cmd != nil {
			rootCmd.AddCommand(cmd)
		}
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// reads `homeDir/config/config.json` into `var config *Config` before each command
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("failed to bind the flag set to the configuration: %v", err)
		}
		if err := ctx.Config.InitConfig(ctx, homePath, configPath, debug, timeout); err != nil {
			return fmt.Errorf("failed to initialize the configuration: %v", err)
		}
		if err := log.InitLogger(logLevel, logFormat, logOutput); err != nil {
			return err
		}
		if err := metrics.InitializeMetrics(metrics.ExporterNull{}); err != nil {
			return fmt.Errorf("failed to initialize the metrics: %v", err)
		}
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		if err := metrics.ShutdownMetrics(cmd.Context()); err != nil {
			return fmt.Errorf("failed to shutdown the metrics subsystem: %v", err)
		}
		return nil
	}

	return rootCmd.Execute()
}

// readLineFromBuf reads one line from stdin.
func readStdin() (string, error) {
	str, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(str), err
}

func noCommand(cmd *cobra.Command, args []string) error {
	cmd.Help()
	return errors.New("specified command does not exist")
}
