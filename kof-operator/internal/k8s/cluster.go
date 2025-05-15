package k8s

import (
	"context"
	"fmt"
)

func (c *KubeClient) GetClusterName(ctx context.Context) (string, error) {
	rawConfig, err := c.Config.RawConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get raw config: %v", err)
	}

	currentContext := rawConfig.CurrentContext
	clusterName := rawConfig.Contexts[currentContext].Cluster

	return clusterName, nil
}
