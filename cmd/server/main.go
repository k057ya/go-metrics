package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

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
}

func main() {
	// Init logger
	if err := logger.Initialize("info"); err != nil {
		fmt.Printf("Error while initializing logger: `%s`\n", err)
		return
	}

	// Prepare config
	ParseFlags()
	if err := ParseEnv(); err != nil {
		fmt.Printf("Unable to parse ENV-vars, falling back to flag values: %s\n", err)
	}

	// Init storage
	storage, err := repository.NewMemStorage(
		config.StorageConfig.BackupFilePath,
		config.StorageConfig.Restore,
		config.StorageConfig.BackupInterval,
	)
	if err != nil {
		fmt.Printf("Unable to initialize storage: %v \n", err)
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

func newRouter(storage *repository.MemStorage) *chi.Mux {
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
	return router
}

func ParseEnv() error {
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
		err := config.StorageConfig.SetBackupInterval(cfg.StoreInterval)
		if err != nil {
			fmt.Printf("cannot set %s from env, falling back to `%s`. Error: %s\n", "StoreInterval", config.StorageConfig.BackupInterval.String(), err)
		}
	}
	return nil
}

func ParseFlags() {
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
	flag.Parse()
}
