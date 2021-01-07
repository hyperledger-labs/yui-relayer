package fabric

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/datachainlab/fabric-ibc/x/auth/types"
)

const (
	handleIBCTxFunc = "handleIBCTx"
)

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	txBytes, err := buildTx(c.encodingConfig.InterfaceRegistry)
	if err != nil {
		return nil, err
	}
	return c.gateway.Contract.SubmitTransaction(handleIBCTxFunc, string(txBytes))
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	_, err := c.SendMsgs(msgs)
	return err == nil
}

// NOTE When uses this tx format, the chaincode must use github.com/datachainlab/fabric-ibc/x/auth/ante/ante.go to validate this.
func buildTx(registry codectypes.InterfaceRegistry, msgs ...sdk.Msg) ([]byte, error) {
	m := codec.NewProtoCodec(registry)
	cfg := authtx.NewTxConfig(m, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	tx := authtypes.NewStdTx(msgs)
	return cfg.TxJSONEncoder()(tx)
}
