package helpers

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

// QueryBalance is a helper function for query balance
func QueryBalance(chain *core.ProvableChain, address sdk.AccAddress, showDenoms bool) (sdk.Coins, error) {
	coins, err := chain.QueryBalance(address)
	if err != nil {
		return nil, err
	}

	if showDenoms {
		return coins, nil
	}

	h, err := chain.GetLatestHeight()
	if err != nil {
		return nil, err
	}

	dts, err := chain.QueryDenomTraces(0, 1000, h)
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
