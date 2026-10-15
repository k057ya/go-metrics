package model

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetrics_Parse: Данные сценариев разбора значений метрики.
func TestMetrics_Parse(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type args struct {
		value string
	}
	type want struct {
		metric     Metrics
		err        bool
		errMessage string
	}

	tests := map[string]struct {
		fields fields
		args   args
		want   want
	}{
		"#1 string value to counter": {
			fields: fields{metric: Metrics{ID: "counterVar1", MType: MetricsTypeCounter}},
			args:   args{value: "this-is-a-rhytm-of-the-night"},
			want:   want{err: true, errMessage: "invalid metric value"},
		},
		"#2 string value to gauge": {
			fields: fields{metric: Metrics{ID: "invalidGauge", MType: MetricsTypeGauge}},
			args:   args{value: "not-a-number"},
			want:   want{err: true, errMessage: "invalid metric value"},
		},
		"#3 counter value overflow": {
			fields: fields{metric: Metrics{ID: "overflowCounter", MType: MetricsTypeCounter}},
			args:   args{value: "9223372036854775808"},
			want:   want{err: true, errMessage: "invalid metric value"},
		},
		"#4 scientific notation gauge": {
			fields: fields{metric: Metrics{ID: "scientificGauge", MType: MetricsTypeGauge}},
			args:   args{value: "1e3"},
			want: want{
				metric: Metrics{ID: "scientificGauge", MType: MetricsTypeGauge, Value: new(float64(1000))},
			},
		},
		"#5 counter_positive": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "42",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(42))},
				err:    false,
			},
		},
		"#6 counter_zero": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "0",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(0))},
				err:    false,
			},
		},
		"#7 counter_negative": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "-5",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(-5))},
				err:    false,
			},
		},
		"#8 counter_min": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "-9223372036854775808",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(math.MinInt64))},
				err:    false,
			},
		},
		"#9 counter_max": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "9223372036854775807",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(math.MaxInt64))},
				err:    false,
			},
		},
		"#10 gauge_integer": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "10",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(10))},
				err:    false,
			},
		},
		"#11 gauge_fraction": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "1.25",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(1.25))},
				err:    false,
			},
		},
		"#12 gauge_negative": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "-3.5",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(-3.5))},
				err:    false,
			},
		},
		"#13 gauge_zero": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "0",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(0))},
				err:    false,
			},
		},
		"#14 gauge_scientific": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "1e3",
			},
			want: want{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(1000))},
				err:    false,
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"counter_empty": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeCounter},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"counter_text": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "not-a-number",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeCounter},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"counter_fraction": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "1.1",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeCounter},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"counter_overflow": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "9223372036854775808",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeCounter},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"counter_underflow": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			args: args{
				value: "-9223372036854775809",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeCounter},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"gauge_empty": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: ошибка разбора не изменяет исходную метрику; текущий Parse присваивает указатель и при ошибке.
		"gauge_text": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "not-a-number",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},

		"gauge_overflow": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "1e309",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		"unknown_type": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: "switcher"},
			},
			args: args{
				value: "1",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: "switcher"},
				err:        true,
				errMessage: "invalid metric type",
			},
		},
		// Допущение: NaN и бесконечности отклоняются, чтобы метрика могла сериализоваться в JSON.
		"gauge_nan": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "NaN",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Допущение: NaN и бесконечности отклоняются, чтобы метрика могла сериализоваться в JSON.
		"gauge_positive_inf": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "+Inf",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Допущение: NaN и бесконечности отклоняются, чтобы метрика могла сериализоваться в JSON.
		"gauge_negative_inf": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			args: args{
				value: "-Inf",
			},
			want: want{
				metric:     Metrics{ID: "metric", MType: MetricsTypeGauge},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
		// Ожидаемый контракт: прежнее значение сохраняется после ошибки Parse.
		"parse_error_preserves_value": {
			fields: fields{
				metric: Metrics{ID: "PollCount", MType: MetricsTypeCounter, Delta: new(int64(42))},
			},
			args: args{
				value: "bad",
			},
			want: want{
				metric:     Metrics{ID: "PollCount", MType: MetricsTypeCounter, Delta: new(int64(42))},
				err:        true,
				errMessage: "invalid metric value",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric
			err := metric.Parse(test.args.value)
			if test.want.err != false {
				require.EqualError(t, err, test.want.errMessage)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want.metric, metric)
		})
	}
}

// TestMetrics_ValidateID: Данные проверки имени.
func TestMetrics_ValidateID(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type want struct {
		err bool
	}

	tests := map[string]struct {
		fields fields
		want   want
	}{
		"nonempty": {
			fields: fields{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter, Delta: new(int64(1))},
			},
			want: want{
				err: false,
			},
		},
		"empty": {
			fields: fields{
				metric: Metrics{ID: "", MType: MetricsTypeCounter, Delta: new(int64(1))},
			},
			want: want{
				err: true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric
			err := metric.ValidateID()
			if test.want.err != false {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestMetrics_ValidateType: Данные проверки типа.
func TestMetrics_ValidateType(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type want struct {
		err        bool
		errMessage string
	}

	tests := map[string]struct {
		fields fields
		want   want
	}{
		MetricsTypeCounter: {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			want: want{
				err: false,
			},
		},
		MetricsTypeGauge: {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			want: want{
				err: false,
			},
		},
		"other": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: "other"},
			},
			want: want{
				err:        true,
				errMessage: "invalid metric type",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric
			err := metric.ValidateType()
			if test.want.err != false {
				require.EqualError(t, err, test.want.errMessage)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestMetrics_ValidateValue: Данные проверки обязательного значения.
func TestMetrics_ValidateValue(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type want struct {
		err        bool
		errMessage string
	}

	tests := map[string]struct {
		fields fields
		want   want
	}{
		"counter_missing": {
			fields: fields{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter},
			},
			want: want{
				err:        true,
				errMessage: "metrics delta is not set",
			},
		},
		"counter_zero": {
			fields: fields{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter, Delta: new(int64(0))},
			},
			want: want{
				err: false,
			},
		},
		"counter_negative": {
			fields: fields{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter, Delta: new(int64(-5))},
			},
			want: want{
				err: false,
			},
		},
		"gauge_missing": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge},
			},
			want: want{
				err:        true,
				errMessage: "metrics value is not set",
			},
		},
		"gauge_zero": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(0))},
			},
			want: want{
				err: false,
			},
		},
		"gauge_fraction": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(1.25))},
			},
			want: want{
				err: false,
			},
		},
		"unknown_type": {
			fields: fields{
				metric: Metrics{ID: "x", MType: "other"},
			},
			want: want{
				err:        true,
				errMessage: "invalid metric type",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric
			err := metric.ValidateValue()
			if test.want.err != false {
				require.EqualError(t, err, test.want.errMessage)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestMetrics_StringValue: Данные строкового представления значения.
func TestMetrics_StringValue(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type want struct {
		value string
	}

	tests := map[string]struct {
		fields fields
		want   want
	}{
		"counter_positive": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(42))},
			},
			want: want{
				value: "42",
			},
		},
		"counter_zero": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(0))},
			},
			want: want{
				value: "0",
			},
		},
		"counter_negative": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter, Delta: new(int64(-5))},
			},
			want: want{
				value: "-5",
			},
		},
		"gauge_fraction": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(1.25))},
			},
			want: want{
				value: "1.25",
			},
		},
		"gauge_zero": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(0))},
			},
			want: want{
				value: "0",
			},
		},
		"gauge_negative": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge, Value: new(float64(-3.5))},
			},
			want: want{
				value: "-3.5",
			},
		},
		"unknown_type": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: "other"},
			},
			want: want{
				value: "-",
			},
		},
		"counter_missing_value": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeCounter},
			},
			want: want{
				value: "-",
			},
		},
		"gauge_missing_value": {
			fields: fields{
				metric: Metrics{ID: "metric", MType: MetricsTypeGauge},
			},
			want: want{
				value: "-",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric

			require.Equal(t, test.want.value, metric.StringValue())
		})
	}
}

// TestMetrics_URL: Данные построения URL.
func TestMetrics_URL(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type want struct {
		url string
	}

	tests := map[string]struct {
		fields fields
		want   want
	}{
		MetricsTypeCounter: {
			fields: fields{
				metric: Metrics{ID: "PollCount", MType: MetricsTypeCounter, Delta: new(int64(1))},
			},
			want: want{
				url: "/value/counter/PollCount",
			},
		},
		MetricsTypeGauge: {
			fields: fields{
				metric: Metrics{ID: "Alloc", MType: MetricsTypeGauge, Value: new(float64(1.25))},
			},
			want: want{
				url: "/value/gauge/Alloc",
			},
		},

		"escaped_id": {
			fields: fields{
				metric: Metrics{ID: "heap size/bytes", MType: MetricsTypeGauge, Value: new(float64(1))},
			},
			want: want{
				url: "/value/gauge/heap%20size%2Fbytes",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			metric := test.fields.metric

			require.Equal(t, test.want.url, metric.URL())
		})
	}
}

// TestMetrics_JSON: Данные сериализации и чтения JSON.
func TestMetrics_JSON(t *testing.T) {
	type fields struct {
		metric Metrics
	}
	type args struct {
		data []byte
	}
	type want struct {
		metric Metrics
		json   string
		err    bool
	}

	tests := map[string]struct {
		fields fields
		args   args
		want   want
	}{
		"counter_zero": {
			fields: fields{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter, Delta: new(int64(0))},
			},
			args: args{
				data: []byte("{\"id\":\"count\",\"type\":\"counter\",\"delta\":0}"),
			},
			want: want{
				metric: Metrics{ID: "count", MType: MetricsTypeCounter, Delta: new(int64(0))},
				json:   "{\"id\":\"count\",\"type\":\"counter\",\"delta\":0}",
				err:    false,
			},
		},
		"gauge_zero": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(0))},
			},
			args: args{
				data: []byte("{\"id\":\"alloc\",\"type\":\"gauge\",\"value\":0}"),
			},
			want: want{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(0))},
				json:   "{\"id\":\"alloc\",\"type\":\"gauge\",\"value\":0}",
				err:    false,
			},
		},
		"absent_values": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge},
			},
			args: args{
				data: []byte("{\"id\":\"alloc\",\"type\":\"gauge\"}"),
			},
			want: want{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge},
				json:   "{\"id\":\"alloc\",\"type\":\"gauge\"}",
				err:    false,
			},
		},
		"with_hash": {
			fields: fields{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(1.25)), Hash: "abc"},
			},
			args: args{
				data: []byte("{\"id\":\"alloc\",\"type\":\"gauge\",\"value\":1.25,\"hash\":\"abc\"}"),
			},
			want: want{
				metric: Metrics{ID: "alloc", MType: MetricsTypeGauge, Value: new(float64(1.25)), Hash: "abc"},
				json:   "{\"id\":\"alloc\",\"type\":\"gauge\",\"value\":1.25,\"hash\":\"abc\"}",
				err:    false,
			},
		},
		"invalid_json": {
			fields: fields{
				metric: Metrics{},
			},
			args: args{
				data: []byte("{"),
			},
			want: want{
				metric: Metrics{},
				json:   "{\"id\":\"\",\"type\":\"\"}",
				err:    true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(test.fields.metric)
			require.NoError(t, err)
			assert.JSONEq(t, test.want.json, string(data))

			var metric Metrics
			err = json.Unmarshal(test.args.data, &metric)
			if test.want.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want.metric, metric)
		})
	}
}
