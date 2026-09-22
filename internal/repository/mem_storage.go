package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/k057ya/go-metrics/internal/model"
)

type MemStorage struct {
	data             map[string]model.Metrics
	path             *os.File
	restore          bool
	autosaveInterval time.Duration
	lastSave         time.Time
}

func (storage *MemStorage) Exists(key string) bool {
	_, exists := storage.data[key]
	return exists
}

func (storage *MemStorage) List() []model.Metrics {
	metrics := make([]model.Metrics, 0, len(storage.data))
	for _, v := range storage.data {
		metrics = append(metrics, v)
	}
	return metrics
}

func (storage *MemStorage) Get(key string) (model.Metrics, error) {
	metric, ok := storage.data[key]
	var err error
	if !ok {
		err = fmt.Errorf("key not found: %s", key)
	}
	return metric, err
}

func (storage *MemStorage) store(metrics model.Metrics) (model.Metrics, error) {
	storage.data[metrics.ID] = metrics
	storage.Persist()
	return storage.Get(metrics.ID)
}

func (storage *MemStorage) Update(key string, metrics model.Metrics) (model.Metrics, error) {

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

func (storage *MemStorage) Delete(key string) bool {
	delete(storage.data, key)
	return !storage.Exists(key)
}

func (storage *MemStorage) Clear() bool {
	storage.data = make(map[string]model.Metrics)
	return len(storage.data) == 0
}

func (storage *MemStorage) Persist() error {
	if storage.lastSave.IsZero() || time.Since(storage.lastSave) >= storage.autosaveInterval {
		list, err := json.Marshal(storage.List())
		if err != nil {
			return err
		}
		err = storage.path.Truncate(0)
		if err != nil {
			return err
		}
		_, err = storage.path.Seek(0, 0)
		if err != nil {
			return err
		}
		_, err = storage.path.Write(list)
		if err != nil {
			return err
		}
		storage.lastSave = time.Now()
	}
	return nil
}

func (storage *MemStorage) Restore() error {
	_, err := storage.path.Seek(0, 0)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(storage.path)
	if err != nil {
		return err
	}

	var metrics []model.Metrics

	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	restored := make(map[string]model.Metrics, len(metrics))

	for _, metric := range metrics {
		restored[metric.ID] = metric
	}

	storage.data = restored

	return nil
}

func NewMemStorage(path *os.File, restore bool, autosaveInterval time.Duration) *MemStorage {
	storage := &MemStorage{
		data:             make(map[string]model.Metrics),
		path:             path,
		restore:          restore,
		autosaveInterval: autosaveInterval,
	}
	if restore {
		err := storage.Restore()
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return nil
		}
	}
	return storage
}
