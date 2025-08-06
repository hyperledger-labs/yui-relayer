package debug

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/core"
)

func debugFakeLost(ctx context.Context, chain *Chain, queryHeight ibcexported.Height) error {
	env := fmt.Sprintf("DEBUG_RELAYER_MISSING_TRIE_NODE_HEIGHT_%s", chain.ChainID())
	if val, ok := os.LookupEnv(env); ok {
		fmt.Printf(">%s: chain=%s: '%s'\n", env, chain.ChainID(), val)

		threshold, err := strconv.Atoi(val)
		if err != nil {
			fmt.Printf("malformed %s: %v\n", env, err)
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

func (c *Chain) QueryClientState(ctx core.QueryContext) (*clienttypes.QueryClientStateResponse, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryClientState(ctx)
}

func (c *Chain) QueryConnection(ctx core.QueryContext, connectionID string) (*conntypes.QueryConnectionResponse, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryConnection(ctx, connectionID)
}

func (c *Chain) QueryChannel(ctx core.QueryContext) (chanRes *chantypes.QueryChannelResponse, err error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryChannel(ctx)
}

func (c *Chain) QueryClientConsensusState(
	ctx core.QueryContext, dstClientConsHeight ibcexported.Height) (*clienttypes.QueryConsensusStateResponse, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryClientConsensusState(ctx, dstClientConsHeight)
}

func (c *Chain) QueryBalance(ctx core.QueryContext, addr sdk.AccAddress) (sdk.Coins, error) {
	return c.OriginChain.QueryBalance(ctx, addr)
}

func (c *Chain) QueryDenomTraces(ctx core.QueryContext, offset, limit uint64) (*transfertypes.QueryDenomTracesResponse, error) {
	return c.OriginChain.QueryDenomTraces(ctx, offset, limit)
}

func (c *Chain) QueryUnreceivedPackets(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryUnreceivedPackets(ctx, seqs)
}

func (c *Chain) QueryUnfinalizedRelayPackets(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	return c.OriginChain.QueryUnfinalizedRelayPackets(ctx, counterparty)
}

func (c *Chain) QueryUnreceivedAcknowledgements(ctx core.QueryContext, seqs []uint64) ([]uint64, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryUnreceivedAcknowledgements(ctx, seqs)
}

func (c *Chain) QueryUnfinalizedRelayAcknowledgements(ctx core.QueryContext, counterparty core.LightClientICS04Querier) (core.PacketInfoList, error) {
	return c.OriginChain.QueryUnfinalizedRelayAcknowledgements(ctx, counterparty)
}

func (c *Chain) QueryChannelUpgrade(ctx core.QueryContext) (*chantypes.QueryUpgradeResponse, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return nil, err
	}
	return c.OriginChain.QueryChannelUpgrade(ctx)
}

func (c *Chain) QueryChannelUpgradeError(ctx core.QueryContext) (*chantypes.QueryUpgradeErrorResponse, error) {
	return c.OriginChain.QueryChannelUpgradeError(ctx)
}

func (c *Chain) QueryCanTransitionToFlushComplete(ctx core.QueryContext) (bool, error) {
	if err := debugFakeLost(ctx.Context(), c, ctx.Height()); err != nil {
		return false, err
	}
	return c.OriginChain.QueryCanTransitionToFlushComplete(ctx)
}

