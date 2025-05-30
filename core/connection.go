package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	retry "github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	rtyAttNum = uint(5)
	rtyAtt    = retry.Attempts(rtyAttNum)
	rtyDel    = retry.Delay(time.Millisecond * 400)
	rtyErr    = retry.LastErrorOnly(true)
)

// CreateConnection sends connection creation messages every interval until a connection is created
func CreateConnection(ctx context.Context, pathName string, src, dst *ProvableChain, interval time.Duration) error {
	ctx, span := tracer.Start(ctx, "CreateConnection", WithConnectionPairAttributes(src, dst))
	defer span.End()
	logger := GetConnectionPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "CreateConnection")

	if cont, err := checkConnectionCreateReady(ctx, src, dst, logger); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if !cont {
		return nil
	}

	failed := 0
	err := runUntilComplete(ctx, interval, func() (bool, error) {
		connSteps, err := createConnectionStep(ctx, src, dst)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create connection step", err)
			return false, err
		}

		if !connSteps.Ready() {
			logger.DebugContext(ctx, "Waiting for next connection step ...")
			return false, nil
		}

		connSteps.Send(ctx, src, dst)

		if connSteps.Success() {
			// In the case of success, synchronize the config file from generated connection identifiers.
			if err := SyncChainConfigsFromEvents(ctx, pathName, connSteps.SrcMsgIDs, connSteps.DstMsgIDs, src, dst); err != nil {
				return false, err
			}

			// In the case of success and this being the last transaction
			// debug logging, log created connection and break
			if connSteps.Last {
				logger.InfoContext(ctx, "★ Connection created")
				return true, nil
			}

			// In the case of success, reset the failures counter
			failed = 0
		} else {
			// In the case of failure, increment the failures counter and exit if this is the 3rd failure
			if failed++; failed > 2 {
				err := errors.New("Connection handshake failed")
				logger.ErrorContext(ctx, err.Error(), err)
				return false, err
			}

			logger.WarnContext(ctx, "Retrying transaction...")
			if err := wait(ctx, 5*time.Second); err != nil {
				return false, err
			}
		}

		return false, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func checkConnectionCreateReady(ctx context.Context, src, dst *ProvableChain, logger *log.RelayLogger) (bool, error) {
	srcID := src.Chain.Path().ConnectionID
	dstID := dst.Chain.Path().ConnectionID

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (conntypes.State, error) {
		if pc.Chain.Path().ConnectionID == "" {
			return conntypes.UNINITIALIZED, nil
		}

		latestHeight, err := pc.LatestHeight(ctx)
		if err != nil {
			return conntypes.UNINITIALIZED, err
		}
		res, err2 := pc.QueryConnection(NewQueryContext(ctx, latestHeight), pc.Path().ConnectionID)
		if err2 != nil {
			return conntypes.UNINITIALIZED, err2
		}
		return res.Connection.State, nil
	}

	srcState, srcErr := getState(src)
	if srcErr != nil {
		return false, srcErr
	}

	dstState, dstErr := getState(dst)
	if dstErr != nil {
		return false, dstErr
	}

	if srcID != "" && srcState == conntypes.UNINITIALIZED {
		return false, fmt.Errorf("src connection id is given but that connection does not exist: %s", srcID)
	}
	if dstID != "" && dstState == conntypes.UNINITIALIZED {
		return false, fmt.Errorf("dst connection id is given but that connection does not exist: %s", dstID)
	}

	if srcState == conntypes.OPEN && dstState == conntypes.OPEN {
		logger.WarnContext(ctx, "connections are already created", "src_connection_id", srcID, "dst_connection_id", dstID)
		return false, nil
	}
	return true, nil
}

func createConnectionStep(ctx context.Context, src, dst *ProvableChain) (*RelayMsgs, error) {
	out := NewRelayMsgs()
	if err := validatePaths(src, dst); err != nil {
		return nil, err
	}
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		return nil, err
	}
	// Query a number of things all at once
	var (
		srcUpdateHeaders, dstUpdateHeaders []Header
		srcCsRes, dstCsRes                 *clienttypes.QueryClientStateResponse
		srcCS, dstCS                       ibcexported.ClientState
		srcConsRes, dstConsRes             *clienttypes.QueryConsensusStateResponse
		srcCons, dstCons                   ibcexported.ConsensusState
		srcConsH, dstConsH                 ibcexported.Height
		srcHostConsProof, dstHostConsProof []byte
	)
	err = retry.Do(func() error {
		srcUpdateHeaders, dstUpdateHeaders, err = sh.SetupBothHeadersForUpdate(ctx, src, dst)
		return err
	}, rtyAtt, rtyDel, rtyErr, retry.Context(ctx), retry.OnRetry(func(n uint, err error) {
		// logRetryUpdateHeaders(src, dst, n, err)
		if err := sh.Updates(ctx, src, dst); err != nil {
			panic(err)
		}
	}))
	if err != nil {
		return nil, err
	}

	srcConn, dstConn, settled, err := querySettledConnectionPair(
		sh.GetQueryContext(ctx, src.ChainID()),
		sh.GetQueryContext(ctx, dst.ChainID()),
		src,
		dst,
		true,
	)
	if err != nil {
		return nil, err
	} else if !settled {
		return out, nil
	}

	if !(srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.UNINITIALIZED) {
		// Query client state from each chain's client
		srcCsRes, dstCsRes, err = QueryClientStatePair(sh.GetQueryContext(ctx, src.ChainID()), sh.GetQueryContext(ctx, dst.ChainID()), src, dst, true)
		if err != nil {
			return nil, err
		}
		if err := src.Codec().UnpackAny(srcCsRes.ClientState, &srcCS); err != nil {
			return nil, err
		}
		if err := dst.Codec().UnpackAny(dstCsRes.ClientState, &dstCS); err != nil {
			return nil, err
		}

		// Store the heights
		srcConsH, dstConsH = srcCS.GetLatestHeight(), dstCS.GetLatestHeight()
		srcConsRes, dstConsRes, err = QueryClientConsensusStatePair(
			sh.GetQueryContext(ctx, src.ChainID()), sh.GetQueryContext(ctx, dst.ChainID()),
			src, dst, srcConsH, dstConsH, true)
		if err != nil {
			return nil, err
		}
		if err := src.Codec().UnpackAny(srcConsRes.ConsensusState, &srcCons); err != nil {
			return nil, err
		}
		if err := dst.Codec().UnpackAny(dstConsRes.ConsensusState, &dstCons); err != nil {
			return nil, err
		}
		srcHostConsProof, err = src.ProveHostConsensusState(sh.GetQueryContext(ctx, src.ChainID()), dstCS.GetLatestHeight(), dstCons)
		if err != nil {
			return nil, err
		}
		dstHostConsProof, err = dst.ProveHostConsensusState(sh.GetQueryContext(ctx, dst.ChainID()), srcCS.GetLatestHeight(), srcCons)
		if err != nil {
			return nil, err
		}
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `connOpenInit` to src
	case srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(ctx, src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnInit(dst.Path(), addr))
		// Handshake has started on dst (1 step done), relay `connOpenTry` and `updateClient` on src
	case srcConn.Connection.State == conntypes.UNINITIALIZED && dstConn.Connection.State == conntypes.INIT:
		logConnectionStates(ctx, src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnTry(dst.Path(), dstCsRes, dstConn, dstConsRes, srcHostConsProof, addr))
	// Handshake has started on src (1 step done), relay `connOpenTry` and `updateClient` on dst
	case srcConn.Connection.State == conntypes.INIT && dstConn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(ctx, dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnTry(src.Path(), srcCsRes, srcConn, srcConsRes, dstHostConsProof, addr))

	// Handshake has started on src end (2 steps done), relay `connOpenAck` and `updateClient` to dst end
	case srcConn.Connection.State == conntypes.TRYOPEN && dstConn.Connection.State == conntypes.INIT:
		logConnectionStates(ctx, dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnAck(src.Path(), srcCsRes, srcConn, srcConsRes, dstHostConsProof, addr))

	// Handshake has started on dst end (2 steps done), relay `connOpenAck` and `updateClient` to src end
	case srcConn.Connection.State == conntypes.INIT && dstConn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(ctx, src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnAck(dst.Path(), dstCsRes, dstConn, dstConsRes, srcHostConsProof, addr))

	// Handshake has confirmed on dst (3 steps done), relay `connOpenConfirm` and `updateClient` to src end
	case srcConn.Connection.State == conntypes.TRYOPEN && dstConn.Connection.State == conntypes.OPEN:
		logConnectionStates(ctx, src, dst, srcConn, dstConn)
		addr := mustGetAddress(src)
		if len(dstUpdateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstUpdateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnConfirm(dstConn, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `connOpenConfirm` and `updateClient` to dst end
	case srcConn.Connection.State == conntypes.OPEN && dstConn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(ctx, dst, src, dstConn, srcConn)
		addr := mustGetAddress(dst)
		if len(srcUpdateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcUpdateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnConfirm(srcConn, addr))
		out.Last = true

	default:
		panic(fmt.Sprintf("not implemented error: %v %v", srcConn.Connection.State, dstConn.Connection.State))
	}

	return out, nil
}

// validatePaths takes two chains and validates their paths
func validatePaths(src, dst Chain) error {
	if err := src.Path().Validate(); err != nil {
		return fmt.Errorf("path on chain %s failed to set: %w", src.ChainID(), err)
	}
	if err := dst.Path().Validate(); err != nil {
		return fmt.Errorf("path on chain %s failed to set: %w", dst.ChainID(), err)
	}
	return nil
}

func logConnectionStates(ctx context.Context, src, dst Chain, srcConn, dstConn *conntypes.QueryConnectionResponse) {
	logger := GetConnectionPairLogger(src, dst)
	logger.InfoContext(ctx,
		"connection states",
		slog.Group("src",
			slog.Uint64("proof_height", mustGetHeight(srcConn.ProofHeight)),
			slog.String("state", srcConn.Connection.State.String()),
		),
		slog.Group("dst",
			slog.Uint64("proof_height", mustGetHeight(dstConn.ProofHeight)),
			slog.String("state", dstConn.Connection.State.String()),
		))
}

// mustGetHeight takes the height inteface and returns the actual height
func mustGetHeight(h ibcexported.Height) uint64 {
	height, ok := h.(clienttypes.Height)
	if !ok {
		panic("height is not an instance of height! wtf")
	}
	return height.GetRevisionHeight()
}

func mustGetAddress(chain interface {
	GetAddress() (sdk.AccAddress, error)
}) sdk.AccAddress {
	addr, err := chain.GetAddress()
	if err != nil {
		panic(err)
	}
	return addr
}

func querySettledConnectionPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*conntypes.QueryConnectionResponse, *conntypes.QueryConnectionResponse, bool, error) {
	logger := GetConnectionPairLogger(src, dst)
	logger = &log.RelayLogger{Logger: logger.With(
		"src_height", srcCtx.Height().String(),
		"dst_height", dstCtx.Height().String(),
		"prove", prove,
	)}

	srcConn, dstConn, err := QueryConnectionPair(srcCtx, dstCtx, src, dst, prove)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query connection pair at the latest finalized height", err)
		return nil, nil, false, err
	}

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

	srcLatestConn, dstLatestConn, err := QueryConnectionPair(srcLatestCtx, dstLatestCtx, src, dst, false)
	if err != nil {
		logger.ErrorContext(srcCtx.Context(), "failed to query connection pair at the latest height", err)
		return nil, nil, false, err
	}

	if srcConn.Connection.String() != srcLatestConn.Connection.String() {
		logger.DebugContext(srcCtx.Context(), "src connection end in transition",
			"from", srcConn.Connection.String(),
			"to", srcLatestConn.Connection.String(),
		)
		return srcConn, dstConn, false, nil
	}
	if dstConn.Connection.String() != dstLatestConn.Connection.String() {
		logger.DebugContext(dstCtx.Context(), "dst connection end in transition",
			"from", dstConn.Connection.String(),
			"to", dstLatestConn.Connection.String(),
		)
		return srcConn, dstConn, false, nil
	}
	return srcConn, dstConn, true, nil
}

func GetConnectionPairLogger(src, dst Chain) *log.RelayLogger {
	return log.GetLogger().
		WithConnectionPair(
			src.ChainID(),
			src.Path().ClientID,
			src.Path().ConnectionID,
			dst.ChainID(),
			dst.Path().ClientID,
			dst.Path().ConnectionID,
		).
		WithModule("core.connection")
}

func WithConnectionPairAttributes(src, dst Chain) trace.SpanStartOption {
	return trace.WithAttributes(slices.Concat(
		semconv.AttributeGroup("src",
			semconv.ChainIDKey.String(src.ChainID()),
			semconv.ClientIDKey.String(src.Path().ClientID),
			semconv.ConnectionIDKey.String(src.Path().ConnectionID),
		),
		semconv.AttributeGroup("dst",
			semconv.ChainIDKey.String(dst.ChainID()),
			semconv.ClientIDKey.String(dst.Path().ClientID),
			semconv.ConnectionIDKey.String(dst.Path().ConnectionID),
		),
	)...)
}
