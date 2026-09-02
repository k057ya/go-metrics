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

func jsonEncode(value any) string {
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
		method      string
		url         string
		want        want
		contentType string
	}{
		{
			name: "#1 create counter",
			url:  "counter/counterVar1/1",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(1))}),
				contentType: "application/json",
			},
		},
		{
			name: "#2 increment counter",
			url:  "counter/counterVar1/1",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(2))}),
				contentType: "application/json",
			},
		},
		{
			name: "#3 increment counter by 5",
			url:  "counter/counterVar1/5",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "counterVar1", MType: "counter", Delta: new(int64(7))}),
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
			name: "#7 empty metric segment redirects",
			url:  "counter//123",
			want: want{
				code:     http.StatusTemporaryRedirect,
				response: "",
			},
		},
		{
			name: "#8 create gauge",
			url:  "gauge/gaugeVar1/1.25",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "gaugeVar1", MType: "gauge", Value: new(float64(1.25))}),
				contentType: "application/json",
			},
		},
		{
			name: "#9 replace gauge value",
			url:  "gauge/gaugeVar1/-3.5",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "gaugeVar1", MType: "gauge", Value: new(float64(-3.5))}),
				contentType: "application/json",
			},
		},
		{
			name: "#10 create zero counter",
			url:  "counter/zeroCounter/0",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "zeroCounter", MType: "counter", Delta: new(int64(0))}),
				contentType: "application/json",
			},
		},
		{
			name: "#11 add negative counter delta",
			url:  "counter/zeroCounter/-5",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "zeroCounter", MType: "counter", Delta: new(int64(-5))}),
				contentType: "application/json",
			},
		},
		{
			name: "#12 string value to gauge",
			url:  "gauge/invalidGauge/not-a-number",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#13 counter value overflow",
			url:  "counter/overflowCounter/9223372036854775808",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#14 counter cannot become gauge",
			url:  "gauge/counterVar1/1.5",
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric type change is not supported: counter\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#15 gauge cannot become counter",
			url:  "counter/gaugeVar1/1",
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric type change is not supported: gauge\n",
				contentType: "text/plain",
			},
		},
		{
			name:        "#16 unsupported content type",
			url:         "gauge/contentTypeGauge/1",
			contentType: "application/json",
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "invalid content-type\n",
				contentType: "text/plain",
			},
		},
		{
			name:   "#17 unsupported method",
			method: http.MethodGet,
			url:    "gauge/methodGauge/1",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "method is not supported by server\n",
				contentType: "text/plain",
			},
		},
		{
			name: "#18 scientific notation gauge",
			url:  "gauge/scientificGauge/1e3",
			want: want{
				code:        http.StatusOK,
				response:    jsonEncode(models.Metrics{ID: "scientificGauge", MType: "gauge", Value: new(float64(1000))}),
				contentType: "application/json",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := test.method
			if method == "" {
				method = http.MethodPost
			}

			request := httptest.NewRequest(
				method,
				"/update/"+test.url,
				nil,
			)

			requestContentType := test.contentType
			if requestContentType == "" {
				requestContentType = "text/plain"
			}
			request.Header.Set("Content-Type", requestContentType)

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
			if test.want.contentType == "" {
				assert.Empty(t, res.Header.Get("Content-Type"))
			} else {
				assert.Contains(t, res.Header.Get("Content-Type"), test.want.contentType)
			}
		})
	}
}
