package k8s

import (
	"context"

	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterSummaries(ctx context.Context, client client.Client, opts ...client.ListOption) (*addoncontrollerv1beta1.ClusterSummaryList, error) {
	summaries := new(addoncontrollerv1beta1.ClusterSummaryList)
	err := client.List(ctx, summaries, opts...)
	return summaries, err
}
