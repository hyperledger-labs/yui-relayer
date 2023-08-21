package core

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

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

	metricSrcProcessedBlockHeight *Int64SyncGauge
	metricDstProcessedBlockHeight *Int64SyncGauge
	metricSrcBacklogSize          *Int64SyncGauge
	metricDstBacklogSize          *Int64SyncGauge
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

	if metricSrcProcessedBlockHeight, err = newProcessedBlockHeightMetric(meter, "src"); err != nil {
		return err
	}
	if metricDstProcessedBlockHeight, err = newProcessedBlockHeightMetric(meter, "dst"); err != nil {
		return err
	}

	if metricSrcBacklogSize, err = newBacklogSizeMetric(meter, "src"); err != nil {
		return err
	}
	if metricDstBacklogSize, err = newBacklogSizeMetric(meter, "dst"); err != nil {
		return err
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

func newProcessedBlockHeightMetric(meter api.Meter, side string) (*Int64SyncGauge, error) {
	name := fmt.Sprintf("%s.%s.processed_block_height", namespaceRoot, side)
	if gauge, err := NewInt64SyncGauge(
		meter,
		name,
		0,
		api.WithUnit("1"),
		api.WithDescription(fmt.Sprintf("%s chain's latest finalized height", side)),
	); err != nil {
		return nil, fmt.Errorf("failed to create the metric %s: %v", name, err)
	} else {
		return gauge, nil
	}
}

func newBacklogSizeMetric(meter api.Meter, side string) (*Int64SyncGauge, error) {
	name := fmt.Sprintf("%s.%s.backlog_size", namespaceRoot, side)
	if gauge, err := NewInt64SyncGauge(
		meter,
		name,
		0,
		api.WithUnit("1"),
		api.WithDescription(fmt.Sprintf("%s chain's number of unreceived (or received but unfinalized) packets", side)),
	); err != nil {
		return nil, fmt.Errorf("failed to create the metric %s: %v", name, err)
	} else {
		return gauge, nil
	}
}

// Int64SyncGauge is an implementation of int64 synchronous gauge made from the official asynchronous one (Int64ObservableGauge).
// In the near future, the synchronous version will be supported officially, but is not now.
// Ref: https://github.com/open-telemetry/opentelemetry-go/issues/3984
type Int64SyncGauge struct {
	value *atomic.Int64
	gauge api.Int64ObservableGauge
}

// NewInt64SyncGauge creates a int64 synchronous gauge.
func NewInt64SyncGauge(meter api.Meter, name string, initialValue int64, options ...api.Int64ObservableGaugeOption) (*Int64SyncGauge, error) {
	value := atomic.Int64{}

	callback := func(ctx context.Context, observer api.Int64Observer) error {
		observer.Observe(value.Load())
		return nil
	}
	options = append(options, api.WithInt64Callback(callback))
	gauge, err := meter.Int64ObservableGauge(name, options...)
	if err != nil {
		return nil, err
	}

	return &Int64SyncGauge{&value, gauge}, nil
}

// Set sets the new value to the gauge in the thread safe manner.
func (g *Int64SyncGauge) Set(value int64) {
	g.value.Store(value)
}
