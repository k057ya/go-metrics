package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/repository"
)

func PrintMetricHandler(resp http.ResponseWriter, req *http.Request, storage repository.MetricsStorage) {

	resp.Header().Set("Content-Type", "text/html; charset=utf-8")

	metricName := chi.URLParam(req, "metric")
	metricType := chi.URLParam(req, "type")
	metric, err := storage.Get(metricName)

	if err != nil {
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Metric not found"))
		return
	}

	if metric.MType != metricType {
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Metric has invalid type"))
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(metric.StringValue()))

}
