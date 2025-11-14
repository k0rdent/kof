package k8s

import (
	"context"
	"fmt"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultKCMSystemNamespace = "kcm-system"

// If a Kubeconfig secret exists in the management cluster, we assume the cluster is not in the region
func CreatedInKCMRegion(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (bool, error) {
	if utils.IsAdopted(cd) {
		creds := new(kcmv1beta1.CredentialList)
		if err := client.List(ctx, creds); err != nil {
			return false, err
		}

		for _, cred := range creds.Items {
			if cred.Name != cd.Spec.Credential {
				continue
			}

			if cred.Spec.Region == "" {
				return false, nil
			}
		}
		return true, nil
	}

	secret := new(corev1.Secret)
	namespacedName := types.NamespacedName{
		Name:      GetSecretName(cd),
		Namespace: DefaultKCMSystemNamespace,
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

func CreatedInKCMRegionByName(ctx context.Context, k8sClient client.Client, clusterName string) (bool, error) {
	clusterSecretName := fmt.Sprintf("%s-%s", clusterName, ClusterSecretSuffix)
	adoptedClusterSecretName := fmt.Sprintf("%s-%s", clusterName, AdoptedClusterSecretSuffix)

	secretNames := []string{clusterSecretName, adoptedClusterSecretName}
	kubeconfigSecret := new(corev1.Secret)

	for _, secretName := range secretNames {
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: DefaultKCMSystemNamespace,
		}, kubeconfigSecret); err == nil {
			return false, nil
		} else if !errors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get kubeconfig secret '%s': %v", secretName, err)
		}
	}

	return true, nil
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

	kubeconfigSecretName := GetSecretName(regional)
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

func GetClusterDeploymentsInSameKcmRegion(ctx context.Context, client client.Client, clusterDeployment *kcmv1beta1.ClusterDeployment) ([]*kcmv1beta1.ClusterDeployment, error) {
	clusterList := new(kcmv1beta1.ClusterDeploymentList)
	regionList := new(kcmv1beta1.RegionList)
	credList := new(kcmv1beta1.CredentialList)

	if err := client.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list cluster deployments: %v", err)
	}

	if err := client.List(ctx, regionList); err != nil {
		return nil, fmt.Errorf("failed to list regions: %v", err)
	}

	if err := client.List(ctx, credList); err != nil {
		return nil, fmt.Errorf("failed to list credentials: %v", err)
	}

	var result []*kcmv1beta1.ClusterDeployment
	var regionName string
	var regionClusterName string

	for _, creds := range credList.Items {
		if creds.Name == clusterDeployment.Spec.Credential {
			regionName = creds.Spec.Region
			break
		}
	}

	for _, region := range regionList.Items {
		if region.Name != regionName {
			continue
		}

		if region.Spec.ClusterDeployment != nil {
			regionClusterName = region.Spec.ClusterDeployment.Name
			break
		}

		if region.Spec.KubeConfig != nil && region.Spec.KubeConfig.Name != "" {
			if clusterName, found := strings.CutSuffix(region.Spec.KubeConfig.Name, ClusterSecretSuffix); found {
				regionClusterName = clusterName
				break
			}

			if clusterName, found := strings.CutSuffix(region.Spec.KubeConfig.Name, AdoptedClusterSecretSuffix); found {
				regionClusterName = clusterName
				break
			}
		}
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
