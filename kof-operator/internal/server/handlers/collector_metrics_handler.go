package handlers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CollectorMetricsService struct {
	kubeClient *k8s.KubeClient
	logger     *logr.Logger
}

type ClusterMetrics struct {
	PodMetrics  PodMetricsMap
	ClusterName string
	Err         error
}

type PodMetrics struct {
	Metrics utils.Metrics
	PodName string
	Err     error
}

type ClusterMetricsMap map[string]PodMetricsMap
type PodMetricsMap map[string]utils.Metrics

type MetricsResponse struct {
	Clusters ClusterMetricsMap `json:"clusters"`
}

const (
	MaxResponseTime               = 60 * time.Second
	MetricsPortName               = "metrics"
	MetricsPath                   = "metrics"
	DefaultCollectorContainerName = "otc-container"
	MetricsPortAnnotation         = "kof.k0rdent.mirantis.com/collector-metrics-port"
)

const (
	ConditionReadyHealthyMetric        = "otel_condition_ready_healthy"
	ConditionReadyReasonMetric         = "otel_condition_ready_reason"
	ConditionReadyMessageMetric        = "otel_condition_ready_message"
	ContainerResourceCpuUsageMetric    = "otel_container_resource_cpu_usage"
	ContainerResourceCpuLimitMetric    = "otel_container_resource_cpu_limit"
	ContainerResourceMemoryUsageMetric = "otel_container_resource_memory_usage"
	ContainerResourceMemoryLimitMetric = "otel_container_resource_memory_limit"
)

func newCollectorMetricsHandler(res *server.Response) (*CollectorMetricsService, error) {
	kubeClient, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	return &CollectorMetricsService{
		kubeClient: kubeClient,
		logger:     res.Logger,
	}, nil
}

func CollectorMetricsHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	h, err := newCollectorMetricsHandler(res)
	if err != nil {
		res.Logger.Error(err, "Failed to create prometheus handler")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	metrics, err := h.getCollectorsMetrics(ctx)
	if err != nil {
		res.Logger.Error(err, "Failed to get collector metrics")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.Send(metrics, http.StatusOK)
}

func (h *CollectorMetricsService) getCollectorsMetrics(ctx context.Context) (*MetricsResponse, error) {
	resp := &MetricsResponse{
		Clusters: make(ClusterMetricsMap),
	}

	cdList, err := k8s.GetKofClusterDeployments(ctx, h.kubeClient.Client)
	if err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}
	metricsChan := make(chan *ClusterMetrics)
	ctx, cancel := context.WithTimeout(ctx, MaxResponseTime)
	defer cancel()

	getLocalCollectorMetricsAsync(ctx, h.kubeClient, metricsChan, wg)

	for _, cd := range cdList.Items {
		getCollectorsMetricsAsync(ctx, h.kubeClient.Client, &cd, metricsChan, wg)
	}

	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	for metrics := range metricsChan {
		resp.Clusters[metrics.ClusterName] = metrics.PodMetrics
		if metrics.Err != nil {
			h.logger.Error(metrics.Err, "failed to receive metrics", "clusterName", metrics.ClusterName)
		}
	}

	return resp, nil
}

func getLocalCollectorMetricsAsync(ctx context.Context, client *k8s.KubeClient, metricsChan chan *ClusterMetrics, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := collectMetrics(ctx, client)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to collect metrics: %v", err), ClusterName: MothershipClusterName, PodMetrics: metrics}
			return
		}
		metricsChan <- &ClusterMetrics{
			PodMetrics:  metrics,
			ClusterName: MothershipClusterName,
		}
	}()
}

func getCollectorsMetricsAsync(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment, metricsChan chan *ClusterMetrics, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		secretName := k8s.GetSecretName(cd)
		secret, err := k8s.GetSecret(ctx, client, secretName, cd.Namespace)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to get secret: %v", err), ClusterName: cd.Name}
			return
		}

		kubeconfig := k8s.GetSecretValue(secret)
		if kubeconfig == nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("kubeconfig is empty: %v", err), ClusterName: cd.Name}
			return
		}

		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to create new client from kubeconfig: %v", err), ClusterName: cd.Name}
			return
		}

		metrics, err := collectMetrics(ctx, client)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to collect metrics: %v", err), ClusterName: cd.Name, PodMetrics: metrics}
			return
		}

		metricsChan <- &ClusterMetrics{
			PodMetrics:  metrics,
			ClusterName: cd.Name,
		}
	}()
}

func collectMetrics(ctx context.Context, kubeClient *k8s.KubeClient) (PodMetricsMap, error) {
	podList, err := k8s.GetCollectorPods(ctx, kubeClient.Client, client.HasLabels{k8s.CollectorMetricsLabel})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	if len(podList.Items) == 0 {
		return PodMetricsMap{}, nil
	}

	metrics := make(PodMetricsMap, len(podList.Items))
	errs := make([]error, 0, len(podList.Items))
	podMetricsCh := make(chan *PodMetrics)

	wg := sync.WaitGroup{}

	for _, pod := range podList.Items {
		wg.Add(1)
		go func(pod corev1.Pod) {
			defer wg.Done()
			podMetricsCh <- collectPodMetrics(ctx, kubeClient, pod)
		}(pod)
	}

	go func() {
		wg.Wait()
		close(podMetricsCh)
	}()

	for result := range podMetricsCh {
		metrics[result.PodName] = result.Metrics
		if result.Err != nil {
			errs = append(errs, result.Err)
		}
	}

	return metrics, errors.Join(errs...)
}

func collectPodMetrics(ctx context.Context, kubeClient *k8s.KubeClient, pod corev1.Pod) *PodMetrics {
	metrics := utils.Metrics{}
	errs := make([]error, 0, 3)

	healthMetrics, err := collectHealthMetrics(pod)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to collect health metrics: %v, podName: %s", err, pod.Name))
	}
	maps.Copy(metrics, healthMetrics)

	resourceMetrics, err := collectResourceMetrics(ctx, kubeClient, pod)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to collect resource metrics: %v, podName: %s", err, pod.Name))
	}
	maps.Copy(metrics, resourceMetrics)

	port, err := getMetricsPort(pod)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get metrics port: %v, podName: %s", err, pod.Name))
		return &PodMetrics{PodName: pod.Name, Metrics: metrics, Err: errors.Join(errs...)}
	}

	resp, err := k8s.Proxy(ctx, kubeClient.Clientset, pod, port, MetricsPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to proxy pod %s: %v", pod.Name, err))
		return &PodMetrics{PodName: pod.Name, Metrics: metrics, Err: errors.Join(errs...)}
	}

	parsed, err := utils.ParsePrometheusMetrics(string(resp))
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse prometheus metrics: %v, podName: %s", err, pod.Name))
		return &PodMetrics{PodName: pod.Name, Metrics: metrics, Err: errors.Join(errs...)}
	}

	maps.Copy(metrics, parsed)

	return &PodMetrics{
		PodName: pod.Name,
		Metrics: metrics,
		Err:     errors.Join(errs...),
	}
}

func collectHealthMetrics(pod corev1.Pod) (utils.Metrics, error) {
	metrics := make(utils.Metrics, 3)
	readyCondition := findPodReadyCondition(pod.Status.Conditions)

	if readyCondition == nil {
		metrics[ConditionReadyHealthyMetric] = "unhealthy"
		metrics[ConditionReadyReasonMetric] = "MissingReadyCondition"
		metrics[ConditionReadyMessageMetric] = "Pod status does not contain Ready condition"
		return metrics, fmt.Errorf("status PodReady not found in conditions")
	}

	if readyCondition.Status == corev1.ConditionTrue {
		metrics[ConditionReadyHealthyMetric] = "healthy"
	} else {
		metrics[ConditionReadyHealthyMetric] = "unhealthy"
		metrics[ConditionReadyReasonMetric] = readyCondition.Reason
		metrics[ConditionReadyMessageMetric] = readyCondition.Message
	}

	return metrics, nil
}

func collectResourceMetrics(ctx context.Context, client *k8s.KubeClient, pod corev1.Pod) (utils.Metrics, error) {
	metrics := utils.Metrics{}

	podMetrics, err := k8s.GetPodMetrics(ctx, client.MetricsClient, pod.Name, pod.Namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return metrics, nil
		}
		return metrics, fmt.Errorf("failed to get pod metrics: %v", err)
	}

	metricsContainer, err := findContainerMetrics(podMetrics.Containers, DefaultCollectorContainerName)
	if err != nil {
		return metrics, fmt.Errorf("failed to find collector container metrics: %v", err)
	}

	container := k8s.GetContainer(pod.Spec.Containers, DefaultCollectorContainerName)
	if container == nil {
		return metrics, fmt.Errorf("failed to find collector container spec: %v", err)
	}

	metrics[ContainerResourceCpuUsageMetric] = metricsContainer.Usage.Cpu().MilliValue()
	metrics[ContainerResourceMemoryUsageMetric] = metricsContainer.Usage.Memory().Value()

	containerCpuLimit := container.Resources.Limits.Cpu().MilliValue()
	containerMemoryLimit := container.Resources.Limits.Memory().Value()

	nodeMetrics, err := k8s.GetNodeMetrics(ctx, client.MetricsClient, pod.Spec.NodeName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return metrics, nil
		}
		return metrics, fmt.Errorf("failed to get node metrics: %v", err)
	}

	node, err := k8s.GetNode(ctx, client.Client, pod.Spec.NodeName)
	if err != nil {
		return metrics, fmt.Errorf("failed to get node spec: %v", err)
	}

	cpuLimit := getResourceLimit(node, nodeMetrics, containerCpuLimit, corev1.ResourceCPU)
	memoryLimit := getResourceLimit(node, nodeMetrics, containerMemoryLimit, corev1.ResourceMemory)

	metrics[ContainerResourceCpuLimitMetric] = cpuLimit
	metrics[ContainerResourceMemoryLimitMetric] = memoryLimit

	return metrics, nil
}

func findContainerMetrics(containers []v1beta1.ContainerMetrics, name string) (*v1beta1.ContainerMetrics, error) {
	for _, container := range containers {
		if container.Name == name {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("container %s not found in metrics", name)
}

func findPodReadyCondition(conditions []corev1.PodCondition) *corev1.PodCondition {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady {
			return &condition
		}
	}
	return nil
}

func getResourceLimit(node *corev1.Node, nodeMetrics *v1beta1.NodeMetrics, containerLimit int64, resourceType corev1.ResourceName) int64 {
	if containerLimit > 0 {
		return containerLimit
	}

	var totalResource, usedResource int64

	switch resourceType {
	case corev1.ResourceCPU:
		resourceQuantity := node.Status.Allocatable[corev1.ResourceCPU]
		totalResource = resourceQuantity.MilliValue()
		usedResource = nodeMetrics.Usage.Cpu().MilliValue()
	case corev1.ResourceMemory:
		resourceQuantity := node.Status.Allocatable[corev1.ResourceMemory]
		totalResource = resourceQuantity.Value()
		usedResource = nodeMetrics.Usage.Memory().Value()
	}

	return totalResource - usedResource
}

func getMetricsPort(pod corev1.Pod) (string, error) {
	if port, ok := pod.Annotations[MetricsPortAnnotation]; ok {
		return port, nil
	}

	return k8s.ExtractContainerPort(&pod, DefaultCollectorContainerName, MetricsPortName)
}
