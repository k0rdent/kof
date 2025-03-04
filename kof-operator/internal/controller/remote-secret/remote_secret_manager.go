package remotesecret

import (
	"context"
	"encoding/base64"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	"github.com/go-logr/logr"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	"istio.io/istio/istioctl/pkg/multicluster"
	"istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RemoteSecretManager struct {
	client client.Client
	IIstioRemoteSecretCreator
}

type IRemoteSecretManager interface {
	Create(*kcmv1alpha1.ClusterDeployment, logr.Logger, context.Context, ctrl.Request) error
}

const (
	IstioRoleLabelKey = "k0rdent.mirantis.com/istio-role"
	IstioRoleChild    = "child"
)

func New(c client.Client) IRemoteSecretManager {
	return &RemoteSecretManager{
		client:                    c,
		IIstioRemoteSecretCreator: NewIstioRemoteSecret(),
	}
}

// Function handles the creation of a remote secret
func (rs *RemoteSecretManager) Create(clusterDeployment *kcmv1alpha1.ClusterDeployment, logger logr.Logger, ctx context.Context, request ctrl.Request) error {
	if !rs.isClusterDeploymentReady(*clusterDeployment.GetConditions()) {
		return nil
	}

	if !rs.hasIstioChildRoleLabel(clusterDeployment.Labels) {
		return nil
	}

	kubeconfig, err := rs.GetKubeconfigFromSecret(logger, ctx, request)
	if err != nil {
		return err
	}

	remoteSecret, err := rs.CreateRemoteSecret(kubeconfig, logger, request.Name)
	if err != nil {
		return err
	}

	remoteSecret.Namespace = request.Namespace
	if err := rs.client.Create(ctx, remoteSecret); err != nil {
		logger.Error(err, "failed to create secret on mothership cluster")
		return err
	}

	return nil
}

// Function retrieves and decodes a kubeconfig from a Secret
func (rs *RemoteSecretManager) GetKubeconfigFromSecret(logger logr.Logger, ctx context.Context, request ctrl.Request) ([]byte, error) {
	kubeconfigSecret := &corev1.Secret{}
	secretFullName := rs.getFullSecretName(request.Name)

	if err := rs.client.Get(ctx, types.NamespacedName{
		Name:      secretFullName,
		Namespace: request.Namespace,
	}, kubeconfigSecret); err != nil {
		logger.Error(err, fmt.Sprintf("Unable to fetch Secret '%s'", secretFullName))
		return nil, err
	}

	logger.Info("Secret found", "name", kubeconfigSecret.Name, "namespace", kubeconfigSecret.Namespace)

	kubeconfigRaw, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig secret does not contain 'value' key")
	}

	kubeconfigDecoded, err := base64.StdEncoding.DecodeString(string(kubeconfigRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 kubeconfig data: %v", err)
	}
	return kubeconfigDecoded, nil
}

// Function checks if the provided labels map contains the Istio child role label
func (rs *RemoteSecretManager) hasIstioChildRoleLabel(labels map[string]string) bool {
	return labels[IstioRoleLabelKey] == IstioRoleChild
}

// Function checks if the cluster deployment is in a ready state
func (rs *RemoteSecretManager) isClusterDeploymentReady(conditions []metav1.Condition) bool {
	for _, condition := range conditions {
		if condition.Type != kcmv1alpha1.ReadyCondition {
			continue
		}

		if condition.Status == metav1.ConditionTrue {
			return true
		}

		break
	}
	return false
}

// Function generates the secret name based on the cluster name
func (rs *RemoteSecretManager) getFullSecretName(clusterName string) string {
	return fmt.Sprintf("%s-kubeconfig", clusterName)
}

type IstioRemoteSecretCreator struct{}

type IIstioRemoteSecretCreator interface {
	CreateRemoteSecret([]byte, logr.Logger, string) (*corev1.Secret, error)
}

func NewIstioRemoteSecret() IIstioRemoteSecretCreator {
	return &IstioRemoteSecretCreator{}
}

// Function creates a remote secret for Istio using the provided kubeconfig
func (rs *IstioRemoteSecretCreator) CreateRemoteSecret(kubeconfig []byte, logger logr.Logger, clusterName string) (*corev1.Secret, error) {
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		logger.Error(err, "failed to create new client config")
		return nil, err
	}

	kubeClient, err := kube.NewCLIClient(config)
	if err != nil {
		logger.Error(err, "failed to create cli client")
		return nil, err
	}

	secret, warn, err := istio.CreateRemoteSecret(multicluster.RemoteSecretOptions{
		ClusterName:          clusterName,
		CreateServiceAccount: true,
	}, kubeClient)
	if err != nil {
		logger.Error(err, "failed to create remote secret")
		return nil, err
	}

	if warn != nil {
		logger.Info("warning when creating remote secret", "warning", warn)
	}

	return secret, nil
}
