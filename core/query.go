package core

import (
	"context"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	"golang.org/x/sync/errgroup"
)

// QueryClientStatePair returns a pair of connection responses
func QueryClientStatePair(
	ctx context.Context,
	src, dst ChainI,
	srch, dsth int64) (srcCsRes, dstCsRes *clienttypes.QueryClientStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientState(ctx, srch, true)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientState(ctx, dsth, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryConnectionPair returns a pair of connection responses
func QueryConnectionPair(
	ctx context.Context,
	src, dst ChainI,
	srcH, dstH int64) (srcConn, dstConn *conntypes.QueryConnectionResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcConn, err = src.QueryConnection(ctx, srcH, true)
		return err
	})
	eg.Go(func() error {
		dstConn, err = dst.QueryConnection(ctx, dstH, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryChannelPair returns a pair of channel responses
func QueryChannelPair(ctx context.Context, src, dst ChainI, srcH, dstH int64) (srcChan, dstChan *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcChan, err = src.QueryChannel(ctx, srcH, true)
		return err
	})
	eg.Go(func() error {
		dstChan, err = dst.QueryChannel(ctx, dstH, true)
		return err
	})
	err = eg.Wait()
	return
}

// QueryClientConsensusStatePair allows for the querying of multiple client states at the same time
func QueryClientConsensusStatePair(
	ctx context.Context,
	src, dst ChainI,
	srch, dsth int64, srcClientConsH,
	dstClientConsH ibcexported.Height) (srcCsRes, dstCsRes *clienttypes.QueryConsensusStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		srcCsRes, err = src.QueryClientConsensusState(ctx, srch, srcClientConsH, true)
		return err
	})
	eg.Go(func() error {
		dstCsRes, err = dst.QueryClientConsensusState(ctx, dsth, dstClientConsH, true)
		return err
	})
	err = eg.Wait()
	return
}
