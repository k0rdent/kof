package k8s

import (
	"context"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const CollectorMetricsAnnotation = "k0rdent.mirantis.com/kof-collector-metrics"

func GetCollectorPods(ctx context.Context, k8sClient client.Client, annotation string) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	if err := k8sClient.List(
		ctx,
		podList,
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "opentelemetry-collector"}),
	); err != nil {
		return podList, err
	}

	podList.Items = slices.DeleteFunc(podList.Items, func(pod corev1.Pod) bool {
		return pod.GetAnnotations()[annotation] != "true"
	})
	return podList, nil
}
