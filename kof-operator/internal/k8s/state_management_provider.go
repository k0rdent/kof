package k8s

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetStateManagementProviders(ctx context.Context, client client.Client, opts ...client.ListOption) (*kcmv1beta1.StateManagementProviderList, error) {
	smpList := new(kcmv1beta1.StateManagementProviderList)
	err := client.List(ctx, smpList, opts...)
	return smpList, err
}
