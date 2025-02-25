package mock

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	fmt "fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
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
	logger := log.GetLogger()
	logger.Info("mock prover is initialized.")
	return nil
}

// SetRelayInfo sets source's path and counterparty's info to the chain
func (pr *Prover) SetRelayInfo(_ *core.PathEnd, _ *core.ProvableChain, _ *core.PathEnd) error {
	return nil // prover uses chain's path instead
}

func (pr *Prover) SetupForRelay(ctx context.Context) error {
	return nil
}

// CreateInitialLightClientState creates a pair of ClientState and ConsensusState for building MsgCreateClient submitted to the counterparty chain
func (pr *Prover) CreateInitialLightClientState(ctx context.Context, height exported.Height) (exported.ClientState, exported.ConsensusState, error) {
	if head, err := pr.GetLatestFinalizedHeader(context.TODO()); err != nil {
		return nil, nil, fmt.Errorf("failed to get the latest finalized header: %v", err)
	} else if height == nil {
		height = head.GetHeight()
	} else if height.GT(head.GetHeight()) {
		return nil, nil, fmt.Errorf("the given height is greater than the latest finalized height: %v > %v", height, head)
	}

	clientState := &mocktypes.ClientState{
		LatestHeight: clienttypes.NewHeight(
			height.GetRevisionNumber(),
			height.GetRevisionHeight(),
		),
	}

	var consensusState exported.ConsensusState
	if timestamp, err := pr.chain.Timestamp(context.TODO(), height); err != nil {
		return nil, nil, fmt.Errorf("get timestamp at height@%v: %v", height, err)
	} else {
		consensusState = &mocktypes.ConsensusState{
			Timestamp: uint64(timestamp.UnixNano()),
		}
	}

	return clientState, consensusState, nil
}

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterparty chain
func (pr *Prover) SetupHeadersForUpdate(ctx context.Context, counterparty core.FinalityAwareChain, latestFinalizedHeader core.Header) ([]core.Header, error) {
	return []core.Header{latestFinalizedHeader.(*mocktypes.Header)}, nil
}

func (pr *Prover) createMockHeader(ctx context.Context, height exported.Height) (core.Header, error) {
	timestamp, err := pr.chain.Timestamp(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("failed to get block timestamp at height:%v", height)
	}
	return &mocktypes.Header{
		Height: clienttypes.Height{
			RevisionNumber: height.GetRevisionNumber(),
			RevisionHeight: height.GetRevisionHeight(),
		},
		Timestamp: uint64(timestamp.UnixNano()),
	}, nil
}

func (pr *Prover) getDelayedLatestFinalizedHeight(ctx context.Context) (exported.Height, error) {
	height, err := pr.chain.LatestHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest height: %v", err)
	}
	for i := uint64(0); i < pr.config.FinalityDelay; i++ {
		if h, ok := height.Decrement(); ok {
			height = h
		} else {
			break
		}
	}
	return height, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader(ctx context.Context) (core.Header, error) {
	if latestFinalizedHeight, err := pr.getDelayedLatestFinalizedHeight(ctx); err != nil {
		return nil, err
	} else {
		return pr.createMockHeader(ctx, latestFinalizedHeight)
	}
}

// CheckRefreshRequired always returns false because mock clients don't need refresh.
func (pr *Prover) CheckRefreshRequired(ctx context.Context, dst core.ChainInfoICS02Querier) (bool, error) {
	return false, nil
}

// ProveState returns the proof of an IBC state specified by `path` and `value`
func (pr *Prover) ProveState(ctx core.QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	height := ctx.Height().(clienttypes.Height)
	return makeProof(height, path, value), height, nil
}

// ProveHostConsensusState returns the proof of the consensus state at `height`
func (pr *Prover) ProveHostConsensusState(ctx core.QueryContext, height exported.Height, consensusState exported.ConsensusState) ([]byte, error) {
	return clienttypes.MarshalConsensusState(pr.chain.Codec(), consensusState)
}

func makeProof(height exported.Height, path string, bz []byte) []byte {
	revisionNumber := height.GetRevisionNumber()
	revisionHeight := height.GetRevisionHeight()

	heightBuf := make([]byte, 16)
	binary.BigEndian.PutUint64(heightBuf[:8], revisionNumber)
	binary.BigEndian.PutUint64(heightBuf[8:], revisionHeight)

	hashPrefix := sha256.Sum256(core.DefaultChainPrefix.Bytes())
	hashPath := sha256.Sum256([]byte(path))
	hashValue := sha256.Sum256([]byte(bz))

	var combined []byte
	combined = append(combined, heightBuf...)
	combined = append(combined, hashPrefix[:]...)
	combined = append(combined, hashPath[:]...)
	combined = append(combined, hashValue[:]...)

	h := sha256.Sum256(combined)
	return h[:]
}
