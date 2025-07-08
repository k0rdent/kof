package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const KofClusterDeploymentLabelKey = "k0rdent.mirantis.com/kof-cluster-role"

func GetKofClusterDeployments(ctx context.Context, k8sClient client.Client) (*kcmv1beta1.ClusterDeploymentList, error) {
	cdList := &kcmv1beta1.ClusterDeploymentList{
		Items: make([]kcmv1beta1.ClusterDeployment, 0),
	}

	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement(KofClusterDeploymentLabelKey, selection.Exists, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector requirement: %v", err)
	}

	selector = selector.Add(*requirement)

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	err = k8sClient.List(ctx, cdList, options)
	return cdList, err
}
