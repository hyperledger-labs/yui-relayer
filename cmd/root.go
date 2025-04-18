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
	"github.com/hyperledger-labs/yui-relayer/internal/telemetry"
	"github.com/hyperledger-labs/yui-relayer/log"
	"golang.org/x/sys/unix"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configPath          = "config/config.json"
	flagEnableTelemetry = "enable-telemetry"
)

var (
	defaultHome = os.ExpandEnv("$HOME/.yui-relayer")
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

	// Register top level flags
	rootCmd.PersistentFlags().String(flags.FlagHome, defaultHome, "set home directory")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "debug output")
	rootCmd.PersistentFlags().Bool(flagEnableTelemetry, false, "enable telemetry")
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		return err
	}

	// Allow viper.BindEnv("some-flag") to use "YRLY_SOME_FLAG" environment variable
	viper.SetEnvPrefix("yrly")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	if err := viper.BindEnv(flagEnableTelemetry); err != nil {
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

	shutdown := func(_ context.Context) error { return nil }
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// reads `homeDir/config/config.json` into `var config *Config` before each command
		homePath := viper.GetString(flags.FlagHome)
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("failed to bind the flag set to the configuration: %v", err)
		}
		if err := ctx.Config.UnmarshalConfig(homePath, configPath); err != nil {
			return fmt.Errorf("failed to initialize the configuration: %v", err)
		}
		if err := initLogger(ctx, viper.GetBool(flagEnableTelemetry)); err != nil {
			return err
		}
		if err := ctx.InitConfig(homePath, viper.GetBool("debug")); err != nil {
			return fmt.Errorf("failed to initialize the configuration: %v", err)
		}

		if err := telemetry.InitializeMetrics(); err != nil {
			return fmt.Errorf("failed to initialize the metrics: %v", err)
		}

		if viper.GetBool(flagEnableTelemetry) {
			var err error
			shutdown, err = telemetry.SetupOTelSDK(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to initialize the telemetry: %v", err)
			}
		}

		cmd.SetContext(notifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM))
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		if err := shutdown(context.Background()); err != nil {
			return fmt.Errorf("failed to shutdown the telemetries: %v", err)
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

func initLogger(ctx *config.Context, enableTelemetry bool) error {
	c := ctx.Config.Global.LoggerConfig
	return log.InitLogger(c.Level, c.Format, c.Output, enableTelemetry)
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
