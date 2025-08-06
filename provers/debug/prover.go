package debug

import (
	"context"
	fmt "fmt"
	"os"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	//mocktypes "github.com/datachainlab/ibc-mock-client/modules/light-clients/xx-mock/types"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/hyperledger-labs/yui-relayer/log"
)

func debugFakeLost(ctx context.Context, chain core.Chain, queryHeight exported.Height) error {
	env := fmt.Sprintf("DEBUG_RELAYER_MISSING_TRIE_NODE_HEIGHT_PROVER_%s", chain.ChainID())
	if val, ok := os.LookupEnv(env); ok {
		fmt.Printf(">%s: chain=%s: '%v'\n", env, chain.ChainID(), val)

		threshold, err := strconv.Atoi(val)
		if err != nil {
			fmt.Printf("malformed %s: Expected format: <chainid> <height threshold>\n", env)
			return nil
		}

		qh := int64(queryHeight.GetRevisionHeight())

		latestHeight, err := chain.LatestHeight(ctx)
		if err != nil {
			return err
		}
		lh := int64(latestHeight.GetRevisionHeight())

		if qh+int64(threshold) < lh {
			return fmt.Errorf("fake missing trie node: %v + %v < %v", qh, threshold, lh)
		}
	}
	return nil
}

type Prover struct {
	chain        core.Chain
	OriginProver core.Prover
}

var _ core.Prover = (*Prover)(nil)

func NewProver(chain core.Chain, config ProverConfig, originProver core.Prover) *Prover {
	return &Prover{OriginProver: originProver, chain: chain}
}

func (pr *Prover) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	logger := log.GetLogger()
	logger.Info("debug prover is initialized.")
	return pr.OriginProver.Init(homePath, timeout, codec, debug)
}

// SetRelayInfo sets source's path and counterparty's info to the chain
func (pr *Prover) SetRelayInfo(path *core.PathEnd, counterparty *core.ProvableChain, counterpartyPath *core.PathEnd) error {
	return pr.OriginProver.SetRelayInfo(path, counterparty, counterpartyPath)
}

func (pr *Prover) SetupForRelay(ctx context.Context) error {
	return pr.OriginProver.SetupForRelay(ctx)
}

// GetChainID returns the chain ID
func (pr *Prover) GetChainID() string {
	return pr.chain.ChainID()
}

/* LightClient implementation */

// CreateInitialLightClientState creates a pair of ClientState and ConsensusState for building MsgCreateClient submitted to the counterparty chain
func (pr *Prover) CreateInitialLightClientState(ctx context.Context, height exported.Height) (exported.ClientState, exported.ConsensusState, error) {
	return pr.OriginProver.CreateInitialLightClientState(ctx, height)
}

// SetupHeadersForUpdate returns the finalized header and any intermediate headers needed to apply it to the client on the counterparty chain
func (pr *Prover) SetupHeadersForUpdate(ctx context.Context, counterparty core.FinalityAwareChain, latestFinalizedHeader core.Header) (<-chan *core.HeaderOrError, error) {
	headerStream, err := pr.OriginProver.SetupHeadersForUpdate(ctx, counterparty, latestFinalizedHeader)
	if err != nil {
		return headerStream, err
	}

	env := fmt.Sprintf("DEBUG_RELAYER_SHFU_WAIT_%s", pr.chain.ChainID())

	if val, ok := os.LookupEnv(env); ok {
		fmt.Printf(">%s: chain=%s, cp=%s: '%s'\n", env, pr.chain.ChainID(), counterparty.ChainID(), val)
		t, _ := strconv.Atoi(val)

		{
			var items []*core.HeaderOrError
			for i := range headerStream {
				items = append(items, i)
			}
			ch := make(chan *core.HeaderOrError, len(items))
			for _, i := range items {
				ch <- i
			}
			close(ch)
			headerStream = ch
		}

		n := t / 60
		for i := 0; i < n; i++ {
			fmt.Printf(" %s: chain=%s, cp=%s, %v/%v\n", env, pr.chain.ChainID(), counterparty.ChainID(), (i+1)*60, t)
			time.Sleep(time.Duration(60) * time.Second)
		}
		fmt.Printf(" %s: chain=%s, cp=%s, %v/%v\n", env, pr.chain.ChainID(), counterparty.ChainID(), t-n*60, t)
		time.Sleep(time.Duration(t-n*60) * time.Second)
		fmt.Printf("<%s: chain=%s, cp=%s, %v\n", env, pr.chain.ChainID(), counterparty.ChainID(), t)
	}
	return headerStream, nil
}

// GetLatestFinalizedHeader returns the latest finalized header
func (pr *Prover) GetLatestFinalizedHeader(ctx context.Context) (core.Header, error) {
	return pr.OriginProver.GetLatestFinalizedHeader(ctx)
}

// CheckRefreshRequired always returns false because mock clients don't need refresh.
func (pr *Prover) CheckRefreshRequired(ctx context.Context, dst core.ChainInfoICS02Querier) (bool, error) {
	return pr.OriginProver.CheckRefreshRequired(ctx, dst)
}

// ProveState returns the proof of an IBC state specified by `path` and `value`
func (pr *Prover) ProveState(ctx core.QueryContext, path string, value []byte) ([]byte, clienttypes.Height, error) {
	if err := debugFakeLost(ctx.Context(), pr.chain, ctx.Height()); err != nil {
		return nil, clienttypes.Height{}, err
	}
	//height := ctx.Height().(clienttypes.Height)
	return pr.OriginProver.ProveState(ctx, path, value)
}

// ProveHostConsensusState returns the proof of the consensus state at `height`
func (pr *Prover) ProveHostConsensusState(ctx core.QueryContext, height exported.Height, consensusState exported.ConsensusState) ([]byte, error) {
	return pr.OriginProver.ProveHostConsensusState(ctx, height, consensusState)
}
