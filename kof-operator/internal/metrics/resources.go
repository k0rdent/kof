package metrics

import (
	"fmt"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

func (s *Service) CollectResources() {
	usage, err := s.getContainerUsage()
	if err != nil {
		s.error(fmt.Errorf("failed to get container resources usage: %v", err))
		return
	}

	if usage == nil {
		return
	}

	s.send(ContainerCPUUsage, usage.CPU)
	s.send(ContainerMemoryUsage, usage.Memory)

	limits, err := s.getContainerLimits()
	if err != nil {
		s.error(fmt.Errorf("failed to get container resources limit: %v", err))
		return
	}

	if limits.CPU > 0 && limits.Memory > 0 {
		s.send(ContainerCPULimit, limits.CPU)
		s.send(ContainerMemoryLimit, limits.Memory)
		return
	}

	nodeAvailableNow, err := s.getNodeAvailableNow()
	if err != nil {
		s.error(fmt.Errorf("failed to get node limits: %v", err))
		return
	}
	s.send(ContainerCPULimit, nodeAvailableNow.CPU)
	s.send(ContainerMemoryLimit, nodeAvailableNow.Memory)
}

func (s *Service) getContainerLimits() (*Resource, error) {
	container := k8s.GetContainer(s.config.Pod.Spec.Containers, s.config.ContainerName)
	if container == nil {
		return nil, fmt.Errorf("container not found")
	}

	return &Resource{
		CPU:    container.Resources.Limits.Cpu().MilliValue(),
		Memory: container.Resources.Limits.Memory().Value(),
	}, nil
}

func (s *Service) getContainerUsage() (*Resource, error) {
	podMetrics, err := k8s.GetPodMetrics(s.config.Ctx, s.config.KubeClient.MetricsClient, s.config.Pod.Name, s.config.Pod.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %v", err)
	}

	metrics, err := findContainerMetric(podMetrics.Containers, s.config.ContainerName)
	if err != nil {
		return nil, fmt.Errorf("failed to find container %s: %v", s.config.ContainerName, err)
	}

	return &Resource{
		CPU:    metrics.Usage.Cpu().MilliValue(),
		Memory: metrics.Usage.Memory().Value(),
	}, nil
}

func (s *Service) getNodeAvailableNow() (*Resource, error) {
	nodeMetrics, err := k8s.GetNodeMetrics(s.config.Ctx, s.config.KubeClient.MetricsClient, s.config.Pod.Spec.NodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %v", err)
	}

	node, err := k8s.GetNode(s.config.Ctx, s.config.KubeClient.Client, s.config.Pod.Spec.NodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %v", err)
	}

	cpuResourceQuantity, ok := node.Status.Allocatable[corev1.ResourceCPU]
	if !ok {
		return nil, fmt.Errorf("CPU resource not found in node %q", node.Name)
	}

	memoryResourceQuantity, ok := node.Status.Allocatable[corev1.ResourceMemory]
	if !ok {
		return nil, fmt.Errorf("memory resource not found in node %q", node.Name)
	}

	return &Resource{
		CPU:    cpuResourceQuantity.MilliValue() - nodeMetrics.Usage.Cpu().MilliValue(),
		Memory: memoryResourceQuantity.Value() - nodeMetrics.Usage.Memory().Value(),
	}, nil
}
