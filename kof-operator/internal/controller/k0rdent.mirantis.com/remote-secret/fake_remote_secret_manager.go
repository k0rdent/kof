package remotesecret

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeRemoteSecret struct{}

func NewFakeManager(c client.Client) IRemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewFakeRemoteSecret(),
	}
}

func NewFakeRemoteSecret() IIstioRemoteSecretCreator {
	return &FakeRemoteSecret{}
}

func (f *FakeRemoteSecret) CreateRemoteSecret(kubeconfig []byte, logger logr.Logger, clusterName string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterName,
			Labels: map[string]string{},
		},
		StringData: map[string]string{
			"value": "Fake values",
		},
	}, nil
}
