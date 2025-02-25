package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	retry "github.com/avast/retry-go"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/hyperledger-labs/yui-relayer/log"
)

// CreateChannel runs the channel creation messages on timeout until they pass
// TODO: add max retries or something to this function
func CreateChannel(pathName string, src, dst *ProvableChain, to time.Duration) error {
	logger := GetChannelPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "CreateChannel")

	if cont, err := checkChannelCreateReady(src, dst, logger); err != nil {
		return err
	} else if !cont {
		return nil
	}

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

		chanSteps.Send(context.TODO(), src, dst)

		if chanSteps.Success() {
			// In the case of success, synchronize the config file from generated channel identifiers
			if err := SyncChainConfigsFromEvents(pathName, chanSteps.SrcMsgIDs, chanSteps.DstMsgIDs, src, dst); err != nil {
				return err
			}

			// In the case of success and this being the last transaction
			// debug logging, log created connection and break
			if chanSteps.Last {
				logger.Info("★ Channel created")
				return nil
			}

			// In the case of success, reset the failures counter
			failures = 0
		} else {
			// In the case of failure, increment the failures counter and exit if this is the 3rd failure
			if failures++; failures > 2 {
				err := errors.New("Channel handshake failed")
				logger.Error(err.Error(), err)
				return err
			}

			logger.Warn("Retrying transaction...")
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

func checkChannelCreateReady(src, dst *ProvableChain, logger *log.RelayLogger) (bool, error) {
	srcID := src.Chain.Path().ChannelID
	dstID := dst.Chain.Path().ChannelID

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (chantypes.State, error) {
		if pc.Chain.Path().ChannelID == "" {
			return chantypes.UNINITIALIZED, nil
		}

		latestHeight, err := pc.LatestHeight(context.TODO())
		if err != nil {
			return chantypes.UNINITIALIZED, err
		}
		res, err2 := pc.QueryChannel(NewQueryContext(context.TODO(), latestHeight))
		if err2 != nil {
			return chantypes.UNINITIALIZED, err2
		}
		return res.Channel.State, nil
	}

	srcState, srcErr := getState(src)
	if srcErr != nil {
		return false, srcErr
	}

	dstState, dstErr := getState(dst)
	if dstErr != nil {
		return false, dstErr
	}

	if srcID != "" && srcState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("src channel id is given but that channel does not exist: %s", srcID)
	}
	if dstID != "" && dstState == chantypes.UNINITIALIZED {
		return false, fmt.Errorf("dst channel id is given but that channel does not exist: %s", dstID)
	}

	if srcState == chantypes.OPEN && dstState == chantypes.OPEN {
		logger.Warn("channels are already created", "src_channel_id", srcID, "dst_channel_id", dstID)
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
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(context.TODO(), src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(context.TODO(), src, dst); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	srcChan, dstChan, settled, err := querySettledChannelPair(
		sh.GetQueryContext(context.TODO(), src.ChainID()),
		sh.GetQueryContext(context.TODO(), dst.ChainID()),
		src,
		dst,
		true,
	)
	if err != nil {
		return nil, err
	} else if !settled {
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

func querySettledChannelPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*chantypes.QueryChannelResponse, *chantypes.QueryChannelResponse, bool, error) {
	logger := GetChannelPairLogger(src, dst)
	logger = &log.RelayLogger{Logger: logger.With(
		"src_height", srcCtx.Height().String(),
		"dst_height", dstCtx.Height().String(),
		"prove", prove,
	)}

	srcChan, dstChan, err := QueryChannelPair(srcCtx, dstCtx, src, dst, prove)
	if err != nil {
		logger.Error("failed to query channel pair at the latest finalized height", err)
		return nil, nil, false, err
	}

	var srcLatestCtx, dstLatestCtx QueryContext
	if h, err := src.LatestHeight(context.TODO()); err != nil {
		logger.Error("failed to get the latest height of the src chain", err)
		return nil, nil, false, err
	} else {
		srcLatestCtx = NewQueryContext(context.TODO(), h)
	}
	if h, err := dst.LatestHeight(context.TODO()); err != nil {
		logger.Error("failed to get the latest height of the dst chain", err)
		return nil, nil, false, err
	} else {
		dstLatestCtx = NewQueryContext(context.TODO(), h)
	}

	srcLatestChan, dstLatestChan, err := QueryChannelPair(srcLatestCtx, dstLatestCtx, src, dst, false)
	if err != nil {
		logger.Error("failed to query channel pair at the latest height", err)
		return nil, nil, false, err
	}

	if srcChan.Channel.String() != srcLatestChan.Channel.String() {
		logger.Debug("src channel end in transition",
			"from", srcChan.Channel.String(),
			"to", srcLatestChan.Channel.String(),
		)
		return srcChan, dstChan, false, nil
	}
	if dstChan.Channel.String() != dstLatestChan.Channel.String() {
		logger.Debug("dst channel end in transition",
			"from", dstChan.Channel.String(),
			"to", dstLatestChan.Channel.String(),
		)
		return srcChan, dstChan, false, nil
	}
	return srcChan, dstChan, true, nil
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
