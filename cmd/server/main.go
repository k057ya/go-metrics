package main

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/handler"
	"github.com/k057ya/go-metrics/internal/repository"
)

func main() {

	storage := repository.NewMemStorage()

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
	err := http.ListenAndServe(":"+strconv.Itoa(config.ServerConfig.Port), router)

	if err != nil {
		panic(err)
	}
}
