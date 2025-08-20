package metrics

import (
	"fmt"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

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
			default:
				value = m.GetUntyped().GetValue()
			}

			metricLabels := m.GetLabel()
			metricValue := &MetricValue{
				Labels: make(map[string]string),
				Value:  value,
			}

			for _, label := range metricLabels {
				metricValue.Labels[*label.Name] = *label.Value
			}
			metrics.Add(name, metricValue)
		}
	}

	return metrics, nil
}

func getReadyCondition(conditions []corev1.PodCondition) *corev1.PodCondition {
	for _, cond := range conditions {
		if cond.Type == corev1.PodReady {
			return &cond
		}
	}
	return nil
}

func findContainerMetric(containers []v1beta1.ContainerMetrics, name string) (*v1beta1.ContainerMetrics, error) {
	for _, c := range containers {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("metrics not found for container: %s", name)
}

func (s *Service) send(name string, metricValue *MetricValue) {
	s.config.Metrics <- &Metric{
		Cluster: s.config.ClusterName,
		Pod:     s.config.Pod.Name,
		Name:    name,
		Data:    metricValue,
	}
}

func (s *Service) error(err error) {
	s.config.Metrics <- &Metric{Err: err}
}

func (s *Service) getPort() (string, error) {
	if port, ok := s.config.Pod.Annotations[s.config.PortAnnotation]; ok {
		return port, nil
	}

	return k8s.ExtractContainerPort(s.config.Pod, s.config.ContainerName, s.config.PortName)
}
