package tendermint

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/tendermint/tendermint/light"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type Prover struct {
	chain  *Chain
	config ProverConfig
}

var _ core.ProverI = (*Prover)(nil)

func NewProver(chain *Chain, config ProverConfig) *Prover {
	return &Prover{chain: chain, config: config}
}

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
}

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(height int64, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return pr.chain.queryClientConsensusState(height, dstClientConsHeight, true)
}

// QueryClientStateWithProof returns the ClientState and its proof
func (pr *Prover) QueryClientStateWithProof(height int64) (*clienttypes.QueryClientStateResponse, error) {
	return pr.chain.queryClientState(height, true)
}

// QueryConnectionWithProof returns the Connection and its proof
func (pr *Prover) QueryConnectionWithProof(height int64) (*conntypes.QueryConnectionResponse, error) {
	return pr.chain.queryConnection(height, true)
}

// QueryChannelWithProof returns the Channel and its proof
func (pr *Prover) QueryChannelWithProof(height int64) (chanRes *chantypes.QueryChannelResponse, err error) {
	return pr.chain.queryChannel(height, true)
}

// QueryPacketCommitmentWithProof returns the packet commitment and its proof
func (pr *Prover) QueryPacketCommitmentWithProof(height int64, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return pr.chain.queryPacketCommitment(height, seq, true)
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(height int64, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return pr.chain.queryPacketAcknowledgementCommitment(height, seq, true)
}

// QueryLatestHeader returns the latest header from the chain
func (pr *Prover) QueryLatestHeader() (out core.HeaderI, err error) {
	var h int64
	if h, err = pr.chain.GetLatestHeight(); err != nil {
		return nil, err
	}
	return pr.queryHeaderAtHeight(h)
}

// GetLatestLightHeight uses the CLI utilities to pull the latest height from a given chain
func (pr *Prover) GetLatestLightHeight() (int64, error) {
	db, df, err := pr.NewLightDB()
	if err != nil {
		return -1, err
	}
	defer df()

	client, err := pr.LightClient(db)
	if err != nil {
		return -1, err
	}

	return client.LastTrustedHeight()
}

// CreateMsgCreateClient creates a CreateClientMsg to this chain
func (pr *Prover) CreateMsgCreateClient(clientID string, dstHeader core.HeaderI, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error) {
	ubdPeriod, err := pr.chain.QueryUnbondingPeriod()
	if err != nil {
		return nil, err
	}
	consensusParams, err := pr.chain.QueryConsensusParams()
	if err != nil {
		return nil, err
	}
	return createClient(
		dstHeader.(*tmclient.Header),
		pr.getTrustingPeriod(),
		ubdPeriod,
		consensusParams,
		signer,
	), nil
}

// SetupHeader creates a new header based on a given header
func (pr *Prover) SetupHeader(dstChain core.LightClientIBCQueryierI, srcHeader core.HeaderI) (core.HeaderI, error) {
	srcChain := pr.chain
	// make copy of header stored in mop
	tmp := srcHeader.(*tmclient.Header)
	h := *tmp

	dsth, err := dstChain.GetLatestLightHeight()
	if err != nil {
		return nil, err
	}

	// retrieve counterparty client from dst chain
	counterpartyClientRes, err := dstChain.QueryClientState(dsth)
	if err != nil {
		return nil, err
	}

	var cs exported.ClientState
	if err := srcChain.codec.UnpackAny(counterpartyClientRes.ClientState, &cs); err != nil {
		return nil, err
	}

	// inject TrustedHeight as latest height stored on counterparty client
	h.TrustedHeight = cs.GetLatestHeight().(clienttypes.Height)

	// query TrustedValidators at Trusted Height from srcChain
	valSet, err := srcChain.QueryValsetAtHeight(h.TrustedHeight)
	if err != nil {
		return nil, err
	}

	// inject TrustedValidators into header
	h.TrustedValidators = valSet
	return &h, nil
}

func lightError(err error) error { return fmt.Errorf("light client: %w", err) }

// UpdateLightWithHeader calls client.Update and then .
func (pr *Prover) UpdateLightWithHeader() (header core.HeaderI, provableHeight int64, queryableHeight int64, err error) {
	// create database connection
	db, df, err := pr.NewLightDB()
	if err != nil {
		return nil, 0, 0, lightError(err)
	}
	defer df()

	client, err := pr.LightClient(db)
	if err != nil {
		return nil, 0, 0, lightError(err)
	}

	sh, err := client.Update(context.Background(), time.Now())
	if err != nil {
		return nil, 0, 0, lightError(err)
	}

	if sh == nil {
		sh, err = client.TrustedLightBlock(0)
		if err != nil {
			return nil, 0, 0, lightError(err)
		}
	}

	protoVal, err := tmtypes.NewValidatorSet(sh.ValidatorSet.Validators).ToProto()
	if err != nil {
		return nil, 0, 0, err
	}

	h := &tmclient.Header{
		SignedHeader: sh.SignedHeader.ToProto(),
		ValidatorSet: protoVal,
	}
	queryableHeight = int64(h.GetHeight().GetRevisionHeight())
	// NOTE: We query connection at height - 1 because of the way tendermint returns
	// proofs the commit for height n is contained in the header of height n + 1
	provableHeight = queryableHeight - 1
	return h, provableHeight, queryableHeight, nil
}

// TrustOptions returns light.TrustOptions given a height and hash
func (pr *Prover) TrustOptions(height int64, hash []byte) light.TrustOptions {
	return light.TrustOptions{
		Period: pr.getTrustingPeriod(),
		Height: height,
		Hash:   hash,
	}
}

/// internal method ///

// getTrustingPeriod returns the trusting period for the chain
func (pr *Prover) getTrustingPeriod() time.Duration {
	tp, _ := time.ParseDuration(pr.config.TrustingPeriod)
	return tp
}

// queryHeaderAtHeight returns the header at a given height
func (c *Prover) queryHeaderAtHeight(height int64) (*tmclient.Header, error) {
	var (
		page    int = 1
		perPage int = 100000
	)
	if height <= 0 {
		return nil, fmt.Errorf("must pass in valid height, %d not valid", height)
	}

	res, err := c.chain.Client.Commit(context.Background(), &height)
	if err != nil {
		return nil, err
	}

	val, err := c.chain.Client.Validators(context.Background(), &height, &page, &perPage)
	if err != nil {
		return nil, err
	}

	protoVal, err := tmtypes.NewValidatorSet(val.Validators).ToProto()
	if err != nil {
		return nil, err
	}

	return &tmclient.Header{
		// NOTE: This is not a SignedHeader
		// We are missing a light.Commit type here
		SignedHeader: res.SignedHeader.ToProto(),
		ValidatorSet: protoVal,
	}, nil
}
