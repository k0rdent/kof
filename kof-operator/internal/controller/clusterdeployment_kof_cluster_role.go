package controller

import (
	"context"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const CLUSTER_DEPLOYMENT_GENERATION_KEY = "cluster_deployment_generation"
const REGIONAL_CLUSTER_NAME_KEY = "regional_cluster_name"
const REGIONAL_DOMAIN_KEY = "regional_domain"

const KOF_ISTIO_SECRET_TEMPLATE = "kof-istio-secret-template"

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
	if !ok {
		err := fmt.Errorf("regional cluster name not found")
		log.Error(
			err, "in",
			"childClusterDeployment", childClusterDeployment.Name,
			"clusterLabel", labelName,
		)
		return err
	}

	regionalClusterDeployment := &kcmv1alpha1.ClusterDeployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      regionalClusterName,
		Namespace: childClusterDeployment.Namespace,
	}, regionalClusterDeployment); err != nil {
		log.Error(
			err, "regional ClusterDeployment not found",
			"name", regionalClusterName,
		)
		return err
	}

	regionalConfig, err := ReadClusterDeploymentConfig(
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
	regionalDomain, ok := regionalConfig.ClusterLabels[labelName]
	if !ok {
		err := fmt.Errorf("regional domain not found")
		log.Error(
			err, "in",
			"regionalClusterDeployment", regionalClusterName,
			"clusterLabel", labelName,
		)
		return err
	}

	if err := r.createProfile(childClusterDeployment, regionalClusterDeployment, ctx); err != nil {
		log.Error(err, "Failed to create profile")
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getConfigMapName(childClusterDeployment.Name),
			Namespace: childClusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				// Auto-delete ConfigMap when child ClusterDeployment is deleted.
				GetOwnerReference(
					childClusterDeployment.Name,
					childClusterDeployment.GetUID(),
				),
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

func (r *ClusterDeploymentReconciler) createProfile(childClusterDeployment, regionalClusterDeployment *kcmv1alpha1.ClusterDeployment, ctx context.Context) error {
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
				GetOwnerReference(
					childClusterDeployment.Name,
					childClusterDeployment.GetUID(),
				),
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
					Name:      KOF_ISTIO_SECRET_TEMPLATE,
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
