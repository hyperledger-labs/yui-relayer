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

// SetupHeadersForUpdate returns a header slice that contains intermediate headers needed to submit the `latestFinalizedHeader`
func (pr *Prover) SetupHeadersForUpdate(dstChain core.ChainI, latestFinalizedHeader core.HeaderI) ([]core.HeaderI, error) {
	return []core.HeaderI{latestFinalizedHeader.(*mocktypes.Header)}, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader() (latestFinalizedHeader core.HeaderI, provableHeight int64, queryableHeight int64, err error) {
	var h = mocktypes.Header{
		Height: &clienttypes.Height{
			RevisionNumber: 0,
			RevisionHeight: pr.sequence,
		},
		Timestamp: uint64(time.Now().UnixNano()),
	}
	chainHeight, err := pr.chain.GetLatestHeight()
	if err != nil {
		return nil, -1, -1, err
	}
	return &h, chainHeight, chainHeight, nil
}

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	res, err := pr.chain.QueryClientConsensusState(height, dstClientConsHeight)
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
func (pr *Prover) QueryClientStateWithProof(height int64) (*clienttypes.QueryClientStateResponse, error) {
	res, err := pr.chain.QueryClientState(height)
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
func (pr *Prover) QueryConnectionWithProof(height int64) (*conntypes.QueryConnectionResponse, error) {
	res, err := pr.chain.QueryConnection(height)
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
func (pr *Prover) QueryChannelWithProof(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	res, err := pr.chain.QueryChannel(height)
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
func (pr *Prover) QueryPacketCommitmentWithProof(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	res, err := pr.chain.QueryPacketCommitment(height, seq)
	if err != nil {
		return nil, err
	}
	res.Proof = res.Commitment
	res.ProofHeight = clienttypes.NewHeight(0, pr.sequence)
	return res, nil
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	res, err := pr.chain.QueryPacketAcknowledgementCommitment(height, seq)
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
