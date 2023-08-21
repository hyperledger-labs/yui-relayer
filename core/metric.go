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
	fallbackAddr = "localhost:0"
)

var (
	meterProvider *metric.MeterProvider
	meter         api.Meter
)

func InitializeMetrics(addr string) error {
	var err error

	if addr == "" {
		addr = fallbackAddr
	}
	meterProvider, err = NewPrometheusMeterProvider(addr)
	if err != nil {
		return fmt.Errorf("failed to create the MeterProvider with the Prometheus Exporter: %v", err)
	}

	meter = meterProvider.Meter(
		"github.com/hyperledger-labs/yui-relayer",
	)

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
