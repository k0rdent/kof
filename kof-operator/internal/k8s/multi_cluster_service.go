package k8s

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetMultiClusterService(ctx context.Context, client client.Client, opts ...client.ListOption) (*kcmv1beta1.MultiClusterServiceList, error) {
	mcsList := new(kcmv1beta1.MultiClusterServiceList)
	err := client.List(ctx, mcsList, opts...)
	return mcsList, err
}
