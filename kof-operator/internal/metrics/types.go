package metrics

import (
	"context"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

type MetricChannel chan *Metric
type Cluster map[string]Pod
type Pod map[string]Metrics
type Metrics map[string][]*MetricValue
type MetricValue struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  any               `json:"value"`
}

type Resource struct {
	CPU    int64
	Memory int64
}

type Metric struct {
	Cluster string
	Pod     string
	Name    string
	Err     error
	Data    *MetricValue
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
