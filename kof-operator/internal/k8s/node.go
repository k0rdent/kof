package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetNode(ctx context.Context, client client.Client, nodeName string) (*corev1.Node, error) {
	node := &corev1.Node{}
	namespaced := types.NamespacedName{Name: nodeName}

	err := client.Get(ctx, namespaced, node)
	return node, err
}
