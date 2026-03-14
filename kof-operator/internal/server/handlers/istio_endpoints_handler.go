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

type IstioMesh map[string]NamespaceMap

type NamespaceMap map[string]NamespaceEntry

type NamespaceEntry struct {
	Shards          map[string][]Endpoint        `json:"Shards"`
	ServiceAccounts map[string]ServiceAccountRaw `json:"ServiceAccounts"`
}

type Endpoint struct {
	Labels                 map[string]string  `json:"Labels"`
	Addresses              []string           `json:"Addresses"`
	ServicePortName        string             `json:"ServicePortName"`
	LegacyClusterPortKey   int                `json:"LegacyClusterPortKey"`
	ServiceAccount         string             `json:"ServiceAccount"`
	Network                string             `json:"Network"`
	Locality               Locality           `json:"Locality"`
	EndpointPort           int                `json:"EndpointPort"`
	LbWeight               int                `json:"LbWeight"`
	TLSMode                string             `json:"TLSMode"`
	Namespace              string             `json:"Namespace"`
	WorkloadName           string             `json:"WorkloadName"`
	HostName               string             `json:"HostName"`
	SubDomain              string             `json:"SubDomain"`
	HealthStatus           model.HealthStatus `json:"HealthStatus"`
	SendUnhealthyEndpoints bool               `json:"SendUnhealthyEndpoints"`
	NodeName               string             `json:"NodeName"`
}

type Locality struct {
	Label     string `json:"Label"`
	ClusterID string `json:"ClusterID"`
}

type ServiceAccountRaw map[string]interface{}

type ClusterConnectivity struct {
	SourceCluster          string              `json:"sourceCluster"`
	SourceClusterNamespace string              `json:"sourceClusterNamespace"`
	ConnectedClusters      []*ConnectedCluster `json:"remoteClusters"`
}

type ConnectedCluster struct {
	ClusterID string             `json:"clusterId"`
	Services  []*ServiceEndpoint `json:"services"`
}

type ServiceEndpoint struct {
	ServiceFQDN    string   `json:"serviceFqdn"`
	Namespace      string   `json:"namespace"`
	WorkloadName   string   `json:"workloadName"`
	Addresses      []string `json:"addresses"`
	Port           int      `json:"port"`
	ServiceAccount string   `json:"serviceAccount"`
	TLSMode        string   `json:"tlsMode"`
	Healthy        bool     `json:"healthy"`
}

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
	if clusterName == MothershipClusterName {
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

	connectivity := buildClusterConnectivity(clusterName, clusterNamespace, merged)
	return ClusterEndpoints{Cluster: clusterName, Endpoints: connectivity}, nil
}

func buildClusterConnectivity(sourceCluster, sourceClusterNamespace string, mesh IstioMesh) ClusterConnectivity {
	byCluster := map[string][]*ServiceEndpoint{}

	for fqdn, nsMap := range mesh {
		for _, entry := range nsMap {
			for shardKey, endpoints := range entry.Shards {
				clusterID := shardKeyToClusterID(shardKey)

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
						TLSMode:        ep.TLSMode,
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

func shardKeyToClusterID(shardKey string) string {
	if idx := strings.Index(shardKey, "/"); idx >= 0 {
		return shardKey[idx+1:]
	}
	return shardKey
}
