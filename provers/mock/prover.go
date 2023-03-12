package mock

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"

	mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Prover struct {
	chain core.ChainI

	sequence uint64
}

var _ core.ProverI = (*Prover)(nil)

func NewProver(chain core.ChainI, sequence uint64) *Prover {
	return &Prover{chain: chain, sequence: sequence}
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

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
}

// CreateMsgCreateClient creates a CreateClientMsg to this chain
func (pr *Prover) CreateMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error) {
	h := dstHeader.(*mocktypes.Header)
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
func (pr *Prover) SetupHeadersForUpdate(dstChain core.ChainI, latestFinalizedHeader core.HeaderI) ([]core.HeaderI, error) {
	return []core.HeaderI{latestFinalizedHeader.(*mocktypes.Header)}, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader() (latestFinalizedHeader core.HeaderI, err error) {
	return &mocktypes.Header{
		Height: &clienttypes.Height{
			RevisionNumber: 0,
			RevisionHeight: pr.sequence,
		},
		Timestamp: uint64(time.Now().UnixNano()),
	}, nil
}

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	res, err := pr.chain.QueryClientConsensusState(ctx, dstClientConsHeight)
	if err != nil {
		return nil, err
	}
	bz, err := pr.chain.Codec().Marshal(res.ConsensusState)
	if err != nil {
		return nil, err
	}
	res.Proof = makeProof(bz)
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryClientStateWithProof returns the ClientState and its proof
func (pr *Prover) QueryClientStateWithProof(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	res, err := pr.chain.QueryClientState(ctx)
	if err != nil {
		return nil, err
	}
	bz, err := pr.chain.Codec().Marshal(res.ClientState)
	if err != nil {
		return nil, err
	}
	res.Proof = makeProof(bz)
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryConnectionWithProof returns the Connection and its proof
func (pr *Prover) QueryConnectionWithProof(ctx core.QueryContext) (*conntypes.QueryConnectionResponse, error) {
	res, err := pr.chain.QueryConnection(ctx)
	if err != nil {
		return nil, err
	}
	bz, err := pr.chain.Codec().Marshal(res.Connection)
	if err != nil {
		return nil, err
	}
	res.Proof = makeProof(bz)
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryChannelWithProof returns the Channel and its proof
func (pr *Prover) QueryChannelWithProof(ctx core.QueryContext) (chanRes *chantypes.QueryChannelResponse, err error) {
	res, err := pr.chain.QueryChannel(ctx)
	if err != nil {
		return nil, err
	}
	bz, err := pr.chain.Codec().Marshal(res.Channel)
	if err != nil {
		return nil, err
	}
	res.Proof = makeProof(bz)
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryPacketCommitmentWithProof returns the packet commitment and its proof
func (pr *Prover) QueryPacketCommitmentWithProof(ctx core.QueryContext, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	res, err := pr.chain.QueryPacketCommitment(ctx, seq)
	if err != nil {
		return nil, err
	}
	res.Proof = res.Commitment
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(ctx core.QueryContext, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	res, err := pr.chain.QueryPacketAcknowledgementCommitment(ctx, seq)
	if err != nil {
		return nil, err
	}
	res.Proof = res.Acknowledgement
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

func makeProof(bz []byte) []byte {
	h := sha256.Sum256(bz)
	return h[:]
}
