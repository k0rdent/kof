package utils

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type Metrics map[string]any

func ParsePrometheusMetrics(metricsText string) (Metrics, error) {
	metrics := Metrics{}
	parser := expfmt.TextParser{}
	reader := strings.NewReader(metricsText)

	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return metrics, fmt.Errorf("failed to parse metrics: %w", err)
	}

	for name, mf := range metricFamilies {
		for _, m := range mf.GetMetric() {
			var value float64
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				value = m.GetCounter().GetValue()
			case dto.MetricType_GAUGE:
				value = m.GetGauge().GetValue()
			case dto.MetricType_HISTOGRAM:
				value = m.GetHistogram().GetSampleSum()
			case dto.MetricType_SUMMARY:
				value = m.GetSummary().GetSampleSum()
			}

			metrics[name] = value
		}
	}

	return metrics, nil
}
