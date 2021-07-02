package cmd

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
)

// NOTE: These commands are registered over in cmd/raw.go

func xfersend(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [path-name] [src-chain-id] [dst-chain-id] [amount] [dst-addr]",
		Short: "Initiate a transfer from one chain to another",
		Long: "Sends the first step to transfer tokens in an IBC transfer." +
			" The created packet must be relayed to another chain",
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := ctx.Config.ChainsFromPath(args[0])
			if err != nil {
				return err
			}
			src := args[1]
			dst := args[2]

			if _, ok := c[src]; !ok {
				return fmt.Errorf("not found chain '%v' in the path", src)
			}

			if _, ok := c[dst]; !ok {
				return fmt.Errorf("not found chain '%v' in the path", dst)
			}

			toHeightOffset, err := cmd.Flags().GetUint64(flagTimeoutHeightOffset)
			if err != nil {
				return err
			}

			toTimeOffset, err := cmd.Flags().GetDuration(flagTimeoutTimeOffset)
			if err != nil {
				return err
			}

			// XXX allow `-` in denom traces
			sdk.SetCoinDenomRegex(func() string {
				return `[a-zA-Z][a-zA-Z0-9/\-]{2,127}`
			})
			amount, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}
			// XXX want to support all denom format
			denom := transfertypes.ParseDenomTrace(amount.Denom)
			if denom.Path != "" {
				amount.Denom = denom.IBCDenom()
			}

			dstAddr, err := sdk.AccAddressFromBech32(args[4])
			if err != nil {
				return err
			}

			switch {
			case toHeightOffset > 0 && toTimeOffset > 0:
				return fmt.Errorf("cannot set both --timeout-height-offset and --timeout-time-offset, choose one")
			case toHeightOffset > 0:
				return core.SendTransferMsg(c[src], c[dst], amount, dstAddr, toHeightOffset, 0)
			case toTimeOffset > 0:
				return core.SendTransferMsg(c[src], c[dst], amount, dstAddr, 0, toTimeOffset)
			case toHeightOffset == 0 && toTimeOffset == 0:
				return core.SendTransferMsg(c[src], c[dst], amount, dstAddr, 0, 0)
			default:
				return fmt.Errorf("shouldn't be here")
			}
		},
	}
	return timeoutFlags(cmd)
}
