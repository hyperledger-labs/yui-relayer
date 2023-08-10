package core

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger-labs/yui-relayer/logger"
	"go.uber.org/zap"
)

func SendTransferMsg(src, dst *ProvableChain, amount sdk.Coin, dstAddr fmt.Stringer, toHeightOffset uint64, toTimeOffset time.Duration) error {
	zapLogger := logger.GetLogger()
	defer zapLogger.SugaredLogger.Sync()
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
		GetChainLoggerFromProvaleChain(zapLogger.SugaredLogger, src, dst).Errorw(
			"failed to get address for send transfer",
			err,
			zap.Stack("stack"),
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
		GetChainLoggerFromProvaleChain(zapLogger.SugaredLogger, src, dst).Errorw(
			"failed to send transfer message",
			errors.New("failed to send transfer message"),
			zap.Stack("stack"),
		)
		return fmt.Errorf("failed to send transfer message")
	}
	return nil
}
