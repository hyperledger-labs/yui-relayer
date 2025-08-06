package debug

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/hyperledger-labs/yui-relayer/core"
)

type Chain struct {
	config ChainConfig
	OriginChain core.Chain
}

var _ core.Chain = (*Chain)(nil)

func (c *Chain) ChainID() string {
	return c.OriginChain.ChainID()
}

func (c *Chain) Config() ChainConfig {
	return c.config
}

func (c *Chain) Codec() codec.ProtoCodecMarshaler {
	return c.OriginChain.Codec()
}

func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	return c.OriginChain.GetAddress()
}

// SetRelayInfo sets source's path and counterparty's info to the chain
func (c *Chain) SetRelayInfo(path *core.PathEnd, counterparty *core.ProvableChain, counterpartyPath *core.PathEnd) error {
	return c.OriginChain.SetRelayInfo(path, counterparty, counterpartyPath)
}

func (c *Chain) Path() *core.PathEnd {
	return c.OriginChain.Path()
}

func (c *Chain) Init(homePath string, timeout time.Duration, codec codec.ProtoCodecMarshaler, debug bool) error {
	return c.OriginChain.Init(homePath, timeout, codec, debug)
}

func (c *Chain) SetupForRelay(ctx context.Context) error {
	return c.OriginChain.SetupForRelay(ctx)
}

// LatestHeight queries the chain for the latest height and returns it
func (c *Chain) LatestHeight(ctx context.Context) (ibcexported.Height, error) {
	return c.OriginChain.LatestHeight(ctx)
}

func (c *Chain) Timestamp(ctx context.Context, height ibcexported.Height) (time.Time, error) {
	return c.OriginChain.Timestamp(ctx, height)
}

func (c *Chain) AverageBlockTime() time.Duration {
	return c.OriginChain.AverageBlockTime()
}

func (c *Chain) RegisterMsgEventListener(listener core.MsgEventListener) {
	c.OriginChain.RegisterMsgEventListener(listener)
}

func (c *Chain) SendMsgs(ctx context.Context, msgs []sdk.Msg) ([]core.MsgID, error) {
	return c.OriginChain.SendMsgs(ctx, msgs)
}

func (c *Chain) GetMsgResult(ctx context.Context, id core.MsgID) (core.MsgResult, error) {
	return c.OriginChain.GetMsgResult(ctx, id)
}

