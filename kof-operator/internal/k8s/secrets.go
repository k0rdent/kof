package k8s

import (
	"context"
	"fmt"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AdoptedClusterSecretSuffix = "kubeconf"
	ClusterSecretSuffix        = "kubeconfig"
)

func GetSecret(ctx context.Context, k8sClient client.Client, name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret)
	return secret, err
}

func GetSecretName(cd *kcmv1beta1.ClusterDeployment) string {
	if strings.Contains(cd.Spec.Template, "adopted") {
		return fmt.Sprintf("%s-%s", cd.Name, AdoptedClusterSecretSuffix)
	}
	return fmt.Sprintf("%s-%s", cd.Name, ClusterSecretSuffix)
}

func GetSecretValue(secret *corev1.Secret) []byte {
	if kubeconfig, ok := secret.Data["value"]; ok {
		return kubeconfig
	}
	return nil
}
