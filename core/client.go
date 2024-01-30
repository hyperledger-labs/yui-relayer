package core

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/hyperledger-labs/yui-relayer/log"
)

func CreateClients(pathName string, src, dst *ProvableChain, srcHeight, dstHeight exported.Height) error {
	logger := GetChainPairLogger(src, dst)
	defer logger.TimeTrack(time.Now(), "CreateClients")
	var (
		clients = &RelayMsgs{Src: []sdk.Msg{}, Dst: []sdk.Msg{}}
	)

	srcAddr, err := src.GetAddress()
	if err != nil {
		logger.Error(
			"failed to get address for create client",
			err,
		)
		return err
	}
	dstAddr, err := dst.GetAddress()
	if err != nil {
		logger.Error(
			"failed to get address for create client",
			err,
		)
		return err
	}

	{
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

	{
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
