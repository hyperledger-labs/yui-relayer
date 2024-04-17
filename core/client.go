package core

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
)

func checkCreateClientsReady(src, dst *ProvableChain, logger *log.RelayLogger) (bool, error) {
	srcID := src.Chain.Path().ClientID;
	dstID := dst.Chain.Path().ClientID;

	if srcID == "" && dstID == "" {
		return true, nil
	}

	getState := func(pc *ProvableChain) (*clienttypes.QueryClientStateResponse, error) {
		latestHeight, err := pc.LatestHeight()
		if err != nil {
			return nil, err
		}

		ctx := NewQueryContext(context.TODO(), latestHeight)
		return pc.QueryClientState(ctx)
	}

	srcState := (*clienttypes.QueryClientStateResponse)(nil)
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

	dstState := (*clienttypes.QueryClientStateResponse)(nil)
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
		logger.Warn(fmt.Sprintf("clients are already created: src=%s, dst=%s", srcID, dstID))
		return false, nil
	} else {
		return true, nil
	}
}

func CreateClients(pathName string, src, dst *ProvableChain, srcHeight, dstHeight exported.Height) error {
	logger := GetChainPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "CreateClients")

	if cont, err := checkCreateClientsReady(src, dst, logger); err != nil {
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
			logger.Error(
				"failed to get address for create client",
				err,
			)
			return err
		}

		cs, cons, err := dst.CreateInitialLightClientState(dstHeight)
		if err != nil {
			logger.Error("failed to create initial light client state", err)
			return err
		}
		msg, err := clienttypes.NewMsgCreateClient(cs, cons, srcAddr.String())
		if err != nil {
			return fmt.Errorf("failed to create MsgCreateClient: %v", err)
		}
		clients.Src = append(clients.Src, msg)
	}

	if dst.Chain.Path().ClientID == "" {
		dstAddr, err := dst.GetAddress()
		if err != nil {
			logger.Error(
				"failed to get address for create client",
				err,
			)
			return err
		}

		cs, cons, err := src.CreateInitialLightClientState(srcHeight)
		if err != nil {
			logger.Error("failed to create initial light client state", err)
			return err
		}
		msg, err := clienttypes.NewMsgCreateClient(cs, cons, dstAddr.String())
		if err != nil {
			logger.Error("failed to create MsgCreateClient: %v", err)
			return err
		}
		clients.Dst = append(clients.Dst, msg)
	}

	// Send msgs to both chains
	if clients.Ready() {
		// TODO: Add retry here for out of gas or other errors
		clients.Send(src, dst)
		if clients.Success() {
			logger.Info(
				"★ Clients created",
			)
			if err := SyncChainConfigsFromEvents(pathName, clients.SrcMsgIDs, clients.DstMsgIDs, src, dst); err != nil {
				return err
			}
		}
	}
	return nil
}

func UpdateClients(src, dst *ProvableChain) error {
	logger := GetClientPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "UpdateClients")
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)
	// First, update the light clients to the latest header and return the header
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		logger.Error(
			"failed to create sync headers for update client",
			err,
		)
		return err
	}
	srcUpdateHeaders, dstUpdateHeaders, err := sh.SetupBothHeadersForUpdate(src, dst)
	if err != nil {
		logger.Error(
			"failed to setup both headers for update client",
			err,
		)
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
		if clients.Send(src, dst); clients.Success() {
			logger.Info(
				"★ Clients updated",
			)
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
