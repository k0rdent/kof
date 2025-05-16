package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	client, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Client:    client,
		Clientset: clientset,
		Config:    config,
	}, nil
}
