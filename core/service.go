package core

import (
	"context"
	"fmt"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/hyperledger-labs/yui-relayer/logger"
	"go.uber.org/zap"
)

// StartService starts a relay service
func StartService(ctx context.Context, st StrategyI, src, dst *ProvableChain, relayInterval time.Duration) error {
	sh, err := NewSyncHeaders(src, dst)
	if err != nil {
		return err
	}
	srv := NewRelayService(st, src, dst, sh, relayInterval)
	return srv.Start(ctx)
}

type RelayService struct {
	src      *ProvableChain
	dst      *ProvableChain
	st       StrategyI
	sh       SyncHeaders
	interval time.Duration
}

// NewRelayService returns a new service
func NewRelayService(st StrategyI, src, dst *ProvableChain, sh SyncHeaders, interval time.Duration) *RelayService {
	return &RelayService{
		src:      src,
		dst:      dst,
		st:       st,
		sh:       sh,
		interval: interval,
	}
}

// Start starts a relay service
func (srv *RelayService) Start(ctx context.Context) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
	for {
		if err := retry.Do(func() error {
			select {
			case <-ctx.Done():
				return retry.Unrecoverable(ctx.Err())
			default:
				return srv.Serve(ctx)
			}
		}, rtyAtt, rtyDel, rtyErr, retry.OnRetry(func(n uint, err error) {
			logger.Info(fmt.Sprintf("- [%s][%s]try(%d/%d) relay-service: %s", srv.src.ChainID(), srv.dst.ChainID(), n+1, rtyAttNum, err))
		})); err != nil {
			return err
		}
		time.Sleep(srv.interval)
	}
}

// Serve performs packet-relay
func (srv *RelayService) Serve(ctx context.Context) error {
	logger := logger.ZapLogger()
	defer logger.Sync()
	// First, update the latest headers for src and dst
	if err := srv.sh.Updates(srv.src, srv.dst); err != nil {
		logger.Error("failed to update headers", zap.Error(err))
		return err
	}

	// relay packets if unrelayed seqs exist

	pseqs, err := srv.st.UnrelayedSequences(srv.src, srv.dst, srv.sh)
	if err != nil {
		logger.Error("failed to get unrelayed sequences", zap.Error(err))
		return err
	}
	if err := srv.st.RelayPackets(srv.src, srv.dst, pseqs, srv.sh); err != nil {
		logger.Error("failed to relay packets", zap.Error(err))
		return err
	}

	// relay acks if unrelayed seqs exist

	aseqs, err := srv.st.UnrelayedAcknowledgements(srv.src, srv.dst, srv.sh)
	if err != nil {
		logger.Error("failed to get unrelayed acknowledgements", zap.Error(err))
		return err
	}
	if err := srv.st.RelayAcknowledgements(srv.src, srv.dst, aseqs, srv.sh); err != nil {
		logger.Error("failed to relay acknowledgements", zap.Error(err))
		return err
	}

	return nil
}
