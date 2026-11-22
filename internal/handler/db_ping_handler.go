package handler

import (
	"net/http"

	"github.com/k057ya/go-metrics/internal/repository"
)

func PingDB(resp http.ResponseWriter, req *http.Request, storage repository.Storage) {

	dbStorage, ok := storage.(*repository.DBStorage)

	if !ok || dbStorage == nil {
		http.Error(resp, "database storage is not configured",
			http.StatusInternalServerError)
		return
	}

	if err := dbStorage.Ping(req.Context()); err != nil {
		http.Error(resp, "database is unavailable",
			http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
}
