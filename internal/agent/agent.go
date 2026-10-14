package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/errorshandler"
	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/service"
)

func NewHTTPClient(baseURL string) HTTPClient {
	return HTTPClient{resty.New().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept-Encoding", "gzip").
		SetBaseURL(baseURL),
	}
}

type Metric struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type HTTPClient struct {
	*resty.Client
}

// Короткий вызов со всеми настройками
func (c *HTTPClient) Request(url string, body []byte) (*resty.Response, error) {
	return c.RequestContext(context.Background(), url, body)
}

func (c *HTTPClient) RequestContext(ctx context.Context, url string, body []byte) (*resty.Response, error) {
	var compressed bytes.Buffer
	gzw := gzip.NewWriter(&compressed)
	_, err := gzw.Write(body)
	if err != nil {
		fmt.Printf("%s", err)
		return nil, err
	}
	err = gzw.Close()
	if err != nil {
		return nil, err
	}

	var response *resty.Response
	err = errorshandler.WithRetry(
		ctx,
		func() error {
			var requestError error
			response, requestError = c.R().
				SetContext(ctx).
				SetBody(compressed.Bytes()).
				SetHeader("Content-Type", "application/json").
				SetHeader("Accept-Encoding", "gzip").
				SetHeader("Content-Encoding", "gzip").
				SetHeader("HashSHA256", service.GenSignature(body, "KEY")).
				Post(url)
			return requestError
		},
		[]errorshandler.Classifier{errorshandler.NewNetworkErrorClassifier()},
		[]time.Duration{
			1 * time.Second,
			3 * time.Second,
			5 * time.Second,
		}, 3)

	return response, err
}

type Agent struct {
	Client         HTTPClient
	PollInterval   time.Duration
	ReportInterval time.Duration
	PollCount      int64
}

func Run(agent Agent) error {

	if agent.ReportInterval < agent.PollInterval {
		fmt.Println("poll interval must be less than report interval, falling back to default values")
		agent.PollInterval = config.DefaultPollInterval.Interval
		agent.ReportInterval = config.DefaultReportInterval.Interval
	}

	pollsPerReport := int(agent.ReportInterval / agent.PollInterval)

	for {
		var collected []Metric
		for range pollsPerReport {
			time.Sleep(agent.PollInterval)
			collected = agent.fetchMetrics()
		}

		if _, err := sendBatch(collected, agent.Client); err != nil {
			// TODO
			continue
		}
	}
}

func sendMetric(metric Metric, client HTTPClient) error {

	m := model.Metrics{
		ID:    metric.ID,
		MType: metric.Type,
	}
	err := m.Parse(metric.Value)
	if err != nil {
		return fmt.Errorf("cannot parse metric")
	}
	mjson, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error marshalling metric")
	}
	response, err := client.Request(`/update`, mjson)

	if err != nil {
		return err
	}

	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return fmt.Errorf("server returned status: %s", response.Status())
	}

	return nil
}

func sendBatch(mlist []Metric, client HTTPClient) (int64, error) {
	var metrics []model.Metrics
	for _, m := range mlist {
		metric := model.Metrics{
			ID:    m.ID,
			MType: m.Type,
		}
		err := metric.Parse(m.Value)
		if err != nil {
			fmt.Printf("cannot parse metric: %s %s", m.ID, err)
			continue
		}
		metrics = append(metrics, metric)
	}

	if len(metrics) == 0 {
		return 0, nil
	}

	mjson, err := json.Marshal(metrics)
	if err != nil {
		fmt.Printf("error marshalling metric list")
		return 0, err
	}
	response, err := client.Request(`/updates`, mjson)

	if err != nil {
		return 0, err
	}

	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("server returned status: %s", response.Status())
	}

	return int64(len(metrics)), nil
}

func (a Agent) fetchMetrics() []Metric {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	a.PollCount += 1

	gauge := func(id string, value float64) Metric {
		return Metric{
			ID:    id,
			Type:  "gauge",
			Value: strconv.FormatFloat(value, 'f', -1, 64),
		}
	}

	counter := func(id string, value int64) Metric {
		return Metric{
			ID:    id,
			Type:  "counter",
			Value: strconv.FormatInt(value, 10),
		}
	}

	return []Metric{
		gauge("Alloc", float64(stats.Alloc)),
		gauge("BuckHashSys", float64(stats.BuckHashSys)),
		gauge("Frees", float64(stats.Frees)),
		gauge("GCCPUFraction", stats.GCCPUFraction),
		gauge("GCSys", float64(stats.GCSys)),
		gauge("HeapAlloc", float64(stats.HeapAlloc)),
		gauge("HeapIdle", float64(stats.HeapIdle)),
		gauge("HeapInuse", float64(stats.HeapInuse)),
		gauge("HeapObjects", float64(stats.HeapObjects)),
		gauge("HeapReleased", float64(stats.HeapReleased)),
		gauge("HeapSys", float64(stats.HeapSys)),
		gauge("LastGC", float64(stats.LastGC)),
		gauge("Lookups", float64(stats.Lookups)),
		gauge("MCacheInuse", float64(stats.MCacheInuse)),
		gauge("MCacheSys", float64(stats.MCacheSys)),
		gauge("MSpanInuse", float64(stats.MSpanInuse)),
		gauge("MSpanSys", float64(stats.MSpanSys)),
		gauge("Mallocs", float64(stats.Mallocs)),
		gauge("NextGC", float64(stats.NextGC)),
		gauge("NumForcedGC", float64(stats.NumForcedGC)),
		gauge("NumGC", float64(stats.NumGC)),
		gauge("OtherSys", float64(stats.OtherSys)),
		gauge("PauseTotalNs", float64(stats.PauseTotalNs)),
		gauge("StackInuse", float64(stats.StackInuse)),
		gauge("StackSys", float64(stats.StackSys)),
		gauge("Sys", float64(stats.Sys)),
		gauge("TotalAlloc", float64(stats.TotalAlloc)),
		counter("PollCount", a.PollCount),
		gauge("RandomValue", rand.Float64()),
	}
}
