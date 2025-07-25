package utils

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type Metrics map[string]any

func ParsePrometheusMetrics(metrics Metrics, metricsText string) error {
	parser := expfmt.TextParser{}
	reader := strings.NewReader(metricsText)

	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
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

	return nil
}
