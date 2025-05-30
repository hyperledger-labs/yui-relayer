package core

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/hyperledger-labs/yui-relayer/otelcore/semconv"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func checkCreateClientsReady(ctx context.Context, src, dst *ProvableChain, logger *log.RelayLogger) (bool, error) {
	srcID := src.Chain.Path().ClientID
	dstID := dst.Chain.Path().ClientID

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (*clienttypes.QueryClientStateResponse, error) {
		latestHeight, err := pc.LatestHeight(ctx)
		if err != nil {
			return nil, err
		}

		ctx := NewQueryContext(ctx, latestHeight)
		return pc.QueryClientState(ctx)
	}

	var srcState *clienttypes.QueryClientStateResponse
	if srcID != "" {
		s, err := getState(src)
		if err != nil {
			return false, err
		}
		if s == nil {
			return false, fmt.Errorf("src client id is given but that client does not exist: %s", srcID)
		}
		srcState = s
	}

	var dstState *clienttypes.QueryClientStateResponse
	if dstID != "" {
		s, err := getState(dst)
		if err != nil {
			return false, err
		}
		if s == nil {
			return false, fmt.Errorf("dst client id is given but that client does not exist: %s", dstID)
		}
		dstState = s
	}

	if srcState != nil && dstState != nil {
		logger.WarnContext(ctx, "clients are already created", "src_client_id", srcID, "dst_client_id", dstID)
		return false, nil
	} else {
		return true, nil
	}
}

func CreateClients(ctx context.Context, pathName string, src, dst *ProvableChain, srcHeight, dstHeight exported.Height) error {
	ctx, span := tracer.Start(ctx, "CreateClients", WithChainPairAttributes(src, dst))
	defer span.End()
	logger := GetChainPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "CreateClients")

	if cont, err := checkCreateClientsReady(ctx, src, dst, logger); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	} else if !cont {
		return nil
	}

	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	if src.Chain.Path().ClientID == "" {
		srcAddr, err := src.GetAddress()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get address for create client", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		cs, cons, err := dst.CreateInitialLightClientState(ctx, dstHeight)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create initial light client state", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		msg, err := clienttypes.NewMsgCreateClient(cs, cons, srcAddr.String())
		if err != nil {
			err = fmt.Errorf("failed to create MsgCreateClient: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		clients.Src = append(clients.Src, msg)
	}

	if dst.Chain.Path().ClientID == "" {
		dstAddr, err := dst.GetAddress()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get address for create client", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		cs, cons, err := src.CreateInitialLightClientState(ctx, srcHeight)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create initial light client state", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		msg, err := clienttypes.NewMsgCreateClient(cs, cons, dstAddr.String())
		if err != nil {
			logger.ErrorContext(ctx, "failed to create MsgCreateClient: %v", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		clients.Send(ctx, src, dst)
		if clients.Success() {
			logger.InfoContext(ctx, "★ Clients created")
			if err := SyncChainConfigsFromEvents(ctx, pathName, clients.SrcMsgIDs, clients.DstMsgIDs, src, dst); err != nil {
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}
	}
	return nil
}

func UpdateClients(ctx context.Context, src, dst *ProvableChain) error {
	ctx, span := tracer.Start(ctx, "UpdateClients", WithChainPairAttributes(src, dst))
	defer span.End()
	logger := GetClientPairLogger(src, dst)
	defer logger.TimeTrackContext(ctx, time.Now(), "UpdateClients")
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(ctx, src, dst)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create sync headers for update client", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	srcUpdateHeaders, dstUpdateHeaders, err := sh.SetupBothHeadersForUpdate(ctx, src, dst)
	if err != nil {
		logger.ErrorContext(ctx, "failed to setup both headers for update client", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if len(dstUpdateHeaders) > 0 {
		clients.Src = append(clients.Src, src.Path().UpdateClients(dstUpdateHeaders, mustGetAddress(src))...)
	}
	if len(srcUpdateHeaders) > 0 {
		clients.Dst = append(clients.Dst, dst.Path().UpdateClients(srcUpdateHeaders, mustGetAddress(dst))...)
	}
	// Send msgs to both chains
	if clients.Ready() {
		if clients.Send(ctx, src, dst); clients.Success() {
			logger.InfoContext(ctx, "★ Clients updated")
		}
	}
	return nil
}

func GetClientPairLogger(src, dst Chain) *log.RelayLogger {
	return log.GetLogger().
		WithClientPair(
			src.ChainID(), src.Path().ClientID,
			dst.ChainID(), dst.Path().ClientID,
		).
		WithModule("core.client")
}

func WithClientAttributes(c Chain) trace.SpanStartOption {
	return trace.WithAttributes(
		semconv.ChainIDKey.String(c.ChainID()),
		semconv.ClientIDKey.String(c.Path().ClientID),
	)
}
