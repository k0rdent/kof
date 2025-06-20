package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/k0rdent/kof/kof-operator/internal/models/target"
	v1 "github.com/prometheus/prometheus/web/api/v1"
)

var PrometheusReceiverPort string

const PrometheusEndpoint = "api/v1/targets"

func CollectPrometheusTargets(ctx context.Context, logger *logr.Logger, kubeClient *KubeClient, clusterName string) (*target.Targets, error) {
	response := &target.Targets{Clusters: make(target.Clusters)}

	podList, err := GetCollectorPods(ctx, kubeClient.Client)
	if err != nil {
		return response, fmt.Errorf("failed to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		byteResponse, err := Proxy(ctx, kubeClient.Clientset, pod, PrometheusReceiverPort, PrometheusEndpoint)
		if err != nil {
			logger.Error(err, "failed to connect to the pod", "podName", pod.Name, "response", string(byteResponse), "clusterName", clusterName)
			continue
		}

		podResponse := &v1.Response{}
		if err := json.Unmarshal(byteResponse, podResponse); err != nil {
			logger.Error(err, "failed to unmarshal pod response", "podName", pod.Name, "response", string(byteResponse), "clusterName", clusterName)
			continue
		}

		response.AddPodResponse(clusterName, pod.Spec.NodeName, pod.Name, podResponse)
	}

	return response, nil
}
