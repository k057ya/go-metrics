package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/model"
)

type StorageWriter interface {
	Update(key string, metrics model.Metrics) (model.Metrics, error)
}

func UpdateMetricsHandler(resp http.ResponseWriter, req *http.Request, storage StorageWriter) {
	// Проверить метод запроса
	if req.Method != http.MethodPost {
		http.Error(resp, "method is not supported by server", http.StatusMethodNotAllowed)
		return
	}

	// Проверить корректность заголовков
	contentType := req.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/plain" {
		http.Error(resp, "invalid content-type", http.StatusUnsupportedMediaType)
		return
	}

	// Извлечь значения из сегментов URL
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "metric")
	metricValue := chi.URLParam(req, "value")

	// Проверить заполненность имени метрики
	if metricName == "" {
		http.Error(resp, "metric name is not specified", http.StatusBadRequest)
		return
	}

	// Проверить корректность типа и значения метрики
	var (
		gaugeVal   float64
		counterVal int64
		err        error
	)
	Metric := model.Metrics{
		ID:    metricName,
		MType: metricType,
	}
	switch metricType {
	case model.MetricsTypeCounter:
		counterVal, err = strconv.ParseInt(metricValue, 10, 64)
		Metric.Delta = &counterVal

	case model.MetricsTypeGauge:
		gaugeVal, err = strconv.ParseFloat(metricValue, 64)
		Metric.Value = &gaugeVal

	default:
		http.Error(resp, "invalid metric type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(resp, "invalid metric value", http.StatusBadRequest)
		return
	}

	// Сохранить метрику в хранилище
	savedMetric, err := storage.Update(metricName, Metric)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// готовим вывод
	marshal, err := json.Marshal(savedMetric)
	if err != nil {
		http.Error(resp, "error marshalling json", http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	// устанавливаем код 200
	resp.WriteHeader(http.StatusOK)
	// пишем тело ответа
	_, err = resp.Write(marshal)
	if err != nil {
		return
	}

}
