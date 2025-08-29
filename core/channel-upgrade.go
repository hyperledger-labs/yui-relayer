package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/log"
	"go.opentelemetry.io/otel/codes"
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
func InitChannelUpgrade(ctx context.Context, chain, cp *ProvableChain, upgradeFields chantypes.UpgradeFields, permitUnsafe bool) error {
	ctx, span := tracer.Start(ctx, "InitChannelUpgrade", WithChannelAttributes(chain.Chain))
	defer span.End()
	logger := GetChannelLogger(chain.Chain)
	defer logger.TimeTrackContext(ctx, time.Now(), "InitChannelUpgrade")

	if h, err := chain.LatestHeight(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to get the latest height", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if cpH, err := cp.LatestHeight(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to get the latest height of the counterparty chain", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if chann, cpChann, err := QueryChannelPair(
		NewQueryContext(ctx, h),
		NewQueryContext(ctx, cpH),
		chain,
		cp,
		false,
	); err != nil {
		logger.ErrorContext(ctx, "failed to query for the channel pair", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if chann.Channel.State != chantypes.OPEN || cpChann.Channel.State != chantypes.OPEN {
		logger = &log.RelayLogger{Logger: logger.With(
			"channel_state", chann.Channel.State,
			"cp_channel_state", cpChann.Channel.State,
		)}

		if permitUnsafe {
			logger.InfoContext(ctx, "unsafe channel upgrade is permitted")
		} else {
			err := errors.New("unsafe channel upgrade initialization")
			logger.ErrorContext(ctx, "unsafe channel upgrade is not permitted", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	addr, err := chain.GetAddress()
	if err != nil {
		logger.ErrorContext(ctx, "failed to get address", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	msg := chain.Path().ChanUpgradeInit(upgradeFields, addr)

	if _, err := chain.SendMsgs(ctx, []sdk.Msg{msg}); err != nil {
		logger.ErrorContext(ctx, "failed to send MsgChannelUpgradeInit", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	} else {
		logger.InfoContext(ctx, "successfully initialized channel upgrade")
	}

	return nil
}

// ExecuteChannelUpgrade carries out channel upgrade handshake until both chains transition to the OPEN state.
// This function repeatedly checks the states of both chains and decides the next action.
func ExecuteChannelUpgrade(ctx context.Context, pathName string, src, dst *ProvableChain, interval time.Duration, targetSrcState, targetDstState UpgradeState) error {
	ctx, span := tracer.Start(ctx, "ExecuteChannelUpgrade", WithChannelPairAttributes(src, dst))
	defer span.End()
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "ExecuteChannelUpgrade")

	if (targetSrcState == UPGRADE_STATE_UNINIT && targetDstState == UPGRADE_STATE_INIT) ||
		(targetSrcState == UPGRADE_STATE_INIT && targetDstState == UPGRADE_STATE_UNINIT) {
		return fmt.Errorf("unreachable target state pair: (%s, %s)", targetSrcState, targetDstState)
	}

	failures := 0
	firstCall := true
	err := runUntilComplete(ctx, interval, func() (bool, error) {
		steps, err := upgradeChannelStep(ctx, src, dst, targetSrcState, targetDstState, firstCall)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create channel upgrade step", err)
			return false, err
		}

		firstCall = false

		if steps.Last {
			logger.InfoContext(ctx, "Channel upgrade completed")
			return true, nil
		}

		if !steps.Ready() {
			logger.DebugContext(ctx, "Waiting for next channel upgrade step ...")
			return false, nil
		}

		steps.Send(ctx, src, dst)

		if steps.Success() {
			if err := SyncChainConfigsFromEvents(ctx, pathName, steps.SrcMsgIDs, steps.DstMsgIDs, src, dst); err != nil {
				logger.ErrorContext(ctx, "failed to synchronize the updated path config to the config file", err)
				return false, err
			}

			failures = 0
		} else {
			if failures++; failures > 2 {
				err := errors.New("channel upgrade failed")
				logger.ErrorContext(ctx, err.Error(), err)
				return false, err
			}

			logger.WarnContext(ctx, "Retrying transaction...")
		}

		return false, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// CancelChannelUpgrade executes chanUpgradeCancel on `chain`.
func CancelChannelUpgrade(ctx context.Context, chain, cp *ProvableChain, settlementInterval time.Duration) error {
	ctx, span := tracer.Start(ctx, "CancelChannelUpgrade", WithChannelPairAttributes(chain, cp))
	defer span.End()
	logger := GetChannelPairLogger(chain, cp)
	defer logger.TimeTrackContext(ctx, time.Now(), "CancelChannelUpgrade")

	// wait for settlement
	err := runUntilComplete(ctx, settlementInterval, func() (bool, error) {
		sh, err := NewSyncHeaders(ctx, chain, cp)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create a SyncHeaders", err)
			return false, err
		}
		queryCtx := sh.GetQueryContext(ctx, chain.ChainID())
		cpQueryCtx := sh.GetQueryContext(ctx, cp.ChainID())

		chann, _, settled, err := querySettledChannelPair(queryCtx, cpQueryCtx, chain, cp, false)
		if err != nil {
			logger.ErrorContext(ctx, "failed to query for settled channel pair", err)
			return false, err
		} else if !settled {
			logger.InfoContext(ctx, "waiting for settlement of channel pair ...")
			return false, nil
		}

		if _, _, settled, err := querySettledChannelUpgradePair(queryCtx, cpQueryCtx, chain, cp, false); err != nil {
			logger.ErrorContext(ctx, "failed to query for settled channel upgrade pair", err)
			return false, err
		} else if !settled {
			logger.InfoContext(ctx, "waiting for settlement of channel upgrade pair")
			return false, nil
		}

		cpHeaders, err := SetupHeadersForUpdateSync(cp, ctx, chain, sh.GetLatestFinalizedHeader(cp.ChainID()))
		if err != nil {
			logger.ErrorContext(ctx, "failed to set up headers for LC update", err)
			return false, err
		}

		upgErr, err := QueryChannelUpgradeError(cpQueryCtx, cp, true)
		if err != nil {
			logger.ErrorContext(ctx, "failed to query the channel upgrade error receipt", err)
			return false, err
		} else if chann.Channel.State == chantypes.FLUSHCOMPLETE &&
			(upgErr == nil || upgErr.ErrorReceipt.Sequence != chann.Channel.UpgradeSequence) {
			var err error
			if upgErr == nil {
				err = fmt.Errorf("upgrade error receipt not found")
			} else {
				err = fmt.Errorf("upgrade sequences don't match: channel.upgrade_sequence=%d, error_receipt.sequence=%d",
					chann.Channel.UpgradeSequence, upgErr.ErrorReceipt.Sequence)
			}
			logger.ErrorContext(ctx, "cannot cancel the upgrade in FLUSHCOMPLETE state", err)
			return false, err
		} else if upgErr == nil {
			// NOTE: Even if an error receipt is not found, anyway try to execute ChanUpgradeCancel.
			// If the sender is authority and the channel state is anything other than FLUSHCOMPLETE,
			// the cancellation will be successful.
			upgErr = &chantypes.QueryUpgradeErrorResponse{}
		}

		addr, err := chain.GetAddress()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get address", err)
			return false, err
		}

		var msgs []sdk.Msg
		msgs = append(msgs, chain.Path().UpdateClients(cpHeaders, addr)...)
		msgs = append(msgs, chain.Path().ChanUpgradeCancel(upgErr, addr))

		// NOTE: A call of SendMsgs for each msg is executed separately to avoid using multicall for eth.
		//       This is just a workaround and should be fixed in the future.
		for _, msg := range msgs {
			if _, err := chain.SendMsgs(ctx, []sdk.Msg{msg}); err != nil {
				logger.ErrorContext(ctx, "failed to send a msg to cancel the channel upgrade", err)
				return false, err
			}
		}
		logger.InfoContext(ctx, "successfully cancelled the channel upgrade")

		return true, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
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

func upgradeChannelStep(ctx context.Context, src, dst *ProvableChain, targetSrcState, targetDstState UpgradeState, firstCall bool) (*RelayMsgs, error) {
	logger := GetChannelPairLogger(src, dst)
	logger = &log.RelayLogger{Logger: logger.With("first_call", firstCall)}

	if err := validatePaths(src, dst); err != nil {
		logger.ErrorContext(ctx, "failed to validate paths", err)
		return nil, err
	}

	out := NewRelayMsgs()

	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create SyncHeaders", err)
		return nil, err
	}

	// Query a number of things all at once
	var srcUpdateHeaders, dstUpdateHeaders []Header
	if err := retry.Do(func() error {
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(ctx, src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(uint, error) {
		if err := sh.Updates(ctx, src, dst); err != nil {
			panic(err)
		}
	})); err != nil {
		logger.ErrorContext(ctx, "failed to set up headers for LC update on both chains", err)
		return nil, err
	}

	srcCtx := sh.GetQueryContext(ctx, src.ChainID())
	dstCtx := sh.GetQueryContext(ctx, dst.ChainID())

	// query finalized channels with proofs
	srcChan, dstChan, settled, err := querySettledChannelPair(srcCtx, dstCtx, src, dst, true)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query the channel pair with proofs", err)
		return nil, err
	} else if !settled {
		return out, nil
	}

	// query finalized channel upgrades with proofs
	srcChanUpg, dstChanUpg, settled, err := querySettledChannelUpgradePair(
		srcCtx,
		dstCtx,
		src,
		dst,
		true,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to query the channel upgrade pair with proofs", err)
		return nil, err
	} else if !settled {
		return out, nil
	}

	// determine upgrade states
	srcState, err := NewUpgradeState(srcChan.Channel.State, srcChanUpg != nil)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create UpgradeState of the src chain", err)
		return nil, err
	}
	dstState, err := NewUpgradeState(dstChan.Channel.State, dstChanUpg != nil)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create UpgradeState of the dst chain", err)
		return nil, err
	}

	logger = &log.RelayLogger{Logger: logger.With(
		slog.Group("current_channel_upgrade_states",
			"src", srcState.String(),
			"dst", dstState.String(),
		),
	)}

	if hasReachedOrPassedTargetState(srcState, targetSrcState, dstState) && hasReachedOrPassedTargetState(dstState, targetDstState, srcState) {
		if firstCall {
			if srcState == UPGRADE_STATE_UNINIT && targetSrcState == UPGRADE_STATE_UNINIT {
				logger.InfoContext(ctx, "both chains have already reached or passed the target states, or the channel upgrade has not been initialized")
			} else {
				logger.InfoContext(ctx, "both chains have already reached or passed the target states")
			}
		} else {
			logger.InfoContext(ctx, "both chains have reached or passed the target states")
		}
		out.Last = true
		return out, nil
	}

	// determine next actions for src/dst chains
	srcAction := UPGRADE_ACTION_NONE
	dstAction := UPGRADE_ACTION_NONE
	switch {
	case srcState == UPGRADE_STATE_UNINIT && dstState == UPGRADE_STATE_UNINIT:
		// This line should never be reached because the channel upgrade is considered completed
		return nil, errors.New("unexpected transition")
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
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtx, dst, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already completed", err)
			return nil, err
		} else if complete {
			dstAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtx, src, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else {
			dstAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_UNINIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtx, src, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already completed", err)
			return nil, err
		} else if complete {
			srcAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtx, dst, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already timed out", err)
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
		if complete, err := upgradeAlreadyComplete(srcChan, dstCtx, dst, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already completed", err)
			return nil, err
		} else if complete {
			dstAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(srcCtx, src, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else {
			dstAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_INIT:
		if complete, err := upgradeAlreadyComplete(dstChan, srcCtx, src, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already completed", err)
			return nil, err
		} else if complete {
			srcAction = UPGRADE_ACTION_OPEN
		} else if timedout, err := upgradeAlreadyTimedOut(dstCtx, dst, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		} else {
			srcAction = UPGRADE_ACTION_CANCEL_FLUSHCOMPLETE
		}
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_FLUSHING:
		if timedout, err := upgradeAlreadyTimedOut(srcCtx, src, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		}
		if timedout, err := upgradeAlreadyTimedOut(dstCtx, dst, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		}

		// if either chain has already timed out, never execute chanUpgradeConfirm
		if srcAction == UPGRADE_ACTION_TIMEOUT || dstAction == UPGRADE_ACTION_TIMEOUT {
			break
		}

		if completable, err := queryCanTransitionToFlushComplete(srcCtx.Context(), src); err != nil {
			logger.ErrorContext(ctx, "failed to check if the src channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			srcAction = UPGRADE_ACTION_CONFIRM
		}
		if completable, err := queryCanTransitionToFlushComplete(dstCtx.Context(), dst); err != nil {
			logger.ErrorContext(ctx, "failed to check if the dst channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			dstAction = UPGRADE_ACTION_CONFIRM
		}
	case srcState == UPGRADE_STATE_FLUSHING && dstState == UPGRADE_STATE_FLUSHCOMPLETE:
		if timedout, err := upgradeAlreadyTimedOut(srcCtx, src, dstChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the src side has already timed out", err)
			return nil, err
		} else if timedout {
			dstAction = UPGRADE_ACTION_TIMEOUT
		} else if completable, err := queryCanTransitionToFlushComplete(srcCtx.Context(), src); err != nil {
			logger.ErrorContext(ctx, "failed to check if the src channel can transition to FLUSHCOMPLETE", err)
			return nil, err
		} else if completable {
			srcAction = UPGRADE_ACTION_CONFIRM
		}
	case srcState == UPGRADE_STATE_FLUSHCOMPLETE && dstState == UPGRADE_STATE_FLUSHING:
		if timedout, err := upgradeAlreadyTimedOut(dstCtx, dst, srcChanUpg); err != nil {
			logger.ErrorContext(ctx, "failed to check if the upgrade on the dst side has already timed out", err)
			return nil, err
		} else if timedout {
			srcAction = UPGRADE_ACTION_TIMEOUT
		} else if completable, err := queryCanTransitionToFlushComplete(dstCtx.Context(), dst); err != nil {
			logger.ErrorContext(ctx, "failed to check if the dst channel can transition to FLUSHCOMPLETE", err)
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
			dstCtx,
			dst,
			dstChan,
			dstChanUpg,
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to build Msg for the src chain", err)
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
			srcCtx,
			src,
			srcChan,
			srcChanUpg,
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to build Msg for the dst chain", err)
			return nil, err
		}

		out.Dst = append(out.Dst, msg)
	}

	logger.InfoContext(ctx, "successfully generates the next step of the channel upgrade")
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

func queryCanTransitionToFlushComplete(ctx context.Context, chain interface {
	ChainInfo
	ICS04Querier
}) (bool, error) {
	if h, err := chain.LatestHeight(ctx); err != nil {
		return false, err
	} else {
		return chain.QueryCanTransitionToFlushComplete(NewQueryContext(ctx, h))
	}
}

func querySettledChannelUpgradePair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*chantypes.QueryUpgradeResponse, *chantypes.QueryUpgradeResponse, bool, error) {
	logger := GetChannelPairLogger(src, dst)
	logger = &log.RelayLogger{Logger: logger.With(
		"src_height", srcCtx.Height().String(),
		"dst_height", dstCtx.Height().String(),
		"prove", prove,
	)}

	// query channel upgrade pair at latest finalized heights
	srcChanUpg, dstChanUpg, err := QueryChannelUpgradePair(srcCtx, dstCtx, src, dst, prove)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query a channel upgrade pair at the latest finalized heights", err)
		return nil, nil, false, err
	}

	// prepare QueryContext's based on the latest heights
	var srcLatestCtx, dstLatestCtx QueryContext
	if h, err := src.LatestHeight(srcCtx.Context()); err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to get the latest height of the src chain", err)
		return nil, nil, false, err
	} else {
		srcLatestCtx = NewQueryContext(srcCtx.Context(), h)
	}
	if h, err := dst.LatestHeight(dstCtx.Context()); err != nil {
		logger.ErrorContext(dstCtx.Context(), "failed to get the latest height of the dst chain", err)
		return nil, nil, false, err
	} else {
		dstLatestCtx = NewQueryContext(dstCtx.Context(), h)
	}

	// query channel upgrade pair at latest heights
	srcLatestChanUpg, dstLatestChanUpg, err := QueryChannelUpgradePair(srcLatestCtx, dstLatestCtx, src, dst, false)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query a channel upgrade pair at the latest heights", err)
		return nil, nil, false, err
	}

	if !compareUpgrades(srcChanUpg, srcLatestChanUpg) {
		logger.DebugContext(srcCtx.Context(), "src channel upgrade in transition")
		return srcChanUpg, dstChanUpg, false, nil
	}
	if !compareUpgrades(dstChanUpg, dstLatestChanUpg) {
		logger.DebugContext(dstCtx.Context(), "dst channel upgrade in transition")
		return srcChanUpg, dstChanUpg, false, nil
	}

	return srcChanUpg, dstChanUpg, true, nil
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
	timestamp, err := chain.Timestamp(ctx.Context(), height)
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

// hasReachedOrPassedTargetState checks if the current state has reached or passed the target state,
// including cases where the target state is skipped.
func hasReachedOrPassedTargetState(currentState, targetState, counterpartyCurrentState UpgradeState) bool {
	return currentState == targetState || hasPassedTargetState(currentState, targetState, counterpartyCurrentState)
}

// hasPassedTargetState checks if the current state has passed the target state,
// including cases where the target state is skipped.
func hasPassedTargetState(currentState, targetState, counterpartyCurrentState UpgradeState) bool {
	// Check if the current state has passed the target state.
	// For simplicity, we consider that any state that can be reached after the target state has passed the target state.
	// NOTE: Each chain can cancel the upgrade and initialize another upgrade at any time. This means that the state
	// can transition to UNINIT from any state except the case where the counterparty is in INIT state.

	isUninitReachable := counterpartyCurrentState != UPGRADE_STATE_INIT
	switch targetState {
	case UPGRADE_STATE_INIT:
		return currentState == UPGRADE_STATE_FLUSHING || currentState == UPGRADE_STATE_FLUSHCOMPLETE ||
			(isUninitReachable && currentState == UPGRADE_STATE_UNINIT)
	case UPGRADE_STATE_FLUSHING:
		return currentState == UPGRADE_STATE_FLUSHCOMPLETE || (isUninitReachable && currentState == UPGRADE_STATE_UNINIT)
	case UPGRADE_STATE_FLUSHCOMPLETE:
		return isUninitReachable && currentState == UPGRADE_STATE_UNINIT
	default:
		return false
	}
}
