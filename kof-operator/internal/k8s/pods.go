package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCollectorPods(ctx context.Context, k8sClient client.Client) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	if err := k8sClient.List(
		ctx,
		podList,
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "opentelemetry-collector"}),
	); err != nil {
		return podList, err
	}

	return podList, nil
}
