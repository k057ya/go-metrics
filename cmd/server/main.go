package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/handler"
	"github.com/k057ya/go-metrics/internal/logger"
	"github.com/k057ya/go-metrics/internal/middleware"
	"github.com/k057ya/go-metrics/internal/repository"
)

type EnvConfig struct {
	Address       string `env:"ADDRESS"`
	StoreInterval string `env:"STORE_INTERVAL"`
	StoragePath   string `env:"FILE_STORAGE_PATH"`
	Restore       bool   `env:"RESTORE"`
	DatabaseDsn   string `env:"DATABASE_DSN"`
}

func main() {
	// Init logger
	if err := logger.Initialize("info"); err != nil {
		fmt.Printf("Error while initializing logger: `%s`\n", err)
		return
	}

	// Prepare config
	parseFlags()
	if err := parseEnv(); err != nil {
		fmt.Printf("Unable to parse ENV-vars, falling back to flag values: %s\n", err)
	}

	// Init storage
	storage, err := getStorage()
	if err != nil {
		fmt.Printf("Error while initializing storage: `%s`\n", err)
		return
	}

	// Init router
	router := newRouter(storage)

	// Start server
	fmt.Printf("Starting server on %s...", config.ServerConfig.String())
	err = http.ListenAndServe(config.ServerConfig.String(), router)

	if err != nil {
		fmt.Printf("Error starting server: `%s`\n", err)
	}
}

func newRouter(storage repository.Storage) *chi.Mux {
	router := chi.NewRouter()
	// Add Middlewares:
	// - chi middleware to strip trailing slash
	// - gzip compression middleware
	// - logger middleware for all routes
	router.Use(chimiddleware.StripSlashes, middleware.Compress, middleware.Log)

	listController := func(w http.ResponseWriter, req *http.Request) {
		handler.ListAllMetrics(w, req, storage)
	}
	valueController := func(w http.ResponseWriter, req *http.Request) {
		handler.PrintMetricHandler(w, req, storage)
	}
	updateController := func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateMetricsHandler(w, req, storage)
	}

	pingDBController := func(w http.ResponseWriter, req *http.Request) {
		handler.PingDB(w, req, storage)
	}

	// List all metrics
	router.Get("/", listController)

	// Get specific metric value
	router.Route("/value", func(router chi.Router) {
		router.Get("/{type}/{metric}", valueController) // Plain
		router.Post("/", valueController)               // JSON
	})

	// Insert or update metric
	router.Route("/update", func(router chi.Router) {
		router.Post("/{type}/{metric}/{value}", updateController) // Plain
		router.Post("/", updateController)                        // JSON
	})

	// Ping DB-connection
	router.Get("/ping", pingDBController)

	return router
}

func getStorage() (repository.Storage, error) {

	var storage repository.Storage

	// Init DB connection
	storage, err := repository.NewDBStorage(config.DatabaseConfig.Dsn)
	if err != nil {
		fmt.Printf("Error setting up database connection: %v \n", err)

		storage, err = repository.NewMemStorage(
			context.Background(),
			config.StorageConfig.BackupFilePath,
			config.StorageConfig.Restore,
			config.StorageConfig.BackupInterval,
		)
		if err != nil {
			fmt.Printf("Unable to initialize mem/file storage: %v \n", err)
			return nil, err
		}

	}

	return storage, nil
}

func parseEnv() error {
	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
		return err
	}
	if cfg.Address != "" {
		if err := config.ServerConfig.Set(cfg.Address); err != nil {
			fmt.Printf("cannot set %s from env, falling back to `%s`. Error: %s\n", "Address", config.ServerConfig.String(), err)
		}
	}
	if cfg.Restore {
		config.StorageConfig.Restore = cfg.Restore
	}
	if cfg.StoragePath != "" {
		config.StorageConfig.BackupFilePath = cfg.StoragePath
	}
	if cfg.StoreInterval != "" {
		if err := config.StorageConfig.SetBackupInterval(cfg.StoreInterval); err != nil {
			fmt.Printf("cannot set %s from env, falling back to `%s`. Error: %s\n", "StoreInterval", config.StorageConfig.BackupInterval.String(), err)
		}
	}
	if cfg.DatabaseDsn != "" {
		if err := config.DatabaseConfig.Set(cfg.DatabaseDsn); err != nil {
			fmt.Printf("cannot set %s from env. Error %s\n", "DatabaseDsn", err)
		}
	}
	return nil
}

func parseFlags() {
	flag.Var(config.ServerConfig, "a", "Server host and port")
	flag.Func("i", "Backup interval in seconds, `0` for sync",
		func(value string) error {
			if err := config.StorageConfig.SetBackupInterval(value); err != nil {
				fmt.Printf("invalid interval, using default %s\n", config.StorageConfig.BackupInterval)
			}
			return nil
		},
	)
	flag.StringVar(&config.StorageConfig.BackupFilePath, "f", "", "Path of a storage file")
	flag.BoolVar(&config.StorageConfig.Restore, "r", false, "`true` if need to restore at startup")
	flag.StringVar(&config.DatabaseConfig.Dsn, "d", "", "Database connection DSN string")
	flag.Parse()
}
