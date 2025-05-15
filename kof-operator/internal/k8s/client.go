package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeClient struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
	Config    clientcmd.OverridingClientConfig
}

func NewClient() (client.Client, error) {
	kubeCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restCfg, err := kubeCfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func NewKubeClientFromKubeconfig(kubeconfig []byte) (*KubeClient, error) {
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

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
