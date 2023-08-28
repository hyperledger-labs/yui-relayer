package core

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"
)

type Int64WithAttributes struct {
	value int64
	attrs attribute.Set
}

type Int64SyncGauge struct {
	gauge         api.Int64ObservableGauge
	mutex         *sync.RWMutex
	attrsValueMap map[string]*Int64WithAttributes
}

func NewInt64SyncGauge(meter api.Meter, name string, options ...api.Int64ObservableGaugeOption) (*Int64SyncGauge, error) {
	mutex := &sync.RWMutex{}
	attrsValueMap := make(map[string]*Int64WithAttributes)
	callback := func(ctx context.Context, observer api.Int64Observer) error {
		mutex.RLock()
		defer mutex.RUnlock()
		for _, entry := range attrsValueMap {
			observer.Observe(entry.value, api.WithAttributeSet(entry.attrs))
		}
		return nil
	}
	options = append(options, api.WithInt64Callback(callback))
	gauge, err := meter.Int64ObservableGauge(name, options...)
	if err != nil {
		return nil, err
	}
	return &Int64SyncGauge{gauge, mutex, attrsValueMap}, nil
}

func (g *Int64SyncGauge) Set(value int64, attr ...attribute.KeyValue) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	attrs := attribute.NewSet(attr...)
	encodedAttrs := attrs.Encoded(attribute.DefaultEncoder())
	g.attrsValueMap[encodedAttrs] = &Int64WithAttributes{value, attrs}
}
