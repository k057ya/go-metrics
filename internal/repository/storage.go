package repository

import (
	"context"

	"github.com/k057ya/go-metrics/internal/model"
)

type Storage interface {
	StorageReader
	StorageWriter
	StorageLister
}

type StorageReader interface {
	Get(ctx context.Context, key string) (model.Metrics, error)
}

type StorageWriter interface {
	Update(ctx context.Context, key string, metrics model.Metrics) (model.Metrics, error)
}

type StorageLister interface {
	List(ctx context.Context) []model.Metrics
}

type StorageDBPinger interface {
	PingDB()
}
