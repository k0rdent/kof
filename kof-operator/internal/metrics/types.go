package metrics

import (
	"context"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

type MetricChannel chan *Metric
type ClusterMetrics map[string]PodMetrics
type PodMetrics map[string]Metrics
type Metrics map[string]any

type Resource struct {
	CPU    int64
	Memory int64
}

type Metric struct {
	Cluster string
	Pod     string
	Name    string
	Value   any
	Err     error
}

type ServiceConfig struct {
	Ctx            context.Context
	KubeClient     *k8s.KubeClient
	Pod            *corev1.Pod
	Metrics        MetricChannel
	ClusterName    string
	ContainerName  string
	PortAnnotation string
	PortName       string
	ProxyEndpoint  string
}

type Service struct {
	config *ServiceConfig
}
