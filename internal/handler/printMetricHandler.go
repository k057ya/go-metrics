package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/model"
)

type StorageReader interface {
	Get(key string) (model.Metrics, error)
}

func PrintMetricHandler(resp http.ResponseWriter, req *http.Request, storage StorageReader) {

	var metricName string
	var metricType string

	if req.Method == http.MethodPost {
		// POST JSON
		var rm model.Metrics
		var buf bytes.Buffer
		// читаем тело запроса
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		// десериализуем JSON в RequestMetric
		if err = json.Unmarshal(buf.Bytes(), &rm); err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		metricName = rm.ID
		metricType = rm.MType

	} else {
		// GET URL PARAMS
		metricName = chi.URLParam(req, "metric")
		metricType = chi.URLParam(req, "type")
	}

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
	obj, err := json.Marshal(metric)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	resp.Write(obj)
}
