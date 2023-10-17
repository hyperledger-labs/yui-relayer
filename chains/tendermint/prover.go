package tendermint

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/light"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/client"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type Prover struct {
	chain  *Chain
	config ProverConfig
}

var _ core.Prover = (*Prover)(nil)

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

// ProveState returns the proof of an IBC state specified by `path` and `value`
func (pr *Prover) ProveState(ctx core.QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	clientCtx := pr.chain.CLIContext(int64(ctx.Height().GetRevisionHeight()))
	if v, proof, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, []byte(path)); err != nil {
		return nil, clienttypes.Height{}, err
	} else if !bytes.Equal(v, value) {
		return nil, clienttypes.Height{}, fmt.Errorf("value unmatch: %x != %x", v, value)
	} else {
		return proof, proofHeight, nil
	}
}

/* LightClient implementation */

// CreateMsgCreateClient creates a CreateClientMsg to this chain
func (pr *Prover) CreateMsgCreateClient(clientID string, dstHeader core.Header, signer sdk.AccAddress) (*clienttypes.MsgCreateClient, error) {
	ubdPeriod, err := pr.chain.QueryUnbondingPeriod()
	if err != nil {
		return nil, err
	}
	return createClient(
		dstHeader.(*tmclient.Header),
		pr.getTrustingPeriod(),
		ubdPeriod,
		signer,
	), nil
}

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
func (pr *Prover) SetupHeadersForUpdate(dstChain core.ChainInfoICS02Querier, latestFinalizedHeader core.Header) ([]core.Header, error) {
	srcChain := pr.chain
	// make copy of header stored in mop
	tmp := latestFinalizedHeader.(*tmclient.Header)
	h := *tmp

	dsth, err := dstChain.LatestHeight()
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
	return []core.Header{&h}, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader() (latestFinalizedHeader core.Header, err error) {
	h, err := pr.UpdateLightClient()
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (pr *Prover) CheckRefreshRequired(counterparty core.Chain) (bool, error) {
	cpQueryHeight, err := counterparty.LatestHeight()
	if err != nil {
		return false, fmt.Errorf("failed to get the latest height of the counterparty chain: %v", err)
	}
	cpQueryCtx := core.NewQueryContext(context.TODO(), cpQueryHeight)

	resCs, err := counterparty.QueryClientState(cpQueryCtx)
	if err != nil {
		return false, fmt.Errorf("failed to query the client state on the counterparty chain: %v", err)
	}

	var cs tmclient.ClientState
	if err := pr.chain.codec.UnpackAny(resCs.ClientState, &cs); err != nil {
		return false, fmt.Errorf("failed to unpack Any into tendermint client state: %v", err)
	}

	resCons, err := counterparty.QueryClientConsensusState(cpQueryCtx, cs.LatestHeight)
	if err != nil {
		return false, fmt.Errorf("failed to query the consensus state on the counterparty chain: %v", err)
	}

	var cons tmclient.ConsensusState
	if err := pr.chain.codec.UnpackAny(resCons.ConsensusState, &cons); err != nil {
		return false, fmt.Errorf("failed to unpack Any into tendermint consensus state: %v", err)
	}

	selfQueryHeight, err := pr.chain.LatestHeight()
	if err != nil {
		return false, fmt.Errorf("failed to get the latest height of the self chain: %v", err)
	}

	selfTimestamp, err := pr.chain.Timestamp(selfQueryHeight)
	if err != nil {
		return false, fmt.Errorf("failed to get timestamp of the self chain: %v", err)
	}

	elapsedTime := selfTimestamp.Sub(cons.Timestamp)

	durationMulByFloat := func(d time.Duration, f float64) time.Duration {
		nsec := float64(d.Nanoseconds())
		nsec *= f
		return time.Duration(nsec) * time.Nanosecond
	}
	needsRefresh := elapsedTime > durationMulByFloat(cs.TrustingPeriod, pr.config.RefreshThresholdRate)

	return needsRefresh, nil
}

/* Local LightClient implementation */

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

func (pr *Prover) UpdateLightClient() (core.Header, error) {
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

func lightError(err error) error { return fmt.Errorf("light client: %w", err) }
