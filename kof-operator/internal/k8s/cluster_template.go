package k8s

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterTemplate(ctx context.Context, client client.Client, name, namespace string) (*kcmv1beta1.ClusterTemplate, error) {
	clusterTemplate := new(kcmv1beta1.ClusterTemplate)
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, clusterTemplate)
	return clusterTemplate, err
}
