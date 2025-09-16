package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(kcmv1beta1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(addoncontrollerv1beta1.AddToScheme(scheme))
	utilruntime.Must(libsveltosv1beta1.AddToScheme(scheme))
}

type KubeClient struct {
	Client        client.Client
	Config        clientcmd.ClientConfig
	Clientset     *kubernetes.Clientset
	MetricsClient *versioned.Clientset
}

func NewClient() (*KubeClient, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	return newKubeClient(config)
}

func NewKubeClientFromKubeconfig(kubeconfig []byte) (*KubeClient, error) {
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	return newKubeClient(config)
}

func NewKubeClientFromClusterDeployment(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (*KubeClient, error) {
	secretName := GetSecretName(cd)
	secret, err := GetSecret(ctx, client, secretName, cd.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %v", err)
	}

	kubeconfig := GetSecretValue(secret)
	if kubeconfig == nil {
		return nil, fmt.Errorf("kubeconfig is empty")
	}

	kubeClient, err := NewKubeClientFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create new client from kubeconfig: %v", err)
	}

	return kubeClient, nil
}

func newKubeClient(config clientcmd.ClientConfig) (*KubeClient, error) {
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(restConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	mc, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Client:        client,
		Clientset:     clientset,
		Config:        config,
		MetricsClient: mc,
	}, nil
}
