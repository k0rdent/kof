package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterSecretSuffix = "kubeconfig"
)

func GetSecret(ctx context.Context, k8sClient client.Client, name string, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret)
	return secret, err
}

func GetKubeconfigSecretName(ctx context.Context, k8sClient client.Client, cd *kcmv1beta1.ClusterDeployment) (string, error) {
	if utils.IsAdopted(cd) {
		return GetAdoptedClusterSecretName(ctx, k8sClient, cd)
	}
	return GetCloudClusterSecretName(cd), nil
}

func GetAdoptedClusterSecretName(ctx context.Context, k8sClient client.Client, cd *kcmv1beta1.ClusterDeployment) (string, error) {
	cred := new(kcmv1beta1.Credential)
	namespacedName := types.NamespacedName{
		Name:      cd.Spec.Credential,
		Namespace: cd.Namespace,
	}

	if err := k8sClient.Get(ctx, namespacedName, cred); err != nil {
		return "", fmt.Errorf("failed to get credential: %v", err)
	}

	if cred.Spec.IdentityRef.Kind != "Secret" {
		return "", fmt.Errorf("unsupported Credential IdentityRef kind %s for adopted cluster", cred.Spec.IdentityRef.Kind)
	}

	return cred.Spec.IdentityRef.Name, nil
}

func GetCloudClusterSecretName(cd *kcmv1beta1.ClusterDeployment) string {
	return fmt.Sprintf("%s-%s", cd.Name, ClusterSecretSuffix)
}

func GetSecretValue(secret *corev1.Secret) []byte {
	if kubeconfig, ok := secret.Data["value"]; ok {
		return kubeconfig
	}
	return nil
}

func GetKubeconfigFromSecret(ctx context.Context, k8sClient client.Client, secretName, namespace string) (*api.Config, error) {
	secret, err := GetSecret(ctx, k8sClient, secretName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s in namespace %s: %v", secretName, namespace, err)
	}

	kubeconfig := GetSecretValue(secret)
	return GetConfig(kubeconfig)
}
