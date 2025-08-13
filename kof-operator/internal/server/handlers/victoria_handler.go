package handlers

import (
	"net/http"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	VictoriaMaxResponseTime       = 10 * time.Second
	VictoriaMetricsPortAnnotation = "kof.k0rdent.mirantis.com/victoria-metrics-port"
	VictoriaMetricsLabel          = "k0rdent.mirantis.com/kof-victoria-metrics"
	VictoriaPortName              = "http"
	VictoriaMetricsEndpoint       = "metrics"
)

func newVictoriaHandler(res *server.Response, req *http.Request) (*BaseMetricsHandler, error) {
	kubeClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	return NewBaseMetricsHandler(
		req.Context(),
		kubeClient,
		res.Logger,
		&MetricsConfig{
			MaxResponseTime:       VictoriaMaxResponseTime,
			MetricsPortAnnotation: VictoriaMetricsPortAnnotation,
			PortName:              VictoriaPortName,
			MetricsEndpoint:       VictoriaMetricsEndpoint,
			PodFilter: []client.ListOption{
				client.HasLabels{VictoriaMetricsLabel},
			},
		},
	), nil
}

func VictoriaHandler(res *server.Response, req *http.Request) {
	h, err := newVictoriaHandler(res, req)
	if err != nil {
		res.Logger.Error(err, "Failed to create victoria handler")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.Send(&Response{
		Clusters: h.GetMetrics(),
	}, http.StatusOK)
}
