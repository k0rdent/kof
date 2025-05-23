package k8s

import (
	corev1 "k8s.io/api/core/v1"
)

type Cluster struct {
	Name   string
	Secret *corev1.Secret
}

func (c *Cluster) GetKubeconfig() []byte {
	kubeconfig, ok := c.Secret.Data["value"]
	if !ok {
		return []byte{}
	}
	return kubeconfig
}
