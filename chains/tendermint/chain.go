package tendermint

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/relayer/relayer"

	"github.com/datachainlab/relayer/core"
)

// Chain represents the necessary data for connecting to and indentifying a chain and its counterparites
type Chain struct {
	relayer.Chain
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) ClientType() string {
	return "tendermint"
}

func (c *Chain) QueryLatestHeader() (core.HeaderI, error) {
	return c.Chain.QueryLatestHeader()
}

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	res, err := c.Chain.SendMsgs(msgs)
	if err != nil {
		return nil, err
	}
	return []byte(res.Logs.String()), nil
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	res, err := c.Chain.SendMsgs(msgs)
	if err != nil || res.Code != 0 {
		c.LogFailedTx(res, err, msgs)
		return false
	}
	// NOTE: Add more data to this such as identifiers
	c.LogSuccessTx(res, msgs)

	return true
}

func (c *Chain) CreateClients(dst core.ChainI) (err error) {
	panic("not implemented error")
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}
