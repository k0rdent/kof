package cert

import (
	"context"

	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/istio"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

func (cm *CertManager) TryCreate(ctx context.Context, clusterDeployment *kcmv1alpha1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	log.Info("Trying to create certificate")

	cert := cm.generateClusterCACertificate(clusterDeployment)
	return cm.createCertificate(ctx, cert, clusterDeployment)
}

func (cm *CertManager) TryDelete(ctx context.Context, req ctrl.Request) error {
	certName := GetCertName(req.Name)
	log := log.FromContext(ctx)

	log.Info("Trying to delete istio certificate", "certificateName", certName)
	if err := cm.k8sClient.Delete(ctx, &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
		},
	}); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Istio Certificate already deleted", "certificateName", certName)
			return nil
		}
		return err
	}

	log.Info("Istio Certificate successfully deleted", "certificateName", certName)
	cm.sendDeletionEvent(req)
	return nil
}

func (cm *CertManager) createCertificate(ctx context.Context, cert *cmv1.Certificate, clusterDeployment *kcmv1alpha1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	log.Info("Creating Intermediate Istio CA certificate", "certificateName", cert.Name)

	if err := cm.k8sClient.Create(ctx, cert); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Istio CA certificate already exists", "certificateName", cert.Name)
			return nil
		}
		return err
	}
	cm.sendCreationEvent(clusterDeployment)
	return nil
}

func (cm *CertManager) generateClusterCACertificate(clusterDeployment *kcmv1alpha1.ClusterDeployment) *cmv1.Certificate {
	certName := GetCertName(clusterDeployment.Name)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kof-operator",
			},
		},
		Spec: cmv1.CertificateSpec{
			IsCA:       true,
			CommonName: fmt.Sprintf("%s CA", clusterDeployment.Name),
			Subject: &cmv1.X509Subject{
				Organizations: []string{"Istio"},
			},
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: cmv1.ECDSAKeyAlgorithm,
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

func (cm *CertManager) sendCreationEvent(cd *kcmv1alpha1.ClusterDeployment) {
	record.Eventf(
		cd,
		utils.GetEventsAnnotations(cd),
		"CertificateCreated",
		"Istio certificate '%s' is successfully created",
		GetCertName(cd.Name),
	)
}

func (cm *CertManager) sendDeletionEvent(req ctrl.Request) {
	cd := utils.GetClusterDeploymentStub(req.Name, req.Namespace)
	record.Eventf(
		cd,
		nil,
		"CertificateDeleted",
		"Istio certificate '%s' is successfully deleted",
		GetCertName(cd.Name),
	)
}

func GetCertName(clusterName string) string {
	return fmt.Sprintf("%s-%s-ca", istioReleaseName, clusterName)
}
