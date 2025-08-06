package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type BaseMetricsHandler struct {
	kubeClient *k8s.KubeClient
	logger     *logr.Logger
	wg         *sync.WaitGroup
	ctx        context.Context
	metricCh   chan *metrics.Metric
	config     *MetricsConfig
}

type MetricsConfig struct {
	PodFilter             []client.ListOption
	MaxResponseTime       time.Duration
	MetricsPortAnnotation string
	PortName              string
	MetricsEndpoint       string
	ContainerName         string
}

func NewBaseMetricsHandler(ctx context.Context, kubeClient *k8s.KubeClient, logger *logr.Logger, cfg *MetricsConfig) *BaseMetricsHandler {
	return &BaseMetricsHandler{
		kubeClient: kubeClient,
		logger:     logger,
		ctx:        ctx,
		metricCh:   make(chan *metrics.Metric),
		wg:         &sync.WaitGroup{},
		config:     cfg,
	}
}

func (h *BaseMetricsHandler) GetMetrics() metrics.ClusterMetrics {
	var cancel context.CancelFunc
	h.ctx, cancel = context.WithTimeout(h.ctx, CollectorMaxResponseTime)
	defer cancel()

	h.CollectLocalMetricsAsync()
	h.CollectRemoteMetricsAsync()

	go func() {
		h.wg.Wait()
		close(h.metricCh)
	}()

	metrics := make(metrics.ClusterMetrics)
	errs := make([]error, 0)

	for metric := range h.metricCh {
		if metric.Err != nil {
			errs = append(errs, metric.Err)
			continue
		}

		metrics.Add(metric)
	}

	if len(errs) > 0 {
		h.logger.Error(fmt.Errorf("%v", errs), "Some errors occurred during metrics fetching")
	}

	return metrics
}

func (h *BaseMetricsHandler) CollectLocalMetricsAsync() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.CollectMetrics(h.kubeClient, MothershipClusterName)
	}()
}

func (h *BaseMetricsHandler) CollectRemoteMetricsAsync() {
	cdList, err := k8s.GetKofClusterDeployments(h.ctx, h.kubeClient.Client)
	if err != nil {
		h.metricCh <- &metrics.Metric{Err: fmt.Errorf("failed to get ClusterDeployments: %v", err)}
		return
	}

	for _, cd := range cdList.Items {
		h.wg.Add(1)
		go func(cd kcmv1beta1.ClusterDeployment) {
			defer h.wg.Done()
			kubeClient, err := k8s.NewKubeClientFromClusterDeployment(h.ctx, h.kubeClient.Client, &cd)
			if err != nil {
				h.metricCh <- &metrics.Metric{Err: fmt.Errorf("failed to create client from ClusterDeployment: %v", err)}
				return
			}

			h.CollectMetrics(kubeClient, cd.Name)
		}(cd)
	}
}

func (h *BaseMetricsHandler) CollectMetrics(kubeClient *k8s.KubeClient, clusterName string) {
	podList, err := k8s.GetPods(h.ctx, kubeClient.Client, h.config.PodFilter...)
	if err != nil {
		h.metricCh <- &metrics.Metric{Err: fmt.Errorf("failed to list pods: %v", err)}
		return
	}

	if len(podList.Items) == 0 {
		return
	}

	log := log.FromContext(h.ctx)
	for _, pod := range podList.Items {
		containerName := h.config.ContainerName
		if containerName == "" {
			containerName = pod.Spec.Containers[0].Name
			log.Info(
				"Container name is not defined in the metrics service config; using the first container from the pod",
				"ContainerName", containerName,
			)
		}

		h.wg.Add(1)
		go func(pod corev1.Pod) {
			defer h.wg.Done()
			cfg := &metrics.ServiceConfig{
				KubeClient:     kubeClient,
				Pod:            &pod,
				ClusterName:    clusterName,
				ContainerName:  containerName,
				Ctx:            h.ctx,
				Metrics:        h.metricCh,
				PortAnnotation: h.config.MetricsPortAnnotation,
				PortName:       h.config.PortName,
				ProxyEndpoint:  h.config.MetricsEndpoint,
			}
			metrics.New(cfg).CollectAll()
		}(pod)
	}
}
