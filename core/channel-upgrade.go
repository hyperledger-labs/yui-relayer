package core

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/log"
)

type UpgradeState int

const (
	UPGRADE_STATE_UNINIT UpgradeState = iota
	UPGRADE_STATE_INIT
	UPGRADE_STATE_FLUSHING
	UPGRADE_STATE_FLUSHCOMPLETE
)

func (state UpgradeState) String() string {
	switch state {
	case UPGRADE_STATE_UNINIT:
		return "UNINIT"
	case UPGRADE_STATE_INIT:
		return "INIT"
	case UPGRADE_STATE_FLUSHING:
		return "FLUSHING"
	case UPGRADE_STATE_FLUSHCOMPLETE:
		return "FLUSHCOMPLETE"
	default:
		panic(fmt.Errorf("unexpected UpgradeState: %d", state))
	}
}

type UpgradeAction int

const (
	UPGRADE_ACTION_NONE UpgradeAction = iota
	UPGRADE_ACTION_TRY
	UPGRADE_ACTION_ACK
	UPGRADE_ACTION_CONFIRM
	UPGRADE_ACTION_OPEN
	UPGRADE_ACTION_CANCEL
	UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
	UPGRADE_ACTION_TIMEOUT
)

func (action UpgradeAction) String() string {
	switch action {
	case UPGRADE_ACTION_NONE:
		return "NONE"
	case UPGRADE_ACTION_TRY:
		return "TRY"
	case UPGRADE_ACTION_ACK:
		return "ACK"
	case UPGRADE_ACTION_CONFIRM:
		return "CONFIRM"
	case UPGRADE_ACTION_OPEN:
		return "OPEN"
	case UPGRADE_ACTION_CANCEL:
		return "CANCEL"
	case UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE:
		return "CANCEL_FLUSHCOMPLETE"
	case UPGRADE_ACTION_TIMEOUT:
		return "TIMEOUT"
	default:
		panic(fmt.Errorf("unexpected UpgradeAction: %d", action))
	}
}

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
		logger.Error("failed to get the finalized result of MsgChannelUpgradeInit", err)
		return err
	} else if ok, desc := result.Status(); !ok {
		err := fmt.Errorf("failed to initialize channel upgrade: %s", desc)
		logger.Error(err.Error(), err)
		return err
	} else {
		logger.Info("successfully initialized channel upgrade")
	}

	return nil
}

// ExecuteChannelUpgrade carries out channel upgrade handshake until both chains transition to the OPEN state.
// This function repeatedly checks the states of both chains and decides the next action.
func ExecuteChannelUpgrade(pathName string, src, dst *ProvableChain, interval time.Duration, targetSrcState, targetDstState UpgradeState) error {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "ExecuteChannelUpgrade")

	tick := time.Tick(interval)
	failures := 0
	firstCall := true
	for {
		<-tick

		steps, err := upgradeChannelStep(src, dst, targetSrcState, targetDstState, firstCall)
		if err != nil {
			logger.Error("failed to create channel upgrade step", err)
			return err
		}

		if steps.Last {
			logger.Info("Channel upgrade completed")
			return nil
		}

		if !steps.Ready() {
			logger.Debug("Waiting for next channel upgrade step ...")
			continue
		}

		steps.Send(src, dst)

		if steps.Success() {
			if err := SyncChainConfigsFromEvents(pathName, steps.SrcMsgIDs, steps.DstMsgIDs, src, dst); err != nil {
				logger.Error("failed to synchronize the updated path config to the config file", err)
				return err
			}

			firstCall = false
			failures = 0
		} else {
			if failures++; failures > 2 {
				err := errors.New("channel upgrade failed")
				logger.Error(err.Error(), err)
				return err
			}

			logger.Warn("Retrying transaction...")
			time.Sleep(5 * time.Second)
		}
	}
}

// CancelChannelUpgrade executes chanUpgradeCancel on `chain`.
func CancelChannelUpgrade(chain, cp *ProvableChain) error {
	logger := GetChannelPairLogger(chain, cp)
	defer logger.TimeTrack(time.Now(), "CancelChannelUpgrade")

	addr, err := chain.GetAddress()
	if err != nil {
		logger.Error("failed to get address", err)
		return err
	}

	var ctx, cpCtx QueryContext
	if sh, err := NewSyncHeaders(chain, cp); err != nil {
		logger.Error("failed to create a SyncHeaders", err)
		return err
	} else {
		ctx = sh.GetQueryContext(chain.ChainID())
		cpCtx = sh.GetQueryContext(cp.ChainID())
	}

	chann, err := chain.QueryChannel(ctx)
	if err != nil {
		logger.Error("failed to get the channel state", err)
		return err
	}

	upgErr, err := QueryChannelUpgradeError(cpCtx, cp, true)
	if err != nil {
		logger.Error("failed to query the channel upgrade error receipt", err)
		return err
	} else if chann.Channel.State == chantypes.FLUSHCOMPLETE &&
		(upgErr == nil || upgErr.ErrorReceipt.Sequence != chann.Channel.UpgradeSequence) {
		var err error
		if upgErr == nil {
			err = fmt.Errorf("upgrade error receipt not found")
		} else {
			err = fmt.Errorf("upgrade sequences don't match: channel.upgrade_sequence=%d, error_receipt.sequence=%d",
				chann.Channel.UpgradeSequence, upgErr.ErrorReceipt.Sequence)
		}
		logger.Error("cannot cancel the upgrade in FLUSHCOMPLETE state", err)
		return err
	} else if upgErr == nil {
		// NOTE: Even if an error receipt is not found, anyway try to execute ChanUpgradeCancel.
		// If the sender is authority and the channel state is anything other than FLUSHCOMPLETE,
		// the cancellation will be successful.
		upgErr = &chantypes.QueryUpgradeErrorResponse{}
	}

	msg := chain.Path().ChanUpgradeCancel(upgErr, addr)

	if msgIDs, err := chain.SendMsgs([]sdk.Msg{msg}); err != nil {
		logger.Error("failed to send MsgChannelUpgradeCancel", err)
		return err
	} else if len(msgIDs) != 1 {
		panic(fmt.Sprintf("len(msgIDs) == %d", len(msgIDs)))
	} else if result, err := GetFinalizedMsgResult(*chain, msgIDs[0]); err != nil {
		logger.Error("failed to get the finalized result of MsgChannelUpgradeCancel", err)
		return err
	} else if ok, desc := result.Status(); !ok {
		err := fmt.Errorf("failed to cancel the channel upgrade: %s", desc)
		logger.Error(err.Error(), err)
		return err
	} else {
		logger.Info("successfully cancelled the channel upgrade")
	}

	return nil
}

func NewUpgradeState(chanState chantypes.State, upgradeExists bool) (UpgradeState, error) {
	switch chanState {
	case chantypes.OPEN:
		if upgradeExists {
			return UPGRADE_STATE_INIT, nil
		} else {
			return UPGRADE_STATE_UNINIT, nil
		}
	case chantypes.FLUSHING:
		return UPGRADE_STATE_FLUSHING, nil
	case chantypes.FLUSHCOMPLETE:
		return UPGRADE_STATE_FLUSHCOMPLETE, nil
	default:
		return 0, fmt.Errorf("channel not opened yet: state=%s", chanState)
	}
}

func upgradeChannelStep(src, dst *ProvableChain, targetSrcState, targetDstState UpgradeState, firstCall bool) (*RelayMsgs, error) {
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
		logger.Error("failed to get query the context based on the src chain's latest finalized height", err)
		return nil, err
	}
	if dstCtxFinalized, err = getQueryContext(dst, sh, true); err != nil {
		logger.Error("failed to get query the context based on the dst chain's latest finalized height", err)
		return nil, err
	}
	if srcCtxLatest, err = getQueryContext(src, sh, false); err != nil {
		logger.Error("failed to get query the context based on the src chain's latest height", err)
		return nil, err
	}
	if dstCtxLatest, err = getQueryContext(dst, sh, false); err != nil {
		logger.Error("failed to get query the context based on the dst chain's latest height", err)
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
		logger.Error("failed to query the channel pair with proofs", err)
		return nil, err
	} else if finalized, err := checkChannelFinality(src, dst, srcChan.Channel, dstChan.Channel); err != nil {
		logger.Error("failed to check if the queried channels have been finalized", err)
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
		logger.Error("failed to query the finalized channel upgrade pair with proofs", err)
		return nil, err
	} else if !finalized {
		return out, nil
	}

	// determine upgrade states
	srcState, err := NewUpgradeState(srcChan.Channel.State, srcChanUpg != nil)
	if err != nil {
		logger.Error("failed to create UpgradeState of the src chain", err)
		return nil, err
	}
	dstState, err := NewUpgradeState(dstChan.Channel.State, dstChanUpg != nil)
	if err != nil {
		logger.Error("failed to create UpgradeState of the dst chain", err)
		return nil, err
	}

	logger = &log.RelayLogger{Logger: logger.With(
		slog.Group("current_channel_upgrade_states",
			"src", srcState.String(),
			"dst", dstState.String(),
		),
	)}

	// check if both chains have reached the target states or UNINIT states
	if !firstCall && srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_UNINIT ||
		srcState != UPGRADE_STATE_UNINIT && dstState != UPGRADE_STATE_UNINIT && srcState == targetSrcState && dstState == targetDstState {
		logger.Info("both chains have reached the target states")
		out.Last = true
		return out, nil
	}

	// determine next actions for src/dst chains
	srcAction := UPGRADE_ACTION_NONE
	dstAction := UPGRADE_ACTION_NONE
	switch {
	case srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_UNINIT:
		return nil, errors.New("channel upgrade is not initialized")
	case srcState == UPGRADE_STATE_INIT && dstState == UPGRADE_STATE_UNINIT:
		if dstChan.Channel.UpgradeSequence >= srcChan.Channel.UpgradeSequence {
			srcAction = UPGRADE_ACTION_CANCEL
		} else {
			dstAction = UPGRADE_ACTION_TRY
		}
	case srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_INIT:
		if srcChan.Channel.UpgradeSequence >= dstChan.Channel.UpgradeSequence {
			dstAction = UPGRADE_ACTION_CANCEL
		} else {
			srcAction = UPGRADE_ACTION_TRY
		}
	case srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_FLUSHING:
		dstAction = UPGRADE_ACTION_CANCEL
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_UNINIT:
		srcAction = UPGRADE_ACTION_CANCEL
	case srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_FLUSHCOMPLETE:
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtxFinalized, dst, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already completed", err)
			return nil, err
		} else if complete {
			dstAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else {
			dstAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_UNINIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtxFinalized, src, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already completed", err)
			return nil, err
		} else if complete {
			srcAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		} else {
			srcAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_INIT && dstState == UPGRADE_STATE_INIT: // crossing hellos
		// it is intentional to execute chanUpgradeTry on both sides if upgrade sequences
		// are identical to each other. this is for testing purpose.
		if srcChan.Channel.UpgradeSequence >= dstChan.Channel.UpgradeSequence {
			dstAction = UPGRADE_ACTION_TRY
		}
		if srcChan.Channel.UpgradeSequence <= dstChan.Channel.UpgradeSequence {
			srcAction = UPGRADE_ACTION_TRY
		}
	case srcState == UPGRADE_STATE_INIT && dstState == UPGRADE_STATE_FLUSHING:
		if srcChan.Channel.UpgradeSequence != dstChan.Channel.UpgradeSequence {
			dstAction = UPGRADE_ACTION_CANCEL
		} else {
			// chanUpgradeAck checks if counterparty-specified timeout has exceeded.
			// if it has, chanUpgradeAck aborts the upgrade handshake.
			// Therefore the relayer need not check timeout by itself.
			srcAction = UPGRADE_ACTION_ACK
		}
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_INIT:
		if srcChan.Channel.UpgradeSequence != dstChan.Channel.UpgradeSequence {
			srcAction = UPGRADE_ACTION_CANCEL
		} else {
			// chanUpgradeAck checks if counterparty-specified timeout has exceeded.
			// if it has, chanUpgradeAck aborts the upgrade handshake.
			// Therefore the relayer need not check timeout by itself.
			dstAction = UPGRADE_ACTION_ACK
		}
	case srcState == UPGRADE_STATE_INIT && dstState == UPGRADE_STATE_FLUSHCOMPLETE:
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtxFinalized, dst, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already completed", err)
			return nil, err
		} else if complete {
			dstAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else {
			dstAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_INIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtxFinalized, src, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already completed", err)
			return nil, err
		} else if complete {
			srcAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		} else {
			srcAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_FLUSHING:
		if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		}
		if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		}

		// if either chain has already timed out, never execute chanUpgradeConfirm
		if srcAction == UPGRADE_ACTION_TIMEOUT || dstAction == UPGRADE_ACTION_TIMEOUT {
			break
		}

		if completable, err := src.QueryCanTransitionToFlushComplete(srcCtxFinalized); err != nil {
			logger.Error("failed to check if the src channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			srcAction = UPGRADE_ACTION_CONFIRM
		}
		if completable, err := dst.QueryCanTransitionToFlushComplete(dstCtxFinalized); err != nil {
			logger.Error("failed to check if the dst channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			dstAction = UPGRADE_ACTION_CONFIRM
		}
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_FLUSHCOMPLETE:
		if timedout, err := upgradeAlreadyTimedOut(srcCtxFinalized, src, dstChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else if completable, err := src.QueryCanTransitionToFlushComplete(srcCtxFinalized); err != nil {
			logger.Error("failed to check if the src channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			srcAction = UPGRADE_ACTION_CONFIRM
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_FLUSHING:
		if timedout, err := upgradeAlreadyTimedOut(dstCtxFinalized, dst, srcChanUpg); err != nil {
			logger.Error("failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		} else if completable, err := dst.QueryCanTransitionToFlushComplete(dstCtxFinalized); err != nil {
			logger.Error("failed to check if the dst channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			dstAction = UPGRADE_ACTION_CONFIRM
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_FLUSHCOMPLETE:
		srcAction = UPGRADE_ACTION_OPEN
		dstAction = UPGRADE_ACTION_OPEN
	default:
		return nil, errors.New("unexpected state")
	}

	logger = &log.RelayLogger{Logger: logger.With(
		slog.Group("next_channel_upgrade_actions",
			"src", srcAction.String(),
			"dst", dstAction.String(),
		),
	)}

	if srcAction != UPGRADE_ACTION_NONE {
		addr := mustGetAddress(src)

		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}

		msg, err := buildActionMsg(
			src,
			srcAction,
			srcChan,
			addr,
			dstCtxFinalized,
			dst,
			dstChan,
			dstChanUpg,
		)
		if err != nil {
			logger.Error("failed to build Msg for the src chain", err)
			return nil, err
		}

		out.Src = append(out.Src, msg)
	}

	if dstAction != UPGRADE_ACTION_NONE {
		addr := mustGetAddress(dst)

		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}

		msg, err := buildActionMsg(
			dst,
			dstAction,
			dstChan,
			addr,
			srcCtxFinalized,
			src,
			srcChan,
			srcChanUpg,
		)
		if err != nil {
			logger.Error("failed to build Msg for the dst chain", err)
			return nil, err
		}

		out.Dst = append(out.Dst, msg)
	}

	logger.Info("successfully generates the next step of the channel upgrade")
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

// buildActionMsg builds and returns a MsgChannelUpgradeXXX message corresponding to `action`.
// This function also returns `UpgradeState` to which the channel will transition after the message is processed.
func buildActionMsg(
	chain *ProvableChain,
	action UpgradeAction,
	selfChan *chantypes.QueryChannelResponse,
	addr sdk.AccAddress,
	cpCtx QueryContext,
	cp *ProvableChain,
	cpChan *chantypes.QueryChannelResponse,
	cpUpg *chantypes.QueryUpgradeResponse,
) (sdk.Msg, error) {
	pathEnd := chain.Path()

	switch action {
	case UPGRADE_ACTION_TRY:
		proposedConnectionID, err := queryProposedConnectionID(cpCtx, cp, cpUpg)
		if err != nil {
			return nil, err
		}
		return pathEnd.ChanUpgradeTry(proposedConnectionID, cpChan, cpUpg, addr), nil
	case UPGRADE_ACTION_ACK:
		return pathEnd.ChanUpgradeAck(cpChan, cpUpg, addr), nil
	case UPGRADE_ACTION_CONFIRM:
		return pathEnd.ChanUpgradeConfirm(cpChan, cpUpg, addr), nil
	case UPGRADE_ACTION_OPEN:
		return pathEnd.ChanUpgradeOpen(cpChan, addr), nil
	case UPGRADE_ACTION_CANCEL:
		upgErr, err := QueryChannelUpgradeError(cpCtx, cp, true)
		if err != nil {
			return nil, err
		} else if upgErr == nil {
			// NOTE: Even if an error receipt is not found, anyway try to execute ChanUpgradeCancel.
			// If the sender is authority and the channel state is anything other than FLUSHCOMPLETE,
			// the cancellation will be successful.
			upgErr = &chantypes.QueryUpgradeErrorResponse{}
		}
		return pathEnd.ChanUpgradeCancel(upgErr, addr), nil
	case UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE:
		upgErr, err := QueryChannelUpgradeError(cpCtx, cp, true)
		if err != nil {
			return nil, err
		} else if upgErr == nil {
			return nil, fmt.Errorf("upgrade error receipt not found")
		} else if upgErr.ErrorReceipt.Sequence != selfChan.Channel.UpgradeSequence {
			return nil, fmt.Errorf(
				"upgrade sequences don't match: channel.upgrade_sequence=%d, error_receipt.sequence=%d",
				selfChan.Channel.UpgradeSequence, upgErr.ErrorReceipt.Sequence)
		}
		return pathEnd.ChanUpgradeCancel(upgErr, addr), nil
	case UPGRADE_ACTION_TIMEOUT:
		return pathEnd.ChanUpgradeTimeout(cpChan, addr), nil
	default:
		panic(fmt.Errorf("unexpected action: %s", action))
	}
}
