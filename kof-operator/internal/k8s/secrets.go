package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetSecret(ctx context.Context, k8sClient client.Client, name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret)
	return secret, err
}
