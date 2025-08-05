package handlers

import (
	"net/http"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Response struct {
	Clusters metrics.ClusterMetrics `json:"clusters"`
}

const (
	CollectorMaxResponseTime       = 60 * time.Second
	CollectorPortName              = "metrics"
	CollectorMetricsEndpoint       = "metrics"
	CollectorContainerName         = "otc-container"
	CollectorMetricsPortAnnotation = "kof.k0rdent.mirantis.com/collector-metrics-port"
	CollectorMetricsLabel          = "k0rdent.mirantis.com/kof-collector-metrics"
)

func newCollectorHandler(res *server.Response, req *http.Request) (*BaseMetricsHandler, error) {
	kubeClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	return NewBaseMetricsHandler(
		req.Context(),
		kubeClient,
		res.Logger,
		&MetricsConfig{
			MaxResponseTime:       CollectorMaxResponseTime,
			MetricsPortAnnotation: CollectorMetricsPortAnnotation,
			PortName:              CollectorPortName,
			MetricsEndpoint:       CollectorMetricsEndpoint,
			ContainerName:         CollectorContainerName,
			PodFilter: []client.ListOption{
				client.HasLabels{CollectorMetricsLabel},
				client.MatchingLabels(map[string]string{
					"app.kubernetes.io/component": "opentelemetry-collector",
				}),
			},
		},
	), nil
}

func CollectorHandle(res *server.Response, req *http.Request) {
	h, err := newCollectorHandler(res, req)
	if err != nil {
		res.Logger.Error(err, "Failed to create prometheus handler")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.Send(&Response{
		Clusters: h.GetMetrics(),
	}, http.StatusOK)
}
