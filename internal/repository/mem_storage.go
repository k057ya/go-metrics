package repository

import (
	"errors"
	"fmt"

	"github.com/k057ya/go-metrics/internal/model"
)

type MetricsStorage interface {
	Exists(key string) bool
	Get(key string) (model.Metrics, error)
	Upsert(metrics model.Metrics) (model.Metrics, error)
	Put(key string, metrics model.Metrics) (bool, error)
	List() []model.Metrics
	Delete(key string) bool
	Clear() bool
}

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

func (storage MemStorage) Upsert(metrics model.Metrics) (model.Metrics, error) {
	storage.data[metrics.ID] = metrics
	return metrics, nil
}

func (storage MemStorage) Put(key string, metrics model.Metrics) (bool, error) {

	if storage.Exists(key) {
		savedMetric, err := storage.Get(key)

		if err != nil {
			return false, err
		}

		if savedMetric.MType != metrics.MType {
			return false, fmt.Errorf("metric type change is not supported: %s", savedMetric.MType)
		}

		switch metrics.MType {
		case model.MetricsTypeCounter:
			if metrics.Delta == nil || savedMetric.Delta == nil {
				return false, errors.New("counter delta is not specified")
			}

			delta := *metrics.Delta + *savedMetric.Delta
			metrics.Delta = &delta
		default:
		}
	}

	_, err := storage.Upsert(metrics)
	if err != nil {
		return false, err
	}

	return true, nil
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
