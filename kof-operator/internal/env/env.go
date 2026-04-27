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
