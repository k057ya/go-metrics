package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	models "github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
)

func UpdateMetricsHandler(resp http.ResponseWriter, req *http.Request, storage repository.MetricsStorage) {
	// Проверить метод запроса
	if req.Method != http.MethodPost {
		http.Error(resp, "method is not supported by server", http.StatusMethodNotAllowed)
		return
	}

	// Проверить корректность заголовков
	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(resp, "invalid content-type", http.StatusUnsupportedMediaType)
		return
	}

	// Извлечь значения из сегментов URL
	metricType := req.PathValue("type")
	metricName := req.PathValue("metric")
	metricValue := req.PathValue("value")

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
	Metric := models.Metrics{
		ID:    metricName,
		MType: metricType,
	}
	switch metricType {
	case models.MetricsTypeCounter:
		counterVal, err = strconv.ParseInt(metricValue, 10, 64)
		Metric.Delta = &counterVal

	case models.MetricsTypeGauge:
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
	_, err = storage.Put(metricName, Metric)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	savedMetric, _ := storage.Get(metricName)
	marshal, err := json.Marshal(savedMetric)

	if err != nil {
		http.Error(resp, "error marshalling json", http.StatusInternalServerError)
	}
	resp.Header().Set("content-type", "application/json")
	// устанавливаем код 200
	resp.WriteHeader(http.StatusOK)
	// пишем тело ответа
	_, err = resp.Write(marshal)
	if err != nil {
		return
	}

}
