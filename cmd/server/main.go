package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/handler"
	"github.com/k057ya/go-metrics/internal/logger"
	"github.com/k057ya/go-metrics/internal/middleware"
	"github.com/k057ya/go-metrics/internal/repository"
	"go.uber.org/zap"
)

type EnvConfig struct {
	Address       string `env:"ADDRESS"`
	StoreInterval string `env:"STORE_INTERVAL"`
	StoragePath   string `env:"FILE_STORAGE_PATH"`
	Restore       bool   `env:"RESTORE"`
	DatabaseDsn   string `env:"DATABASE_DSN"`
	SignatureKey  string `env:"KEY"`
}

func main() {
	// Init logger
	if err := logger.Initialize("info"); err != nil {
		fmt.Fprintln(os.Stderr, "Error while initializing logger:", err)
		return
	}
	defer logger.Log.Sync()

	// Prepare config
	parseFlags()
	if err := parseEnv(); err != nil {
		logger.Log.Warn("Unable to parse ENV-vars, falling back to flag values", zap.Error(err))
	}

	// Init storage
	storage, err := getStorage()
	if err != nil {
		logger.Log.Error("Error while initializing storage", zap.Error(err))
		return
	}

	// Init router
	router := newRouter(storage, config.SignatureConfig.Key)

	// Start server
	logger.Log.Sugar().Infof("Starting server on %s...", config.ServerConfig.String())
	err = http.ListenAndServe(config.ServerConfig.String(), router)

	if err != nil {
		logger.Log.Error("Error starting server", zap.Error(err))
	}
}

func newRouter(storage repository.Storage, signatureKey string) *chi.Mux {
	router := chi.NewRouter()
	// Add Middlewares:
	// - chi middleware to strip trailing slash
	router.Use(chimiddleware.StripSlashes)

	// - gzip compression middleware
	router.Use(middleware.Compress(logger.Log.Sugar().Named("gzip")))
	// If key provided - require and validate signature header, sign responses
	if signatureKey != "" {
		router.Use(middleware.CheckAndSign(signatureKey, logger.Log.Sugar().Named("signer")))
	}

	// - logger middleware for all routes
	router.Use(middleware.Log)

	handlerLog := logger.Log.Sugar().Named("handler")

	listController := func(w http.ResponseWriter, req *http.Request) {
		handler.ListAllMetrics(w, req, storage, handlerLog)
	}
	valueController := func(w http.ResponseWriter, req *http.Request) {
		handler.PrintMetricHandler(w, req, storage, handlerLog)
	}
	updateController := func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateMetricsHandler(w, req, storage, handlerLog)
	}

	pingDBController := func(w http.ResponseWriter, req *http.Request) {
		handler.PingDB(w, req, storage)
	}

	updateBatchController := func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateBatchHandler(w, req, storage, handlerLog)
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

	router.Post("/updates", updateBatchController)

	// Ping DB-connection
	router.Get("/ping", pingDBController)

	return router
}

func getStorage() (repository.Storage, error) {
	var (
		storage repository.Storage
		err     error
	)

	// Init DB connection
	if config.DatabaseConfig.Dsn != "" {
		storage, err = repository.NewDBStorage(config.DatabaseConfig.Dsn)
		if err != nil {
			logger.Log.Warn("Error setting up database connection", zap.Error(err))
		} else {
			return storage, err
		}
	}

	// Init mem storage
	storage, err = repository.NewMemStorage(
		context.Background(),
		config.StorageConfig.BackupFilePath,
		config.StorageConfig.Restore,
		config.StorageConfig.BackupInterval,
		logger.Log.Sugar().Named("mem_storage"),
	)
	if err != nil {
		logger.Log.Warn("Unable to initialize mem/file storage", zap.Error(err))
		return nil, err
	}

	return storage, nil
}

func parseEnv() error {
	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		return err
	}
	if cfg.Address != "" {
		if err := config.ServerConfig.Set(cfg.Address); err != nil {
			logger.Log.Sugar().Warnf("cannot set %s from env, falling back to `%s`. Error: %s", "Address", config.ServerConfig.String(), err)
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
			logger.Log.Sugar().Warnf("cannot set %s from env, falling back to `%s`. Error: %s", "StoreInterval", config.StorageConfig.BackupInterval.String(), err)
		}
	}
	if cfg.DatabaseDsn != "" {
		if err := config.DatabaseConfig.Set(cfg.DatabaseDsn); err != nil {
			logger.Log.Sugar().Warnf("cannot set %s from env. Error %s", "DatabaseDsn", err)
		}
	}
	if cfg.SignatureKey != "" {
		config.SignatureConfig.Key = cfg.SignatureKey
	}
	return nil
}

func parseFlags() {
	flag.Var(config.ServerConfig, "a", "Server host and port")
	flag.Func("i", "Backup interval in seconds, `0` for sync",
		func(value string) error {
			if err := config.StorageConfig.SetBackupInterval(value); err != nil {
				logger.Log.Sugar().Warnf("invalid interval, using default %s", config.StorageConfig.BackupInterval)
			}
			return nil
		},
	)
	flag.StringVar(&config.StorageConfig.BackupFilePath, "f", "", "Path of a storage file")
	flag.BoolVar(&config.StorageConfig.Restore, "r", false, "`true` if need to restore at startup")
	flag.StringVar(&config.DatabaseConfig.Dsn, "d", "", "Database connection DSN string")
	flag.StringVar(&config.SignatureConfig.Key, "k", "", "Key for signing requests")
	flag.Parse()
}
