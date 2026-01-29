package k8s

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CachedClusterData struct {
	Regions     *kcmv1beta1.RegionList
	Credentials *kcmv1beta1.CredentialList
	Clusters    *kcmv1beta1.ClusterDeploymentList
}

// If a Kubeconfig secret exists in the management cluster, we assume the cluster is not in the region
func CreatedInKCMRegion(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (bool, error) {

	if utils.IsAdopted(cd) {
		cred := new(kcmv1beta1.Credential)
		namespacedName := types.NamespacedName{
			Name:      cd.Spec.Credential,
			Namespace: DefaultSystemNamespace,
		}

		if err := client.Get(ctx, namespacedName, cred); err != nil {
			return false, err
		}

		if cred.Spec.Region == "" {
			return false, nil
		}

		return true, nil
	}

	secret := new(corev1.Secret)
	namespacedName := types.NamespacedName{
		Name:      GetCloudClusterSecretName(cd),
		Namespace: DefaultSystemNamespace,
	}
	err := client.Get(ctx, namespacedName, secret)
	if errors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func IsClusterInSameKcmRegion(ctx context.Context, client client.Client, childName, childNamespace, regionalName, regionalNamespace string) (bool, error) {
	child := new(kcmv1beta1.ClusterDeployment)
	regional := new(kcmv1beta1.ClusterDeployment)
	regionList := new(kcmv1beta1.RegionList)
	credList := new(kcmv1beta1.CredentialList)

	if err := client.Get(ctx, types.NamespacedName{
		Name:      childName,
		Namespace: childNamespace,
	}, child); err != nil {
		return false, fmt.Errorf("failed to get clusterDeployment %s: %v", childName, err)
	}

	if err := client.Get(ctx, types.NamespacedName{
		Name:      regionalName,
		Namespace: regionalNamespace,
	}, regional); err != nil {
		return false, fmt.Errorf("failed to get clusterDeployment %s: %v", regionalName, err)
	}

	// If both clusters have the same credential, they are in the same region
	// But if kof-regional is deployed as to the KCM Region cluster they will not have the same credential
	if child.Spec.Credential == regional.Spec.Credential {
		return true, nil
	}

	if err := client.List(ctx, regionList); err != nil {
		return false, fmt.Errorf("failed to list regions: %v", err)
	}

	kubeconfigSecretName, err := GetKubeconfigSecretName(ctx, client, regional)
	if err != nil {
		return false, fmt.Errorf("failed to get secret name: %v", err)
	}

	regionName := ""
	for _, region := range regionList.Items {
		if region.Spec.KubeConfig != nil && region.Spec.KubeConfig.Name == kubeconfigSecretName {
			regionName = region.Name
			break
		}

		if region.Spec.ClusterDeployment != nil && region.Spec.ClusterDeployment.Name == regional.Name && region.Spec.ClusterDeployment.Namespace == regional.Namespace {
			regionName = region.Name
			break
		}
	}

	if regionName == "" {
		return false, nil
	}

	if err := client.List(ctx, credList); err != nil {
		return false, fmt.Errorf("failed to list credentials: %v", err)
	}

	regionCredsName := ""
	for _, cred := range credList.Items {
		if cred.Spec.Region == regionName {
			regionCredsName = cred.Name
			break
		}
	}

	return child.Spec.Credential == regionCredsName, nil
}

// GetClusterDeploymentsInSameKcmRegion returns the list of ClusterDeployments that are in the same KCM region
// as the specified ClusterDeployment. The specified ClusterDeployment must be a child cluster.
func GetClusterDeploymentsInSameKcmRegion(ctx context.Context, client client.Client, clusterDeployment *kcmv1beta1.ClusterDeployment) ([]*kcmv1beta1.ClusterDeployment, error) {
	cdCred := new(kcmv1beta1.Credential)
	if err := client.Get(ctx, types.NamespacedName{
		Name:      clusterDeployment.Spec.Credential,
		Namespace: clusterDeployment.Namespace,
	}, cdCred); err != nil {
		return nil, fmt.Errorf("failed to get cluster deployment credential: %v", err)
	}

	var result []*kcmv1beta1.ClusterDeployment
	if cdCred.Spec.Region == "" {
		return result, nil
	}

	regionName := cdCred.Spec.Region
	region := new(kcmv1beta1.Region)
	if err := client.Get(ctx, types.NamespacedName{Name: regionName}, region); err != nil {
		return nil, fmt.Errorf("failed to get region %s: %v", regionName, err)
	}

	clusterList := new(kcmv1beta1.ClusterDeploymentList)
	if err := client.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list cluster deployments: %v", err)
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

	for _, cd := range clusterList.Items {
		if cd.Name == regionClusterName {
			result = append(result, &cd)
			continue
		}

		if cd.Spec.Credential == clusterDeployment.Spec.Credential {
			result = append(result, &cd)
		}
	}

	return result, nil
}

// GetKcmRegionChildClusters returns the list of ClusterDeployments that are inside the specified KCM region.
func GetKcmRegionChildClusters(ctx context.Context, kubeClient client.Client, regionCluster *kcmv1beta1.ClusterDeployment, cache CachedClusterData) ([]*kcmv1beta1.ClusterDeployment, error) {
	creds := cache.Credentials
	clusters := cache.Clusters

	region, err := GetRegionByClusterDeployment(ctx, kubeClient, regionCluster, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to get region by cluster deployment %s/%s: %w", regionCluster.Namespace, regionCluster.Name, err)
	}

	if region == nil {
		return nil, fmt.Errorf("region for cluster %s not found", regionCluster.Name)
	}

	if creds == nil {
		creds, err = GetCredentials(ctx, kubeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get credentials: %w", err)
		}
	}

	if clusters == nil {
		clusters, err = GetKofChildClusterDeployments(ctx, kubeClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get kof child cluster deployments: %w", err)
		}
	}

	regionCredNames := make(map[string]struct{}, len(creds.Items))
	for _, cred := range creds.Items {
		if cred.Spec.Region == region.Name {
			regionCredNames[cred.Name] = struct{}{}
		}
	}

	childCds := make([]*kcmv1beta1.ClusterDeployment, 0, len(clusters.Items))
	for _, childCd := range clusters.Items {
		if _, ok := regionCredNames[childCd.Spec.Credential]; ok {
			childCds = append(childCds, &childCd)
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
			LabelSelector: labels.Set{
				kofClusterRoleLabel: kofRoleRegional,
				kofKcmRegionLabel:   utils.True,
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
