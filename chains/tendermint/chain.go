package tendermint

import (
	"log"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	tmclient "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
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

func (c *Chain) Marshaler() codec.Marshaler {
	return c.base.Encoding.Marshaler
}

// GetAddress returns the sdk.AccAddress associated with the configred key
func (c *Chain) GetAddress() (sdk.AccAddress, error) {
	return c.base.GetAddress()
}

func (c *Chain) SetPath(p *core.PathEnd) error {
	return c.base.SetPath(p)
}

func (c *Chain) Path() *core.PathEnd {
	return c.base.PathEnd
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

func (srcChain *Chain) CreateTrustedHeader(dstChain core.ChainI, srcHeader core.HeaderI) (core.HeaderI, error) {
	// make copy of header stored in mop
	tmp := srcHeader.(*tmclient.Header)
	h := *tmp

	dsth, err := dstChain.GetLatestLightHeight()
	if err != nil {
		return nil, err
	}

	// retrieve counterparty client from dst chain
	counterpartyClientRes, err := dstChain.QueryClientState(dsth, true)
	if err != nil {
		return nil, err
	}

	var cs exported.ClientState
	if err := srcChain.base.Encoding.Marshaler.UnpackAny(counterpartyClientRes.ClientState, &cs); err != nil {
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

func (c *Chain) GetLatestLightHeight() (int64, error) {
	return c.base.GetLatestLightHeight()
}
