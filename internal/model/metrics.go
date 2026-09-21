package model

import (
	"errors"
	"fmt"
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
		return strconv.FormatInt(*metrics.Delta, 10)
	case MetricsTypeGauge:
		return strconv.FormatFloat(*metrics.Value, 'f', -1, 64)
	}
	return "-"
}

func (metrics *Metrics) URL() string {
	return fmt.Sprintf("/value/%s/%s", metrics.MType, metrics.ID)
}

func (metrics *Metrics) Parse(v string) error {
	var (
		gaugeVal   float64
		counterVal int64
		err        error
	)
	switch metrics.MType {
	case MetricsTypeCounter:
		counterVal, err = strconv.ParseInt(v, 10, 64)
		metrics.Delta = &counterVal
	case MetricsTypeGauge:
		gaugeVal, err = strconv.ParseFloat(v, 64)
		metrics.Value = &gaugeVal
	default:
		err = errors.New("invalid metric type")
	}
	return err
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
		err = errors.New("unknown metric type")
	}
	if err != nil {
		return err
	}
	return nil

}
