package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
	"k8s.io/apimachinery/pkg/api/errors"
)

func IstioMeshEndpointsHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	clusterNamespace := req.URL.Query().Get("namespace")
	clusterName := req.URL.Query().Get("cluster")
	if clusterName == "" {
		res.Fail("missing required query parameter: cluster", http.StatusBadRequest)
		return
	}

	graph, err := getIstioEndpoints(ctx, res.Logger, clusterName, clusterNamespace)
	if err != nil {
		res.Logger.Error(err, "Failed to get Istio endpoints", "cluster", clusterName)
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.SendObj(graph, http.StatusOK)
}

func getIstioEndpoints(ctx context.Context, logger *logr.Logger, clusterName, clusterNamespace string) (MeshEndpointsResponse, error) {
	var kubeClient *k8s.KubeClient
	if clusterName == ManagementClusterName {
		kubeClient = k8s.LocalKubeClient
	} else {
		if clusterNamespace == "" {
			return MeshEndpointsResponse{}, fmt.Errorf("missing required query parameter: namespace")
		}

		cluster, err := k8s.GetClusterDeployment(ctx, k8s.LocalKubeClient.Client, clusterName, clusterNamespace)
		if err != nil && !errors.IsNotFound(err) {
			return MeshEndpointsResponse{}, fmt.Errorf("failed to get ClusterDeployment for cluster %q: %w", clusterName, err)
		}

		if errors.IsNotFound(err) {
			return MeshEndpointsResponse{}, fmt.Errorf("cluster %q not found in namespace %q", clusterName, clusterNamespace)
		}

		rc, err := k8s.NewKubeClientFromClusterDeployment(ctx, k8s.LocalKubeClient.Client, cluster)
		if err != nil {
			return MeshEndpointsResponse{}, fmt.Errorf("failed to create kube client for cluster %q: %w", clusterName, err)
		}

		kubeClient = rc
	}

	ce, err := collectEndpointsFromCluster(ctx, logger, kubeClient, clusterName, clusterNamespace)
	if err != nil {
		return MeshEndpointsResponse{}, fmt.Errorf("failed to collect Istio endpoints for cluster %q: %w", clusterName, err)
	}

	return MeshEndpointsResponse{Endpoints: []ClusterEndpoints{ce}}, nil
}

func collectEndpointsFromCluster(
	ctx context.Context,
	logger *logr.Logger,
	kubeClient *k8s.KubeClient,
	clusterName string,
	clusterNamespace string,
) (ClusterEndpoints, error) {
	istioClient, err := kube.NewCLIClient(kubeClient.Config)
	if err != nil {
		return ClusterEndpoints{Cluster: clusterName}, fmt.Errorf("failed to create Istio CLI client for %s: %w", clusterName, err)
	}

	results, err := istioClient.AllDiscoveryDo(ctx, constants.IstioSystemNamespace, "debug/endpointz")
	if err != nil {
		return ClusterEndpoints{Cluster: clusterName}, fmt.Errorf("istio AllDiscoveryDo endpointz for %s: %w", clusterName, err)
	}

	merged := IstioMesh{}
	for _, data := range results {
		var entries IstioMesh
		if err := json.Unmarshal(data, &entries); err != nil {
			logger.Error(err, "Failed to unmarshal endpointz response", "cluster", clusterName)
			continue
		}
		for svc, nsMap := range entries {
			merged[svc] = nsMap
		}
	}

	connectivity := BuildClusterConnectivity(clusterName, clusterNamespace, merged)
	return ClusterEndpoints{Cluster: clusterName, Endpoints: connectivity}, nil
}

// BuildClusterConnectivity converts raw IstioMesh data into a
// ClusterConnectivity summary. Endpoints whose shard key resolves to
// sourceCluster are excluded (self-references are not useful to the caller).
//
// Shard keys use the format "Kubernetes/<clusterID>".
func BuildClusterConnectivity(sourceCluster, sourceClusterNamespace string, mesh IstioMesh) ClusterConnectivity {
	byCluster := map[string][]*ServiceEndpoint{}

	for fqdn, nsMap := range mesh {
		for _, entry := range nsMap {
			for shardKey, endpoints := range entry.Shards {
				clusterID := ShardKeyToClusterID(shardKey)

				if clusterID == sourceCluster {
					continue
				}
				for _, ep := range endpoints {
					byCluster[clusterID] = append(byCluster[clusterID], &ServiceEndpoint{
						ServiceFQDN:    fqdn,
						Namespace:      ep.Namespace,
						WorkloadName:   ep.WorkloadName,
						Addresses:      ep.Addresses,
						Port:           ep.EndpointPort,
						ServiceAccount: ep.ServiceAccount,
						Healthy:        model.Healthy == ep.HealthStatus,
					})
				}
			}
		}
	}

	remotes := make([]*ConnectedCluster, 0, len(byCluster))
	for clusterID, svcs := range byCluster {
		remotes = append(remotes, &ConnectedCluster{
			ClusterID: clusterID,
			Services:  svcs,
		})
	}

	return ClusterConnectivity{
		SourceCluster:          sourceCluster,
		SourceClusterNamespace: sourceClusterNamespace,
		ConnectedClusters:      remotes,
	}
}

// ShardKeyToClusterID extracts the cluster ID from a shard key.
// Expected format: "Kubernetes/<clusterID>". Falls back to the full key if the
// format is unexpected.
func ShardKeyToClusterID(shardKey string) string {
	if idx := strings.Index(shardKey, "/"); idx >= 0 {
		return shardKey[idx+1:]
	}
	return shardKey
}
