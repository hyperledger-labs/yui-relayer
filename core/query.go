package core

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	"golang.org/x/sync/errgroup"
)

// QueryClientStatePair returns a pair of connection responses
func QueryClientState(
	ctx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	prove bool,
) (csRes *clienttypes.QueryClientStateResponse, err error) {
	csRes, err = chain.QueryClientState(ctx)
	if err != nil {
		return nil, err
	}
	if prove {
		path := host.FullClientStatePath(chain.Path().ClientID)
		var value []byte
		value, err = chain.Codec().Marshal(csRes.ClientState)
		if err != nil {
			return nil, err
		}
		csRes.Proof, csRes.ProofHeight, err = chain.ProveState(ctx, path, value)
	}
	return csRes, nil
}

func QueryClientStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*clienttypes.QueryClientStateResponse, *clienttypes.QueryClientStateResponse, error) {
	srcDsRes, err := QueryClientState(srcCtx, src, prove)
	if err != nil {
		return nil, nil, err
	}

	dstDsRes, err := QueryClientState(dstCtx, dst, prove)
	if err != nil {
		return nil, nil, err
	}
	return srcDsRes, dstDsRes, nil
}

func QueryClientConsensusState(
	ctx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	clientConsH ibcexported.Height,
	prove bool,
) (*clienttypes.QueryConsensusStateResponse, error) {
	csRes, err := chain.QueryClientConsensusState(ctx, clientConsH)
	if err != nil {
		return nil, err
	}
	if prove {
		path := host.FullConsensusStatePath(chain.Path().ClientID, clientConsH)
		value, err := chain.Codec().Marshal(csRes.ConsensusState)
		if err != nil {
			return nil, err
		}
		csRes.Proof, csRes.ProofHeight, err = chain.ProveState(ctx, path, value)
		if err != nil {
			return nil, err
		}
	}
	return csRes, nil
}

// QueryClientConsensusStatePair allows for the querying of multiple client states at the same time
func QueryClientConsensusStatePair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	srcClientConsH,
	dstClientConsH ibcexported.Height,
	prove bool,
) (*clienttypes.QueryConsensusStateResponse, *clienttypes.QueryConsensusStateResponse, error) {
	srcCsRes, err := QueryClientConsensusState(srcCtx, src, srcClientConsH, prove)
	if err != nil {
		return nil, nil, err
	}
	dstCsRes, err := QueryClientConsensusState(dstCtx, dst, dstClientConsH, prove)
	if err != nil {
		return nil, nil, err
	}
	return srcCsRes, dstCsRes, nil
}

func QueryConnection(
	queryCtx QueryContext,
	chain interface {
		Chain
		StateProver
	},
	prove bool,
) (*conntypes.QueryConnectionResponse, error) {
	if chain.Path().ConnectionID == "" {
		conn := &conntypes.QueryConnectionResponse{
			Connection: &conntypes.ConnectionEnd{
				State: conntypes.UNINITIALIZED,
			},
		}
		return conn, nil
	}
	conn, err := chain.QueryConnection(queryCtx, chain.Path().ConnectionID)
	if err != nil {
		return nil, err
	} else if conn.Connection.State == conntypes.UNINITIALIZED {
		return conn, nil
	}
	if prove {
		path := host.ConnectionPath(chain.Path().ConnectionID)
		value, err := chain.Codec().Marshal(conn.Connection)
		if err != nil {
			return nil, err
		}
		conn.Proof, conn.ProofHeight, err = chain.ProveState(queryCtx, path, value)
	}
	return conn, nil
}

// QueryConnectionPair returns a pair of connection responses
func QueryConnectionPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (*conntypes.QueryConnectionResponse, *conntypes.QueryConnectionResponse, error) {
	srcConnRes, err := QueryConnection(srcCtx, src, prove)
	if err != nil {
		return nil, nil, err
	}
	dstConnRes, err := QueryConnection(dstCtx, dst, prove)
	if err != nil {
		return nil, nil, err
	}
	return srcConnRes, dstConnRes, nil
}

// QueryChannelPair returns a pair of channel responses
func QueryChannelPair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChan, dstChan *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		if src.Path().ChannelID == "" {
			srcChan = &chantypes.QueryChannelResponse{
				Channel: &chantypes.Channel{
					State: chantypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		srcChan, err = src.QueryChannel(srcCtx)
		if err != nil {
			return err
		} else if srcChan.Channel.State == chantypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ChannelPath(src.Path().PortID, src.Path().ChannelID)
			var value []byte
			value, err = src.Codec().Marshal(srcChan.Channel)
			if err != nil {
				return err
			}
			srcChan.Proof, srcChan.ProofHeight, err = src.ProveState(srcCtx, path, value)
		}
		return err
	})
	eg.Go(func() error {
		if dst.Path().ChannelID == "" {
			dstChan = &chantypes.QueryChannelResponse{
				Channel: &chantypes.Channel{
					State: chantypes.UNINITIALIZED,
				},
			}
			return nil
		}
		var err error
		dstChan, err = dst.QueryChannel(dstCtx)
		if err != nil {
			return err
		} else if dstChan.Channel.State == chantypes.UNINITIALIZED {
			return nil
		}
		if prove {
			path := host.ChannelPath(dst.Path().PortID, dst.Path().ChannelID)
			var value []byte
			value, err = dst.Codec().Marshal(dstChan.Channel)
			if err != nil {
				return err
			}
			dstChan.Proof, dstChan.ProofHeight, err = dst.ProveState(dstCtx, path, value)
		}
		return err
	})
	err = eg.Wait()
	return
}

func QueryChannelUpgradePair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChanUpg, dstChanUpg *chantypes.QueryUpgradeResponse, err error) {
	eg := new(errgroup.Group)

	// get channel upgrade from src chain
	eg.Go(func() error {
		var err error
		srcChanUpg, err = src.QueryChannelUpgrade(srcCtx)
		if err != nil {
			return err
		} else if srcChanUpg == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := src.Codec().Marshal(&srcChanUpg.Upgrade); err != nil {
			return err
		} else {
			path := host.ChannelUpgradePath(src.Path().PortID, src.Path().ChannelID)
			srcChanUpg.Proof, srcChanUpg.ProofHeight, err = src.ProveState(srcCtx, path, value)
			return err
		}
	})

	// get channel upgrade from dst chain
	eg.Go(func() error {
		var err error
		dstChanUpg, err = dst.QueryChannelUpgrade(dstCtx)
		if err != nil {
			return err
		} else if dstChanUpg == nil {
			return nil
		}

		if !prove {
			return nil
		}

		if value, err := dst.Codec().Marshal(&dstChanUpg.Upgrade); err != nil {
			return err
		} else {
			path := host.ChannelUpgradePath(dst.Path().PortID, dst.Path().ChannelID)
			dstChanUpg.Proof, dstChanUpg.ProofHeight, err = dst.ProveState(dstCtx, path, value)
			return err
		}
	})
	err = eg.Wait()
	return
}

func QueryChannelUpgradeError(ctx QueryContext, chain interface {
	Chain
	StateProver
}, prove bool) (*chantypes.QueryUpgradeErrorResponse, error) {
	if chanUpgErr, err := chain.QueryChannelUpgradeError(ctx); err != nil {
		return nil, err
	} else if chanUpgErr == nil {
		return nil, nil
	} else if !prove {
		return chanUpgErr, nil
	} else if value, err := chain.Codec().Marshal(&chanUpgErr.ErrorReceipt); err != nil {
		return nil, err
	} else {
		path := host.ChannelUpgradeErrorPath(chain.Path().PortID, chain.Path().ChannelID)
		chanUpgErr.Proof, chanUpgErr.ProofHeight, err = chain.ProveState(ctx, path, value)
		return chanUpgErr, err
	}
}
