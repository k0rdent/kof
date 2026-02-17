package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Victoria struct {
	ctx        context.Context
	kubeClient *k8s.KubeClient
	pods       []corev1.Pod
	name       string
}

const (
	VictoriaMaxResponseTime       = 10 * time.Second
	VictoriaMetricsPortAnnotation = "kof.k0rdent.mirantis.com/victoria-metrics-port"
	VictoriaMetricsLabel          = "k0rdent.mirantis.com/kof-victoria-metrics"
	VictoriaPortName              = "http"
	VictoriaMetricsEndpoint       = "metrics"

	KubernetesAppLabel        = "app.kubernetes.io/name"
	VictoriaLogsAppLabelValue = "victoria-logs-cluster"

	VictoriaLogsResource    = "VictoriaLogs"
	VictoriaMetricsResource = "VictoriaMetrics"
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
			GetCustomResourcesFn:  GetVictoriaResources,
			MaxResponseTime:       VictoriaMaxResponseTime,
			MetricsPortAnnotation: VictoriaMetricsPortAnnotation,
			PortName:              VictoriaPortName,
			MetricsEndpoint:       VictoriaMetricsEndpoint,
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

	res.SendObj(&Response{
		Clusters: h.GetMetrics(),
	}, http.StatusOK)
}

func GetVictoriaResources(ctx context.Context, kubeClient *k8s.KubeClient) ([]ICustomResource, error) {
	podList := new(corev1.PodList)

	options := []client.ListOption{
		client.HasLabels{VictoriaMetricsLabel},
	}

	if err := kubeClient.Client.List(ctx, podList, options...); err != nil {
		return nil, fmt.Errorf("failed to get victoria metrics/logs pods: %v", err)
	}

	vlPods := make([]corev1.Pod, 0)
	vmPods := make([]corev1.Pod, 0)
	for _, pod := range podList.Items {
		if value := pod.Labels[KubernetesAppLabel]; value == VictoriaLogsAppLabelValue {
			vlPods = append(vlPods, pod)
		} else {
			vmPods = append(vmPods, pod)
		}
	}

	customResources := make([]ICustomResource, 0, 2)
	if len(vlPods) > 0 {
		customResources = append(customResources, NewVictoriaResource(ctx, kubeClient, VictoriaLogsResource, vlPods))
	}
	if len(vmPods) > 0 {
		customResources = append(customResources, NewVictoriaResource(ctx, kubeClient, VictoriaMetricsResource, vmPods))
	}

	return customResources, nil
}

func NewVictoriaResource(ctx context.Context, kubeClient *k8s.KubeClient, resourceName string, pods []corev1.Pod) ICustomResource {
	return &Victoria{
		ctx:        ctx,
		kubeClient: kubeClient,
		pods:       pods,
		name:       resourceName,
	}
}

func (v *Victoria) GetName() string {
	return v.name
}

func (v *Victoria) GetPods() ([]corev1.Pod, error) {
	return v.pods, nil
}

func (v *Victoria) GetStatus() *ResourceStatus {
	return nil
}
