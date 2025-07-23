package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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

func GetKofClusterDeployments(ctx context.Context, k8sClient client.Client) (*kcmv1beta1.ClusterDeploymentList, error) {
	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement("k0rdent.mirantis.com/kof-cluster-role", selection.Exists, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector requirement: %v", err)
	}

	selector = selector.Add(*requirement)

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	return GetClusterDeployments(ctx, k8sClient, options)
}

func GetClusterDeployment(ctx context.Context, client client.Client, name, namespace string) (*kcmv1beta1.ClusterDeployment, error) {
	cd := &kcmv1beta1.ClusterDeployment{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, cd)
	return cd, err
}
