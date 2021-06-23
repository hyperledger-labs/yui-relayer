package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	accounttypes "github.com/datachainlab/cross/x/core/account/types"
	"github.com/hyperledger-labs/yui-relayer/tests/tendermint/simapp/samplemod/types"
	"github.com/spf13/cobra"
)

func GetCounter() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "counter",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadQueryCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			keyName := args[0]
			keyInfo, err := clientCtx.Keyring.Key(keyName)
			if err != nil {
				return err
			}

			q := types.NewQueryClient(clientCtx)
			res, err := q.Counter(
				context.Background(),
				&types.QueryCounterRequest{
					Account: accounttypes.AccountIDFromAccAddress(keyInfo.GetAddress()),
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintString(fmt.Sprint(res.Value))
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test)")
	return cmd
}
