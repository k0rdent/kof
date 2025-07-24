package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GetContainer(containers []corev1.Container, name string) (*corev1.Container, error) {
	for _, container := range containers {
		if container.Name == name {
			return &container, nil
		}
	}
	return nil, fmt.Errorf("container %s not found in spec", name)
}

func GetContainerPort(ports []corev1.ContainerPort, name string) (*corev1.ContainerPort, error) {
	for _, port := range ports {
		if port.Name == name {
			return &port, nil
		}
	}
	return nil, fmt.Errorf("port %s not found in container ports", name)
}

func ExtractContainerPort(pod *corev1.Pod, containerName, portName string) (string, error) {
	container, err := GetContainer(pod.Spec.Containers, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to find container '%s': %v", containerName, err)
	}

	port, err := GetContainerPort(container.Ports, portName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", port.ContainerPort), nil
}
