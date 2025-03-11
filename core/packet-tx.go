package core

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func SendTransferMsg(ctx context.Context, src, dst *ProvableChain, amount sdk.Coin, dstAddr string, toHeightOffset uint64, toTimeOffset time.Duration) error {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "SendTransferMsg")
	var (
		timeoutHeight    uint64
		timeoutTimestamp uint64
	)

	h, err := dst.LatestHeight(ctx)
	if err != nil {
		return err
	}

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
			dst.Path(), amount, dstAddr, srcAddr, timeoutHeight, timeoutTimestamp, "",
		)},
		Dst: []sdk.Msg{},
	}

	if txs.Send(ctx, src, dst); !txs.Success() {
		err := fmt.Errorf("failed to send transfer message")
		logger.Error(err.Error(), err)
		return err
	}
	return nil
}
