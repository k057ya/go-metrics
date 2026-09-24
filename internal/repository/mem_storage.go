package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/model"
)

type MemStorage struct {
	dataMu         sync.Mutex
	data           map[string]model.Metrics
	backupFilePath string
	syncBackup     bool
	backupMu       sync.Mutex
}

func (storage *MemStorage) List() []model.Metrics {
	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	return storage.list()
}

func (storage *MemStorage) list() []model.Metrics {
	metrics := make([]model.Metrics, 0, len(storage.data))
	for _, v := range storage.data {
		metrics = append(metrics, v)
	}
	return metrics
}

func (storage *MemStorage) Get(key string) (model.Metrics, error) {
	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	return storage.get(key)
}

func (storage *MemStorage) get(key string) (model.Metrics, error) {
	metric, ok := storage.data[key]
	var err error
	if !ok {
		err = fmt.Errorf("key not found: %s", key)
	}
	return metric, err
}

func (storage *MemStorage) Update(key string, metrics model.Metrics) (model.Metrics, error) {

	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	if savedMetric, err := storage.get(key); err == nil {

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

	storage.data[metrics.ID] = metrics

	// Backup data if sync backup is on
	if storage.syncBackup {
		err := storage.Backup(storage.list())
		if err != nil {
			return metrics, err
		}
	}
	return metrics, nil
}

func (storage *MemStorage) Delete(key string) bool {
	storage.dataMu.Lock()
	delete(storage.data, key)
	defer storage.dataMu.Unlock()
	_, exists := storage.data[key]
	return !exists
}

func (storage *MemStorage) Clear() bool {
	storage.dataMu.Lock()
	storage.data = make(map[string]model.Metrics)
	defer storage.dataMu.Unlock()
	return len(storage.data) == 0
}

func (storage *MemStorage) Backup(data []model.Metrics) error {
	storage.backupMu.Lock()
	defer storage.backupMu.Unlock()
	bak, err := storage.openBackupFile(os.O_RDWR | os.O_CREATE)
	if err != nil {
		return err
	}
	defer bak.Close()

	list, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = bak.Truncate(0)
	if err != nil {
		return err
	}
	_, err = bak.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = bak.Write(list)
	if err != nil {
		return err
	}
	return nil
}

func (storage *MemStorage) Restore() error {
	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()
	storage.backupMu.Lock()
	defer storage.backupMu.Unlock()

	bak, err := storage.openBackupFile(os.O_RDONLY | os.O_CREATE)
	if err != nil {
		return err
	}
	defer bak.Close()

	if _, err := bak.Seek(0, 0); err != nil {
		return err
	}
	data, err := io.ReadAll(bak)
	if err != nil {
		return err
	}
	// разрешить пустые файлы
	if len(data) == 0 {
		return nil
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

func (storage *MemStorage) openBackupFile(mask int) (*os.File, error) {
	var (
		err  error
		file *os.File
	)
	file, err = os.OpenFile(config.StorageConfig.BackupFilePath, mask, 0666)
	if err != nil {
		tmp, tmpErr := os.CreateTemp("", "go-metrics-storage-*")
		if tmpErr != nil {
			return nil, fmt.Errorf("cannot open storage: %v; cannot create temp file: %v",
				err, tmpErr)
		}

		tmpPath, err := filepath.Abs(tmp.Name())
		if err != nil {
			tmp.Close()
			return nil, fmt.Errorf("cannot get temp file path: %v", err)
		}

		fmt.Printf(
			"Error opening storage file: %v. Falling back to temp file: %s\n",
			err,
			tmpPath,
		)

		// дальше работаем с временным файлом
		file = tmp
		storage.backupFilePath = tmpPath
	}
	return file, nil
}

func NewMemStorage(backupFilePath string, restore bool, backupInterval time.Duration) (*MemStorage, error) {

	syncBackup := backupInterval == 0

	storage := &MemStorage{
		data:           make(map[string]model.Metrics),
		backupFilePath: backupFilePath,
		syncBackup:     syncBackup,
	}
	if restore {
		err := storage.Restore()
		if err != nil {
			return nil, fmt.Errorf("restore storage: %w", err)
		}
	}

	if !syncBackup {
		go func() {
			for {
				time.Sleep(backupInterval)
				data := storage.List()
				storage.Backup(data)
			}
		}()
	}

	return storage, nil
}
