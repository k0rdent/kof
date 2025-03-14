package controller

import (
	"context"
	"fmt"
	"strings"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Labels:
const labelPrefix = "k0rdent.mirantis.com/"
const KofClusterRoleLabel = labelPrefix + "kof-cluster-role"
const KofRegionalClusterNameLabel = labelPrefix + "kof-regional-cluster-name"
const KofRegionalDomainLabel = labelPrefix + "kof-regional-domain"

// ConfigMap data keys:
const ClusterDeploymentGenerationKey = "cluster_deployment_generation"
const RegionalClusterNameKey = "regional_cluster_name"
const RegionalDomainKey = "regional_domain"

const KofIstioSecretTemplate = "kof-istio-secret-template"

func getConfigMapName(clusterDeploymentName string) string {
	return "kof-cluster-config-" + clusterDeploymentName
}

func (r *ClusterDeploymentReconciler) ReconcileKofClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1alpha1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)

	configMap := &corev1.ConfigMap{}
	configMapName := getConfigMapName(clusterDeployment.Name)
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: clusterDeployment.Namespace,
	}, configMap)
	if err == nil &&
		configMap.Data[ClusterDeploymentGenerationKey] ==
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

	role := clusterDeployment.Labels[KofClusterRoleLabel]
	if role == "child" {
		return r.reconcileChildClusterRole(ctx, clusterDeployment)
	} // TODO: else if role == "regional" {...}

	return nil
}

func (r *ClusterDeploymentReconciler) reconcileChildClusterRole(
	ctx context.Context,
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)

	regionalClusterName, ok := childClusterDeployment.Labels[KofRegionalClusterNameLabel]
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
		); err != nil {
			log.Error(
				err, "regional ClusterDeployment not found both by label and by location",
				"childClusterDeployment", childClusterDeployment.Name,
				"label", KofRegionalClusterNameLabel,
			)
			return err
		}
		regionalClusterName = regionalClusterDeployment.Name
	}

	regionalDomain, ok := regionalClusterDeployment.Labels[KofRegionalDomainLabel]
	if !ok {
		err := fmt.Errorf("regional domain not found")
		log.Error(
			err, "in",
			"regionalClusterDeployment", regionalClusterName,
			"label", KofRegionalDomainLabel,
		)
		return err
	}

	if err := r.createProfile(ctx, childClusterDeployment, regionalClusterDeployment); err != nil {
		log.Error(err, "Failed to create profile")
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfigMapName(childClusterDeployment.Name),
			Namespace: childClusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				// Auto-delete ConfigMap when child ClusterDeployment is deleted.
				GetOwnerReference(childClusterDeployment),
			},
		},
		Data: map[string]string{
			ClusterDeploymentGenerationKey: fmt.Sprintf("%d", childClusterDeployment.Generation),
			RegionalClusterNameKey:         regionalClusterName,
			RegionalDomainKey:              regionalDomain,
		},
	}

	if err := r.Create(ctx, configMap); err != nil {
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
			RegionalClusterNameKey, regionalClusterName,
			RegionalDomainKey, regionalDomain,
		)
		return nil
	}

	log.Info(
		"Created child cluster ConfigMap",
		"name", configMap.Name,
		RegionalClusterNameKey, regionalClusterName,
		RegionalDomainKey, regionalDomain,
	)
	return nil
}

func (r *ClusterDeploymentReconciler) createProfile(ctx context.Context, childClusterDeployment, regionalClusterDeployment *kcmv1alpha1.ClusterDeployment) error {
	log := log.FromContext(ctx)
	remoteSecretName := istio.RemoteSecretNameFromClusterName(regionalClusterDeployment.Name)

	log.Info("Creating profile")

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteSecretName,
			Namespace: childClusterDeployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kof-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				GetOwnerReference(childClusterDeployment),
			},
		},
		Spec: sveltosv1beta1.Spec{
			ClusterRefs: []corev1.ObjectReference{
				{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       clusterv1.ClusterKind,
					Name:       childClusterDeployment.Name,
					Namespace:  childClusterDeployment.Namespace,
				},
			},
			TemplateResourceRefs: []sveltosv1beta1.TemplateResourceRef{
				{
					Identifier: "Secret",
					Resource: corev1.ObjectReference{
						APIVersion: corev1.SchemeGroupVersion.Version,
						Kind:       "Secret",
						Name:       remoteSecretName,
						Namespace:  istio.IstioSystemNamespace,
					},
				},
			},
			PolicyRefs: []sveltosv1beta1.PolicyRef{
				{
					Kind:      "ConfigMap",
					Name:      KofIstioSecretTemplate,
					Namespace: istio.IstioSystemNamespace,
				},
			},
		},
	}

	if err := r.Create(ctx, profile); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Profile is already created")
			return nil
		}
		return err
	}

	log.Info("Profile successfully created", "profile", profile.Name)
	return nil
}

func getCloud(clusterDeployment *kcmv1alpha1.ClusterDeployment) string {
	cloud, _, _ := strings.Cut(clusterDeployment.Spec.Template, "-")
	return cloud
}

func (r *ClusterDeploymentReconciler) discoverRegionalClusterDeploymentByLocation(
	ctx context.Context,
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
) (*kcmv1alpha1.ClusterDeployment, error) {
	log := log.FromContext(ctx)
	childCloud := getCloud(childClusterDeployment)

	childClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
		childClusterDeployment.Spec.Config.Raw,
	)
	if err != nil {
		log.Error(
			err, "cannot read child ClusterDeployment config",
			"name", childClusterDeployment.Name,
		)
		return nil, err
	}

	regionalClusterDeploymentList := &kcmv1alpha1.ClusterDeploymentList{}
	for {
		opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "regional"}}
		if regionalClusterDeploymentList.Continue != "" {
			opts = append(opts, client.Continue(regionalClusterDeploymentList.Continue))
		}

		if err := r.List(ctx, regionalClusterDeploymentList, opts...); err != nil {
			log.Error(err, "cannot list regional ClusterDeployments")
			return nil, err
		}

		for _, regionalClusterDeployment := range regionalClusterDeploymentList.Items {
			if childCloud != getCloud(&regionalClusterDeployment) {
				continue
			}

			regionalClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
				regionalClusterDeployment.Spec.Config.Raw,
			)
			if err != nil {
				continue
			}

			if locationIsTheSame(
				childCloud,
				childClusterDeploymentConfig,
				regionalClusterDeploymentConfig,
			) {
				return &regionalClusterDeployment, nil
			}
		}

		if regionalClusterDeploymentList.Continue == "" {
			break
		}
	}

	return nil, fmt.Errorf(
		"regional ClusterDeployment with matching location is not found, "+
			`please set .metadata.labels["%s"] explicitly`,
		KofRegionalClusterNameLabel,
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
