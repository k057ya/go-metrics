package handler

import (
	"database/sql"
	"net/http"
)

func PingDB(resp http.ResponseWriter, req *http.Request, db *sql.DB) {
	if err := db.Ping(); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(http.StatusOK)
}
