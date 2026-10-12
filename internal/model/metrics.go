package model

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
)

const (
	MetricsTypeCounter = "counter"
	MetricsTypeGauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (metrics *Metrics) StringValue() string {
	switch metrics.MType {
	case MetricsTypeCounter:
		if metrics.Delta == nil {
			break
		}
		return strconv.FormatInt(*metrics.Delta, 10)
	case MetricsTypeGauge:
		if metrics.Value == nil {
			break
		}
		return strconv.FormatFloat(*metrics.Value, 'f', -1, 64)
	}
	return "-"
}

func (metrics *Metrics) URL() string {
	return fmt.Sprintf("/value/%s/%s", url.PathEscape(metrics.MType), url.PathEscape(metrics.ID))
}

func (metrics *Metrics) Parse(v string) error {
	var (
		gaugeVal   float64
		counterVal int64
		err        error
	)
	isFinite := func(num float64) bool {
		return !math.IsInf(num, 0) && !math.IsNaN(num)
	}
	switch metrics.MType {
	case MetricsTypeCounter:
		counterVal, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			break
		}
		if !isFinite(float64(counterVal)) {
			err = errors.New("counter value is NaN or Inf")
		}
		metrics.Delta = &counterVal
	case MetricsTypeGauge:
		gaugeVal, err = strconv.ParseFloat(v, 64)
		if err != nil {
			break
		}
		if !isFinite(gaugeVal) {
			err = errors.New("gauge value is NaN or Inf")
		}
		metrics.Value = &gaugeVal
	default:
		return errors.New("invalid metric type")
	}
	if err != nil {
		return errors.New("invalid metric value")
	}
	return nil
}

func (metrics *Metrics) ValidateValue() error {
	var err error
	switch metrics.MType {
	case MetricsTypeCounter:
		if metrics.Delta == nil {
			err = errors.New("metrics delta is not set")
		}
	case MetricsTypeGauge:
		if metrics.Value == nil {
			err = errors.New("metrics value is not set")
		}
	default:
		err = errors.New("invalid metric type")
	}
	if err != nil {
		return err
	}
	return nil
}

func (metrics *Metrics) ValidateType() error {
	var err error
	switch metrics.MType {
	case MetricsTypeCounter:
	case MetricsTypeGauge:
	default:
		err = errors.New("invalid metric type")
	}
	if err != nil {
		return err
	}
	return nil
}

func (metrics *Metrics) ValidateID() error {
	if metrics.ID == "" {
		return errors.New("metric name is not specified")
	}
	return nil
}

func (metrics *Metrics) Validate() error {
	if err := metrics.ValidateID(); err != nil {
		return err
	}
	if err := metrics.ValidateType(); err != nil {
		return err
	}
	if err := metrics.ValidateValue(); err != nil {
		return err
	}

	return nil
}
