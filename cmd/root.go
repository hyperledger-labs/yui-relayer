package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/metrics"
	"golang.org/x/sys/unix"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	homePath    string
	debug       bool
	defaultHome = os.ExpandEnv("$HOME/.yui-relayer")
	configPath  = "config/config.json"
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
		if err := ctx.Config.UnmarshalConfig(homePath, configPath); err != nil {
			return fmt.Errorf("failed to initialize the configuration: %v", err)
		}
		if err := initLogger(ctx); err != nil {
			return err
		}
		if err := ctx.InitConfig(homePath, debug); err != nil {
			return fmt.Errorf("failed to initialize the configuration: %v", err)
		}
		if err := metrics.InitializeMetrics(metrics.ExporterNull{}); err != nil {
			return fmt.Errorf("failed to initialize the metrics: %v", err)
		}
		cmd.SetContext(notifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM))
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

// readStdin reads one line from stdin.
func readStdin() (string, error) {
	str, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(str), err
}

func initLogger(ctx *config.Context) error {
	c := ctx.Config.Global.LoggerConfig
	return log.InitLogger(c.Level, c.Format, c.Output)
}

func noCommand(cmd *cobra.Command, args []string) error {
	cmd.Help()
	return errors.New("specified command does not exist")
}

func notifyContext(ctx context.Context, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, signals...)

	go func() {
		sig := <-sigChan
		var sigName string
		if s, ok := sig.(syscall.Signal); ok {
			sigName = unix.SignalName(s)
		} else {
			sigName = s.String()
		}
		log.GetLogger().Info(fmt.Sprintf("Received %s. Shutting down...", sigName))
		cancel()
	}()

	return ctx
}
