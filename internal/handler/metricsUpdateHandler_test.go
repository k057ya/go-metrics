package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func json_encode(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func TestUpdateMetricsHandler(t *testing.T) {

	storage := repository.NewMemStorage()
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/update/{type}/{metric}/{value}",
		func(w http.ResponseWriter, r *http.Request) {
			UpdateMetricsHandler(w, r, storage)
		},
	)

	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name        string
		url         string
		want        want
		contentType string
	}{
		{
			name: "#1 create counter",
			url:  "counter/counterVar1/1",
			want: want{
				code:        http.StatusOK,
				response:    json_encode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(1))}),
				contentType: "application/json",
			},
		},
		{
			name: "#2 increment counter",
			url:  "counter/counterVar1/1",
			want: want{
				code:        http.StatusOK,
				response:    json_encode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(2))}),
				contentType: "application/json",
			},
		},
		{
			name: "#3 increment counter by 5",
			url:  "counter/counterVar1/5",
			want: want{
				code:        http.StatusOK,
				response:    json_encode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(7))}),
				contentType: "application/json",
			},
		},
		{
			name: "#4 float value to counter",
			url:  "counter/counterVar1/1.1",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#5 string value to counter",
			url:  "counter/counterVar1/this-is-a-rhytm-of-the-night",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#6 unknown type",
			url:  "switcher/name/123",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#7 invalid name",
			url:  "counter//123",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(
				http.MethodPost,
				"/update/"+test.url,
				nil,
			)

			if test.contentType == "" {
				test.contentType = "text/plain"
			}
			request.Header.Set("Content-Type", test.contentType)

			// создаём новый Recorder
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, test.want.response, string(resBody))
			// грязный хак, так как сервер добавляет charset
			assert.Contains(t, res.Header.Get("Content-Type"), test.want.contentType)
		})
	}
}
