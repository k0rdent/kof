package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const PrometheusReceiverAnnotation = "k0rdent.mirantis.com/kof-prometheus-receiver"

func GetCollectorPods(ctx context.Context, k8sClient client.Client) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	if err := k8sClient.List(
		ctx,
		podList,
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "opentelemetry-collector"}),
	); err != nil {
		return podList, err
	}

	filteredItems := make([]corev1.Pod, 0)
	for _, cd := range podList.Items {
		if cd.GetAnnotations()[PrometheusReceiverAnnotation] == "true" {
			filteredItems = append(filteredItems, cd)
		}
	}

	podList.Items = filteredItems
	return podList, nil
}
