package cert

import (
	"context"

	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const istioReleaseName = "kof-istio"

type CertManager struct {
	k8sClient client.Client
}

func New(client client.Client) *CertManager {
	return &CertManager{
		k8sClient: client,
	}
}

func (cm *CertManager) TryCreate(clusterDeployment *kcmv1alpha1.ClusterDeployment, ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Trying to create certificate")

	cert := cm.generateCertificate(clusterDeployment)
	return cm.createCertificate(cert, ctx)

}

func (cm *CertManager) createCertificate(cert *cmv1.Certificate, ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Creating Intermediate Istio CA certificate", "certificateName", cert.Name)

	if err := cm.k8sClient.Create(ctx, cert); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Istio CA certificate already exists", "certificateName", cert.Name)
			return nil
		}
		return err
	}
	return nil
}

func (cm *CertManager) generateCertificate(clusterDeployment *kcmv1alpha1.ClusterDeployment) *cmv1.Certificate {
	certName := cm.getCertName(clusterDeployment.Name)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kof-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "k0rdent.mirantis.com/v1alpha1",
					Kind:       "ClusterDeployment",
					Name:       clusterDeployment.Name,
					UID:        clusterDeployment.GetUID(),
				},
			},
		},
		Spec: cmv1.CertificateSpec{
			IsCA:       true,
			CommonName: fmt.Sprintf("%s CA", clusterDeployment.Name),
			Subject: &cmv1.X509Subject{
				Organizations: []string{"Istio"},
			},
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: "ECDSA",
				Size:      256,
			},
			SecretName: certName,
			IssuerRef: cmmetav1.ObjectReference{
				Name:  fmt.Sprintf("%s-root", istioReleaseName),
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}
}

func (cm *CertManager) getCertName(clusterName string) string {
	return fmt.Sprintf("kof-istio-%s-ca", clusterName)
}
