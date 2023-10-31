package mock

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"

	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Prover struct {
	chain  core.Chain
	config ProverConfig
}

var _ core.Prover = (*Prover)(nil)

func NewProver(chain core.Chain, config ProverConfig) *Prover {
	return &Prover{chain: chain, config: config}
}

func (pr *Prover) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	return nil
}

// SetRelayInfo sets source's path and counterparty's info to the chain
func (pr *Prover) SetRelayInfo(_ *core.PathEnd, _ *core.ProvableChain, _ *core.PathEnd) error {
	return nil // prover uses chain's path instead
}

func (pr *Prover) SetupForRelay(ctx context.Context) error {
	return nil
}

// CreateMsgCreateClient creates a MsgCreateClient for the counterparty chain
func (pr *Prover) CreateMsgCreateClient(clientID string, selfHeader core.Header, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error) {
	h := selfHeader.(*mocktypes.Header)
	clientState := &mocktypes.ClientState{
		LatestHeight: h.Height,
	}
	consensusState := &mocktypes.ConsensusState{
		Timestamp: h.Timestamp,
	}
	return clienttypes.NewMsgCreateClient(
		clientState,
		consensusState,
		signer.String(),
	)
}

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
func (pr *Prover) SetupHeadersForUpdate(_ core.FinalityAwareChain, latestFinalizedHeader core.Header) ([]core.Header, error) {
	return []core.Header{latestFinalizedHeader.(*mocktypes.Header)}, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader() (latestFinalizedHeader core.Header, err error) {
	chainLatestHeight, err := pr.chain.LatestHeight()
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < pr.config.FinalityDelay; i++ {
		if prevHeight, success := chainLatestHeight.Decrement(); success {
			chainLatestHeight = prevHeight
		} else {
			break
		}
	}
	timestamp, err := pr.chain.Timestamp(chainLatestHeight)
	if err != nil {
		return nil, err
	}
	return &mocktypes.Header{
		Height: clienttypes.Height{
			RevisionNumber: chainLatestHeight.GetRevisionNumber(),
			RevisionHeight: chainLatestHeight.GetRevisionHeight(),
		},
		Timestamp: uint64(timestamp.UnixNano()),
	}, nil
}

// CheckRefreshRequired always returns false because mock clients don't need refresh.
func (pr *Prover) CheckRefreshRequired(dst core.ChainInfoICS02Querier) (bool, error) {
	return false, nil
}

// ProveState returns the proof of an IBC state specified by `path` and `value`
func (pr *Prover) ProveState(ctx core.QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	return makeProof(value), ctx.Height().(clienttypes.Height), nil
}

func makeProof(bz []byte) []byte {
	h := sha256.Sum256(bz)
	return h[:]
}
