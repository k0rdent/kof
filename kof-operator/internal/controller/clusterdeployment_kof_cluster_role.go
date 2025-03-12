package controller

import (
	"context"
	"fmt"
	"strings"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const CLUSTER_DEPLOYMENT_GENERATION_KEY = "cluster_deployment_generation"
const REGIONAL_CLUSTER_NAME_KEY = "regional_cluster_name"
const REGIONAL_DOMAIN_KEY = "regional_domain"

func getConfigMapName(clusterDeploymentName string) string {
	return "kof-cluster-config-" + clusterDeploymentName
}

func (r *ClusterDeploymentReconciler) ReconcileKofClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1alpha1.ClusterDeployment,
	clusterDeploymentConfig *ClusterDeploymentConfig,
) error {
	log := log.FromContext(ctx)

	configMap := &corev1.ConfigMap{}
	configMapName := getConfigMapName(clusterDeployment.Name)
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: clusterDeployment.Namespace,
	}, configMap)
	if err == nil &&
		configMap.Data[CLUSTER_DEPLOYMENT_GENERATION_KEY] ==
			fmt.Sprintf("%d", clusterDeployment.Generation) {
		// Logging nothing as we have a lot of frequent `status` updates to ignore here.
		// Cannot add `WithEventFilter(predicate.GenerationChangedPredicate{})`
		// to `SetupWithManager` of reconciler shared with istio which needs `status` updates.
		return nil
	}

	// If this ConfigMap is not found, it's OK, we will create it below.
	// Any other error should be handled:
	if err != nil && !errors.IsNotFound(err) {
		log.Error(
			err, "cannot read existing child cluster ConfigMap",
			"name", configMapName,
		)
		return err
	}

	role := clusterDeploymentConfig.ClusterLabels["k0rdent.mirantis.com/kof-cluster-role"]

	if role == "child" {
		return r.reconcileChildClusterRole(ctx, clusterDeployment, clusterDeploymentConfig)
	} // TODO: else if role == "regional" {...}

	return nil
}

func (r *ClusterDeploymentReconciler) reconcileChildClusterRole(
	ctx context.Context,
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
	childClusterDeploymentConfig *ClusterDeploymentConfig,
) error {
	log := log.FromContext(ctx)

	labelName := "k0rdent.mirantis.com/kof-regional-cluster-name"
	regionalClusterName, ok := childClusterDeploymentConfig.ClusterLabels[labelName]
	regionalClusterDeployment := &kcmv1alpha1.ClusterDeployment{}
	if ok {
		err := r.Get(ctx, types.NamespacedName{
			Name:      regionalClusterName,
			Namespace: childClusterDeployment.Namespace,
		}, regionalClusterDeployment)
		if err != nil {
			log.Error(
				err, "regional ClusterDeployment not found",
				"name", regionalClusterName,
			)
			return err
		}
	} else {
		var err error
		if regionalClusterDeployment, err = r.discoverRegionalClusterDeploymentByLocation(
			ctx,
			childClusterDeployment,
			childClusterDeploymentConfig,
		); err != nil {
			log.Error(
				err, "regional ClusterDeployment not found both by label and by location",
				"childClusterDeployment", childClusterDeployment.Name,
				"clusterLabel", labelName,
			)
			return err
		}
		regionalClusterName = regionalClusterDeployment.Name
	}

	regionalClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
		regionalClusterDeployment.Spec.Config.Raw,
	)
	if err != nil {
		log.Error(
			err, "cannot read regional ClusterDeployment config",
			"name", regionalClusterName,
		)
		return err
	}

	labelName = "k0rdent.mirantis.com/kof-regional-domain"
	regionalDomain, ok := regionalClusterDeploymentConfig.ClusterLabels[labelName]
	if !ok {
		err := fmt.Errorf("regional domain not found")
		log.Error(
			err, "in",
			"regionalClusterDeployment", regionalClusterName,
			"clusterLabel", labelName,
		)
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfigMapName(childClusterDeployment.Name),
			Namespace: childClusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				// Auto-delete ConfigMap when child ClusterDeployment is deleted.
				{
					APIVersion: "k0rdent.mirantis.com/v1alpha1",
					Kind:       "ClusterDeployment",
					Name:       childClusterDeployment.Name,
					UID:        childClusterDeployment.GetUID(),
				},
			},
		},
		Data: map[string]string{
			CLUSTER_DEPLOYMENT_GENERATION_KEY: fmt.Sprintf("%d", childClusterDeployment.Generation),
			REGIONAL_CLUSTER_NAME_KEY:         regionalClusterName,
			REGIONAL_DOMAIN_KEY:               regionalDomain,
		},
	}

	if err = r.Create(ctx, configMap); err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(
				err, "cannot create child cluster ConfigMap",
				"name", configMap.Name,
			)
			return err
		}

		if err = r.Update(ctx, configMap); err != nil {
			log.Error(
				err, "cannot update child cluster ConfigMap",
				"name", configMap.Name,
			)
			return err
		}

		log.Info(
			"Updated child cluster ConfigMap",
			"name", configMap.Name,
			REGIONAL_CLUSTER_NAME_KEY, regionalClusterName,
			REGIONAL_DOMAIN_KEY, regionalDomain,
		)
		return nil
	}

	log.Info(
		"Created child cluster ConfigMap",
		"name", configMap.Name,
		REGIONAL_CLUSTER_NAME_KEY, regionalClusterName,
		REGIONAL_DOMAIN_KEY, regionalDomain,
	)
	return nil
}

func getCloud(clusterDeployment *kcmv1alpha1.ClusterDeployment) string {
	cloud, _, _ := strings.Cut(clusterDeployment.Spec.Template, "-")
	return cloud
}

func (r *ClusterDeploymentReconciler) discoverRegionalClusterDeploymentByLocation(
	ctx context.Context,
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
	childClusterDeploymentConfig *ClusterDeploymentConfig,
) (*kcmv1alpha1.ClusterDeployment, error) {
	childCloud := getCloud(childClusterDeployment)

	clusterDeploymentList := &kcmv1alpha1.ClusterDeploymentList{}
	for {
		var opts []client.ListOption
		if clusterDeploymentList.Continue != "" {
			opts = append(opts, client.Continue(clusterDeploymentList.Continue))
		}

		if err := r.List(ctx, clusterDeploymentList, opts...); err != nil {
			return nil, err
		}

		for _, clusterDeployment := range clusterDeploymentList.Items {
			if childCloud != getCloud(&clusterDeployment) {
				continue
			}

			clusterDeploymentConfig, err := ReadClusterDeploymentConfig(
				clusterDeployment.Spec.Config.Raw,
			)
			if err != nil {
				continue
			}

			role := clusterDeploymentConfig.ClusterLabels["k0rdent.mirantis.com/kof-cluster-role"]
			if role != "regional" {
				continue
			}

			if locationIsTheSame(childCloud, childClusterDeploymentConfig, clusterDeploymentConfig) {
				return &clusterDeployment, nil
			}
		}

		if clusterDeploymentList.Continue == "" {
			break
		}
	}

	return nil, fmt.Errorf(
		"regional ClusterDeployment with matching location is not found, " +
			"please set clusterLabel k0rdent.mirantis.com/kof-regional-cluster-name explicitly",
	)
}

func locationIsTheSame(cloud string, c1, c2 *ClusterDeploymentConfig) bool {
	switch cloud {
	case "adopted":
		return false
	case "aws":
		return c1.Region == c2.Region
	case "azure":
		return c1.Location == c2.Location
	case "docker":
		return true
	case "openstack":
		return c1.IdentityRef.Region == c2.IdentityRef.Region
	case "remote":
		return false
	case "vsphere":
		return c1.VSphere.Datacenter == c2.VSphere.Datacenter
	}

	return false
}
