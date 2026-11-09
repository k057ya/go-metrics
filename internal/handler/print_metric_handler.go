package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
)

func PrintMetricHandler(resp http.ResponseWriter, req *http.Request, storage repository.StorageReader, log Logger) {

	var metricName string
	var metricType string

	if req.Method == http.MethodPost {
		// POST JSON
		var rm model.Metrics
		var buf bytes.Buffer
		// читаем тело запроса
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			log.Errorw("error reading body", err)
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		// десериализуем JSON в RequestMetric
		if err = json.Unmarshal(buf.Bytes(), &rm); err != nil {
			log.Warnw("error unmarsalling body", err)
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

	metric, err := storage.Get(req.Context(), metricName)

	if err != nil {
		var nfe *repository.NotFoundError
		if errors.As(err, &nfe) {
			resp.WriteHeader(http.StatusNotFound)
			resp.Write([]byte("Metric not found"))
			return
		}
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if metric.MType != metricType {
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Metric has invalid type"))
		return
	}

	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "text/plain", "":
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
		_, err = resp.Write([]byte(metric.StringValue()))
		if err != nil {
			log.Warnw("error writing response", err)
		}

	case "application/json":
		obj, err := json.Marshal(metric)
		if err != nil {
			log.Warnw("error marshalling response", err)
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
		_, err = resp.Write(obj)
		if err != nil {
			log.Warnw("error writing response", err)
		}
	}
}
