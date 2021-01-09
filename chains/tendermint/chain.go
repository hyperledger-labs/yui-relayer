package tendermint

import (
	"log"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/relayer/relayer"

	"github.com/datachainlab/relayer/core"
)

// Chain represents the necessary data for connecting to and indentifying a chain and its counterparites
type Chain struct {
	base relayer.Chain
}

var _ core.ChainI = (*Chain)(nil)

func (c *Chain) ClientType() string {
	return "tendermint"
}

func (c *Chain) ChainID() string {
	return c.base.ChainID
}

func (c *Chain) ClientID() string {
	return c.base.PathEnd.ClientID
}

// GetAddress returns the sdk.AccAddress associated with the configred key
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	return c.base.GetAddress()
}

func (c *Chain) SetPath(p *core.PathEnd) error {
	return c.base.SetPath(p)
}

func (c *Chain) Update(key, value string) (core.ChainConfigI, error) {
	out, err := c.base.Update(key, value)
	if err != nil {
		return nil, err
	}
	return &ChainConfig{
		Key:            out.Key,
		ChainId:        out.ChainID,
		RpcAddr:        out.RPCAddr,
		AccountPrefix:  out.AccountPrefix,
		GasAdjustment:  float32(out.GasAdjustment),
		GasPrices:      out.GasPrices,
		TrustingPeriod: out.TrustingPeriod,
	}, nil
}

func (c *Chain) Init(homePath string, timeout time.Duration, debug bool) error {
	err := c.base.Init(homePath, timeout, debug)
	if err != nil {
		return err
	}
	c.base.Encoding = core.MakeEncodingConfig()
	return nil
}

func (c *Chain) QueryLatestHeader() (core.HeaderI, error) {
	return c.base.QueryLatestHeader()
}

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	log.Printf("tendermint.SendMsgs: %v", msgs)
	res, err := c.base.SendMsgs(msgs)
	if err != nil {
		return nil, err
	}
	return []byte(res.Logs.String()), nil
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	log.Printf("tendermint.Send: %v", msgs)
	res, err := c.base.SendMsgs(msgs)
	if err != nil || res.Code != 0 {
		c.base.LogFailedTx(res, err, msgs)
		return false
	}
	// NOTE: Add more data to this such as identifiers
	c.base.LogSuccessTx(res, msgs)

	return true
}

func (c *Chain) StartEventListener(dst core.ChainI, strategy core.StrategyI) {
	panic("not implemented error")
}

func (c *Chain) Base() relayer.Chain {
	return c.base
}
