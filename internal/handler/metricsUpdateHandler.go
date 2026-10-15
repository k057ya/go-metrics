package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
)

func UpdateMetricsHandler(resp http.ResponseWriter, req *http.Request, storage repository.StorageWriter) {

	var metric model.Metrics

	// Проверить метод запроса
	if req.Method != http.MethodPost {
		http.Error(resp, "method is not supported by server", http.StatusMethodNotAllowed)
		return
	}

	// Проверить корректность заголовков
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "text/plain", "":
		// Извлечь значения из сегментов URL
		urlValue := chi.URLParam(req, "value")

		metric = model.Metrics{
			ID:    chi.URLParam(req, "metric"),
			MType: chi.URLParam(req, "type"),
		}
		if err := metric.ValidateType(); err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		err := metric.Parse(urlValue)
		if err != nil {
			http.Error(resp, "invalid metric value", http.StatusBadRequest)
			return
		}

	case "application/json":
		var buf bytes.Buffer
		// читаем тело запроса
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		// десериализуем JSON в Metric
		if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}

	default:
		http.Error(resp, "invalid content-type", http.StatusUnsupportedMediaType)
		return
	}

	// Проверить заполненность имени метрики
	if metric.ID == "" {
		http.Error(resp, "metric name is not specified", http.StatusBadRequest)
		return
	}

	if err := metric.ValidateType(); err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	if err := metric.ValidateValue(); err != nil {
		http.Error(resp, "invalid metric value", http.StatusBadRequest)
		return
	}

	// Сохранить метрику в хранилище
	savedMetric, err := storage.Update(req.Context(), metric.ID, metric)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// готовим вывод
	obj, err := json.Marshal(savedMetric)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	resp.Write(obj)
}
