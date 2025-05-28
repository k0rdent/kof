package k8s

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterDeployments(ctx context.Context, client client.Client) (*kcmv1beta1.ClusterDeploymentList, error) {
	cdList := &kcmv1beta1.ClusterDeploymentList{
		Items: make([]kcmv1beta1.ClusterDeployment, 0),
	}
	err := client.List(ctx, cdList)
	return cdList, err
}
