package cmd

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
)

// transactionCmd represents the tx command
func transactionCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "IBC Transaction Commands",
		Long: strings.TrimSpace(`Commands to create IBC transactions on configured chains. 
		Most of these commands take a '[path]' argument. Make sure:
	1. Chains are properly configured to relay over by using the 'rly chains list' command
	2. Path is properly configured to relay over by using the 'rly paths list' command`),
	}

	cmd.AddCommand(
		xfersend(ctx),
		relayMsgsCmd(ctx),
		relayAcksCmd(ctx),
		flags.LineBreak,
		createClientsCmd(ctx),
		updateClientsCmd(ctx),
		createConnectionCmd(ctx),
		createChannelCmd(ctx),
	)

	return cmd
}

func createClientsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clients [path-name]",
		Short: "create a clients between two configured chains with a configured path",
		Long: "Creates a working ibc client for chain configured on each end of the" +
			" path by querying headers from each chain and then sending the corresponding create-client messages",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.CreateClients(c[src], c[dst])
		},
	}
	return cmd
}

func updateClientsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-clients [path-name]",
		Short: "update the clients between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to updates 
			the clients with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.UpdateClients(c[src], c[dst])
		},
	}
	return cmd
}

func createConnectionCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection [path-name]",
		Short: "create a connection between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to repair or create 
		a connection between two chains with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			to, err := getTimeout(cmd)
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.CreateConnection(c[src], c[dst], to)
		},
	}

	return timeoutFlag(cmd)
}

func createChannelCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [path-name]",
		Short: "create a channel between two configured chains with a configured path",
		Long: strings.TrimSpace(`This command is meant to be used to repair or 
		create a channel between two chains with a configured path in the config file`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}

			to, err := getTimeout(cmd)
			if err != nil {
				return err
			}

			// ensure that keys exist
			if _, err = c[src].GetAddress(); err != nil {
				return err
			}
			if _, err = c[dst].GetAddress(); err != nil {
				return err
			}

			return core.CreateChannel(c[src], c[dst], false, to)
		},
	}

	return timeoutFlag(cmd)
}

func relayMsgsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay [path-name]",
		Short: "relay any packets that remain to be relayed on a given path, in both directions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			path, err := ctx.Config.Paths.Get(args[0])
			if err != nil {
				return err
			}
			sh, err := core.NewSyncHeaders(c[src], c[dst])
			if err != nil {
				return err
			}
			st, err := core.GetStrategy(*path.Strategy)
			if err != nil {
				return err
			}

			sp, err := st.UnrelayedSequences(c[src], c[dst], sh)
			if err != nil {
				return err
			}

			if err = st.RelayPackets(c[src], c[dst], sp, sh); err != nil {
				return err
			}

			return nil
		},
	}
	// TODO add option support for strategy
	return cmd
}

func relayAcksCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "relay-acknowledgements [path-name]",
		Aliases: []string{"acks"},
		Short:   "relay any acknowledgements that remain to be relayed on a given path, in both directions",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, src, dst, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			path, err := ctx.Config.Paths.Get(args[0])
			if err != nil {
				return err
			}
			sh, err := core.NewSyncHeaders(c[src], c[dst])
			if err != nil {
				return err
			}
			st, err := core.GetStrategy(*path.Strategy)
			if err != nil {
				return err
			}

			// sp.Src contains all sequences acked on SRC but acknowledgement not processed on DST
			// sp.Dst contains all sequences acked on DST but acknowledgement not processed on SRC
			sp, err := st.UnrelayedAcknowledgements(c[src], c[dst], sh)
			if err != nil {
				return err
			}

			if err = st.RelayAcknowledgements(c[src], c[dst], sp, sh); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
