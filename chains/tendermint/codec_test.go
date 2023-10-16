package tendermint_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/hyperledger-labs/yui-relayer/chains/tendermint"
	"github.com/hyperledger-labs/yui-relayer/core"
)

func TestCodec(t *testing.T) {
	codec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	tendermint.RegisterInterfaces(codec.InterfaceRegistry())

	orig := tendermint.MsgID{
		TxHash:   "hoge",
		MsgIndex: 123,
	}

	bz, err := codec.MarshalInterface(core.MsgID(&orig))
	if err != nil {
		t.Fatalf("failed to marshal from tendermint.MsgID to Any: %v", err)
	}

	var msgID core.MsgID
	if err := codec.UnmarshalInterface(bz, &msgID); err != nil {
		t.Fatalf("failed to unmarshal from Any to core.MsgID: %v", err)
	}

	tmMsgID, ok := msgID.(*tendermint.MsgID)
	if !ok {
		t.Fatalf("unexpected MsgID instance type: %T", msgID)
	}

	if orig != *tmMsgID {
		t.Fatalf("unmatched MsgID values: %v != %v", orig, *tmMsgID)
	}
}
