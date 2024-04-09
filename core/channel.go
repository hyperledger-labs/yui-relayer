package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	retry "github.com/avast/retry-go"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/log"
)

// CreateChannel runs the channel creation messages on timeout until they pass
// TODO: add max retries or something to this function
func CreateChannel(pathName string, src, dst *ProvableChain, to time.Duration) error {
	if cont, err := checkChannelCreateReady(src, dst); err != nil {
		return err
	} else if !cont {
		return nil
	}

	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "CreateChannel")

	ticker := time.NewTicker(to)
	failures := 0
	for ; true; <-ticker.C {
		chanSteps, err := createChannelStep(src, dst)
		if err != nil {
			logger.Error(
				"failed to create channel step",
				err,
			)
			return err
		}

		if !chanSteps.Ready() {
			logger.Debug("Waiting for next channel step ...")
			continue
		}

		chanSteps.Send(src, dst)
		if chanSteps.Success() {
			if err := SyncChainConfigsFromEvents(pathName, chanSteps.SrcMsgIDs, chanSteps.DstMsgIDs, src, dst); err != nil {
				return err
			}
		}

		switch {
		// In the case of success and this being the last transaction
		// debug logging, log created connection and break
		case chanSteps.Success() && chanSteps.Last:
			logger.Info(
				"★ Channel created",
			)
			return nil
		// In the case of success, reset the failures counter
		case chanSteps.Success():
			failures = 0
			continue
		// In the case of failure, increment the failures counter and exit if this is the 3rd failure
		case !chanSteps.Success():
			failures++
			logger.Info("retrying transaction...")
			time.Sleep(5 * time.Second)
			if failures > 2 {
				logger.Error(
					"! Channel failed",
					err,
				)
				return fmt.Errorf("! Channel failed: [%s]chan{%s}port{%s} -> [%s]chan{%s}port{%s}",
					src.ChainID(), src.Path().ChannelID, src.Path().PortID,
					dst.ChainID(), dst.Path().ChannelID, dst.Path().PortID,
				)
			}
		}
	}

	return nil
}

func checkChannelCreateReady(src, dst *ProvableChain) (bool, error) {
	srcID := src.Chain.Path().ChannelID
	dstID := dst.Chain.Path().ChannelID

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (chantypes.State, error) {
		if pc.Chain.Path().ChannelID == "" {
			return chantypes.UNINITIALIZED, nil
		}

		latestHeight, err := pc.LatestHeight()
		if err != nil {
			return chantypes.UNINITIALIZED, err
		}
		queryHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), 0)
		res, err2 := pc.QueryChannel(NewQueryContext(context.TODO(), queryHeight))
		if err2 != nil {
			return chantypes.UNINITIALIZED, err2
		}
		return res.Channel.State, nil
	}

	srcState, srcErr := getState(src)
	if srcErr != nil {
		return false, srcErr
	}

	dstState, dstErr := getState(src)
	if dstErr != nil {
		return false, dstErr
	}

	if srcID != "" && srcState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("src channel id is given but that channel is not exists: %s", srcID);
	}
	if dstID != "" && dstState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("dst channel id is given but that channel is not exists: %s", dstID);
	}

	if srcState == chantypes.OPEN && dstState == chantypes.OPEN {
		fmt.Printf("channels are already created: src=%s, dst=%s\n", srcID, dstID)
		return false, nil
	}
	return true, nil
}

func createChannelStep(src, dst *ProvableChain) (*RelayMsgs, error) {
	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		return nil, err
	}
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return nil, err
	}

	// Query a number of things all at once
	var (
		srcUpdateHeaders, dstUpdateHeaders []Header
	)

	err = retry.Do(func() error {
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(src, dst); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	srcChan, dstChan, err := QueryChannelPair(sh.GetQueryContext(src.ChainID()), sh.GetQueryContext(dst.ChainID()), src, dst, true)
	if err != nil {
		return nil, err
	}

	if finalized, err := checkChannelFinality(src, dst, srcChan.Channel, dstChan.Channel); err != nil {
		return nil, err
	} else if !finalized {
		return out, nil
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `chanOpenInit` to src
	case srcChan.Channel.State == chantypes.UNINITIALIZED && dstChan.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		out.Src = append(out.Src,
			src.Path().ChanInit(dst.Path(), addr),
		)
	// Handshake has started on dst (1 step done), relay `chanOpenTry` and `updateClient` to src
	case srcChan.Channel.State == chantypes.UNINITIALIZED && dstChan.Channel.State == chantypes.INIT:
		logChannelStates(src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanTry(dst.Path(), dstChan, addr))
	// Handshake has started on src (1 step done), relay `chanOpenTry` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.INIT && dstChan.Channel.State == chantypes.UNINITIALIZED:
		logChannelStates(dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanTry(src.Path(), srcChan, addr))

	// Handshake has started on src (2 steps done), relay `chanOpenAck` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.TRYOPEN && dstChan.Channel.State == chantypes.INIT:
		logChannelStates(dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanAck(src.Path(), srcChan, addr))

	// Handshake has started on dst (2 steps done), relay `chanOpenAck` and `updateClient` to src
	case srcChan.Channel.State == chantypes.INIT && dstChan.Channel.State == chantypes.TRYOPEN:
		logChannelStates(src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanAck(dst.Path(), dstChan, addr))

	// Handshake has confirmed on dst (3 steps done), relay `chanOpenConfirm` and `updateClient` to src
	case srcChan.Channel.State == chantypes.TRYOPEN && dstChan.Channel.State == chantypes.OPEN:
		logChannelStates(src, dst, srcChan, dstChan)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ChanConfirm(dstChan, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `chanOpenConfirm` and `updateClient` to dst
	case srcChan.Channel.State == chantypes.OPEN && dstChan.Channel.State == chantypes.TRYOPEN:
		logChannelStates(dst, src, dstChan, srcChan)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ChanConfirm(srcChan, addr))
		out.Last = true
	default:
		panic(fmt.Sprintf("not implemeneted error: %v <=> %v", srcChan.Channel.State.String(), dstChan.Channel.State.String()))
	}
	return out, nil
}

func logChannelStates(src, dst *ProvableChain, srcChan, dstChan *chantypes.QueryChannelResponse) {
	logger := GetChannelPairLogger(src, dst)
	logger.Info(
		"channel states",
		slog.Group("src",
			slog.Uint64("proof_height", mustGetHeight(srcChan.ProofHeight)),
			slog.String("state", srcChan.Channel.State.String()),
		),
		slog.Group("dst",
			slog.Uint64("proof_height", mustGetHeight(dstChan.ProofHeight)),
			slog.String("state", dstChan.Channel.State.String()),
		))
}

func checkChannelFinality(src, dst *ProvableChain, srcChannel, dstChannel *chantypes.Channel) (bool, error) {
	logger := GetChannelPairLogger(src, dst)
	sh, err := src.LatestHeight()
	if err != nil {
		return false, err
	}
	dh, err := dst.LatestHeight()
	if err != nil {
		return false, err
	}
	srcChanLatest, dstChanLatest, err := QueryChannelPair(NewQueryContext(context.TODO(), sh), NewQueryContext(context.TODO(), dh), src, dst, false)
	if err != nil {
		return false, err
	}
	if srcChannel.State != srcChanLatest.Channel.State {
		logger.Debug("src channel state in transition", "from_state", srcChannel.State, "to_state", srcChanLatest.Channel.State)
		return false, nil
	}
	if dstChannel.State != dstChanLatest.Channel.State {
		logger.Debug("dst channel state in transition", "from_state", dstChannel.State, "to_state", dstChanLatest.Channel.State)
		return false, nil
	}
	return true, nil
}

func GetChannelLogger(c Chain) *log.RelayLogger {
	return log.GetLogger().
		WithChannel(
			c.ChainID(), c.Path().PortID, c.Path().ChannelID,
		).
		WithModule("core.channel")
}

func GetChannelPairLogger(src, dst Chain) *log.RelayLogger {
	return log.GetLogger().
		WithChannelPair(
			src.ChainID(), src.Path().PortID, src.Path().ChannelID,
			dst.ChainID(), dst.Path().PortID, dst.Path().ChannelID,
		).
		WithModule("core.channel")
}
