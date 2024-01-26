package simapp

import (
	cosmossdkerrors "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectionkeeper "github.com/cosmos/ibc-go/v8/modules/core/03-connection/keeper"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
)

func overrideIBCClientKeeper(k ibckeeper.Keeper, cdc codec.BinaryCodec, key storetypes.StoreKey, paramSpace paramtypes.Subspace, stakingKeeper *stakingkeeper.Keeper) *ibckeeper.Keeper {
	clientKeeper := NewClientKeeper(k.ClientKeeper, stakingKeeper)
	k.ConnectionKeeper = connectionkeeper.NewKeeper(cdc, key, paramSpace, clientKeeper)
	return &k
}

var _ connectiontypes.ClientKeeper = (*ClientKeeper)(nil)
var _ channeltypes.ClientKeeper = (*ClientKeeper)(nil)

// ClientKeeper override `ValidateSelfClient` and `GetSelfConsensusState` in the keeper of ibc-client
// original method doesn't yet support a consensus state for general client
type ClientKeeper struct {
	clientkeeper.Keeper
	stakingKeeper *stakingkeeper.Keeper
}

func NewClientKeeper(k clientkeeper.Keeper, stakingKeeper *stakingkeeper.Keeper) ClientKeeper {
	return ClientKeeper{Keeper: k, stakingKeeper: stakingKeeper}
}

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	return nil
}

func (k ClientKeeper) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error) {
	selfHeight, ok := height.(clienttypes.Height)
	if !ok {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrInvalidType, "expected %T, got %T", clienttypes.Height{}, height)
	}
	// check that height revision matches chainID revision
	revision := clienttypes.ParseChainID(ctx.ChainID())
	if revision != height.GetRevisionNumber() {
		return nil, cosmossdkerrors.Wrapf(clienttypes.ErrInvalidHeight, "chainID revision number does not match height revision number: expected %d, got %d", revision, height.GetRevisionNumber())
	}
	histInfo, err := k.stakingKeeper.GetHistoricalInfo(ctx, int64(selfHeight.RevisionHeight))
	if err != nil {
		return nil, cosmossdkerrors.Wrapf(sdkerrors.ErrNotFound, "no historical info found at height %d", selfHeight.RevisionHeight)
	}
	consensusState := &mocktypes.ConsensusState{
		Timestamp: uint64(histInfo.Header.Time.UnixNano()),
	}
	return consensusState, nil
}
