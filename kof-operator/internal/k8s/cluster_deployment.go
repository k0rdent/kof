package k8s

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterDeployments(ctx context.Context, client client.Client, opts ...client.ListOption) (*kcmv1beta1.ClusterDeploymentList, error) {
	cdList := &kcmv1beta1.ClusterDeploymentList{
		Items: make([]kcmv1beta1.ClusterDeployment, 0),
	}
	err := client.List(ctx, cdList, opts...)
	return cdList, err
}

func GetClusterDeployment(ctx context.Context, client client.Client, name, namespace string) (*kcmv1beta1.ClusterDeployment, error) {
	cd := &kcmv1beta1.ClusterDeployment{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, cd)
	return cd, err
}
