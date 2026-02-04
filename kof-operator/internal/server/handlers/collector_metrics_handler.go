package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	otel "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Response struct {
	Clusters metrics.ClusterMap `json:"clusters"`
}

type OpenTelemetryCollector struct {
	ctx        context.Context
	kubeClient *k8s.KubeClient
	collector  *otel.OpenTelemetryCollector
}

const (
	CollectorMaxResponseTime       = 60 * time.Second
	CollectorPortName              = "metrics"
	CollectorMetricsEndpoint       = "metrics"
	CollectorContainerName         = "otc-container"
	CollectorMetricsPortAnnotation = "kof.k0rdent.mirantis.com/collector-metrics-port"
	CollectorMetricsLabel          = "k0rdent.mirantis.com/kof-collector-metrics"
)

const (
	CollectorNoReplicasMessage          = "Collector has no replicas, please check configuration of the collector."
	CollectorFailedFetchStatusMessage   = "Failed to fetch current collector status."
	CollectorZeroReplicasWarningMessage = "Collector was not deployed because the replica count is zero. This may be caused by a mismatched selector."
	CollectorReplicasDownMessage        = "Collector replicas are not ready (%d of %d replicas ready). Check the Collector configuration and pod logs for details."
)

func newCollectorHandler(res *server.Response, req *http.Request) *BaseMetricsHandler {
	return NewBaseMetricsHandler(
		req.Context(),
		k8s.LocalKubeClient,
		res.Logger,
		&MetricsConfig{
			GetCustomResourcesFn:  GetOpenTelemetryCollectors,
			MaxResponseTime:       CollectorMaxResponseTime,
			MetricsPortAnnotation: CollectorMetricsPortAnnotation,
			PortName:              CollectorPortName,
			MetricsEndpoint:       CollectorMetricsEndpoint,
			ContainerName:         CollectorContainerName,
		},
	)
}

func CollectorHandler(res *server.Response, req *http.Request) {
	h := newCollectorHandler(res, req)

	res.SendObj(&Response{
		Clusters: h.GetMetrics(),
	}, http.StatusOK)
}

func GetOpenTelemetryCollectors(ctx context.Context, kubeClient *k8s.KubeClient) ([]ICustomResource, error) {
	otelcList, err := k8s.GetOpenTelemetryCollectors(ctx, kubeClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenTelemetryCollectors: %v", err)
	}
	customResources := make([]ICustomResource, len(otelcList.Items))
	for i, otelc := range otelcList.Items {
		customResources[i] = &OpenTelemetryCollector{
			ctx:        ctx,
			kubeClient: kubeClient,
			collector:  &otelc,
		}
	}
	return customResources, nil
}

func (o *OpenTelemetryCollector) GetPods() ([]corev1.Pod, error) {
	selector, err := k8s.ExtractPodSelectorsFromOTelCollector(o.collector)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pod selectors from collector: %v", err)
	}

	ListOption := []client.ListOption{
		client.HasLabels{CollectorMetricsLabel},
		client.MatchingLabelsSelector{
			Selector: selector,
		},
	}

	podList, err := k8s.GetPods(o.ctx, o.kubeClient.Client, ListOption...)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods of collector: %v", err)
	}
	return podList.Items, nil
}

func (o *OpenTelemetryCollector) GetStatus() *ResourceStatus {
	log := log.FromContext(o.ctx)

	if o.collector.Status.Scale.StatusReplicas == "" {
		return &ResourceStatus{
			MessageType: metrics.MessageTypeError,
			Message:     CollectorNoReplicasMessage,
		}
	}

	expectedReplicas := o.collector.Status.Scale.Replicas
	currentReplicasStr := strings.Split(o.collector.Status.Scale.StatusReplicas, "/")[0]
	currentReplicas, err := strconv.Atoi(currentReplicasStr)
	if err != nil {
		log.Error(err, "Failed to convert current replicas to int", "collector", o.collector.Name, "replicas", currentReplicasStr)
		return &ResourceStatus{
			MessageType: metrics.MessageTypeError,
			Message:     CollectorFailedFetchStatusMessage,
		}
	}

	if expectedReplicas == 0 && currentReplicas == 0 {
		return &ResourceStatus{
			MessageType: metrics.MessageTypeWarning,
			Message:     CollectorZeroReplicasWarningMessage,
		}
	}

	if currentReplicas != int(expectedReplicas) {
		return &ResourceStatus{
			MessageType: metrics.MessageTypeError,
			Message:     fmt.Sprintf(CollectorReplicasDownMessage, currentReplicas, expectedReplicas),
		}
	}
	return nil
}

func (o *OpenTelemetryCollector) GetName() string {
	return o.collector.Name
}
