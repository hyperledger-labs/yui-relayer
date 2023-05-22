package helpers

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
)

// QueryBalance is a helper function for query balance
func QueryBalance(chain *core.ProvableChain, height ibcexported.Height, address sdk.AccAddress, showDenoms bool) (sdk.Coins, error) {
	ctx := core.NewQueryContext(context.TODO(), height)
	coins, err := chain.QueryBalance(ctx, address)
	if err != nil {
		return nil, err
	}

	if showDenoms {
		return coins, nil
	}

	dts, err := chain.QueryDenomTraces(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	if len(dts.DenomTraces) == 0 {
		return coins, nil
	}

	var out sdk.Coins
	for _, c := range coins {
		if c.Amount.Equal(sdk.NewInt(0)) {
			continue
		}

		for i, d := range dts.DenomTraces {
			if c.Denom == d.IBCDenom() {
				out = append(out, sdk.Coin{Denom: d.GetFullDenomPath(), Amount: c.Amount})
				break
			}

			if i == len(dts.DenomTraces)-1 {
				out = append(out, c)
			}
		}
	}
	return out, nil
}
