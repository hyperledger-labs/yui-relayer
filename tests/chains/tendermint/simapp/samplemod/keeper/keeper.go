package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	accounttypes "github.com/datachainlab/cross/x/core/account/types"
	contracttypes "github.com/datachainlab/cross/x/core/contract/types"
	storetypes "github.com/datachainlab/cross/x/core/store/types"
	txtypes "github.com/datachainlab/cross/x/core/tx/types"
	crosstypes "github.com/datachainlab/cross/x/core/types"
	xcctypes "github.com/datachainlab/cross/x/core/xcc/types"
	"github.com/hyperledger-labs/yui-relayer/tests/tendermint/simapp/samplemod/types"
)

type Keeper struct {
	m        codec.Marshaler
	storeKey sdk.StoreKey
	xstore   storetypes.KVStoreI

	exContractCaller contracttypes.ExternalContractCaller
}

func NewKeeper(m codec.Marshaler, storeKey sdk.StoreKey, xstore storetypes.KVStoreI) Keeper {
	return Keeper{
		m:                m,
		storeKey:         storeKey,
		xstore:           xstore,
		exContractCaller: contracttypes.NewExternalContractCaller(),
	}
}

// HandleContractCall is called by ContractModule
func (k Keeper) HandleContractCall(goCtx context.Context, callInfo txtypes.ContractCallInfo) (*txtypes.ContractCallResult, error) {
	var req types.ContractCallRequest
	if err := k.m.UnmarshalJSON(callInfo, &req); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	switch req.Method {
	case "nop":
		return &txtypes.ContractCallResult{}, nil
	case "counter":
		return k.HandleCounter(ctx, req)
	case "external-call":
		return k.HandleExternalCall(ctx, req)
	case "fail":
		return nil, errors.New("failed to process a contract request")
	default:
		panic(fmt.Sprintf("unknown method '%v'", req.Method))
	}
}

var counterKey = []byte("counter")

func (k Keeper) HandleCounter(ctx sdk.Context, req types.ContractCallRequest) (*txtypes.ContractCallResult, error) {
	// use the account ID as namespace
	account := contracttypes.ContractSignersFromContext(ctx.Context())[0]
	v := k.getCounter(ctx, account)
	bz := k.setCounter(ctx, account, v+1)
	return &txtypes.ContractCallResult{Data: bz}, nil
}

func (k Keeper) getCounter(ctx sdk.Context, account accounttypes.AccountID) uint64 {
	var count uint64
	v := k.xstore.Prefix(account).Get(ctx, counterKey)
	if v == nil {
		count = 0
	} else {
		count = sdk.BigEndianToUint64(v)
	}
	return count
}

func (k Keeper) setCounter(ctx sdk.Context, account accounttypes.AccountID, value uint64) []byte {
	bz := sdk.Uint64ToBigEndian(value)
	k.xstore.Prefix(account).Set(ctx, counterKey, bz)
	return bz
}

func (k Keeper) HandleExternalCall(ctx sdk.Context, req types.ContractCallRequest) (*txtypes.ContractCallResult, error) {
	if len(req.Args) != 2 {
		return nil, fmt.Errorf("the number of arguments must be 2")
	}

	accID, err := hex.DecodeString(req.Args[0])
	if err != nil {
		return nil, err
	}
	channelID := req.Args[1]

	r := types.NewContractCallRequest("counter")
	callInfo := txtypes.ContractCallInfo(k.m.MustMarshalJSON(&r))

	ret := k.exContractCaller.Call(
		ctx,
		&xcctypes.ChannelInfo{
			Port:    crosstypes.PortID,
			Channel: channelID,
		},
		callInfo,
		[]accounttypes.AccountID{accID},
	)
	return &txtypes.ContractCallResult{Data: ret}, nil
}
