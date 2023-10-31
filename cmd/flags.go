package cmd

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagJSON                = "json"
	flagYAML                = "yaml"
	flagFile                = "file"
	flagTimeout             = "timeout"
	flagTimeoutHeightOffset = "timeout-height-offset"
	flagTimeoutTimeOffset   = "timeout-time-offset"
	flagIBCDenoms           = "ibc-denoms"
)

func heightFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Uint64(flags.FlagHeight, 0, "Height of headers to fetch")
	if err := viper.BindPFlag(flags.FlagHeight, cmd.Flags().Lookup(flags.FlagHeight)); err != nil {
		panic(err)
	}
	return cmd
}

func fileFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagFile, "f", "", "fetch json data from specified file")
	if err := viper.BindPFlag(flagFile, cmd.Flags().Lookup(flagFile)); err != nil {
		panic(err)
	}
	return cmd
}

func yamlFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().BoolP(flagYAML, "y", false, "output using yaml")
	if err := viper.BindPFlag(flagYAML, cmd.Flags().Lookup(flagYAML)); err != nil {
		panic(err)
	}
	return cmd
}

func jsonFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().BoolP(flagJSON, "j", false, "returns the response in json format")
	if err := viper.BindPFlag(flagJSON, cmd.Flags().Lookup(flagJSON)); err != nil {
		panic(err)
	}
	return cmd
}

func timeoutFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(flagTimeout, "o", "1s", "timeout between relayer runs")
	if err := viper.BindPFlag(flagTimeout, cmd.Flags().Lookup(flagTimeout)); err != nil {
		panic(err)
	}
	return cmd
}

func getTimeout(cmd *cobra.Command) (time.Duration, error) {
	to, err := cmd.Flags().GetString(flagTimeout)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(to)
}

func timeoutFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Uint64P(flagTimeoutHeightOffset, "y", 0, "set timeout height offset for ")
	cmd.Flags().DurationP(flagTimeoutTimeOffset, "c", time.Duration(0), "specify the path to relay over")
	if err := viper.BindPFlag(flagTimeoutHeightOffset, cmd.Flags().Lookup(flagTimeoutHeightOffset)); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag(flagTimeoutTimeOffset, cmd.Flags().Lookup(flagTimeoutTimeOffset)); err != nil {
		panic(err)
	}
	return cmd
}

func ibcDenomFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().BoolP(flagIBCDenoms, "i", false, "Display IBC denominations for sending tokens back to other chains")
	if err := viper.BindPFlag(flagIBCDenoms, cmd.Flags().Lookup(flagIBCDenoms)); err != nil {
		panic(err)
	}
	return cmd
}
