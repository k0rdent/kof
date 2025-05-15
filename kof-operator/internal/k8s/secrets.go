package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKubeconfigSecrets(ctx context.Context, k8sClient client.Client) (*corev1.SecretList, error) {
	secretList := &corev1.SecretList{}

	if err := k8sClient.List(
		ctx,
		secretList,
		client.MatchingLabels(map[string]string{"cluster-secret": "true"}),
	); err != nil {
		return secretList, err
	}

	return secretList, nil
}

func GetKubeconfigFromSecretList(secretList *corev1.SecretList) [][]byte {
	kubeconfigList := make([][]byte, 0, len(secretList.Items))

	for _, secret := range secretList.Items {
		kubeconfig, ok := secret.Data["value"]
		if !ok {
			continue
		}
		kubeconfigList = append(kubeconfigList, kubeconfig)
	}
	return kubeconfigList
}
