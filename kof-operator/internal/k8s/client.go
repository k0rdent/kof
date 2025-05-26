package k8s

import (
	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(kcmv1beta1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

type KubeClient struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
	Config    clientcmd.ClientConfig
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

	return &KubeClient{
		Client:    client,
		Clientset: clientset,
		Config:    config,
	}, nil
}
