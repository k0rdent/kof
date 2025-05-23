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

func CollectPrometheusTargets(ctx context.Context, kubeClient *KubeClient, clusterName string) (*target.PrometheusTargets, error) {
	response := &target.PrometheusTargets{}

	podList, err := GetCollectorPods(ctx, kubeClient.Client)
	if err != nil {
		return response, fmt.Errorf("failed to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		byteResponse, err := Proxy(ctx, kubeClient.Clientset, pod, PrometheusPort, PrometheusEndpoint)
		if err != nil {
			log.Printf("failed to connect to the pod '%s': %v, %s", pod.Name, err, string(byteResponse))
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
