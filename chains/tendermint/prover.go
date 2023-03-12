package tendermint

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v4/modules/light-clients/07-tendermint/types"
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

/* IBCProvableQuerierI implementation */

// QueryClientConsensusState returns the ClientConsensusState and its proof
func (pr *Prover) QueryClientConsensusStateWithProof(ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	return pr.chain.queryClientConsensusState(int64(ctx.Height().GetRevisionHeight()), dstClientConsHeight, true)
}

// QueryClientStateWithProof returns the ClientState and its proof
func (pr *Prover) QueryClientStateWithProof(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	return pr.chain.queryClientState(int64(ctx.Height().GetRevisionHeight()), true)
}

// QueryConnectionWithProof returns the Connection and its proof
func (pr *Prover) QueryConnectionWithProof(ctx core.QueryContext) (*conntypes.QueryConnectionResponse, error) {
	return pr.chain.queryConnection(int64(ctx.Height().GetRevisionHeight()), true)
}

// QueryChannelWithProof returns the Channel and its proof
func (pr *Prover) QueryChannelWithProof(ctx core.QueryContext) (chanRes *chantypes.QueryChannelResponse, err error) {
	return pr.chain.queryChannel(int64(ctx.Height().GetRevisionHeight()), true)
}

// QueryPacketCommitmentWithProof returns the packet commitment and its proof
func (pr *Prover) QueryPacketCommitmentWithProof(ctx core.QueryContext, seq uint64) (comRes *chantypes.QueryPacketCommitmentResponse, err error) {
	return pr.chain.queryPacketCommitment(int64(ctx.Height().GetRevisionHeight()), seq, true)
}

// QueryPacketAcknowledgementCommitmentWithProof returns the packet acknowledgement commitment and its proof
func (pr *Prover) QueryPacketAcknowledgementCommitmentWithProof(ctx core.QueryContext, seq uint64) (ackRes *chantypes.QueryPacketAcknowledgementResponse, err error) {
	return pr.chain.queryPacketAcknowledgementCommitment(int64(ctx.Height().GetRevisionHeight()), seq, true)
}

/* LightClientI implementation */

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
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

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
func (pr *Prover) SetupHeadersForUpdate(dstChain core.ChainI, latestFinalizedHeader core.HeaderI) ([]core.HeaderI, error) {
	srcChain := pr.chain
	// make copy of header stored in mop
	tmp := latestFinalizedHeader.(*tmclient.Header)
	h := *tmp

	dsth, err := dstChain.GetLatestHeight()
	if err != nil {
		return nil, err
	}

	// retrieve counterparty client from dst chain
	counterpartyClientRes, err := dstChain.QueryClientState(core.NewQueryContext(context.TODO(), dsth))
	if err != nil {
		return nil, err
	}

	var cs ibcexported.ClientState
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
	return []core.HeaderI{&h}, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader() (latestFinalizedHeader core.HeaderI, err error) {
	h, err := pr.UpdateLightClient()
	if err != nil {
		return nil, err
	}
	return h, nil
}

/* Local LightClient implementation */

// GetLatestHeight uses the CLI utilities to pull the latest height from a given chain
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

func (pr *Prover) UpdateLightClient() (core.HeaderI, error) {
	// create database connection
	db, df, err := pr.NewLightDB()
	if err != nil {
		return nil, lightError(err)
	}
	defer df()

	client, err := pr.LightClient(db)
	if err != nil {
		return nil, lightError(err)
	}

	sh, err := client.Update(context.Background(), time.Now())
	if err != nil {
		return nil, lightError(err)
	}

	if sh == nil {
		sh, err = client.TrustedLightBlock(0)
		if err != nil {
			return nil, lightError(err)
		}
	}

	valSet := tmtypes.NewValidatorSet(sh.ValidatorSet.Validators)
	protoVal, err := valSet.ToProto()
	if err != nil {
		return nil, err
	}
	protoVal.TotalVotingPower = valSet.TotalVotingPower()

	return &tmclient.Header{
		SignedHeader: sh.SignedHeader.ToProto(),
		ValidatorSet: protoVal,
	}, nil
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

	valSet := tmtypes.NewValidatorSet(val.Validators)
	protoVal, err := valSet.ToProto()
	if err != nil {
		return nil, err
	}
	protoVal.TotalVotingPower = valSet.TotalVotingPower()

	return &tmclient.Header{
		// NOTE: This is not a SignedHeader
		// We are missing a light.Commit type here
		SignedHeader: res.SignedHeader.ToProto(),
		ValidatorSet: protoVal,
	}, nil
}

func lightError(err error) error { return fmt.Errorf("light client: %w", err) }
