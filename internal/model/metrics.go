package model

import (
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

func (metrics Metrics) StringValue() string {
	switch metrics.MType {
	case MetricsTypeCounter:
		return strconv.FormatInt(*metrics.Delta, 10)
	case MetricsTypeGauge:
		return strconv.FormatFloat(*metrics.Value, 'f', -1, 64)
	}
	return "-"
}

func (metrics Metrics) URL() string {
	return fmt.Sprintf("/value/%s/%s", metrics.MType, metrics.ID)
}
