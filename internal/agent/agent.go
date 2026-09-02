package agent

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k057ya/go-metrics/internal/config"
)

func NewHttpClient() *resty.Client {
	return resty.New().
		SetHeader("Content-Type", "text/plain").
		SetBaseURL(config.ServerConfig.Address())
}

type Metric struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

var httpClient = NewHttpClient()

func Run() {

	pollsPerReport := int(config.ClientConfig.ReportInterval / config.ClientConfig.PollInterval)

	var collected []Metric
	for {
		for i := 0; i < pollsPerReport; i++ {
			time.Sleep(config.ClientConfig.PollInterval)
			fresh := fetchMetrics()
			collected = append(collected, fresh...)
		}

		for _, metric := range collected {
			if err := sendMetric(metric); err != nil {
				// TODO
				continue
			}
		}
		collected = collected[:0]
	}
}

func sendMetric(metric Metric) error {

	response, err := httpClient.R().SetBody([]byte("")).
		Post(`/update/` + metric.Type + `/` + metric.ID + `/` + metric.Value)

	if err != nil {
		return err
	}

	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return fmt.Errorf("server returned status: %s", response.Status())
	}

	return nil
}
func fetchMetrics() []Metric {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

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
		counter("PollCount", 1),
		gauge("RandomValue", rand.Float64()),
	}
}
