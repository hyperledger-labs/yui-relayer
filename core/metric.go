package core

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

const (
	meterName     = "github.com/hyperledger-labs/yui-relayer"
	namespaceRoot = "relayer"
	fallbackAddr  = "localhost:0"
)

var (
	meterProvider *metric.MeterProvider
	meter         api.Meter

	metricSrcProcessedBlockHeight   *atomic.Int64
	metricDstProcessedBlockHeight   *atomic.Int64
	metricSrcBacklog                PacketInfoList // take care. this is not thread safe.
	metricDstBacklog                PacketInfoList // take care. this is not thread safe.
	metricSrcBacklogSize            *atomic.Int64
	metricDstBacklogSize            *atomic.Int64
	metricSrcBacklogOldestTimestamp *atomic.Int64
	metricDstBacklogOldestTimestamp *atomic.Int64

	instProcessedBlockHeight    api.Int64ObservableGauge
	instBacklogSize             api.Int64ObservableGauge
	instBacklogOldestHeight     api.Int64ObservableGauge
	instReceivePacketsFinalized api.Int64Counter
)

func InitializeMetrics(addr string) error {
	if addr == "" {
		addr = fallbackAddr
	}

	var err error

	if meterProvider, err = NewPrometheusMeterProvider(addr); err != nil {
		return err
	}

	meter = meterProvider.Meter(meterName)

	metricSrcProcessedBlockHeight = &atomic.Int64{}
	metricDstProcessedBlockHeight = &atomic.Int64{}
	metricSrcBacklogSize = &atomic.Int64{}
	metricDstBacklogSize = &atomic.Int64{}
	metricSrcBacklogOldestTimestamp = &atomic.Int64{}
	metricDstBacklogOldestTimestamp = &atomic.Int64{}

	// create the instrument "relayer.processed_block_height"
	name := fmt.Sprintf("%s.processed_block_height", namespaceRoot)
	callback := func(ctx context.Context, observer api.Int64Observer) error {
		observer.Observe(metricSrcProcessedBlockHeight.Load(), api.WithAttributes(attribute.String("chain", "src")))
		observer.Observe(metricDstProcessedBlockHeight.Load(), api.WithAttributes(attribute.String("chain", "dst")))
		return nil
	}
	if instProcessedBlockHeight, err = meter.Int64ObservableGauge(
		name,
		api.WithUnit("1"),
		api.WithDescription("latest finalized height"),
		api.WithInt64Callback(callback),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.backlog_size"
	name = fmt.Sprintf("%s.backlog_size", namespaceRoot)
	callback = func(ctx context.Context, observer api.Int64Observer) error {
		observer.Observe(metricSrcBacklogSize.Load(), api.WithAttributes(attribute.String("chain", "src")))
		observer.Observe(metricDstBacklogSize.Load(), api.WithAttributes(attribute.String("chain", "dst")))
		return nil
	}
	if instBacklogSize, err = meter.Int64ObservableGauge(
		name,
		api.WithUnit("1"),
		api.WithDescription("number of packets that are unreceived or received but unfinalized"),
		api.WithInt64Callback(callback),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.backlog_oldest_height"
	name = fmt.Sprintf("%s.backlog_oldest_height", namespaceRoot)
	callback = func(ctx context.Context, observer api.Int64Observer) error {
		observer.Observe(metricSrcBacklogOldestTimestamp.Load(), api.WithAttributes(attribute.String("chain", "src")))
		observer.Observe(metricDstBacklogOldestTimestamp.Load(), api.WithAttributes(attribute.String("chain", "dst")))
		return nil
	}
	if instBacklogOldestHeight, err = meter.Int64ObservableGauge(
		name,
		api.WithUnit("1"),
		api.WithDescription("height when the oldest packet in backlog was sent"),
		api.WithInt64Callback(callback),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.receive_packets_finalized"
	name = fmt.Sprintf("%s.receive_packets_finalized", namespaceRoot)
	if instReceivePacketsFinalized, err = meter.Int64Counter(
		name,
		api.WithUnit("1"),
		api.WithDescription("number of packets that are received and finalized"),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	return nil
}

func ShutdownMetrics(ctx context.Context) error {
	if err := meterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown the MeterProvider: %v", err)
	}
	return nil
}

func NewPrometheusMeterProvider(addr string) (*metric.MeterProvider, error) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			// TODO: we should replace this with a more proper logger, slog?
			log.Fatalf("Prometheus exporter server failed: %v", err)
		}
	}()

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create the Prometheus Exporter: %v", err)
	}

	return metric.NewMeterProvider(
		metric.WithReader(exporter),
	), nil
}

func UpdateBlockMetrics(ctx context.Context, newSrcHeight, newDstHeight uint64) {
	metricSrcProcessedBlockHeight.Store(int64(newSrcHeight))
	metricDstProcessedBlockHeight.Store(int64(newDstHeight))
}

func UpdateBacklogMetrics(ctx context.Context, src, dst Chain, newSrcBacklog, newDstBacklog PacketInfoList) error {
	metricSrcBacklogSize.Store(int64(len(newSrcBacklog)))
	metricDstBacklogSize.Store(int64(len(newDstBacklog)))

	if len(newSrcBacklog) > 0 {
		oldestHeight := newSrcBacklog[0].EventHeight
		oldestTimestamp, err := src.Timestamp(oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the src chain: %v", oldestHeight, err)
		}
		metricSrcBacklogOldestTimestamp.Store(oldestTimestamp.UnixNano())
	} else {
		metricSrcBacklogOldestTimestamp.Store(0)
	}
	if len(newDstBacklog) > 0 {
		oldestHeight := newDstBacklog[0].EventHeight
		oldestTimestamp, err := dst.Timestamp(oldestHeight)
		if err != nil {
			return fmt.Errorf("failed to get the timestamp of block[%d] on the dst chain: %v", oldestHeight, err)
		}
		metricDstBacklogOldestTimestamp.Store(oldestTimestamp.UnixNano())
	} else {
		metricDstBacklogOldestTimestamp.Store(0)
	}

	for _, packet := range metricSrcBacklog {
		received := true
		for _, newPacket := range newSrcBacklog {
			if packet.Sequence == newPacket.Sequence {
				received = false
				break
			}
		}
		if received {
			instReceivePacketsFinalized.Add(ctx, 1, api.WithAttributes(attribute.String("chain", "src")))
		}
	}
	metricSrcBacklog = newSrcBacklog

	for _, packet := range metricDstBacklog {
		received := true
		for _, newPacket := range newDstBacklog {
			if packet.Sequence == newPacket.Sequence {
				received = false
				break
			}
		}
		if received {
			instReceivePacketsFinalized.Add(ctx, 1, api.WithAttributes(attribute.String("chain", "dst")))
		}
	}
	metricDstBacklog = newDstBacklog

	return nil
}
