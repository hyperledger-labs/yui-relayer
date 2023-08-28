package core

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	ProcessedBlockHeightGauge      *Int64SyncGauge
	BacklogSizeGauge               *Int64SyncGauge
	BacklogOldestTimestampGauge    *Int64SyncGauge
	ReceivePacketsFinalizedCounter api.Int64Counter
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

	// create the instrument "relayer.processed_block_height"
	name := fmt.Sprintf("%s.processed_block_height", namespaceRoot)
	if ProcessedBlockHeightGauge, err = NewInt64SyncGauge(
		meter,
		name,
		api.WithUnit("1"),
		api.WithDescription("latest finalized height"),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.backlog_size"
	name = fmt.Sprintf("%s.backlog_size", namespaceRoot)
	if BacklogSizeGauge, err = NewInt64SyncGauge(
		meter,
		name,
		api.WithUnit("1"),
		api.WithDescription("number of packets that are unreceived or received but unfinalized"),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.backlog_oldest_timestamp"
	name = fmt.Sprintf("%s.backlog_oldest_timestamp", namespaceRoot)
	if BacklogOldestTimestampGauge, err = NewInt64SyncGauge(
		meter,
		name,
		api.WithUnit("nsec"),
		api.WithDescription("timestamp when the oldest packet in backlog was sent"),
	); err != nil {
		return fmt.Errorf("failed to create the instrument %s: %v", name, err)
	}

	// create the instrument "relayer.receive_packets_finalized"
	name = fmt.Sprintf("%s.receive_packets_finalized", namespaceRoot)
	if ReceivePacketsFinalizedCounter, err = meter.Int64Counter(
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
