package env

import (
	"fmt"
	"os"

	"github.com/k0rdent/kof/kof-operator/internal/strutil"
)

// GetEnvOrDefault returns the value of the environment variable specified by key.
// If the environment variable is not set or is empty, it returns the provided default value.
func GetEnvOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return value
}

// GetPropagationTemplateName returns the ServiceTemplate name for the kof-propagation chart.
// It reads KOF_PROPAGATION_TEMPLATE env var set by the Helm deployment.
func GetPropagationTemplateName() string {
	return GetEnvOrDefault("KOF_PROPAGATION_TEMPLATE", "kof-propagation")
}

// GrafanaEnabled checks if Grafana integration is enabled by reading the KOF_GRAFANA_ENABLED environment variable.
func GrafanaEnabled() bool {
	return GetEnvOrDefault("KOF_GRAFANA_ENABLED", strutil.False) == strutil.True
}

// RegionlessEnabled checks if the regionless setup is enabled by reading the KOF_REGIONLESS_ENABLED environment variable.
// In this setup, there are no regional clusters, and child clusters send metrics/logs/traces to the management cluster for storage.
func RegionlessEnabled() bool {
	return GetEnvOrDefault("KOF_REGIONLESS_ENABLED", strutil.False) == strutil.True
}

// GetRegionlessDomain returns the domain used when child clusters send KOF data to the management cluster without Istio.
func GetRegionlessDomain() string {
	return GetEnvOrDefault("KOF_REGIONLESS_DOMAIN", "mothership.example.com")
}

// GetRegionlessHTTPConfig returns the JSON HTTP config used when child clusters send KOF data to the management cluster.
func GetRegionlessHTTPConfig() string {
	return GetEnvOrDefault("KOF_REGIONLESS_HTTP_CONFIG", `{"tls_config": {"insecure_skip_verify": false}}`)
}

// CrossNamespaceEnabled checks if cross-namespace support is enabled by reading the CROSS_NAMESPACE environment variable.
func CrossNamespaceEnabled() bool {
	return GetEnvOrDefault("CROSS_NAMESPACE", strutil.False) == strutil.True
}

// GetRegionalMCSName returns the name of the regional MultiClusterService
func GetRegionalMCSName() string {
	return GetEnvOrDefault("KOF_REGIONAL_CLUSTER_MCS_NAME", "kof-regional-cluster")
}

// GetChildMCSName returns the name of the child MultiClusterService
func GetChildMCSName() string {
	return GetEnvOrDefault("KOF_CHILD_CLUSTER_MCS_NAME", "kof-child-cluster")
}

// GetIstioRegionalMCSName returns the name of the regional MultiClusterService for Istio clusters
func GetIstioRegionalMCSName() string {
	return GetEnvOrDefault("KOF_ISTIO_REGIONAL_CLUSTER_MCS_NAME", "kof-istio-regional-cluster")
}

// GetIstioChildMCSName returns the name of the child MultiClusterService for Istio clusters
func GetIstioChildMCSName() string {
	return GetEnvOrDefault("KOF_ISTIO_CHILD_CLUSTER_MCS_NAME", "kof-istio-child-cluster")
}

// GetVTClusterName returns the name of the VTCluster to register regional storage nodes with.
// Returns "" when not configured, in which case VMStorageConnection creation is skipped.
func GetVTClusterName() string {
	return os.Getenv("KOF_VT_CLUSTER_NAME")
}

// GetVLClusterName returns the name of the VLCluster to register regional storage nodes with.
// Returns "" when not configured, in which case VMStorageConnection creation is skipped.
func GetVLClusterName() string {
	return os.Getenv("KOF_VL_CLUSTER_NAME")
}

// GetReleaseNamespace returns the namespace in which the operator is deployed.
func GetReleaseNamespace() (string, error) {
	namespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return "", fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}
	if len(namespace) == 0 {
		return "", fmt.Errorf("RELEASE_NAMESPACE env var is set but empty")
	}
	return namespace, nil
}
