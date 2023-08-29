package metrics

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
)

var (
	meterProvider *metric.MeterProvider
	meter         api.Meter

	ProcessedBlockHeightGauge      *Int64SyncGauge
	BacklogSizeGauge               *Int64SyncGauge
	BacklogOldestTimestampGauge    *Int64SyncGauge
	ReceivePacketsFinalizedCounter api.Int64Counter
)

type ExporterConfig interface {
	exporterType() string
}

type ExporterNull struct{}

func (e ExporterNull) exporterType() string { return "null" }

type ExporterProm struct {
	Addr string
}

func (e ExporterProm) exporterType() string { return "prometheus" }

func InitializeMetrics(exporterConf ExporterConfig) error {
	var err error

	switch exporterConf := exporterConf.(type) {
	case ExporterNull:
		meterProvider = metric.NewMeterProvider()
	case ExporterProm:
		if exporter, err := NewPrometheusExporter(exporterConf.Addr); err != nil {
			return err
		} else {
			meterProvider = metric.NewMeterProvider(metric.WithReader(exporter))
		}
	default:
		panic("unexpected exporter type")
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

func NewPrometheusExporter(addr string) (*prometheus.Exporter, error) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil {
			// TODO: we should replace this with a more proper logger, slog?
			log.Fatalf("Prometheus exporter server failed: %v", err)
		}
	}()

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create the Prometheus Exporter: %v", err)
	}

	return exporter, nil
}
