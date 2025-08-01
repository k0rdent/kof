package handlers

import (
	"context"
	"fmt"
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

type Metric struct {
	ClusterName string
	PodName     string
	MetricName  string
	MetricValue any
	Err         error
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
	cdList, err := k8s.GetKofClusterDeployments(ctx, h.kubeClient.Client)
	if err != nil {
		return nil, err
	}

	wg := &sync.WaitGroup{}
	metricCh := make(chan *Metric)
	ctx, cancel := context.WithTimeout(ctx, MaxResponseTime)
	defer cancel()

	getLocalCollectorMetricsAsync(ctx, h.kubeClient, metricCh, wg)

	for _, cd := range cdList.Items {
		getCollectorsMetricsAsync(ctx, h.kubeClient.Client, &cd, metricCh, wg)
	}

	go func() {
		wg.Wait()
		close(metricCh)
	}()

	errs := []error{}
	clustersMetrics := ClusterMetricsMap{}

	for metric := range metricCh {
		if metric.Err != nil {
			errs = append(errs, metric.Err)
		}

		if !isValidMetric(metric) {
			continue
		}

		ensureClusterExists(metric.ClusterName, clustersMetrics)
		ensurePodExists(metric.ClusterName, metric.PodName, clustersMetrics)

		clustersMetrics[metric.ClusterName][metric.PodName][metric.MetricName] = metric.MetricValue
	}

	if len(errs) > 0 {
		h.logger.Error(fmt.Errorf("%v", errs), "Some errors occurred during metrics fetching")
	}

	return &MetricsResponse{
		Clusters: clustersMetrics,
	}, nil
}

func getLocalCollectorMetricsAsync(ctx context.Context, client *k8s.KubeClient, metricCh chan *Metric, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectMetrics(ctx, client, MothershipClusterName, metricCh, wg)
	}()
}

func getCollectorsMetricsAsync(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment, metricCh chan *Metric, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		secretName := k8s.GetSecretName(cd)
		secret, err := k8s.GetSecret(ctx, client, secretName, cd.Namespace)
		if err != nil {
			metricCh <- &Metric{ClusterName: cd.Name, Err: fmt.Errorf("failed to get secret: %v", err)}
			return
		}

		kubeconfig := k8s.GetSecretValue(secret)
		if kubeconfig == nil {
			metricCh <- &Metric{ClusterName: cd.Name, Err: fmt.Errorf("kubeconfig is empty")}
			return
		}

		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			metricCh <- &Metric{ClusterName: cd.Name, Err: fmt.Errorf("failed to create new client from kubeconfig: %v", err)}
			return
		}

		collectMetrics(ctx, client, cd.Name, metricCh, wg)
	}()
}

func collectMetrics(ctx context.Context, kubeClient *k8s.KubeClient, clusterName string, metricCh chan *Metric, wg *sync.WaitGroup) {
	podList, err := k8s.GetCollectorPods(ctx, kubeClient.Client, client.HasLabels{k8s.CollectorMetricsLabel})
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, Err: fmt.Errorf("failed to list pods: %v", err)}
		return
	}

	if len(podList.Items) == 0 {
		return
	}

	for _, pod := range podList.Items {
		wg.Add(1)
		go func(pod corev1.Pod) {
			defer wg.Done()
			collectPodMetrics(ctx, kubeClient, clusterName, pod, metricCh)
		}(pod)
	}
}

func collectPodMetrics(ctx context.Context, kubeClient *k8s.KubeClient, clusterName string, pod corev1.Pod, metricCh chan *Metric) {
	collectHealthMetrics(pod, clusterName, metricCh)
	collectResourceMetrics(ctx, kubeClient, pod, clusterName, metricCh)

	port, err := getMetricsPort(pod)
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to get metrics port: %v", err)}
		return
	}

	resp, err := k8s.Proxy(ctx, kubeClient.Clientset, pod, port, MetricsPath)
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to proxy: %v", err)}
		return
	}

	parsed, err := utils.ParsePrometheusMetrics(string(resp))
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to parse prometheus metrics: %v", err)}
		return
	}

	for k, v := range parsed {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: k, MetricValue: v}
	}
}

func collectHealthMetrics(pod corev1.Pod, clusterName string, metricCh chan *Metric) {
	readyCondition := findPodReadyCondition(pod.Status.Conditions)

	if readyCondition == nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyHealthyMetric, MetricValue: "unhealthy"}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyReasonMetric, MetricValue: "MissingReadyCondition"}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyMessageMetric, MetricValue: "Pod status does not contain Ready condition"}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("status PodReady not found in conditions")}
		return
	}

	if readyCondition.Status == corev1.ConditionTrue {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyHealthyMetric, MetricValue: "healthy"}
	} else {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyHealthyMetric, MetricValue: "unhealthy"}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyReasonMetric, MetricValue: readyCondition.Reason}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ConditionReadyMessageMetric, MetricValue: readyCondition.Message}
	}
}

func collectResourceMetrics(ctx context.Context, client *k8s.KubeClient, pod corev1.Pod, clusterName string, metricCh chan *Metric) {
	podMetrics, err := k8s.GetPodMetrics(ctx, client.MetricsClient, pod.Name, pod.Namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return
		}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to get pod metrics: %v", err)}
		return
	}

	metricsContainer, err := findContainerMetrics(podMetrics.Containers, DefaultCollectorContainerName)
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to find collector container metrics: %v", err)}
		return
	}

	container := k8s.GetContainer(pod.Spec.Containers, DefaultCollectorContainerName)
	if container == nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to find collector container spec: %v", err)}
		return
	}

	metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ContainerResourceCpuUsageMetric, MetricValue: metricsContainer.Usage.Cpu().MilliValue()}
	metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ContainerResourceMemoryUsageMetric, MetricValue: metricsContainer.Usage.Memory().Value()}

	containerCpuLimit := container.Resources.Limits.Cpu().MilliValue()
	containerMemoryLimit := container.Resources.Limits.Memory().Value()

	nodeMetrics, err := k8s.GetNodeMetrics(ctx, client.MetricsClient, pod.Spec.NodeName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return
		}
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to get node metrics: %v", err)}
		return
	}

	node, err := k8s.GetNode(ctx, client.Client, pod.Spec.NodeName)
	if err != nil {
		metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, Err: fmt.Errorf("failed to get node spec: %v", err)}
		return
	}

	cpuLimit := getResourceLimit(node, nodeMetrics, containerCpuLimit, corev1.ResourceCPU)
	memoryLimit := getResourceLimit(node, nodeMetrics, containerMemoryLimit, corev1.ResourceMemory)

	metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ContainerResourceCpuLimitMetric, MetricValue: cpuLimit}
	metricCh <- &Metric{ClusterName: clusterName, PodName: pod.Name, MetricName: ContainerResourceMemoryLimitMetric, MetricValue: memoryLimit}
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

func isValidMetric(metric *Metric) bool {
	return metric.ClusterName != "" &&
		metric.PodName != "" &&
		metric.MetricName != ""
}
func ensureClusterExists(clusterName string, clustersMetrics map[string]PodMetricsMap) {
	if _, exists := clustersMetrics[clusterName]; !exists {
		clustersMetrics[clusterName] = make(PodMetricsMap)
	}
}
func ensurePodExists(clusterName, podName string, clustersMetrics map[string]PodMetricsMap) {
	if _, exists := clustersMetrics[clusterName][podName]; !exists {
		clustersMetrics[clusterName][podName] = make(utils.Metrics)
	}
}
