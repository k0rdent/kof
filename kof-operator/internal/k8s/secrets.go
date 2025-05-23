package k8s

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKubeconfigSecret(ctx context.Context, k8sClient client.Client, clusterName string, namespace string) (*corev1.Secret, error) {
	const ClusterNameLabelKey = "cluster.x-k8s.io/cluster-name"

	secrets := &corev1.SecretList{}

	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{ClusterNameLabelKey: clusterName},
	}

	if err := k8sClient.List(ctx, secrets, listOpts...); err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, "kubeconf") {
			return &secret, nil
		}
	}

	return nil, nil
}

func GetKubeconfigFromSecretList(secretList []*corev1.Secret) [][]byte {
	kubeconfigList := make([][]byte, 0, len(secretList))

	for _, secret := range secretList {
		kubeconfig, ok := secret.Data["value"]
		if !ok {
			continue
		}
		kubeconfigList = append(kubeconfigList, kubeconfig)
	}
	return kubeconfigList
}
