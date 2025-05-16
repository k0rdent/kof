package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKubeconfigSecret(ctx context.Context, k8sClient client.Client, name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}

	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		secret,
	); err != nil {
		return secret, err
	}

	return secret, nil
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
