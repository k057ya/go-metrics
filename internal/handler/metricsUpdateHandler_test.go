package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/k057ya/go-metrics/internal/agent"
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
	router := chi.NewRouter()
	router.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, r *http.Request) {
		UpdateMetricsHandler(w, r, storage)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	var httpClient = agent.NewHttpClient()
	httpClient.
		SetBaseURL(server.URL + "/update/")

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
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#5 string value to counter",
			url:  "counter/counterVar1/this-is-a-rhytm-of-the-night",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#6 unknown type",
			url:  "switcher/name/123",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#7 empty metric segment redirects",
			url:  "counter//123",
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric name is not specified\n",
				contentType: "text/plain; charset=utf-8",
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
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#13 counter value overflow",
			url:  "counter/overflowCounter/9223372036854775808",
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#14 counter cannot become gauge",
			url:  "gauge/counterVar1/1.5",
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric type change is not supported: counter\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "#15 gauge cannot become counter",
			url:  "counter/gaugeVar1/1",
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric type change is not supported: gauge\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "#16 unsupported content type",
			url:         "gauge/contentTypeGauge/1",
			contentType: "application/json",
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "invalid content-type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "#17 unsupported method",
			method: http.MethodGet,
			url:    "gauge/methodGauge/1",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
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

			request := httpClient.R().SetBody("")

			method := test.method
			if method == "" {
				method = http.MethodPost
			}

			requestContentType := test.contentType
			if requestContentType == "" {
				requestContentType = "text/plain"
			}
			request.SetHeader("Content-Type", requestContentType)

			response, err := request.Execute(method, test.url)
			require.NoError(t, err)

			assert.Equal(t, test.want.response, string(response.Body()))

			assert.Equal(t, test.want.contentType, response.Header().Get("Content-Type"))

			assert.Equal(t, test.want.code, response.StatusCode())
		})
	}
}
