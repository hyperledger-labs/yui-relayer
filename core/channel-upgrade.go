package core

import (
	"errors"
	"fmt"
	"math"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// InitChannelUpgrade builds `MsgChannelUpgradeInit` based on the specified UpgradeFields and sends it to the specified chain.
func InitChannelUpgrade(chain *ProvableChain, upgradeFields chantypes.UpgradeFields) error {
	logger := GetChannelLogger(chain.Chain)
	defer logger.TimeTrack(time.Now(), "InitChannelUpgrade")

	addr, err := chain.GetAddress()
	if err != nil {
		logger.Error("failed to get address", err)
		return err
	}

	msg := chain.Path().ChanUpgradeInit(upgradeFields, addr)

	msgIDs, err := chain.SendMsgs([]sdk.Msg{msg})
	if err != nil {
		logger.Error("failed to send MsgChannelUpgradeInit", err)
		return err
	} else if len(msgIDs) != 1 {
		panic(fmt.Sprintf("len(msgIDs) == %d", len(msgIDs)))
	} else if result, err := GetFinalizedMsgResult(*chain, msgIDs[0]); err != nil {
		logger.Error("failed to the finalized result of MsgChannelUpgradeInit", err)
		return err
	} else if ok, desc := result.Status(); !ok {
		err := fmt.Errorf("failed to initialize channel upgrade: %s", desc)
		logger.Error(err.Error(), err)
	} else {
		logger.Info("successfully initialized channel upgrade")
	}

	return nil
}

// ExecuteChannelUpgrade carries out channel upgrade handshake until both chains transition to the OPEN state.
// This function repeatedly checks the states of both chains and decides the next action.
func ExecuteChannelUpgrade(src, dst *ProvableChain, interval time.Duration) error {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "ExecuteChannelUpgrade")

	tick := time.Tick(interval)
	failures := 0
	for {
		<-tick

		steps, err := upgradeChannelStep(src, dst)
		if err != nil {
			logger.Error("failed to create channel upgrade step", err)
			return err
		}

		if !steps.Ready() {
			logger.Debug("Waiting for next channel upgrade step ...")
			continue
		}

		steps.Send(src, dst)

		if steps.Success() {
			if steps.Last {
				logger.Info("Channel upgrade completed")
				return nil
			}

			failures = 0
		} else {
			if failures++; failures > 2 {
				err := errors.New("Channel upgrade failed")
				logger.Error(err.Error(), err)
				return err
			}

			logger.Warn("Retrying transaction...")
			time.Sleep(5 * time.Second)
		}
	}
}

func upgradeChannelStep(src, dst *ProvableChain) (*RelayMsgs, error) {
	logger := GetChannelPairLogger(src, dst)

	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		logger.Error("failed to validate paths", err)
		return nil, err
	}

	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		logger.Error("failed to create SyncHeaders", err)
		return nil, err
	}

	// Query a number of things all at once
	var srcUpdateHeaders, dstUpdateHeaders []Header
	if err := retry.Do(func() error {
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(uint, error) {
		if err := sh.Updates(src, dst); err != nil {
			panic(err)
		}
	})); err != nil {
		logger.Error("failed to set up headers for LC update on both chains", err)
		return nil, err
	}

	// prepare query contexts
	var srcCtxFinalized, dstCtxFinalized, srcCtxLatest, dstCtxLatest QueryContext
	if srcCtxFinalized, err = getQueryContext(src, sh, true); err != nil {
		return nil, err
	}
	if dstCtxFinalized, err = getQueryContext(dst, sh, true); err != nil {
		return nil, err
	}
	if srcCtxLatest, err = getQueryContext(src, sh, false); err != nil {
		return nil, err
	}
	if dstCtxLatest, err = getQueryContext(dst, sh, false); err != nil {
		return nil, err
	}

	// query finalized channels with proofs
	srcChan, dstChan, err := QueryChannelPair(
		srcCtxFinalized,
		dstCtxFinalized,
		src,
		dst,
		true,
	)
	if err != nil {
		return nil, err
	} else if finalized, err := checkChannelFinality(src, dst, srcChan.Channel, dstChan.Channel); err != nil {
		return nil, err
	} else if !finalized {
		return out, nil
	}

	// query finalized channel upgrades with proofs
	srcChanUpg, dstChanUpg, finalized, err := queryFinalizedChannelUpgradePair(
		srcCtxFinalized,
		dstCtxFinalized,
		srcCtxLatest,
		dstCtxLatest,
		src,
		dst,
	)
	if err != nil {
		return nil, err
	} else if !finalized {
		return out, nil
	}

	// translate channel state to channel upgrade state
	type UpgradeState chantypes.State
	const (
		UPGRADEUNINIT = UpgradeState(chantypes.OPEN)
		UPGRADEINIT   = UpgradeState(math.MaxInt32)
		FLUSHING      = UpgradeState(chantypes.FLUSHING)
		FLUSHCOMPLETE = UpgradeState(chantypes.FLUSHCOMPLETE)
	)
	srcState := UpgradeState(srcChan.Channel.State)
	if srcState == UPGRADEUNINIT && srcChanUpg != nil {
		srcState = UPGRADEINIT
	}
	dstState := UpgradeState(dstChan.Channel.State)
	if dstState == UPGRADEUNINIT && dstChanUpg != nil {
		dstState = UPGRADEINIT
	}

	// query finalized channel upgrade error receipts with proofs
	srcChanUpgErr, dstChanUpgErr, finalized, err := queryFinalizedChannelUpgradeErrorPair(
		srcCtxFinalized,
		dstCtxFinalized,
		srcCtxLatest,
		dstCtxLatest,
		src,
		dst,
	)
	if err != nil {
		return nil, err
	} else if !finalized {
		return out, nil
	}

	doTry := func(chain *ProvableChain, cpCtx QueryContext, cp *ProvableChain, cpHeaders []Header, cpChan *chantypes.QueryChannelResponse, cpChanUpg *chantypes.QueryUpgradeResponse) ([]sdk.Msg, error) {
		proposedConnectionID, err := queryProposedConnectionID(cpCtx, cp, cpChanUpg)
		if err != nil {
			return nil, err
		}
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeTry(proposedConnectionID, cpChan, cpChanUpg, addr))
		return msgs, nil
	}

	doAck := func(chain *ProvableChain, cpHeaders []Header, cpChan *chantypes.QueryChannelResponse, cpChanUpg *chantypes.QueryUpgradeResponse) []sdk.Msg {
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeAck(cpChan, cpChanUpg, addr))
		return msgs
	}

	doConfirm := func(chain *ProvableChain, cpHeaders []Header, cpChan *chantypes.QueryChannelResponse, cpChanUpg *chantypes.QueryUpgradeResponse) []sdk.Msg {
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeConfirm(cpChan, cpChanUpg, addr))
		return msgs
	}

	doOpen := func(chain *ProvableChain, cpHeaders []Header, cpChan *chantypes.QueryChannelResponse) []sdk.Msg {
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeOpen(cpChan, addr))
		return msgs
	}

	doCancel := func(chain *ProvableChain, cpHeaders []Header, cpChanUpgErr *chantypes.QueryUpgradeErrorResponse) []sdk.Msg {
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeCancel(cpChanUpgErr, addr))
		return msgs
	}

	doTimeout := func(chain *ProvableChain, cpHeaders []Header, cpChan *chantypes.QueryChannelResponse) []sdk.Msg {
		var msgs []sdk.Msg
		addr := mustGetAddress(chain)
		if len(cpHeaders) > 0 {
			msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		}
		msgs = append(msgs, chain.Path().ChanUpgradeTimeout(cpChan, addr))
		return msgs
	}

	switch {
	case srcState == UPGRADEUNINIT && dstState == UPGRADEUNINIT:
		return nil, errors.New("channel upgrade is not initialized")
	case srcState == UPGRADEINIT && dstState == UPGRADEUNINIT:
		if dstChan.Channel.UpgradeSequence >= srcChan.Channel.UpgradeSequence {
			out.Src = doCancel(src, dstUpdateHeaders, dstChanUpgErr)
		} else {
			if out.Dst, err = doTry(dst, srcCtxFinalized, src, srcUpdateHeaders, srcChan, srcChanUpg); err != nil {
				return nil, err
			}
		}
	case srcState == UPGRADEUNINIT && dstState == UPGRADEINIT:
		if srcChan.Channel.UpgradeSequence >= dstChan.Channel.UpgradeSequence {
			out.Dst = doCancel(dst, srcUpdateHeaders, srcChanUpgErr)
		} else {
			if out.Src, err = doTry(src, dstCtxFinalized, dst, dstUpdateHeaders, dstChan, dstChanUpg); err != nil {
				return nil, err
			}
		}
	case srcState == UPGRADEUNINIT && dstState == FLUSHING:
		out.Dst = doCancel(dst, srcUpdateHeaders, srcChanUpgErr)
	case srcState == FLUSHING && dstState == UPGRADEUNINIT:
		out.Src = doCancel(src, dstUpdateHeaders, dstChanUpgErr)
	case srcState == UPGRADEUNINIT && dstState == FLUSHCOMPLETE:
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtxFinalized, dst, dstChanUpg); err != nil {
			return nil, err
		} else if complete {
			out.Dst = doOpen(dst, srcUpdateHeaders, srcChan)
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Dst = doTimeout(dst, srcUpdateHeaders, srcChan)
		} else {
			out.Dst = doCancel(dst, srcUpdateHeaders, srcChanUpgErr)
		}
	case srcState == FLUSHCOMPLETE && dstState == UPGRADEUNINIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtxFinalized, src, srcChanUpg); err != nil {
			return nil, err
		} else if complete {
			out.Src = doOpen(src, dstUpdateHeaders, dstChan)
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Src = doTimeout(src, dstUpdateHeaders, dstChan)
		} else {
			out.Src = doCancel(src, dstUpdateHeaders, dstChanUpgErr)
		}
	case srcState == UPGRADEINIT && dstState == UPGRADEINIT: // crossing hellos
		// it is intentional to execute chanUpgradeTry on both sides if upgrade sequences
		// are identical to each other. this is for testing purpose.
		if srcChan.Channel.UpgradeSequence >= dstChan.Channel.UpgradeSequence {
			if out.Dst, err = doTry(dst, srcCtxFinalized, src, srcUpdateHeaders, srcChan, srcChanUpg); err != nil {
				return nil, err
			}
		}
		if srcChan.Channel.UpgradeSequence <= dstChan.Channel.UpgradeSequence {
			if out.Src, err = doTry(src, dstCtxFinalized, dst, dstUpdateHeaders, dstChan, dstChanUpg); err != nil {
				return nil, err
			}
		}
	case srcState == UPGRADEINIT && dstState == FLUSHING:
		if srcChan.Channel.UpgradeSequence != dstChan.Channel.UpgradeSequence {
			out.Dst = doCancel(dst, srcUpdateHeaders, srcChanUpgErr)
		} else {
			// chanUpgradeAck checks if counterparty-specified timeout has exceeded.
			// if it has, chanUpgradeAck aborts the upgrade handshake.
			// Therefore the relayer need not check timeout by itself.
			out.Src = doAck(src, dstUpdateHeaders, dstChan, dstChanUpg)
		}
	case srcState == FLUSHING && dstState == UPGRADEINIT:
		if srcChan.Channel.UpgradeSequence != dstChan.Channel.UpgradeSequence {
			out.Src = doCancel(src, dstUpdateHeaders, dstChanUpgErr)
		} else {
			// chanUpgradeAck checks if counterparty-specified timeout has exceeded.
			// if it has, chanUpgradeAck aborts the upgrade handshake.
			// Therefore the relayer need not check timeout by itself.
			out.Dst = doAck(dst, srcUpdateHeaders, srcChan, srcChanUpg)
		}
	case srcState == UPGRADEINIT && dstState == FLUSHCOMPLETE:
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtxFinalized, dst, dstChanUpg); err != nil {
			return nil, err
		} else if complete {
			out.Dst = doOpen(dst, srcUpdateHeaders, srcChan)
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Dst = doTimeout(dst, srcUpdateHeaders, srcChan)
		} else {
			out.Dst = doCancel(dst, srcUpdateHeaders, srcChanUpgErr)
		}
	case srcState == FLUSHCOMPLETE && dstState == UPGRADEINIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtxFinalized, src, srcChanUpg); err != nil {
			return nil, err
		} else if complete {
			out.Src = doOpen(src, dstUpdateHeaders, dstChan)
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Src = doTimeout(src, dstUpdateHeaders, dstChan)
		} else {
			out.Src = doCancel(src, dstUpdateHeaders, dstChanUpgErr)
		}
	case srcState == FLUSHING && dstState == FLUSHING:
		nTimedout := 0
		if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			return nil, err
		} else if timedout {
			nTimedout += 1
			out.Dst = doTimeout(dst, srcUpdateHeaders, srcChan)
		}
		if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			return nil, err
		} else if timedout {
			nTimedout += 1
			out.Src = doTimeout(src, dstUpdateHeaders, dstChan)
		}

		// if any chains have exceeded timeout, never execute chanUpgradeConfirm
		if nTimedout > 0 {
			break
		}

		if completable, err := src.QueryCanTransitionToFlushComplete(srcCtxFinalized); err != nil {
			return nil, err
		} else if completable {
			out.Src = doConfirm(src, dstUpdateHeaders, dstChan, dstChanUpg)
		}
		if completable, err := dst.QueryCanTransitionToFlushComplete(dstCtxFinalized); err != nil {
			return nil, err
		} else if completable {
			out.Dst = doConfirm(dst, srcUpdateHeaders, srcChan, srcChanUpg)
		}
	case srcState == FLUSHING && dstState == FLUSHCOMPLETE:
		if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Dst = doTimeout(dst, srcUpdateHeaders, srcChan)
		} else if completable, err := src.QueryCanTransitionToFlushComplete(srcCtxFinalized); err != nil {
			return nil, err
		} else if completable {
			out.Src = doConfirm(src, dstUpdateHeaders, dstChan, dstChanUpg)
		}
	case srcState == FLUSHCOMPLETE && dstState == FLUSHING:
		if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			return nil, err
		} else if timedout {
			out.Src = doTimeout(src, dstUpdateHeaders, dstChan)
		} else if completable, err := dst.QueryCanTransitionToFlushComplete(dstCtxFinalized); err != nil {
			return nil, err
		} else if completable {
			out.Dst = doConfirm(dst, srcUpdateHeaders, srcChan, srcChanUpg)
		}
	case srcState == FLUSHCOMPLETE && dstState == FLUSHCOMPLETE:
		out.Src = doOpen(src, dstUpdateHeaders, dstChan)
		out.Dst = doOpen(dst, srcUpdateHeaders, srcChan)
	default:
		return nil, errors.New("unexpected state")
	}

	return out, nil
}

func queryProposedConnectionID(cpCtx QueryContext, cp *ProvableChain, cpChanUpg *chantypes.QueryUpgradeResponse) (string, error) {
	if cpConn, err := cp.QueryConnection(
		cpCtx,
		cpChanUpg.Upgrade.Fields.ConnectionHops[0],
	); err != nil {
		return "", err
	} else {
		return cpConn.Connection.Counterparty.ConnectionId, nil
	}
}

func queryFinalizedChannelUpgradePair(
	srcCtxFinalized,
	dstCtxFinalized,
	srcCtxLatest,
	dstCtxLatest QueryContext,
	src,
	dst *ProvableChain,
) (srcChanUpg, dstChanUpg *chantypes.QueryUpgradeResponse, finalized bool, err error) {
	logger := GetChannelPairLogger(src, dst)

	// query channel upgrade pair at latest finalized heights
	srcChanUpgF, dstChanUpgF, err := QueryChannelUpgradePair(
		srcCtxFinalized,
		dstCtxFinalized,
		src,
		dst,
		true,
	)
	if err != nil {
		logger.Error("failed to query a channel upgrade pair at the latest finalized heights", err)
		return nil, nil, false, err
	}

	// query channel upgrade pair at latest heights
	srcChanUpgL, dstChanUpgL, err := QueryChannelUpgradePair(
		srcCtxLatest,
		dstCtxLatest,
		src,
		dst,
		false,
	)
	if err != nil {
		logger.Error("failed to query a channel upgrade pair at the latest heights", err)
		return nil, nil, false, err
	}

	if !compareUpgrades(srcChanUpgF, srcChanUpgL) {
		logger.Debug("channel upgrade is not finalized on src chain")
		return nil, nil, false, nil
	}
	if !compareUpgrades(dstChanUpgF, dstChanUpgL) {
		logger.Debug("channel upgrade is not finalized on dst chain")
		return nil, nil, false, nil
	}

	return srcChanUpgF, dstChanUpgF, true, nil
}

func compareUpgrades(a, b *chantypes.QueryUpgradeResponse) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return false
	}
	return a.Upgrade.String() == b.Upgrade.String()
}

func queryFinalizedChannelUpgradeErrorPair(
	srcCtxFinalized,
	dstCtxFinalized,
	srcCtxLatest,
	dstCtxLatest QueryContext,
	src,
	dst *ProvableChain,
) (srcChanUpg, dstChanUpg *chantypes.QueryUpgradeErrorResponse, finalized bool, err error) {
	logger := GetChannelPairLogger(src, dst)

	// query channel upgrade pair at latest finalized heights
	srcChanUpgErrF, dstChanUpgErrF, err := QueryChannelUpgradeErrorPair(
		srcCtxFinalized,
		dstCtxFinalized,
		src,
		dst,
		true,
	)
	if err != nil {
		logger.Error("failed to query a channel upgrade error pair at the latest finalized heights", err)
		return nil, nil, false, err
	}

	// query channel upgrade pair at latest heights
	srcChanUpgErrL, dstChanUpgErrL, err := QueryChannelUpgradeErrorPair(
		srcCtxFinalized,
		dstCtxFinalized,
		src,
		dst,
		false,
	)
	if err != nil {
		logger.Error("failed to query a channel upgrade error pair at the latest heights", err)
		return nil, nil, false, err
	}

	if !compareUpgradeErrors(srcChanUpgErrF, srcChanUpgErrL) {
		logger.Debug("channel upgrade error is not finalized on src chain")
		return nil, nil, false, nil
	}
	if !compareUpgradeErrors(dstChanUpgErrF, dstChanUpgErrL) {
		logger.Debug("channel upgrade error is not finalized on dst chain")
		return nil, nil, false, nil
	}

	return srcChanUpgErrF, dstChanUpgErrF, true, nil
}

func compareUpgradeErrors(a, b *chantypes.QueryUpgradeErrorResponse) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return false
	}
	return a.ErrorReceipt.String() == b.ErrorReceipt.String()
}

func upgradeAlreadyComplete(
	channel *chantypes.QueryChannelResponse,
	cpCtx QueryContext,
	cp *ProvableChain,
	cpChanUpg *chantypes.QueryUpgradeResponse,
) (bool, error) {
	proposedConnectionID, err := queryProposedConnectionID(cpCtx, cp, cpChanUpg)
	if err != nil {
		return false, err
	}
	result := channel.Channel.Version == cpChanUpg.Upgrade.Fields.Version &&
		channel.Channel.Ordering == cpChanUpg.Upgrade.Fields.Ordering &&
		channel.Channel.ConnectionHops[0] == proposedConnectionID
	return result, nil
}

func upgradeAlreadyTimedOut(
	ctx QueryContext,
	chain *ProvableChain,
	cpChanUpg *chantypes.QueryUpgradeResponse,
) (bool, error) {
	height := ctx.Height().(clienttypes.Height)
	timestamp, err := chain.Timestamp(height)
	if err != nil {
		return false, err
	}
	return cpChanUpg.Upgrade.Timeout.Elapsed(height, uint64(timestamp.UnixNano())), nil
}
