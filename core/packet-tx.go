package core

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/log"
)

func SendTransferMsg(src, dst *ProvableChain, amount sdk.Coin, dstAddr fmt.Stringer, toHeightOffset uint64, toTimeOffset time.Duration) error {
	logger := GetChannelLogger(log.GetLogger(), src, dst)
	var (
		timeoutHeight    uint64
		timeoutTimestamp uint64
	)

	h, err := dst.LatestHeight()
	if err != nil {
		return err
	}

	// Properly render the address string
	dstAddrString := dstAddr.String()

	switch {
	case toHeightOffset > 0 && toTimeOffset > 0:
		return fmt.Errorf("cant set both timeout height and time offset")
	case toHeightOffset > 0:
		timeoutHeight = uint64(h.GetRevisionHeight()) + toHeightOffset
		timeoutTimestamp = 0
	case toTimeOffset > 0:
		timeoutHeight = 0
		timeoutTimestamp = uint64(time.Now().Add(toTimeOffset).UnixNano())
	case toHeightOffset == 0 && toTimeOffset == 0:
		timeoutHeight = uint64(h.GetRevisionHeight() + 1000)
		timeoutTimestamp = 0
	}

	srcAddr, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"failed to get address for send transfer",
			err,
		)
		return err
	}

	// MsgTransfer will call SendPacket on src chain
	txs := RelayMsgs{
		Src: []sdk.Msg{src.Path().MsgTransfer(
			dst.Path(), amount, dstAddrString, srcAddr, timeoutHeight, timeoutTimestamp, "",
		)},
		Dst: []sdk.Msg{},
	}

	if txs.Send(src, dst); !txs.Success() {
		logger.Error(
			"failed to send transfer message",
		)
		return fmt.Errorf("failed to send transfer message")
	}
	return nil
}
