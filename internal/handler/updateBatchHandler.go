package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
)

func UpdateBatchHandler(resp http.ResponseWriter, req *http.Request, storage repository.StorageWriter) {

	var mlist []model.Metrics

	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в mlist
	if err = json.Unmarshal(buf.Bytes(), &mlist); err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	// Сохранить метрики в хранилище
	affected, err := storage.BatchUpdate(req.Context(), mlist)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// готовим вывод
	obj, err := json.Marshal(affected)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	resp.Write(obj)

}
