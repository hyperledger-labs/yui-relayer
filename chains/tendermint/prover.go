package tendermint

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/light"
	"github.com/cometbft/cometbft/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/client"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"go.opentelemetry.io/otel/codes"

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
	clientCtx := pr.chain.CLIContext(int64(ctx.Height().GetRevisionHeight())).WithCmdContext(ctx.Context())
	if v, proof, proofHeight, err := ibcclient.QueryTendermintProof(clientCtx, []byte(path)); err != nil {
		return nil, clienttypes.Height{}, err
	} else if !bytes.Equal(v, value) {
		return nil, clienttypes.Height{}, fmt.Errorf("value unmatch: %x != %x", v, value)
	} else {
		return proof, proofHeight, nil
	}
}

// ProveHostConsensusState returns the existence proof of the consensus state at `height`
// ibc-go doesn't use this proof, so it returns nil
func (pr *Prover) ProveHostConsensusState(ctx core.QueryContext, height ibcexported.Height, consensusState ibcexported.ConsensusState) ([]byte, error) {
	return nil, nil
}

/* LightClient implementation */

// CreateInitialLightClientState creates a pair of ClientState and ConsensusState submitted to the counterparty chain as MsgCreateClient
func (pr *Prover) CreateInitialLightClientState(ctx context.Context, height ibcexported.Height) (ibcexported.ClientState, ibcexported.ConsensusState, error) {
	var tmHeight int64
	if height != nil {
		tmHeight = int64(height.GetRevisionHeight())
	}
	selfHeader, err := pr.UpdateLightClient(ctx, tmHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update the local light client and get the header@%d: %v", tmHeight, err)
	}

	ubdPeriod, err := pr.chain.QueryUnbondingPeriod(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query for the unbonding period: %v", err)
	}

	cs := createClient(
		selfHeader,
		pr.getTrustingPeriod(),
		ubdPeriod,
	)

	cons := selfHeader.ConsensusState()

	return cs, cons, nil
}

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterpaty chain
func (pr *Prover) SetupHeadersForUpdate(ctx context.Context, counterparty core.FinalityAwareChain, latestFinalizedHeader core.Header) (<-chan *core.HeaderOrError, error) {
	self := pr.chain
	// make copy of header stored in mop
	tmp := latestFinalizedHeader.(*tmclient.Header)
	h := *tmp

	cph, err := counterparty.LatestHeight(ctx)
	if err != nil {
		return nil, err
	}

	// retrieve the client state from the counterparty chain
	counterpartyClientRes, err := counterparty.QueryClientState(core.NewQueryContext(ctx, cph))
	if err != nil {
		return nil, err
	}

	var cs ibcexported.ClientState
	if err := self.codec.UnpackAny(counterpartyClientRes.ClientState, &cs); err != nil {
		return nil, err
	}

	// inject TrustedHeight as latest height stored on counterparty client
	h.TrustedHeight = cs.GetLatestHeight().(clienttypes.Height)

	// query TrustedValidators at Trusted Height from the self chain
	valSet, err := self.QueryValsetAtHeight(ctx, h.TrustedHeight)
	if err != nil {
		return nil, err
	}

	// inject TrustedValidators into header
	h.TrustedValidators = valSet
	return core.MakeHeaderStream(&h), nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader(ctx context.Context) (core.Header, error) {
	return pr.UpdateLightClient(ctx, 0)
}

func (pr *Prover) CheckRefreshRequired(ctx context.Context, counterparty core.ChainInfoICS02Querier) (bool, error) {
	cpQueryHeight, err := counterparty.LatestHeight(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get the latest height of the counterparty chain: %v", err)
	}
	cpQueryCtx := core.NewQueryContext(ctx, cpQueryHeight)

	resCs, err := counterparty.QueryClientState(cpQueryCtx)
	if err != nil {
		return false, fmt.Errorf("failed to query the client state on the counterparty chain: %v", err)
	}

	var cs ibcexported.ClientState
	if err := pr.chain.codec.UnpackAny(resCs.ClientState, &cs); err != nil {
		return false, fmt.Errorf("failed to unpack Any into tendermint client state: %v", err)
	}

	resCons, err := counterparty.QueryClientConsensusState(cpQueryCtx, cs.GetLatestHeight())
	if err != nil {
		return false, fmt.Errorf("failed to query the consensus state on the counterparty chain: %v", err)
	}

	var cons ibcexported.ConsensusState
	if err := pr.chain.codec.UnpackAny(resCons.ConsensusState, &cons); err != nil {
		return false, fmt.Errorf("failed to unpack Any into tendermint consensus state: %v", err)
	}
	lcLastTimestamp := time.Unix(0, int64(cons.GetTimestamp()))

	selfQueryHeight, err := pr.chain.LatestHeight(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get the latest height of the self chain: %v", err)
	}

	selfTimestamp, err := pr.chain.Timestamp(ctx, selfQueryHeight)
	if err != nil {
		return false, fmt.Errorf("failed to get timestamp of the self chain: %v", err)
	}

	elapsedTime := selfTimestamp.Sub(lcLastTimestamp)

	durationMulByFraction := func(d time.Duration, f *Fraction) time.Duration {
		nsec := d.Nanoseconds() * int64(f.Numerator) / int64(f.Denominator)
		return time.Duration(nsec) * time.Nanosecond
	}
	needsRefresh := elapsedTime > durationMulByFraction(pr.config.GetTrustingPeriod(), pr.config.RefreshThresholdRate)

	return needsRefresh, nil
}

/* Local LightClient implementation */

// GetLatestLightHeight uses the CLI utilities to pull the latest height from a given chain
func (pr *Prover) GetLatestLightHeight(ctx context.Context) (int64, error) {
	db, df, err := pr.NewLightDB(ctx)
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

func (pr *Prover) UpdateLightClient(ctx context.Context, height int64) (*tmclient.Header, error) {
	ctx, span := tracer.Start(ctx, "Prover.UpdateLightClient", core.WithChainAttributes(pr.chain.ChainID()))
	defer span.End()

	// create database connection
	db, df, err := pr.NewLightDB(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, lightError(err)
	}
	defer df()

	client, err := pr.LightClient(db)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, lightError(err)
	}

	var sh *types.LightBlock
	if height == 0 {
		if sh, err = client.Update(ctx, time.Now()); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, lightError(err)
		} else if sh == nil {
			sh, err = client.TrustedLightBlock(0)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return nil, lightError(err)
			}
		}
	} else {
		if sh, err = client.VerifyLightBlockAtHeight(ctx, height, time.Now()); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, lightError(err)
		}
	}

	valSet := tmtypes.NewValidatorSet(sh.ValidatorSet.Validators)
	protoVal, err := valSet.ToProto()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
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
