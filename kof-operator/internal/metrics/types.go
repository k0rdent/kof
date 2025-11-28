package metrics

import (
	"context"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

type BaseResourceStatus struct {
	Name        string      `json:"name,omitempty"`
	Message     string      `json:"message,omitempty"`
	MessageType MessageType `json:"type,omitempty"`
}

type ClusterInfo struct {
	BaseResourceStatus
	CustomResources CustomResourceMap `json:"customResource,omitempty"`
}

type CustomResourceInfo struct {
	BaseResourceStatus
	Pods PodMap `json:"pods,omitempty"`
}

type PodInfo struct {
	BaseResourceStatus
	Metrics MetricsMap `json:"metrics,omitempty"`
}

type MetricValue struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  any               `json:"value"`
}

type ResourceUsage struct {
	CPU    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
}

type ResourceAddress struct {
	Cluster        string `json:"cluster"`
	CustomResource string `json:"customResource,omitempty"`
	Pod            string `json:"pod,omitempty"`
}

type MessageType string

const (
	MessageTypeInfo    MessageType = "info"
	MessageTypeWarning MessageType = "warning"
	MessageTypeError   MessageType = "error"
)

type StatusMessage struct {
	ResourceAddress
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
	Details string      `json:"details,omitempty"`
}

type MetricData struct {
	ResourceAddress
	Name  string       `json:"name"`
	Value *MetricValue `json:"value,omitempty"`
	Err   error        `json:"error,omitempty"`
}

type ResourceMessage struct {
	Status  *StatusMessage `json:"status,omitempty"`
	Metrics *MetricData    `json:"metrics,omitempty"`
}

type MetricCollectorServiceConfig struct {
	Ctx                context.Context
	KubeClient         *k8s.KubeClient
	Pod                *corev1.Pod
	MetricsChan        ResourceChannel
	ClusterName        string
	CustomResourceName string
	ContainerName      string
	PortAnnotation     string
	PortName           string
	ProxyEndpoint      string
}

type MetricCollectorService struct {
	config *MetricCollectorServiceConfig
}

type (
	ResourceChannel = chan *ResourceMessage
)
