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
		if err != nil {
			return nil, err
		}
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
) (srcCsRes, dstCsRes *clienttypes.QueryClientStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		var err error
		srcCsRes, err = QueryClientState(srcCtx, src, prove)
		return err
	})
	eg.Go(func() error {
		var err error
		dstCsRes, err = QueryClientState(dstCtx, dst, prove)
		return err
	})
	err = eg.Wait()
	return
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
) (srcCsRes, dstCsRes *clienttypes.QueryConsensusStateResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		var err error
		srcCsRes, err = QueryClientConsensusState(srcCtx, src, srcClientConsH, prove)
		return err
	})
	eg.Go(func() error {
		var err error
		dstCsRes, err = QueryClientConsensusState(dstCtx, dst, dstClientConsH, prove)
		return err
	})
	err = eg.Wait()
	return
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
) (srcConn, dstConn *conntypes.QueryConnectionResponse, err error) {
	var eg = new(errgroup.Group)
	eg.Go(func() error {
		var err error
		srcConn, err = QueryConnection(srcCtx, src, prove)
		return err
	})
	eg.Go(func() error {
		var err error
		dstConn, err = QueryConnection(dstCtx, dst, prove)
		return err
	})
	err = eg.Wait()
	return
}

func QueryChannel(queryCtx QueryContext, chain interface {
	Chain
	StateProver
}, prove bool) (*chantypes.QueryChannelResponse, error) {
	if chain.Path().ChannelID == "" {
		ch := &chantypes.QueryChannelResponse{
			Channel: &chantypes.Channel{
				State: chantypes.UNINITIALIZED,
			},
		}
		return ch, nil
	}

	ch, err := chain.QueryChannel(queryCtx)
	if err != nil {
		return nil, err
	} else if ch.Channel.State == chantypes.UNINITIALIZED {
		return ch, nil
	}

	if prove {
		path := host.ChannelPath(chain.Path().PortID, chain.Path().ChannelID)
		var value []byte
		value, err = chain.Codec().Marshal(ch.Channel)
		if err != nil {
			return nil, err
		}
		ch.Proof, ch.ProofHeight, err = chain.ProveState(queryCtx, path, value)
		if err != nil {
			return nil, err
		}
	}
	return ch, nil
}

// QueryChannelPair returns a pair of connection responses
func QueryChannelPair(
	srcCtx, dstCtx QueryContext,
	src, dst interface {
		Chain
		StateProver
	},
	prove bool,
) (srcChanRes, dstChanRes *chantypes.QueryChannelResponse, err error) {
	var eg = new(errgroup.Group)

	eg.Go(func() error {
		var err error
		srcChanRes, err = QueryChannel(srcCtx, src, prove)
		return err
	})
	eg.Go(func() error {
		var err error
		dstChanRes, err = QueryChannel(dstCtx, dst, prove)
		return err
	})
	err = eg.Wait()
	return
}

func QueryChannelUpgrade(queryCtx QueryContext, chain interface {
	Chain
	StateProver
}, prove bool) (*chantypes.QueryUpgradeResponse, error) {
	chanUpg, err := chain.QueryChannelUpgrade(queryCtx)
	if err != nil {
		return nil, err
	} else if chanUpg == nil {
		return nil, nil
	}

	if !prove {
		return chanUpg, nil
	}

	if value, err := chain.Codec().Marshal(&chanUpg.Upgrade); err != nil {
		return nil, err
	} else {
		path := host.ChannelUpgradePath(chain.Path().PortID, chain.Path().ChannelID)
		chanUpg.Proof, chanUpg.ProofHeight, err = chain.ProveState(queryCtx, path, value)
		if err != nil {
			return nil, err
		}
	}
	return chanUpg, nil
}

func QueryChannelUpgradePair(srcCtx, dstCtx QueryContext, src, dst interface {
	Chain
	StateProver
}, prove bool) (srcChanUpg, dstChanUpg *chantypes.QueryUpgradeResponse, err error) {
	eg := new(errgroup.Group)

	eg.Go(func() error {
		var err error
		srcChanUpg, err = QueryChannelUpgrade(srcCtx, src, prove)
		if err != nil {
			return err
		}
		return nil
	})
	eg.Go(func() error {
		dstChanUpg, err = QueryChannelUpgrade(dstCtx, dst, prove)
		if err != nil {
			return err
		}
		return nil
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
