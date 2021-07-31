package corda

import (
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	cordatypes "github.com/hyperledger-labs/yui-corda-ibc/go/x/ibc/light-clients/xx-corda/types"
	"github.com/hyperledger-labs/yui-relayer/core"
)

type Prover struct {
	chain *Chain
}

var _ core.ProverI = (*Prover)(nil)

func NewProver(chain *Chain) *Prover {
	return &Prover{chain: chain}
}

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
}

// QueryLatestHeader returns the latest header from the chain
func (pr *Prover) QueryLatestHeader() (out core.HeaderI, err error) {
	return &cordatypes.Header{}, nil
}

// GetLatestLightHeight returns the latest height on the light client
func (pr *Prover) GetLatestLightHeight() (int64, error) {
	return -1, nil
}

// SetupHeader creates a new header based on a given header
func (pr *Prover) SetupHeader(dst core.LightClientIBCQueryierI, baseSrcHeader core.HeaderI) (core.HeaderI, error) {
	return nil, nil
}

// UpdateLightWithHeader updates a header on the light client and returns the header and the corresponding height
func (pr *Prover) UpdateLightWithHeader() (header core.HeaderI, provableHeight int64, queryableHeight int64, err error) {
	h, err := pr.QueryLatestHeader()
	if err != nil {
		return nil, -1, -1, err
	}
	return h, -1, -1, nil
}

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return pr.chain.QueryClientConsensusState(height, dstClientConsHeight)
}

// QueryClientStateWithProof returns the ClientState and its proof
func (pr *Prover) QueryClientStateWithProof(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return pr.chain.QueryClientState(height)
}

// QueryConnectionWithProof returns the Connection and its proof
func (pr *Prover) QueryConnectionWithProof(height int64) (*conntypes.QueryConnectionResponse, error) {
	return pr.chain.QueryConnection(height)
}

// QueryChannelWithProof returns the Channel and its proof
func (pr *Prover) QueryChannelWithProof(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	return pr.chain.QueryChannel(height)
}

// QueryPacketCommitmentWithProof returns the packet commitment and its proof
func (pr *Prover) QueryPacketCommitmentWithProof(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return pr.chain.QueryPacketCommitment(height, seq)
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return pr.chain.QueryPacketAcknowledgementCommitment(height, seq)
}
