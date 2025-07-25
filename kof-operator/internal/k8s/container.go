package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GetContainer(containers []corev1.Container, name string) *corev1.Container {
	for _, container := range containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

func GetContainerPort(ports []corev1.ContainerPort, name string) *corev1.ContainerPort {
	for _, port := range ports {
		if port.Name == name {
			return &port
		}
	}
	return nil
}

func ExtractContainerPort(pod *corev1.Pod, containerName, portName string) (string, error) {
	container := GetContainer(pod.Spec.Containers, containerName)
	if container == nil {
		return "", fmt.Errorf("failed to find container '%s'", containerName)
	}

	port := GetContainerPort(container.Ports, portName)
	if port == nil {
		return "", fmt.Errorf("port %s not found in container ports", portName)
	}

	return fmt.Sprintf("%d", port.ContainerPort), nil
}
