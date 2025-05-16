package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

const (
	PrometheusPort     = "9090"
	PrometheusEndpoint = "api/v1/targets"
)

func CollectPrometheusTargets(ctx context.Context, kubeClient *KubeClient) (*target.PrometheusTargets, error) {
	response := &target.PrometheusTargets{}

	clusterName, err := kubeClient.GetClusterName(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster name: %v", err)
	}

	podList, err := GetCollectorPods(ctx, kubeClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		byteResponse, err := Proxy(ctx, kubeClient.Clientset, pod, PrometheusPort, PrometheusEndpoint)
		if err != nil {
			log.Printf("failed to connect to the pod '%s': %v", pod.Name, err)
			continue
		}

		podResponse := &v1.Response{}
		if err := json.Unmarshal(byteResponse, podResponse); err != nil {
			log.Printf("failed to unmarshal pod '%s' response: %v", pod.Name, err)
			continue
		}

		response.AddPodResponse(clusterName, pod.Spec.NodeName, pod.Name, podResponse)
	}

	return response, nil
}
