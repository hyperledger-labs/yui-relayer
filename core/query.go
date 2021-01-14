package core

import (
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	conntypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	chantypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"golang.org/x/sync/errgroup"
)

// QueryClientStatePair returns a pair of connection responses
func QueryClientStatePair(
	src, dst ChainI,
	srch, dsth int64) (srcCsRes, dstCsRes *clienttypes.QueryClientStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientState(srch, true)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientState(dsth, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryConnectionPair returns a pair of connection responses
func QueryConnectionPair(
	src, dst ChainI,
	srcH, dstH int64) (srcConn, dstConn *conntypes.QueryConnectionResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcConn, err = src.QueryConnection(srcH, true)
		return err
	})
	eg.Go(func() error {
		dstConn, err = dst.QueryConnection(dstH, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryChannelPair returns a pair of channel responses
func QueryChannelPair(src, dst ChainI, srcH, dstH int64) (srcChan, dstChan *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcChan, err = src.QueryChannel(srcH, true)
		return err
	})
	eg.Go(func() error {
		dstChan, err = dst.QueryChannel(dstH, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryClientConsensusStatePair allows for the querying of multiple client states at the same time
func QueryClientConsensusStatePair(
	src, dst ChainI,
	srch, dsth int64, srcClientConsH,
	dstClientConsH ibcexported.Height) (srcCsRes, dstCsRes *clienttypes.QueryConsensusStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientConsensusState(srch, srcClientConsH, true)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientConsensusState(dsth, dstClientConsH, true)
		return err
	})
	err = eg.Wait()
	return
}
