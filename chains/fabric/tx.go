package fabric

import (
	"log"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

const (
	handleTxFunc = "handleTx"
)

func (c *Chain) SendMsgs(msgs []sdk.Msg) ([]byte, error) {
	log.Printf("fabric.SendMsgs: %v", msgs)
	txBytes, err := c.buildTx(c.codec.InterfaceRegistry(), msgs...)
	if err != nil {
		return nil, err
	}
	res, err := c.gateway.Contract.SubmitTransaction(handleTxFunc, string(txBytes))
	log.Printf("fabric.SendMsgs.result: res='%v' err='%v'", res, err)
	return res, err
}

func (c *Chain) Send(msgs []sdk.Msg) bool {
	_, err := c.SendMsgs(msgs)
	return err == nil
}

// NOTE When uses this tx format, the chaincode must use github.com/hyperledger-labs/yui-fabric-ibc/x/auth/ante/ante.go to validate this.
func (c *Chain) buildTx(registry codectypes.InterfaceRegistry, msgs ...sdk.Msg) ([]byte, error) {
	m := codec.NewProtoCodec(registry)
	cfg := authtx.NewTxConfig(m, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})

	txBuilder := cfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	// TODO use an empty pubkey instead of temporary key
	senderPrivKey := secp256k1.GenPrivKey()
	sig := signing.SignatureV2{
		PubKey: senderPrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
	}
	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}
	tx := txBuilder.GetTx()
	return cfg.TxJSONEncoder()(tx)
}
