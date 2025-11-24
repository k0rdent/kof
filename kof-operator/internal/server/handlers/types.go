package handlers

import (
	"github.com/k0rdent/kof/kof-operator/internal/metrics"
	v1 "k8s.io/api/core/v1"
)

type ResourceStatus struct {
	MessageType metrics.MessageType `json:"type,omitempty"`
	Message     string              `json:"message,omitempty"`
}

type ICustomResource interface {
	GetPods() ([]v1.Pod, error)
	GetName() string
	GetStatus() *ResourceStatus
}
