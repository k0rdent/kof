package handlers

import (
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	"istio.io/istio/pilot/pkg/model"
	v1 "k8s.io/api/core/v1"
)

var ManagementClusterName = "mothership"

type ResourceStatus struct {
	MessageType metrics.MessageType `json:"type,omitempty"`
	Message     string              `json:"message,omitempty"`
}

type ICustomResource interface {
	GetPods() ([]v1.Pod, error)
	GetName() string
	GetStatus() *ResourceStatus
}

// IstioMesh represents the top-level mapping from a service
type IstioMesh map[string]NamespaceMap

// NamespaceMap maps a namespace name to its entry data.
type NamespaceMap map[string]NamespaceEntry

// NamespaceEntry holds shards and service accounts for a namespace.
type NamespaceEntry struct {
	Shards          map[string][]Endpoint        `json:"Shards"`
	ServiceAccounts map[string]ServiceAccountRaw `json:"ServiceAccounts"`
}

// Endpoint is one observed endpoint instance in a shard.
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

// Locality describes endpoint locality information.
type Locality struct {
	Label     string `json:"Label"`
	ClusterID string `json:"ClusterID"`
}

// ServiceAccountRaw is a flexible holder for service-account related metadata.
// The JSON sample shows empty objects; using a map allows future fields.
type ServiceAccountRaw map[string]interface{}

// ClusterConnectivity is the simplified view of which remote clusters a given
// cluster has endpoints for, and what workloads those clusters expose.
type ClusterConnectivity struct {
	SourceCluster          string              `json:"sourceCluster"`
	SourceClusterNamespace string              `json:"sourceClusterNamespace"`
	ConnectedClusters      []*ConnectedCluster `json:"connectedClusters"`
}

// ConnectedCluster groups all service endpoints discovered from a single remote
// cluster.
type ConnectedCluster struct {
	ClusterID string             `json:"clusterId"`
	Services  []*ServiceEndpoint `json:"services"`
}

// ServiceEndpoint is a flattened, human-readable view of one remote endpoint.
type ServiceEndpoint struct {
	ServiceFQDN    string   `json:"serviceFqdn"`
	Namespace      string   `json:"namespace"`
	WorkloadName   string   `json:"workloadName"`
	Addresses      []string `json:"addresses"`
	Port           int      `json:"port"`
	ServiceAccount string   `json:"serviceAccount"`
	Healthy        bool     `json:"healthy"`
}

// MeshNode represents a cluster in the Istio multi-cluster mesh topology.
type MeshNode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Role      string `json:"role"`
}

// MeshLink represents a connection between two clusters established via
// an Istio remote secret.
type MeshLink struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	SecretName string `json:"secretName"`
}

// ClusterEndpoints groups all endpoints discovered from a single cluster's
// istiod instance.
type ClusterEndpoints struct {
	Cluster   string              `json:"cluster"`
	Endpoints ClusterConnectivity `json:"endpoints"`
}

// MeshGraph is the full topology returned by IstioMeshHandler.
type MeshGraph struct {
	Nodes []MeshNode `json:"nodes"`
	Links []MeshLink `json:"links"`
}

// MeshEndpointsResponse is the response type for the dedicated /api/istio/endpoints endpoint.
type MeshEndpointsResponse struct {
	Endpoints []ClusterEndpoints `json:"endpoints"`
}
