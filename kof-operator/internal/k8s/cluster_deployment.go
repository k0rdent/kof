package k8s

import (
	"context"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterDeployments(ctx context.Context, client client.Client) (*kcmv1alpha1.ClusterDeploymentList, error) {
	cdList := &kcmv1alpha1.ClusterDeploymentList{}
	if err := client.List(ctx, cdList); err != nil {
		return nil, err
	}
	return cdList, nil
}
