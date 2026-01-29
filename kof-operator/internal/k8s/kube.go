package k8s

import (
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	DefaultSystemNamespace = "kcm-system"
	KofNamespace           = "kof"
)

const (
	kofClusterRoleLabel = "k0rdent.mirantis.com/kof-cluster-role"
	kofKcmRegionLabel   = "k0rdent.mirantis.com/kcm-region-cluster"

	kofRoleRegional = "regional"
	kofRoleChild    = "child"
)

func GetConfig(kubeconfig []byte) (*clientcmdapi.Config, error) {
	return clientcmd.Load(kubeconfig)
}
