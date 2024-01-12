package tendermint

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// LogFailedTx takes the transaction and the messages to create it and logs the appropriate data
func (c *Chain) LogFailedTx(res *sdk.TxResponse, err error, msgs []sdk.Msg) {
	logger := GetChainLogger()
	if c.debug {
		logger.Info("sending-tx", "chain-id", c.ChainID())
		for _, msg := range msgs {
			c.Print(msg, false, false)
		}
	}

	if err != nil {
		logger.Error("failed-tx", err, "chain-id", c.ChainID())
		if res == nil {
			return
		}
	}

	if res.Code != 0 && res.Codespace != "" {
		logger.Info("res", "chain-id", c.ChainID(), "height", res.Height, "action", getMsgAction(msgs), "codespace", res.Codespace, "code", res.Code, "raw-log", res.RawLog)
	}

	if c.debug && !res.Empty() {
		logger.Info("tx-response", "chain-id", c.ChainID(), "res", res)
		c.Print(res, false, false)
	}
}

// LogSuccessTx take the transaction and the messages to create it and logs the appropriate data
func (c *Chain) LogSuccessTx(res *sdk.TxResponse, msgs []sdk.Msg) {
	logger := GetChainLogger()
	logger.Info("success-tx", "chain-id", c.ChainID(), "height", res.Height, "hash", res.TxHash)
}

// Print fmt.Printlns the json or yaml representation of whatever is passed in
// CONTRACT: The cmd calling this function needs to have the "json" and "indent" flags set
// TODO: better "text" printing here would be a nice to have
// TODO: fix indenting all over the code base
func (c *Chain) Print(toPrint proto.Message, text, indent bool) error {
	var (
		out []byte
		err error
	)

	switch {
	case indent && text:
		return fmt.Errorf("must pass either indent or text, not both")
	case text:
		// TODO: This isn't really a good option,
		out = []byte(fmt.Sprintf("%v", toPrint))
	default:
		out, err = c.codec.MarshalJSON(toPrint)
	}

	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func getMsgAction(msgs []sdk.Msg) string {
	var out string
	for i, msg := range msgs {
		out += fmt.Sprintf("%d:%s,", i, sdk.MsgTypeURL(msg))
	}
	return strings.TrimSuffix(out, ",")
}
