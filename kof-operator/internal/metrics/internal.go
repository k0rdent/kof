package metrics

import (
	"fmt"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
)

func (s *MetricCollectorService) CollectInternal() {
	port, err := s.getPort()
	if err != nil {
		s.error(fmt.Errorf("failed to get metrics port: %v", err))
		return
	}

	resp, err := k8s.Proxy(s.config.Ctx, s.config.KubeClient.Clientset, *s.config.Pod, port, s.config.ProxyEndpoint)
	if err != nil {
		s.error(fmt.Errorf("failed to proxy: %v", err))
		return
	}

	metrics, err := ParsePrometheusMetrics(string(resp))
	if err != nil {
		s.error(fmt.Errorf("failed to parse prometheus metrics: %v", err))
		return
	}

	for name, values := range metrics {
		for _, value := range values {
			s.sendMetric(name, value)
		}
	}
}
