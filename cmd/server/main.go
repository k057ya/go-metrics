package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/handler"
	"github.com/k057ya/go-metrics/internal/repository"
)

type EnvConfig struct {
	Address string `env:"ADDRESS"`
}

func main() {

	storage := repository.NewMemStorage()

	flag.Var(config.ServerConfig, "a", "Server host and port")
	flag.Parse()

	var cfg EnvConfig
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Address != "" {
		err := config.ServerConfig.Set(cfg.Address)
		if err != nil {
			fmt.Println(err, ", falling back to", config.ServerConfig.String())
		}
	}

	router := chi.NewRouter()
	// List all metrics
	router.Get("/", func(w http.ResponseWriter, req *http.Request) {
		handler.ListAllMetrics(w, req, storage)
	})
	// Get specific metric value
	router.Get("/value/{type}/{metric}", func(w http.ResponseWriter, req *http.Request) {
		handler.PrintMetricHandler(w, req, storage)
	})
	// Insert or update metric
	router.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, req *http.Request) {
		handler.UpdateMetricsHandler(w, req, storage)
	})

	fmt.Println("Starting server on " + config.ServerConfig.String() + "...")

	err = http.ListenAndServe(config.ServerConfig.String(), router)

	if err != nil {
		fmt.Println("Error starting server: " + err.Error())
	}
}
