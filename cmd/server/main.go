package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v6"
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
	StoreInterval int    `env:"STORE_INTERVAL"`
	StoragePath   string `env:"FILE_STORAGE_PATH"`
	Restore       bool   `env:"RESTORE"`
}

func main() {

	// Make config
	ParseFlags()
	if err := ParseEnv(); err != nil {
		fmt.Println("Unable to parse ENV-vars, falling back to flag values " + err.Error())
	}

	// Init storage
	file, err := os.OpenFile(config.StorageConfig.StoragePath.String(), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		tmp, tmpErr := os.CreateTemp("", "go-metrics-storage-*")
		if tmpErr != nil {
			fmt.Printf("Cannot open storage: %v; cannot create temp file: %v\n",
				err, tmpErr)
			return
		}

		fullPath, pathErr := filepath.Abs(tmp.Name())
		if pathErr != nil {
			tmp.Close()
			fmt.Printf("Cannot get temp file path: %v\n", pathErr)
			return
		}

		fmt.Printf(
			"Error opening storage file: %v. Falling back to temp file: %s\n",
			err,
			fullPath,
		)

		file = tmp // дальше работаем с временным файлом
	}
	defer file.Close()

	storage := repository.NewMemStorage(
		file,
		bool(config.StorageConfig.Restore),
		config.StorageConfig.StoreInterval.Interval,
	)
	if err := logger.Initialize("info"); err != nil {
		fmt.Println("Error while initializing logger " + err.Error())
		return
	}

	// Init router
	router := chi.NewRouter()
	// Add Middlewares:
	// - chi middleware to strip trailing slash
	// - gzip compression middleware
	// - logger middleware for all routes
	router.Use(chimiddleware.StripSlashes, middleware.Compress, middleware.Log)

	// List all metrics
	router.Get("/", func(w http.ResponseWriter, req *http.Request) {
		handler.ListAllMetrics(w, req, storage)
	})
	// Get specific metric value
	router.Get("/value/{type}/{metric}", func(w http.ResponseWriter, req *http.Request) {
		handler.PrintMetricHandler(w, req, storage)
	})
	// JSON: Get specific metric value
	router.Post("/value", func(w http.ResponseWriter, req *http.Request) {
		handler.PrintMetricHandler(w, req, storage)
	})
	// Insert or update metric
	router.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateMetricsHandler(w, req, storage)
	})
	// JSON: Insert or update metric
	router.Post("/update", func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateMetricsHandler(w, req, storage)
	})

	fmt.Println("Starting server on " + config.ServerConfig.String() + "...")

	err = http.ListenAndServe(config.ServerConfig.String(), router)

	if err != nil {
		fmt.Println("Error starting server: " + err.Error())
	}
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
		config.StorageConfig.Restore = config.StorageRestore(cfg.Restore)
	}
	if cfg.StoragePath != "" {
		if err := config.StorageConfig.StoragePath.Set(cfg.StoragePath); err != nil {
			fmt.Printf("cannot set %s from env, falling back to `%s`. Error: %s\n", "StoragePath", config.StorageConfig.StoragePath.String(), err)
		}
	}
	if cfg.StoreInterval >= 0 {
		if err := config.StorageConfig.StoragePath.Set(cfg.StoragePath); err != nil {
			fmt.Printf("cannot set %s from env, falling back to `%s`. Error: %s\n", "StoragePath", config.StorageConfig.StoragePath.String(), err)
		}
	}
	return nil
}

func ParseFlags() {
	flag.Var(config.ServerConfig, "a", "Server host and port")
	flag.Var(&config.StorageConfig.StoreInterval, "i", "Store interval in seconds, `0` for sync")
	flag.Var(&config.StorageConfig.StoragePath, "f", "Path of a storage file")
	flag.Var(&config.StorageConfig.Restore, "r", "`true` if need to restore at startup")
	flag.Parse()
}
