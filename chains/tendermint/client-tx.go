package tendermint

import (
	"time"

	"github.com/cometbft/cometbft/light"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	tmclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

func createClient(
	dstHeader *tmclient.Header,
	trustingPeriod, unbondingPeriod time.Duration,
) *tmclient.ClientState {
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
	)

	return clientState
}
