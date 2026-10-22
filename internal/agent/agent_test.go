package agent

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/k057ya/go-metrics/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestFetchMetrics(t *testing.T) {

	agent := Agent{}

	tests := []struct {
		name     string
		wantType string
	}{
		{name: "Alloc", wantType: "gauge"},
		{name: "BuckHashSys", wantType: "gauge"},
		{name: "Frees", wantType: "gauge"},
		{name: "GCCPUFraction", wantType: "gauge"},
		{name: "GCSys", wantType: "gauge"},
		{name: "HeapAlloc", wantType: "gauge"},
		{name: "HeapIdle", wantType: "gauge"},
		{name: "HeapInuse", wantType: "gauge"},
		{name: "HeapObjects", wantType: "gauge"},
		{name: "HeapReleased", wantType: "gauge"},
		{name: "HeapSys", wantType: "gauge"},
		{name: "LastGC", wantType: "gauge"},
		{name: "Lookups", wantType: "gauge"},
		{name: "MCacheInuse", wantType: "gauge"},
		{name: "MCacheSys", wantType: "gauge"},
		{name: "MSpanInuse", wantType: "gauge"},
		{name: "MSpanSys", wantType: "gauge"},
		{name: "Mallocs", wantType: "gauge"},
		{name: "NextGC", wantType: "gauge"},
		{name: "NumForcedGC", wantType: "gauge"},
		{name: "NumGC", wantType: "gauge"},
		{name: "OtherSys", wantType: "gauge"},
		{name: "PauseTotalNs", wantType: "gauge"},
		{name: "StackInuse", wantType: "gauge"},
		{name: "StackSys", wantType: "gauge"},
		{name: "Sys", wantType: "gauge"},
		{name: "TotalAlloc", wantType: "gauge"},
		{name: "PollCount", wantType: "counter"},
		{name: "RandomValue", wantType: "gauge"},
	}

	got := agent.fetchMetrics()
	require.Len(t, got, len(tests))

	metricsByID := make(map[string]Metric, len(got))
	for _, metric := range got {
		_, exists := metricsByID[metric.ID]
		assert.False(t, exists, "metric %q is returned more than once", metric.ID)
		metricsByID[metric.ID] = metric
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metric, exists := metricsByID[test.name]
			require.True(t, exists, "metric %q is missing", test.name)
			assert.Equal(t, test.wantType, metric.Type)

			if metric.Type == "counter" {
				_, err := strconv.ParseInt(metric.Value, 10, 64)
				require.NoError(t, err)
				return
			}

			_, err := strconv.ParseFloat(metric.Value, 64)
			require.NoError(t, err)
		})
	}

	assert.Equal(t, "1", metricsByID["PollCount"].Value)
	randomValue, err := strconv.ParseFloat(metricsByID["RandomValue"].Value, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, randomValue, 0.0)
	assert.Less(t, randomValue, 1.0)
}

func TestSendMetric(t *testing.T) {
	type receivedRequest struct {
		method          string
		path            string
		contentType     string
		contentEncoding string
		acceptEncoding  string
		body            []byte
	}

	tests := []struct {
		name         string
		metric       Metric
		statusCode   int
		gzip         bool
		transportErr error
		wantPath     string
		wantErr      bool
		wantReqBody  []byte
	}{
		{
			name:       "#1 send gauge",
			metric:     Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			statusCode: http.StatusOK,
			gzip:       true,
		},
		{
			name:       "#2 send counter",
			metric:     Metric{ID: "PollCount", Type: "counter", Value: "1"},
			statusCode: http.StatusOK,
		},
		{
			name:       "#3 server returns error",
			metric:     Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:         "#4 server is unavailable",
			metric:       Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			transportErr: errors.New("server is unavailable"),
			wantErr:      true,
		},
		{
			name:       "#5 gzip request",
			metric:     Metric{ID: "Alloc", Type: "gauge", Value: "12.54"},
			statusCode: http.StatusOK,
			gzip:       true,
		},
		{
			name:        "#6 body",
			metric:      Metric{ID: "Alloc", Type: "gauge", Value: "12.54"},
			statusCode:  http.StatusOK,
			wantReqBody: []byte(`{"id":"Alloc","type":"gauge","value":12.54}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			var received receivedRequest

			client := NewHTTPClient("http://metrics.test")

			client.SetTransport(roundTripFunc(func(request *http.Request) (*http.Response, error) {

				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)

				received = receivedRequest{
					method:          request.Method,
					path:            request.URL.Path,
					contentType:     request.Header.Get("Content-Type"),
					contentEncoding: request.Header.Get("Content-Encoding"),
					acceptEncoding:  request.Header.Get("Accept-Encoding"),
					body:            body,
				}

				if test.transportErr != nil {
					return nil, test.transportErr
				}

				return &http.Response{
					StatusCode: test.statusCode,
					Status:     strconv.Itoa(test.statusCode) + " " + http.StatusText(test.statusCode),
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
					Request:    request,
				}, nil
			}))

			err := sendMetric(test.metric, client)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, http.MethodPost, received.method)

			assert.Equal(t, "application/json", received.contentType)
			assert.Equal(t, "gzip", received.acceptEncoding)
			assert.Equal(t, "gzip", received.contentEncoding)

			if len(test.wantReqBody) > 0 {
				var compressed bytes.Buffer
				gzw := gzip.NewWriter(&compressed)
				_, err := gzw.Write(test.wantReqBody)
				require.NoError(t, err)
				err = gzw.Close()
				require.NoError(t, err)

				assert.Equal(t, compressed.Bytes(), received.body)
			}
		})
	}
}

func TestSendMetricMalformedGzipBody(t *testing.T) {
	client := NewHTTPClient("http://metrics.test")

	client.SetTransport(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := "this is not gzip"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Encoding": []string{"gzip"},
			},
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       request,
		}, nil
	}))

	err := sendMetric(
		Metric{ID: "Alloc", Type: "gauge", Value: "12.54"},
		client,
	)

	require.ErrorIs(t, err, gzip.ErrHeader)
}

func TestRequestMalformedGzipBody(t *testing.T) {
	server := httptest.NewServer(middleware.Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("malformed gzip reached handler")
		w.WriteHeader(http.StatusNoContent)
	})))
	t.Cleanup(server.Close)

	client := NewHTTPClient(server.URL)

	// Хук заменяющий body перед отправкой Request
	client.OnBeforeRequest(func(_ *resty.Client, request *resty.Request) error {
		request.SetBody([]byte("this is not gzip"))
		return nil
	})

	response, err := client.Request("/update", []byte(`{"id":"Alloc","type":"gauge","value":12.54}`))
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode())
}
