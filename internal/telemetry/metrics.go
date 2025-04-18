package telemetry

import (
	"fmt"
	"net/http"

	"github.com/hyperledger-labs/yui-relayer/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
)

const (
	namespaceRoot = "relayer"
)

var (
	ProcessedBlockHeightGauge      *Int64SyncGauge
	BacklogSizeGauge               *Int64SyncGauge
	BacklogOldestTimestampGauge    *Int64SyncGauge
	ReceivePacketsFinalizedCounter api.Int64Counter

	meter = otel.Meter(name)
)

func InitializeMetrics() error {
	var err error

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

func NewPrometheusExporter(addr string) (*prometheus.Exporter, error) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger := log.GetLogger().WithModule("core.metrics")
			logger.Fatal("Prometheus exporter server failed", err)
		}
	}()

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create the Prometheus Exporter: %v", err)
	}

	return exporter, nil
}
