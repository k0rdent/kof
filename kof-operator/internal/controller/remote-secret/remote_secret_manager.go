package remotesecret

import (
	"context"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	"istio.io/istio/istioctl/pkg/multicluster"
	"istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RemoteSecretManager struct {
	client client.Client
	IIstioRemoteSecretCreator
}

func New(c client.Client) *RemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewIstioRemoteSecret(),
	}
}

// Function tries to delete the remote secret
func (rs *RemoteSecretManager) TryDelete(ctx context.Context, request ctrl.Request) error {
	log := log.FromContext(ctx)
	log.Info("Trying to delete remote secret")

	if err := rs.deleteRemoteSecret(ctx, request); err != nil {
		return fmt.Errorf("failed to delete remote secret: %v", err)
	}
	return nil
}

// Function handles the creation of a remote secret
func (rs *RemoteSecretManager) TryCreate(clusterDeployment *kcmv1alpha1.ClusterDeployment, ctx context.Context, request ctrl.Request) error {
	log := log.FromContext(ctx)
	log.Info("Trying to create remote secret")

	if !rs.isClusterDeploymentReady(*clusterDeployment.GetConditions()) {
		log.Info("Cluster deployment is not ready")
		return nil
	}

	exists, err := rs.remoteSecretExists(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to check remote secret: %v", err)
	}

	if exists {
		log.Info("Remote secret already exists")
		return nil
	}

	kubeconfig, err := rs.GetKubeconfigFromSecret(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig from secret: %v", err)
	}

	remoteSecret, err := rs.CreateRemoteSecret(kubeconfig, ctx, request.Name)
	if err != nil {
		return fmt.Errorf("failed to create remote secret: %v", err)
	}

	if err := rs.createRemoteSecret(ctx, remoteSecret); err != nil {
		log.Error(err, "failed to create remote secret")
		return fmt.Errorf("failed to create remote secret: %v", err)
	}

	log.Info("Remote secret successfully created")
	return nil
}

// Function retrieves and decodes a kubeconfig from a Secret
func (rs *RemoteSecretManager) GetKubeconfigFromSecret(ctx context.Context, request ctrl.Request) ([]byte, error) {
	log := log.FromContext(ctx)
	kubeconfigSecret := &corev1.Secret{}
	secretFullName := rs.getFullSecretName(request.Name)

	if err := rs.client.Get(ctx, types.NamespacedName{
		Name:      secretFullName,
		Namespace: request.Namespace,
	}, kubeconfigSecret); err != nil {
		log.Error(err, fmt.Sprintf("Unable to fetch Secret '%s'", secretFullName))
		return nil, err
	}

	log.Info("Secret found", "name", kubeconfigSecret.Name, "namespace", kubeconfigSecret.Namespace)

	kubeconfigRaw, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig secret does not contain 'value' key")
	}

	return kubeconfigRaw, nil
}

// Function checks if the cluster deployment is in a ready state
func (rs *RemoteSecretManager) isClusterDeploymentReady(conditions []metav1.Condition) bool {
	if len(conditions) == 0 {
		return false
	}

	for _, condition := range conditions {
		if condition.Status != metav1.ConditionTrue {
			return false
		}
	}
	return true
}

// Function generates the secret name based on the cluster name
func (rs *RemoteSecretManager) getFullSecretName(clusterName string) string {
	return fmt.Sprintf("%s-kubeconfig", clusterName)
}

func (rs *RemoteSecretManager) remoteSecretExists(ctx context.Context, req ctrl.Request) (bool, error) {
	secret := &corev1.Secret{}
	if err := rs.client.Get(ctx, types.NamespacedName{
		Name:      istio.RemoteSecretNameFromClusterName(req.Name),
		Namespace: istio.IstioSystemNamespace,
	}, secret); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Function creates the remote secret resource in k8s
func (rs *RemoteSecretManager) createRemoteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := rs.client.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (rs *RemoteSecretManager) deleteRemoteSecret(ctx context.Context, req ctrl.Request) error {
	log := log.FromContext(ctx)

	if err := rs.client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      istio.RemoteSecretNameFromClusterName(req.Name),
			Namespace: istio.IstioSystemNamespace,
		},
	}); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Remote secret already deleted")
			return nil
		}
		return err
	}

	log.Info("Remote secret successfully deleted")
	return nil
}

type IstioRemoteSecretCreator struct{}

type IIstioRemoteSecretCreator interface {
	CreateRemoteSecret([]byte, context.Context, string) (*corev1.Secret, error)
}

func NewIstioRemoteSecret() IIstioRemoteSecretCreator {
	return &IstioRemoteSecretCreator{}
}

// Function creates a remote secret for Istio using the provided kubeconfig
func (rs *IstioRemoteSecretCreator) CreateRemoteSecret(kubeconfig []byte, ctx context.Context, clusterName string) (*corev1.Secret, error) {
	log := log.FromContext(ctx)

	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		log.Error(err, "failed to create new client config")
		return nil, err
	}

	kubeClient, err := kube.NewCLIClient(config)
	if err != nil {
		log.Error(err, "failed to create cli client")
		return nil, err
	}

	secret, warn, err := istio.CreateRemoteSecret(multicluster.RemoteSecretOptions{
		Type:                 multicluster.SecretTypeRemote,
		AuthType:             multicluster.RemoteSecretAuthTypeBearerToken,
		ClusterName:          clusterName,
		CreateServiceAccount: true,
		KubeOptions: multicluster.KubeOptions{
			Namespace: istio.IstioSystemNamespace,
		},
	}, kubeClient, ctx)
	if err != nil {
		log.Error(err, "failed to create remote secret")
		return nil, err
	}

	if warn != nil {
		log.Info("warning when creating remote secret", "warning", warn)
	}

	return secret, nil
}
