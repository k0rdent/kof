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

type RemoteCluster struct {
	Name       string
	KubeClient *k8s.KubeClient
	Error      error
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

func (h *BaseMetricsHandler) GetMetrics() metrics.Cluster {
	var cancel context.CancelFunc
	h.ctx, cancel = context.WithTimeout(h.ctx, CollectorMaxResponseTime)
	defer cancel()

	h.CollectLocalMetricsAsync()
	h.CollectRemoteMetricsAsync()

	go func() {
		h.wg.Wait()
		close(h.metricCh)
	}()

	metrics := make(metrics.Cluster)
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
	h.wg.Go(func() {
		h.CollectMetrics(h.kubeClient, MothershipClusterName)
	})
}

func (h *BaseMetricsHandler) CollectRemoteMetricsAsync() {
	remoteClustersChan := make(chan *RemoteCluster)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go h.collectRemoteClusterKubeClients(remoteClustersChan, wg)

	go func() {
		wg.Wait()
		close(remoteClustersChan)
	}()

	for remoteCluster := range remoteClustersChan {
		h.wg.Add(1)
		go func(remoteCluster *RemoteCluster) {
			defer h.wg.Done()
			h.CollectMetrics(remoteCluster.KubeClient, remoteCluster.Name)
		}(remoteCluster)
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

func (h *BaseMetricsHandler) collectRemoteClusterKubeClients(ch chan *RemoteCluster, wg *sync.WaitGroup) {
	defer wg.Done()

	processed := new(sync.Map)

	regions, creds, clusters, err := h.fetchClusterData()
	if err != nil {
		ch <- &RemoteCluster{Error: err}
		return
	}

	if err := h.handleRegionClusters(ch, regions, creds, clusters, processed); err != nil {
		h.logger.Error(err, "failed to handle region clusters")
	}

	if err := h.handleOtherClusters(ch, clusters, processed); err != nil {
		h.logger.Error(err, "failed to handle other clusters")
	}
}

func (h *BaseMetricsHandler) handleRegionClusters(
	ch chan *RemoteCluster,
	regions *kcmv1beta1.RegionList,
	creds *kcmv1beta1.CredentialList,
	clusters *kcmv1beta1.ClusterDeploymentList,
	processed *sync.Map,
) error {
	wg := &sync.WaitGroup{}

	for _, region := range regions.Items {
		regionSecret := region.Spec.KubeConfig.Name
		regionCluster := k8s.GetClusterNameByKubeconfigSecretName(regionSecret)

		regionClient, err := h.createAndSendKubeClient(ch, regionSecret, regionCluster, h.kubeClient.Client, processed)
		if err != nil {
			h.logger.Error(err, "failed to create region kubeclient", "secret", regionSecret)
			continue
		}

		credName := findCredentialNameForRegion(region.Name, creds)
		if credName == "" {
			continue
		}

		for _, cd := range clusters.Items {
			if cd.Spec.Credential != credName {
				continue
			}

			wg.Add(1)
			go func(cd kcmv1beta1.ClusterDeployment) {
				defer wg.Done()
				secret := k8s.GetSecretName(&cd)
				if _, err := h.createAndSendKubeClient(ch, secret, cd.Name, regionClient.Client, processed); err != nil {
					h.logger.Error(err, "failed to create kubeclient for regional cluster", "secret", secret)
				}
			}(cd)
		}
	}

	wg.Wait()
	return nil
}

func (h *BaseMetricsHandler) handleOtherClusters(
	ch chan *RemoteCluster,
	clusters *kcmv1beta1.ClusterDeploymentList,
	processed *sync.Map,
) error {
	wg := &sync.WaitGroup{}

	for _, cd := range clusters.Items {
		if _, done := processed.Load(cd.Name); done {
			continue
		}

		wg.Add(1)
		go func(cd kcmv1beta1.ClusterDeployment) {
			defer wg.Done()
			secret := k8s.GetSecretName(&cd)
			if _, err := h.createAndSendKubeClient(ch, secret, cd.Name, h.kubeClient.Client, processed); err != nil {
				h.logger.Error(err, "failed to create kubeclient for cluster", "secret", secret)
			}
		}(cd)
	}

	wg.Wait()
	return nil
}

func (h *BaseMetricsHandler) createAndSendKubeClient(
	ch chan *RemoteCluster,
	secretName, clusterName string,
	client client.Client,
	processed *sync.Map,
) (*k8s.KubeClient, error) {
	kc, err := k8s.NewKubeClientFromSecret(h.ctx, client, secretName, k8s.DefaultSystemNamespace)
	if err != nil {
		return nil, err
	}

	ch <- &RemoteCluster{Name: clusterName, KubeClient: kc}
	processed.Store(clusterName, struct{}{})
	return kc, nil
}

func (h *BaseMetricsHandler) fetchClusterData() (*kcmv1beta1.RegionList, *kcmv1beta1.CredentialList, *kcmv1beta1.ClusterDeploymentList, error) {
	ctx, client := h.ctx, h.kubeClient.Client

	regionList := new(kcmv1beta1.RegionList)
	credList := new(kcmv1beta1.CredentialList)
	clusterList := new(kcmv1beta1.ClusterDeploymentList)

	if err := client.List(ctx, regionList); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get region list: %w", err)
	}
	if err := client.List(ctx, credList); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get credential list: %w", err)
	}
	if err := client.List(ctx, clusterList); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get cluster deployment list: %w", err)
	}
	return regionList, credList, clusterList, nil
}

func findCredentialNameForRegion(regionName string, credList *kcmv1beta1.CredentialList) string {
	for _, cred := range credList.Items {
		if cred.Spec.Region == regionName {
			return cred.Name
		}
	}
	return ""
}
