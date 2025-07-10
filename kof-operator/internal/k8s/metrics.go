package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetPodMetrics(ctx context.Context, client *versioned.Clientset, podName, podNamespace string) (*v1beta1.PodMetrics, error) {
	return client.MetricsV1beta1().PodMetricses(podNamespace).Get(ctx, podName, metav1.GetOptions{})
}

func GetNodeMetrics(ctx context.Context, client *versioned.Clientset, nodeName string) (*v1beta1.NodeMetrics, error) {
	return client.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
}
