package repository

import (
	"errors"
	"fmt"

	"github.com/k057ya/go-metrics/internal/model"
)

type MemStorage struct {
	data map[string]model.Metrics
}

func (storage MemStorage) Exists(key string) bool {
	_, exists := storage.data[key]
	return exists
}

func (storage MemStorage) List() []model.Metrics {
	metrics := make([]model.Metrics, 0, len(storage.data))
	for _, v := range storage.data {
		metrics = append(metrics, v)
	}
	return metrics
}

func (storage MemStorage) Get(key string) (model.Metrics, error) {
	metric, ok := storage.data[key]
	var err error
	if !ok {
		err = fmt.Errorf("key not found: %s", key)
	}
	return metric, err
}

func (storage MemStorage) store(metrics model.Metrics) (model.Metrics, error) {
	storage.data[metrics.ID] = metrics
	return storage.Get(metrics.ID)
}

func (storage MemStorage) Update(key string, metrics model.Metrics) (model.Metrics, error) {

	if storage.Exists(key) {
		savedMetric, err := storage.Get(key)

		if err != nil {
			return metrics, err
		}

		if savedMetric.MType != metrics.MType {
			return metrics, fmt.Errorf("metric type change is not supported: %s", savedMetric.MType)
		}

		switch metrics.MType {
		case model.MetricsTypeCounter:
			if metrics.Delta == nil || savedMetric.Delta == nil {
				return metrics, errors.New("counter delta is not specified")
			}

			delta := *metrics.Delta + *savedMetric.Delta
			metrics.Delta = &delta
		default:
		}
	}

	return storage.store(metrics)
}

func (storage MemStorage) Delete(key string) bool {
	delete(storage.data, key)
	return !storage.Exists(key)
}

func (storage MemStorage) Clear() bool {
	storage.data = make(map[string]model.Metrics)
	return len(storage.data) == 0
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]model.Metrics),
	}
}
