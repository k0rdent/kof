package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CachedClusterData struct {
	Regions     *kcmv1beta1.RegionList
	Credentials *kcmv1beta1.CredentialList
	Clusters    *kcmv1beta1.ClusterDeploymentList
}

// Verify if the Cluster was created in a KCM region
func CreatedInKCMRegion(cd *kcmv1beta1.ClusterDeployment) bool {
	return !utils.IsEmptyString(cd.Status.Region)
}

func IsClusterKcmRegion(ctx context.Context, client client.Client, clusterName, namespace string) (bool, error) {
	clusters, err := GetKcmRegionClusters(ctx, client)
	if err != nil {
		return false, fmt.Errorf("failed to get KCM region clusters: %v", err)
	}

	for _, cluster := range clusters {
		if cluster.Name == clusterName && cluster.Namespace == namespace {
			return true, nil
		}
	}
	return false, nil
}

func IsClusterInSameKcmRegion(ctx context.Context, client client.Client, childName, childNamespace, regionalName, regionalNamespace string) (bool, error) {
	regional, err := GetClusterDeployment(ctx, client, regionalName, regionalNamespace)
	if err != nil {
		return false, fmt.Errorf("failed to get regional cluster deployment %s/%s: %v", regionalNamespace, regionalName, err)
	}

	child, err := GetClusterDeployment(ctx, client, childName, childNamespace)
	if err != nil {
		return false, fmt.Errorf("failed to get child cluster deployment %s/%s: %v", childNamespace, childName, err)
	}

	if child.Spec.Credential == regional.Spec.Credential {
		return true, nil
	}

	region, err := GetRegion(ctx, client, child.Status.Region)
	if err != nil {
		return false, fmt.Errorf("failed to get region %s: %v", child.Status.Region, err)
	}

	if region.Spec.ClusterDeployment != nil {
		if region.Spec.ClusterDeployment.Name == regionalName && region.Spec.ClusterDeployment.Namespace == regionalNamespace {
			return true, nil
		}
	}

	if region.Spec.KubeConfig != nil {
		kubeconfigSecretName, err := GetKubeconfigSecretName(ctx, client, regional)
		if err != nil {
			return false, fmt.Errorf("failed to get secret name: %v", err)
		}

		if region.Spec.KubeConfig.Name == kubeconfigSecretName {
			return true, nil
		}
	}

	return false, nil
}

// GetClusterDeploymentsInSameKcmRegion returns the list of ClusterDeployments that are in the same KCM region
// as the specified ClusterDeployment. The specified ClusterDeployment must be a child cluster.
func GetClusterDeploymentsInSameKcmRegion(ctx context.Context, client client.Client, clusterDeployment *kcmv1beta1.ClusterDeployment) ([]*kcmv1beta1.ClusterDeployment, error) {
	if utils.IsEmptyString(clusterDeployment.Status.Region) {
		return nil, fmt.Errorf("cluster deployment is not in KCM region: %s/%s", clusterDeployment.Namespace, clusterDeployment.Name)
	}

	clusters, err := GetClusterDeployments(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster deployments: %v", err)
	}

	region := new(kcmv1beta1.Region)
	if err := client.Get(ctx, types.NamespacedName{Name: clusterDeployment.Status.Region}, region); err != nil {
		return nil, fmt.Errorf("failed to get region %s: %v", clusterDeployment.Status.Region, err)
	}

	var regionClusterName string
	if region.Spec.ClusterDeployment != nil {
		regionClusterName = region.Spec.ClusterDeployment.Name
	}

	if region.Spec.KubeConfig != nil && region.Spec.KubeConfig.Name != "" {
		config, err := GetKubeconfigFromSecret(ctx, client, region.Spec.KubeConfig.Name, region.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig for region %s: %v", region.Name, err)
		}
		regionClusterName = config.CurrentContext
	}

	result := make([]*kcmv1beta1.ClusterDeployment, 0)
	for i := range clusters.Items {
		cluster := &clusters.Items[i]
		if cluster.Name == regionClusterName {
			result = append(result, cluster)
			continue
		}

		if cluster.Status.Region == clusterDeployment.Status.Region {
			result = append(result, cluster)
		}
	}

	return result, nil
}

// GetKcmRegionChildClusters returns the list of ClusterDeployments that are inside the specified KCM region.
func GetKcmRegionChildClusters(ctx context.Context, kubeClient client.Client, regionCluster *kcmv1beta1.ClusterDeployment, cache CachedClusterData) ([]*kcmv1beta1.ClusterDeployment, error) {
	clusters := cache.Clusters

	region, err := GetRegionByClusterDeployment(ctx, kubeClient, regionCluster, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to get region by cluster deployment %s/%s: %w", regionCluster.Namespace, regionCluster.Name, err)
	}

	if region == nil {
		return nil, fmt.Errorf("region for cluster %s not found", regionCluster.Name)
	}

	if clusters == nil {
		clusters, err = GetKofChildClusterDeployments(ctx, kubeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get kof child cluster deployments: %w", err)
		}
	}

	childCds := make([]*kcmv1beta1.ClusterDeployment, 0)
	for i := range clusters.Items {
		cluster := &clusters.Items[i]
		if cluster.Status.Region == region.Name {
			childCds = append(childCds, cluster)
		}
	}

	return childCds, nil
}

func GetRegionByClusterDeployment(ctx context.Context, kubeClient client.Client, cd *kcmv1beta1.ClusterDeployment, cache CachedClusterData) (*kcmv1beta1.Region, error) {
	var err error
	regionList := cache.Regions

	if regionList == nil {
		regionList, err = GetRegions(ctx, kubeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to list regions: %v", err)
		}
	}

	secretName, err := GetKubeconfigSecretName(ctx, kubeClient, cd)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret name for cluster deployment %s/%s: %v", cd.Namespace, cd.Name, err)
	}

	for _, region := range regionList.Items {
		if region.Spec.ClusterDeployment != nil &&
			region.Spec.ClusterDeployment.Name == cd.Name &&
			region.Spec.ClusterDeployment.Namespace == cd.Namespace {
			return &region, nil
		}

		if region.Spec.KubeConfig != nil && region.Spec.KubeConfig.Name == secretName {
			return &region, nil
		}
	}

	return nil, nil
}

// GetKcmRegionClusters returns the list of ClusterDeployments that are used as a KCM region.
func GetKcmRegionClusters(ctx context.Context, kubeClient client.Client) ([]*kcmv1beta1.ClusterDeployment, error) {
	clusterList, err := GetClusterDeployments(
		ctx,
		kubeClient,
		&client.ListOptions{
			LabelSelector: k8slabels.Set{
				labels.KofClusterRoleLabel: KofRoleRegional,
				labels.KofKcmRegionLabel:   utils.True,
			}.AsSelector(),
		},
	)
	if err != nil {
		return nil, err
	}

	clusters := make([]*kcmv1beta1.ClusterDeployment, 0, len(clusterList.Items))
	for i := range clusterList.Items {
		clusters = append(clusters, &clusterList.Items[i])
	}
	return clusters, nil
}

func GetRegion(ctx context.Context, kubeClient client.Client, regionName string) (*kcmv1beta1.Region, error) {
	region := new(kcmv1beta1.Region)
	err := kubeClient.Get(ctx, types.NamespacedName{Name: regionName}, region)
	return region, err
}

func GetCredentials(ctx context.Context, kubeClient client.Client) (*kcmv1beta1.CredentialList, error) {
	credList := new(kcmv1beta1.CredentialList)
	err := kubeClient.List(ctx, credList)
	return credList, err
}

func GetRegions(ctx context.Context, kubeClient client.Client) (*kcmv1beta1.RegionList, error) {
	regionList := new(kcmv1beta1.RegionList)
	err := kubeClient.List(ctx, regionList)
	return regionList, err
}
