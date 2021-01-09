package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagJSON    = "json"
	flagYAML    = "yaml"
	flagFile    = "file"
	flagTimeout = "timeout"
)

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
	cmd.Flags().StringP(flagTimeout, "o", "10s", "timeout between relayer runs")
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
