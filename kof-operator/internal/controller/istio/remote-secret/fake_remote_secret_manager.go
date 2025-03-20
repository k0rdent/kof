package remotesecret

import (
	"context"

	"github.com/k0rdent/kof/kof-operator/internal/controller/istio"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeRemoteSecretCreator struct{}

func NewFakeManager(c client.Client) *RemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewFakeRemoteSecretCreator(),
	}
}

func NewFakeRemoteSecretCreator() IIstioRemoteSecretCreator {
	return &FakeRemoteSecretCreator{}
}

func (f *FakeRemoteSecretCreator) CreateRemoteSecret(kubeconfig []byte, ctx context.Context, clusterName string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: istio.IstioSystemNamespace,
			Name:      RemoteSecretNameFromClusterName(clusterName),
			Labels:    map[string]string{},
		},
		StringData: map[string]string{
			"value": "Fake values",
		},
	}, nil
}
