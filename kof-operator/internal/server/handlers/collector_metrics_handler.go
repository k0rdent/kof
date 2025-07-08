package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/utils"
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

type ClusterMetricsMap map[string]PodMetricsMap
type PodMetricsMap map[string]utils.Metrics

type MetricsResponse struct {
	Clusters ClusterMetricsMap `json:"clusters"`
}

const (
	CollectorPort = "8888"
	MetricsPath   = "metrics"
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

	cdList, err := k8s.GetClusterDeployments(ctx, h.kubeClient.Client)
	if err != nil {
		return nil, err
	}

	if len(cdList.Items) == 0 {
		h.logger.Info("Cluster deployments not found")
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	metricsChan := make(chan *ClusterMetrics)
	wg := &sync.WaitGroup{}

	getLocalCollectorMetricsAsync(ctx, h.kubeClient, cancel, metricsChan, wg)

	for _, cd := range cdList.Items {
		getCollectorsMetricsAsync(ctx, h.kubeClient.Client, &cd, cancel, metricsChan, wg)
	}

	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	for metrics := range metricsChan {
		if metrics.Err == nil {
			resp.Clusters[metrics.ClusterName] = metrics.PodMetrics
			continue
		}

		if errors.Is(metrics.Err, context.Canceled) {
			continue
		}

		h.logger.Error(metrics.Err, "failed to receive metrics", "clusterName", metrics.ClusterName)
	}

	return resp, nil
}

func getLocalCollectorMetricsAsync(ctx context.Context, client *k8s.KubeClient, cancel context.CancelFunc, metricsChan chan *ClusterMetrics, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics, err := collectMetrics(ctx, client)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to collect metrics: %v", err), ClusterName: MothershipClusterName}
			cancel()
			return
		}
		metricsChan <- &ClusterMetrics{
			PodMetrics:  metrics,
			ClusterName: MothershipClusterName,
		}
	}()
}

func getCollectorsMetricsAsync(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment, cancel context.CancelFunc, metricsChan chan *ClusterMetrics, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		secretName := k8s.GetSecretName(cd)
		secret, err := k8s.GetSecret(ctx, client, secretName, cd.Namespace)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to get secret: %v", err), ClusterName: cd.Name}
			cancel()
			return
		}

		kubeconfig := k8s.GetSecretValue(secret)
		if kubeconfig == nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("kubeconfig is empty: %v", err), ClusterName: cd.Name}
			cancel()
			return
		}

		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to create new client from kubeconfig: %v", err), ClusterName: cd.Name}
			cancel()
			return
		}

		metrics, err := collectMetrics(ctx, client)
		if err != nil {
			metricsChan <- &ClusterMetrics{Err: fmt.Errorf("failed to collect metrics: %v", err), ClusterName: cd.Name}
			cancel()
			return
		}

		metricsChan <- &ClusterMetrics{
			PodMetrics:  metrics,
			ClusterName: cd.Name,
		}
	}()
}

func collectMetrics(ctx context.Context, client *k8s.KubeClient) (PodMetricsMap, error) {
	podList, err := k8s.GetCollectorPods(ctx, client.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	if len(podList.Items) == 0 {
		return PodMetricsMap{}, nil
	}

	metrics := make(PodMetricsMap, len(podList.Items))

	for _, pod := range podList.Items {
		response, err := k8s.Proxy(ctx, client.Clientset, pod, CollectorPort, MetricsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to proxy pod %s: %v", pod.Name, err)
		}

		metricsData, err := utils.ParsePrometheusMetrics(string(response))
		if err != nil {
			return nil, fmt.Errorf("failed to parse prometheus metrics: %v, podName: %s", err, pod.Name)
		}

		metrics[pod.Name] = metricsData
	}
	return metrics, nil
}
