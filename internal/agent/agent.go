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
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k057ya/go-metrics/internal/config"
	"github.com/k057ya/go-metrics/internal/errorshandler"
	"github.com/k057ya/go-metrics/internal/model"
	"github.com/k057ya/go-metrics/internal/service"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type HTTPClient struct {
	*resty.Client
}

func NewHTTPClient(baseURL string, gzip bool, signatureKey string) HTTPClient {
	c := HTTPClient{resty.New().
		SetTimeout(10*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept-Encoding", "gzip").
		SetBaseURL(baseURL)}

	if signatureKey != "" {
		c.signRequestsWithKey(signatureKey)
	}

	if gzip {
		c.useGzip()
	}

	return c
}

func (c *HTTPClient) signRequestsWithKey(key string) {
	c.OnBeforeRequest(func(_ *resty.Client, request *resty.Request) error {
		body, ok := request.Body.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte body, got %T", request.Body)
		}
		request.SetHeader("HashSHA256", service.GenSignature(body, key))
		return nil
	})
}

func (c *HTTPClient) useGzip() {
	c.OnBeforeRequest(func(_ *resty.Client, request *resty.Request) error {
		body, ok := request.Body.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte body, got %T", request.Body)
		}
		var compressed bytes.Buffer
		gzw := gzip.NewWriter(&compressed)
		_, err := gzw.Write(body)
		if err != nil {
			fmt.Printf("%s", err)
			return err
		}
		err = gzw.Close()
		if err != nil {
			return err
		}

		request.
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Encoding", "gzip").
			SetBody(compressed.Bytes())

		return nil
	})
}

func (c *HTTPClient) Request(url string, body []byte) (*resty.Response, error) {
	return c.RequestContext(context.Background(), url, body)
}

func (c *HTTPClient) RequestContext(ctx context.Context, url string, body []byte) (*resty.Response, error) {

	var response *resty.Response
	err := errorshandler.WithRetry(
		ctx,
		func() error {
			var requestError error
			response, requestError = c.R().
				SetContext(ctx).
				SetBody(body).
				SetHeader("Content-Type", "application/json").
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
	WorkerCount    int
}

type Metric struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

func Run(ctx context.Context, agent Agent) error {

	if agent.ReportInterval < agent.PollInterval {
		fmt.Println("poll interval must be less than report interval, falling back to default values")
		agent.PollInterval = config.DefaultPollInterval.Interval
		agent.ReportInterval = config.DefaultReportInterval.Interval
	}

	// отдельный контекст для отправки
	sendCtx, cancelSend := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelSend()

	// собирать метрики
	mchan := agent.collector(ctx)

	// готовые к отправке батчи
	bchan := make(chan []Metric)
	go agent.batcher(mchan, bchan)

	// отправлять метрики
	var wg sync.WaitGroup
	if agent.WorkerCount <= 0 {
		agent.WorkerCount = config.ClientConfig.WorkerCount
	}
	for id := range agent.WorkerCount {
		wg.Go(func() { agent.worker(sendCtx, id, bchan) })
	}
	workersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workersDone)
	}()

	<-ctx.Done()

	// пробуем доотправить накопленное
	shutdownTimeout := 5 * time.Second
	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()

	select {
	case <-workersDone:
		return nil

	// по истечению таймаута убиваем все запросы
	// и ждем завершения воркеров
	// и возвращаем ошибку
	case <-timer.C:
		cancelSend()

		<-workersDone

		return fmt.Errorf(
			"workers were not finished in timeout %s: %w",
			shutdownTimeout,
			context.DeadlineExceeded,
		)
	}
}

func (agent Agent) collector(ctx context.Context) chan Metric {
	mchan := make(chan Metric)
	var wg sync.WaitGroup

	collect := func(fetch func() []Metric) {
		defer wg.Done()
		for {
			if ctx.Err() != nil {
				return
			}

			for _, metric := range fetch() {
				select {
				case <-ctx.Done():
					return
				case mchan <- metric:
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(agent.PollInterval):
			}
		}
	}

	wg.Add(2)
	go collect(agent.fetchMetrics)
	go collect(agent.fetchPsUtilMetrics)

	go func() {
		wg.Wait()
		defer close(mchan)
	}()

	return mchan
}

func (agent Agent) batcher(mchan <-chan Metric, bchan chan<- []Metric) {
	defer close(bchan)
	ticker := time.NewTicker(agent.ReportInterval)
	defer ticker.Stop()

	var batch []Metric

	var push = func() {
		if len(batch) != 0 {
			bchan <- batch
		}
	}

	for {
		select {
		case <-ticker.C:
			push()
			batch = nil

		case m, ok := <-mchan:
			if !ok {
				push()
				return
			}
			batch = append(batch, m)
		}
	}
}

func (agent Agent) worker(ctx context.Context, id int, batches <-chan []Metric) {
	for batch := range batches {
		if _, err := sendBatch(ctx, batch, agent.Client); err != nil && ctx.Err() == nil {
			fmt.Printf("Worker %d: send batch %v\n", id, err)
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

func sendBatch(ctx context.Context, mlist []Metric, client HTTPClient) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

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
	response, err := client.RequestContext(ctx, `/updates`, mjson)

	if err != nil {
		return 0, err
	}

	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("server returned status: %s", response.Status())
	}

	return int64(len(metrics)), nil
}

func gauge(id string, value float64) Metric {
	return Metric{
		ID:    id,
		Type:  "gauge",
		Value: strconv.FormatFloat(value, 'f', -1, 64),
	}
}

func counter(id string, value int64) Metric {
	return Metric{
		ID:    id,
		Type:  "counter",
		Value: strconv.FormatInt(value, 10),
	}
}

func (a Agent) fetchMetrics() []Metric {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

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

func (a Agent) fetchPsUtilMetrics() []Metric {
	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil
	}
	metrics := []Metric{
		gauge("TotalMemory", float64(memStats.Total)),
		gauge("FreeMemory", float64(memStats.Available)),
	}
	cpuStats, err := cpu.Percent(0, true)
	if err != nil {
		return metrics
	}

	for i, usage := range cpuStats {
		metrics = append(metrics, gauge(fmt.Sprintf("CPUutilization%d", i+1), usage))
	}
	return metrics
}
