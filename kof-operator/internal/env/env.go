package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

// GetEnvBool returns the boolean value of the environment variable key, or def
// when the variable is unset, empty, or not a valid boolean string.
func GetEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// GetEnvInt returns the integer value of the environment variable key, or def
// when the variable is unset, empty, or not a valid integer string.
func GetEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GetEnvDuration returns the time.Duration value of the environment variable
// key, or def when the variable is unset, empty, or not a valid duration string.
func GetEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
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
