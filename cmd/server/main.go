package main

import (
	"net/http"

	"github.com/k057ya/go-metrics/internal/handler"
	"github.com/k057ya/go-metrics/internal/repository"
)

func main() {

	storage := repository.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{type}/{metric}/{value}`, func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateMetricsHandler(w, r, storage)
	})
	// ...
	err := http.ListenAndServe(`:8080`, mux)
	// ...

	if err != nil {
		panic(err)
	}
}
