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
	"golang.org/x/sync/errgroup"
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

type queryStateResult struct {
	updateHeaders []Header
	conn          *conntypes.QueryConnectionResponse
	settled       bool
	csRes         *clienttypes.QueryClientStateResponse
	cs            ibcexported.ClientState
	consRes       *clienttypes.QueryConsensusStateResponse
	cons          ibcexported.ConsensusState
	consH         ibcexported.Height
}
func	queryState(ctx QueryContext, logger *log.RelayLogger, sh SyncHeaders, prover, counterparty *ProvableChain)  (*queryStateResult, error) {
	var ret queryStateResult
	var err error

	ret.updateHeaders, err = sh.SetupHeadersForUpdate(ctx.Context(), prover, counterparty)
	if err != nil {
		logger.ErrorContext(ctx.Context(), "error setting up headers for update", err)
		return nil, err
	}

	ret.conn, ret.settled, err = querySettledConnection(ctx, logger, prover, true)
	if err != nil {
		return nil, err
	} else if !ret.settled {
		return &ret, nil
	}

	if ret.conn.Connection.State != conntypes.UNINITIALIZED {
		ret.csRes, err = QueryClientState(ctx, prover, true)
		if err != nil {
			return nil, err
		}
		if err := prover.Codec().UnpackAny(ret.csRes.ClientState, &ret.cs); err != nil {
			return nil, err
		}

		// Store the heights
		ret.consH = ret.cs.GetLatestHeight()
		ret.consRes, err = QueryClientConsensusState(ctx, prover, ret.consH, true)
		if err != nil {
			return nil, err
		}
		if err := prover.Codec().UnpackAny(ret.consRes.ConsensusState, &ret.cons); err != nil {
			return nil, err
		}
	}
	return &ret, nil
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
		srcState, dstState                 *queryStateResult
		srcHostConsProof, dstHostConsProof []byte
	)

	if err := EnsureDifferentChains(src, dst); err != nil {
		return nil, err
	}

	{
		var eg = new(errgroup.Group)
		srcStream := make(chan *queryStateResult, 1)
		dstStream := make(chan *queryStateResult, 1)
		defer close(srcStream)
		defer close(dstStream)

		srcHeight := sh.GetQueryContext(ctx, src.ChainID()).Height()
		dstHeight := sh.GetQueryContext(ctx, dst.ChainID()).Height()

		eg.Go(func() error {
			logger := &log.RelayLogger{Logger: GetConnectionPairLogger(src, dst).With(
				"side", "src",
				"src_height", srcHeight.String(),
				"dst_height", dstHeight.String(),
			)}
			queryCtx := sh.GetQueryContext(ctx, src.ChainID())
			state, err := queryState(queryCtx, logger, sh, src, dst) //, doGetProof)
			if err != nil {
				return err
			}
			srcStream <- state
			return nil
		})
		eg.Go(func() error {
			logger := &log.RelayLogger{Logger: GetConnectionPairLogger(src, dst).With(
				"side", "dst",
				"src_height", srcHeight.String(),
				"dst_height", dstHeight.String(),
			)}
			queryCtx := sh.GetQueryContext(ctx, dst.ChainID())
			state, err := queryState(queryCtx, logger, sh, dst, src) //, doGetProof)
			if err != nil {
				return err
			}
			dstStream <- state
			return nil
		})

		err := eg.Wait() // it wait quering to other chain. it may take more time and my chain's state is deleted.
		if  err != nil {
			return nil, err
		}
		srcState, _ = <- srcStream
		dstState, _ = <- dstStream
	}
	if (!srcState.settled || !dstState.settled) {
		return out, nil
	}

	if (srcState.conn.Connection.State != conntypes.UNINITIALIZED) {
		// note that ProveHostConsensusState does not query to its chain.
		dstHostConsProof, err = dst.ProveHostConsensusState(sh.GetQueryContext(ctx, dst.ChainID()), srcState.cs.GetLatestHeight(), srcState.cons)
		if err != nil {
			return nil, err
		}
	}
	if (dstState.conn.Connection.State != conntypes.UNINITIALIZED) {
		srcHostConsProof, err = src.ProveHostConsensusState(sh.GetQueryContext(ctx, src.ChainID()), dstState.cs.GetLatestHeight(), dstState.cons)
		if err != nil {
			return nil, err
		}
	}

	switch {
	// Handshake hasn't been started on src or dst, relay `connOpenInit` to src
	case srcState.conn.Connection.State == conntypes.UNINITIALIZED && dstState.conn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(ctx, src, dst, srcState.conn, dstState.conn)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnInit(dst.Path(), addr))
	// Handshake has started on dst (1 step done), relay `connOpenTry` and `updateClient` on src
	case srcState.conn.Connection.State == conntypes.UNINITIALIZED && dstState.conn.Connection.State == conntypes.INIT:
		logConnectionStates(ctx, src, dst, srcState.conn, dstState.conn)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnTry(dst.Path(), dstState.csRes, dstState.conn, dstState.consRes, srcHostConsProof, addr))
	// Handshake has started on src (1 step done), relay `connOpenTry` and `updateClient` on dst
	case srcState.conn.Connection.State == conntypes.INIT && dstState.conn.Connection.State == conntypes.UNINITIALIZED:
		logConnectionStates(ctx, dst, src, dstState.conn, srcState.conn)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnTry(src.Path(), srcState.csRes, srcState.conn, srcState.consRes, dstHostConsProof, addr))

	// Handshake has started on src end (2 steps done), relay `connOpenAck` and `updateClient` to dst end
	case srcState.conn.Connection.State == conntypes.TRYOPEN && dstState.conn.Connection.State == conntypes.INIT:
		logConnectionStates(ctx, dst, src, dstState.conn, srcState.conn)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnAck(src.Path(), srcState.csRes, srcState.conn, srcState.consRes, dstHostConsProof, addr))

	// Handshake has started on dst end (2 steps done), relay `connOpenAck` and `updateClient` to src end
	case srcState.conn.Connection.State == conntypes.INIT && dstState.conn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(ctx, src, dst, srcState.conn, dstState.conn)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnAck(dst.Path(), dstState.csRes, dstState.conn, dstState.consRes, srcHostConsProof, addr))

	// Handshake has confirmed on dst (3 steps done), relay `connOpenConfirm` and `updateClient` to src end
	case srcState.conn.Connection.State == conntypes.TRYOPEN && dstState.conn.Connection.State == conntypes.OPEN:
		logConnectionStates(ctx, src, dst, srcState.conn, dstState.conn)
		addr := mustGetAddress(src)
		if len(dstState.updateHeaders) > 0 {
			out.Src = append(out.Src, src.Path().UpdateClients(dstState.updateHeaders, addr)...)
		}
		out.Src = append(out.Src, src.Path().ConnConfirm(dstState.conn, addr))
		out.Last = true

	// Handshake has confirmed on src (3 steps done), relay `connOpenConfirm` and `updateClient` to dst end
	case srcState.conn.Connection.State == conntypes.OPEN && dstState.conn.Connection.State == conntypes.TRYOPEN:
		logConnectionStates(ctx, dst, src, dstState.conn, srcState.conn)
		addr := mustGetAddress(dst)
		if len(srcState.updateHeaders) > 0 {
			out.Dst = append(out.Dst, dst.Path().UpdateClients(srcState.updateHeaders, addr)...)
		}
		out.Dst = append(out.Dst, dst.Path().ConnConfirm(srcState.conn, addr))
		out.Last = true

	default:
		panic(fmt.Sprintf("not implemented error: %v %v", srcState.conn.Connection.State, dstState.conn.Connection.State))
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

func querySettledConnection(
	queryCtx QueryContext,
	logger *log.RelayLogger,
	chain interface {
		Chain
		StateProver
	},
	prove bool,
) (*conntypes.QueryConnectionResponse, bool, error) {
	logger.DebugContext(queryCtx.Context(), ">QuerySettledConnection", "chainId", chain.ChainID())

	conn, err := QueryConnection(queryCtx, chain, prove)
	if err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to query connection at the latest finalized height", err)
		return nil, false, err
	}

	var latestCtx QueryContext
	if h, err := chain.LatestHeight(queryCtx.Context()); err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to get the latest height", err)
		return nil, false, err
	} else {
		latestCtx = NewQueryContext(queryCtx.Context(), h)
	}

	latestConn, err := QueryConnection(latestCtx, chain, false)
	if err != nil {
		logger.ErrorContext(queryCtx.Context(), "failed to query connection pair at the latest height", err)
		return nil, false, err
	}

	if conn.Connection.String() != latestConn.Connection.String() {
		logger.DebugContext(queryCtx.Context(), "connection end in transition",
			"from", conn.Connection.String(),
			"to", latestConn.Connection.String(),
		)
		return conn, false, nil
	}
	return conn, true, nil
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
	src_latest, _ := src.LatestHeight(context.TODO())
	dst_latest, _ := dst.LatestHeight(context.TODO())
	logger.DebugContext(srcCtx.Context(), ">QuerySettledConnectionPair", "src", src.ChainID(), "src_latest", src_latest, "dst", dst.ChainID(), "dst_latest", dst_latest)

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
