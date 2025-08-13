package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPods(ctx context.Context, k8sClient client.Client, opt ...client.ListOption) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	err := k8sClient.List(ctx, podList, opt...)
	return podList, err
}
