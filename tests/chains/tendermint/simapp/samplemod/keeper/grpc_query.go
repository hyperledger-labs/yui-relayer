package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/tests/tendermint/simapp/samplemod/types"
)

var _ types.QueryServer = (*Keeper)(nil)

func (q Keeper) Counter(c context.Context, req *types.QueryCounterRequest) (*types.QueryCounterResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	v := q.getCounter(ctx, req.Account)
	return &types.QueryCounterResponse{Value: v}, nil
}
