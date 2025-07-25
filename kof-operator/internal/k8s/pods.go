package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const CollectorMetricsLabel = "k0rdent.mirantis.com/kof-collector-metrics"

func GetCollectorPods(ctx context.Context, k8sClient client.Client, additionalOptions ...client.ListOption) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	baseSelector := client.MatchingLabels(map[string]string{
		"app.kubernetes.io/component": "opentelemetry-collector",
	})

	optionsCount := 1 + len(additionalOptions)
	options := make([]client.ListOption, 0, optionsCount)
	options = append(options, baseSelector)
	options = append(options, additionalOptions...)

	if err := k8sClient.List(
		ctx,
		podList,
		options...,
	); err != nil {
		return podList, err
	}

	return podList, nil
}
