package agent

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Это собрано с помощью AI, так как еще не достаточно разобрался в подмене
type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestFetchMetrics(t *testing.T) {
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

	got := fetchMetrics()
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
		method      string
		path        string
		contentType string
	}

	tests := []struct {
		name         string
		metric       Metric
		statusCode   int
		transportErr error
		wantPath     string
		wantErr      bool
	}{
		{
			name:       "#1 send gauge",
			metric:     Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			statusCode: http.StatusOK,
			wantPath:   "/update/gauge/Alloc/12.5",
		},
		{
			name:       "#2 send counter",
			metric:     Metric{ID: "PollCount", Type: "counter", Value: "1"},
			statusCode: http.StatusOK,
			wantPath:   "/update/counter/PollCount/1",
		},
		{
			name:       "#3 server returns error",
			metric:     Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			statusCode: http.StatusInternalServerError,
			wantPath:   "/update/gauge/Alloc/12.5",
			wantErr:    true,
		},
		{
			name:         "#4 server is unavailable",
			metric:       Metric{ID: "Alloc", Type: "gauge", Value: "12.5"},
			transportErr: errors.New("server is unavailable"),
			wantPath:     "/update/gauge/Alloc/12.5",
			wantErr:      true,
		},
	}

	// Это собрано с помощью AI, так как еще не достаточно разобрался в подмене
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldHTTPClient := httpClient

			var received receivedRequest

			httpClient = newHttpClient().
				SetBaseURL("http://metrics.test").
				SetTransport(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					received = receivedRequest{
						method:      request.Method,
						path:        request.URL.Path,
						contentType: request.Header.Get("Content-Type"),
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

			t.Cleanup(func() {
				httpClient = oldHTTPClient
			})

			err := sendMetric(test.metric)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, http.MethodPost, received.method)
			assert.Equal(t, test.wantPath, received.path)
			assert.Equal(t, "text/plain", received.contentType)
		})
	}
}
