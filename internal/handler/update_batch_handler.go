package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
)

func UpdateBatchHandler(resp http.ResponseWriter, req *http.Request, storage repository.StorageWriter, log Logger) {

	var mlist []model.Metrics

	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		log.Warnw("error reading request body", err)
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в mlist
	if err = json.Unmarshal(buf.Bytes(), &mlist); err != nil {
		log.Warnw("unable to unmarshal variable", err)
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	// Сохранить метрики в хранилище
	affected, err := storage.BatchUpdate(req.Context(), mlist)
	if err != nil {
		log.Warnw("unable to save batch", err)
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	// готовим вывод
	obj, err := json.Marshal(affected)
	if err != nil {
		log.Warnw("unable to marshal response", err)
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	_, err = resp.Write(obj)
	if err != nil {
		log.Warnw("unable to write response", err)
		return
	}

}
