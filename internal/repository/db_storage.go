package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/k057ya/go-metrics/internal/model"
)

type DBStorage struct {
	connection *sql.DB
}

func (storage *DBStorage) List(ctx context.Context) []model.Metrics {
	return storage.list(ctx)
}

func (storage *DBStorage) list(ctx context.Context) []model.Metrics {
	var metrics []model.Metrics

	rows, err := storage.connection.QueryContext(ctx, "SELECT id, \"type\", delta, \"value\" FROM metrics")
	if err != nil {
		fmt.Printf("error getting metrics from database: %v", err)
	}
	defer rows.Close()

	if rows.Err() != nil {
		fmt.Printf("error getting metrics from database: %v", err)
	}

	for rows.Next() {
		var m model.Metrics
		err = rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
		if err != nil {
			fmt.Printf("error scanning metric row: %v", err)
			continue
		}

		metrics = append(metrics, m)
	}

	return metrics
}

func (storage *DBStorage) Get(ctx context.Context, key string) (model.Metrics, error) {
	return storage.get(ctx, key)
}

func (storage *DBStorage) get(ctx context.Context, key string) (model.Metrics, error) {

	row := storage.connection.QueryRowContext(
		ctx,
		"SELECT id, \"type\", delta, value from metrics WHERE id=$1 LIMIT 1",
		key,
	)
	var m model.Metrics

	err := row.Scan(&m.ID, &m.MType, &m.Delta, &m.Value)
	if err != nil {
		return model.Metrics{}, err
	}

	return m, err
}

func (storage *DBStorage) Update(ctx context.Context, _ string, m model.Metrics) (model.Metrics, error) {

	var (
		updated   model.Metrics
		validated []model.Metrics
	)
	if err := m.Validate(); err != nil {
		return updated, err
	}
	validated = append(validated, m)

	upserted, err := storage.upsert(ctx, validated)
	if err != nil || len(upserted) != 1 {
		return updated, err
	}

	updated = upserted[0]
	return updated, nil
}

func (storage *DBStorage) BatchUpdate(ctx context.Context, mlist []model.Metrics) ([]model.Metrics, error) {

	var validated []model.Metrics

	for _, m := range mlist {
		if err := m.Validate(); err != nil {
			continue
		}
		validated = append(validated, m)
	}

	return storage.upsert(ctx, validated)
}

func (storage *DBStorage) upsert(ctx context.Context, mlist []model.Metrics) ([]model.Metrics, error) {

	var updated []model.Metrics

	tx, err := storage.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(
		ctx,
		"INSERT INTO metrics (id, \"type\", delta, \"value\", updated_at) "+
			"VALUES ($1, $2, $3, $4, NOW()) "+
			"ON CONFLICT(id) "+
			"DO UPDATE SET "+
			"delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, "+
			"value = EXCLUDED.value, "+
			"updated_at = EXCLUDED.updated_at "+
			"WHERE metrics.type = EXCLUDED.type "+
			"RETURNING id, type, delta, value",
	)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for _, m := range mlist {

		var metric model.Metrics

		err := stmt.
			QueryRowContext(ctx, m.ID, m.MType, m.Delta, m.Value).
			Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)

		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"metric %q: type change is not supported", m.ID,
			)
		}
		if err != nil {
			return nil, fmt.Errorf("upsert error: %q %w", m.ID, err)
		}

		updated = append(updated, metric)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (storage *DBStorage) Ping(ctx context.Context) error {
	return storage.connection.PingContext(ctx)
}

func (storage *DBStorage) Close() error {
	return storage.connection.Close()
}

func (storage *DBStorage) Migrate(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	var storage = &DBStorage{
		connection: db,
	}
	err = storage.Migrate(db)
	return storage, err
}
