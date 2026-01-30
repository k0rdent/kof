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
)

type BaseMetricsHandler struct {
	kubeClient *k8s.KubeClient
	logger     *logr.Logger
	wg         *sync.WaitGroup
	ctx        context.Context
	resourceCh metrics.ResourceChannel
	config     *MetricsConfig
}

type MetricsConfig struct {
	GetCustomResourcesFn  func(context.Context, *k8s.KubeClient) ([]ICustomResource, error)
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
		resourceCh: make(metrics.ResourceChannel),
		wg:         &sync.WaitGroup{},
		config:     cfg,
	}
}

func (h *BaseMetricsHandler) GetMetrics() metrics.ClusterMap {
	var cancel context.CancelFunc
	h.ctx, cancel = context.WithTimeout(h.ctx, CollectorMaxResponseTime)
	defer cancel()

	h.CollectLocalMetricsAsync()
	h.CollectRemoteMetricsAsync()

	go func() {
		h.wg.Wait()
		close(h.resourceCh)
	}()

	clusters := make(metrics.ClusterMap)
	errs := make([]error, 0)

	for resource := range h.resourceCh {
		if resource.Metrics != nil {
			if resource.Metrics.Err != nil {
				errs = append(errs, resource.Metrics.Err)
				continue
			}
			clusters.AddMetric(resource.Metrics)
		}

		if resource.Status != nil {
			clusters.AddStatus(resource.Status)
		}
	}

	if len(errs) > 0 {
		h.logger.Error(fmt.Errorf("%v", errs), "Some errors occurred during metrics fetching")
	}

	return clusters
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
		if remoteCluster.Error != nil {
			h.logger.Error(remoteCluster.Error, "failed to get kubeclient for remote cluster", "cluster", remoteCluster.Name)
			continue
		}

		h.wg.Add(1)
		go func(remoteCluster *RemoteCluster) {
			defer h.wg.Done()
			h.CollectMetrics(remoteCluster.KubeClient, remoteCluster.Name)
		}(remoteCluster)
	}
}

func (h *BaseMetricsHandler) CollectMetrics(kubeClient *k8s.KubeClient, clusterName string) {
	customResources, err := h.config.GetCustomResourcesFn(h.ctx, kubeClient)
	if err != nil {
		h.logger.Error(err, "failed to get pods for metrics collection", "cluster", clusterName)
		return
	}

	if len(customResources) == 0 {
		return
	}

	for _, cr := range customResources {
		customResourcesName := cr.GetName()
		pods, err := cr.GetPods()
		if err != nil {
			h.logger.Error(err, "failed to get pods of custom resource", "customResource", customResourcesName, "cluster", clusterName)
			continue
		}

		status := cr.GetStatus()
		if status != nil {
			h.resourceCh <- &metrics.ResourceMessage{
				Status: &metrics.StatusMessage{
					ResourceAddress: metrics.ResourceAddress{
						Cluster:        clusterName,
						CustomResource: customResourcesName,
					},
					Type:    status.MessageType,
					Message: status.Message,
				},
			}
		}

		for _, pod := range pods {
			containerName := h.config.ContainerName
			if containerName == "" {
				containerName = pod.Spec.Containers[0].Name
			}

			h.wg.Add(1)
			go func(pod corev1.Pod) {
				defer h.wg.Done()
				cfg := &metrics.MetricCollectorServiceConfig{
					KubeClient:         kubeClient,
					Pod:                &pod,
					ClusterName:        clusterName,
					ContainerName:      containerName,
					CustomResourceName: customResourcesName,
					Ctx:                h.ctx,
					MetricsChan:        h.resourceCh,
					PortAnnotation:     h.config.MetricsPortAnnotation,
					PortName:           h.config.PortName,
					ProxyEndpoint:      h.config.MetricsEndpoint,
				}
				metrics.New(cfg).CollectAll()
			}(pod)
		}
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

	h.handleRegionClusters(ch, regions, creds, clusters, processed, wg)
	h.handleOtherClusters(ch, clusters, processed, wg)
}

func (h *BaseMetricsHandler) handleRegionClusters(
	ch chan *RemoteCluster,
	regions *kcmv1beta1.RegionList,
	creds *kcmv1beta1.CredentialList,
	clusters *kcmv1beta1.ClusterDeploymentList,
	processed *sync.Map,
	wg *sync.WaitGroup,
) {
	regionClusters, err := k8s.GetKcmRegionClusters(h.ctx, h.kubeClient.Client)
	if err != nil {
		h.logger.Error(err, "failed to get region clusters")
		return
	}

	cache := k8s.CachedClusterData{
		Regions:     regions,
		Credentials: creds,
		Clusters:    clusters,
	}

	for _, cd := range regionClusters {
		childs, err := k8s.GetKcmRegionChildClusters(h.ctx, h.kubeClient.Client, cd, cache)
		if err != nil {
			h.logger.Error(err, "failed to get child clusters for region cluster", "clusterDeployment", cd.Name)
			continue
		}

		regionSecretName, err := k8s.GetKubeconfigSecretName(h.ctx, k8s.LocalKubeClient.Client, cd)
		if err != nil {
			h.logger.Error(err, "Failed to get secret name for cluster deployment", "clusterDeployment", cd.Name)
			continue
		}

		regionClient, err := h.createAndSendKubeClient(ch, regionSecretName, cd.Name, h.kubeClient.Client, processed)
		if err != nil {
			h.logger.Error(err, "failed to create region kubeclient", "secret", regionSecretName)
			continue
		}

		for _, child := range childs {
			wg.Add(1)
			go func(cd *kcmv1beta1.ClusterDeployment, client *k8s.KubeClient) {
				defer wg.Done()
				secret, err := k8s.GetKubeconfigSecretName(h.ctx, k8s.LocalKubeClient.Client, cd)
				if err != nil {
					h.logger.Error(err, "failed to get secret name for regional cluster", "clusterDeployment", cd.Name)
					return
				}

				if _, err := h.createAndSendKubeClient(ch, secret, cd.Name, client.Client, processed); err != nil {
					h.logger.Error(err, "failed to create kubeclient for regional cluster", "secret", secret)
				}
			}(child, regionClient)
		}
	}
}

func (h *BaseMetricsHandler) handleOtherClusters(
	ch chan *RemoteCluster,
	clusters *kcmv1beta1.ClusterDeploymentList,
	processed *sync.Map,
	wg *sync.WaitGroup,
) {
	for _, cd := range clusters.Items {
		if _, done := processed.Load(cd.Name); done {
			continue
		}

		wg.Add(1)
		go func(cd kcmv1beta1.ClusterDeployment) {
			defer wg.Done()
			secret, err := k8s.GetKubeconfigSecretName(h.ctx, k8s.LocalKubeClient.Client, &cd)
			if err != nil {
				h.logger.Error(err, "failed to get secret name for regional cluster", "clusterDeployment", cd.Name)
				return
			}

			if _, err := h.createAndSendKubeClient(ch, secret, cd.Name, h.kubeClient.Client, processed); err != nil {
				h.logger.Error(err, "failed to create kubeclient for cluster", "secret", secret)
			}
		}(cd)
	}
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
	ctx := h.ctx
	client := h.kubeClient.Client

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
