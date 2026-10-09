package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/k057ya/go-metrics/internal/model"
)

type DBStorage struct {
	dataMu     sync.Mutex
	connection *sql.DB
}

func (storage *DBStorage) List(ctx context.Context) []model.Metrics {
	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	return storage.list(ctx)
}

func (storage *DBStorage) list(ctx context.Context) []model.Metrics {
	var metrics []model.Metrics

	rows, err := storage.connection.QueryContext(ctx, "SELECT `id`, `type`, `delta`, `value` from `metrics`")
	if err != nil {
		fmt.Printf("error getting metrics from database: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m model.Metrics
		err = rows.Scan(&m.ID, m.MType, &m.Delta, &m.Value)
		if err != nil {
			fmt.Printf("error scanning metric row: %v", err)
		}

		metrics = append(metrics, m)
	}

	return metrics
}

func (storage *DBStorage) Get(ctx context.Context, key string) (model.Metrics, error) {
	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	return storage.get(ctx, key)
}

func (storage *DBStorage) get(ctx context.Context, key string) (model.Metrics, error) {

	row := storage.connection.QueryRowContext(
		ctx,
		"SELECT `id`, `type`, `delta`, `value` from `metrics` WHERE `id`=$1 LIMIT 1",
		key,
	)
	var m model.Metrics

	err := row.Scan(&m.ID, m.MType, &m.Delta, &m.Value)
	if err != nil {
		return model.Metrics{}, err
	}

	return m, err
}

func (storage *DBStorage) Update(ctx context.Context, key string, m model.Metrics) (model.Metrics, error) {

	storage.dataMu.Lock()
	defer storage.dataMu.Unlock()

	if savedMetric, err := storage.get(ctx, key); err == nil {

		if savedMetric.MType != m.MType {
			return m, fmt.Errorf("metric type change is not supported: %s", savedMetric.MType)
		}

		switch m.MType {
		case model.MetricsTypeCounter:
			if m.Delta == nil || savedMetric.Delta == nil {
				return m, errors.New("counter delta is not specified")
			}

			delta := *m.Delta + *savedMetric.Delta
			m.Delta = &delta
		default:
		}
	}

	result, err := storage.connection.ExecContext(
		ctx,
		"INSERT INTO metrics (id, type, delta, value) "+
			"VALUES ($1, $2, $3, $4) "+
			"ON CONFLICT(id) "+
			"DO UPDATE SET "+
			"delta = EXCLUDED.delta, "+
			"value = EXCLUDED.value",
		m.ID,
		m.MType,
		m.Delta,
		m.Value,
	)
	if err != nil {
		return m, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return m, err
	}
	if affected == 0 {
		return m, errors.New("error saving metrics")
	}
	return m, nil
}

func (storage *DBStorage) Ping(ctx context.Context) error {
	return storage.connection.PingContext(ctx)
}

func NewDbStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	var storage = &DBStorage{
		connection: db,
	}
	return storage, nil
}
