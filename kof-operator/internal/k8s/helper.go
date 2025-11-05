package k8s

import (
	"strings"
)

func GetClusterNameByKubeconfigSecretName(secretName string) string {
	if clusterName, ok := strings.CutSuffix(secretName, "-"+ClusterSecretSuffix); ok {
		return clusterName
	}

	if clusterName, ok := strings.CutSuffix(secretName, "-"+AdoptedClusterSecretSuffix); ok {
		return clusterName
	}

	return ""
}
