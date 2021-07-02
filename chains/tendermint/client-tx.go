package tendermint

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	tmclient "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/light"
)

// MakeMsgCreateClient creates a Msg to this chain
func (dst *Chain) MakeMsgCreateClient(_ string, dstHeader core.HeaderI, signer sdk.AccAddress) (sdk.Msg, error) {
	ubdPeriod, err := dst.QueryUnbondingPeriod()
	if err != nil {
		return nil, err
	}
	consensusParams, err := dst.QueryConsensusParams()
	if err != nil {
		return nil, err
	}
	return createClient(
		dstHeader.(*tmclient.Header),
		dst.GetTrustingPeriod(),
		ubdPeriod,
		consensusParams,
		signer,
	), nil
}

func createClient(
	dstHeader *tmclient.Header,
	trustingPeriod, unbondingPeriod time.Duration,
	consensusParams *abci.ConsensusParams,
	signer sdk.AccAddress) sdk.Msg {
	if err := dstHeader.ValidateBasic(); err != nil {
		panic(err)
	}

	// Blank Client State
	clientState := tmclient.NewClientState(
		dstHeader.GetHeader().GetChainID(),
		tmclient.NewFractionFromTm(light.DefaultTrustLevel),
		trustingPeriod,
		unbondingPeriod,
		time.Minute*10,
		dstHeader.GetHeight().(clienttypes.Height),
		commitmenttypes.GetSDKSpecs(),
		[]string{"upgrade", "upgradedIBCState"},
		false,
		false,
	)

	msg, err := clienttypes.NewMsgCreateClient(
		clientState,
		dstHeader.ConsensusState(),
		signer.String(),
	)

	if err != nil {
		panic(err)
	}
	if err = msg.ValidateBasic(); err != nil {
		panic(err)
	}
	return msg
}
