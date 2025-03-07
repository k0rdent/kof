package controller

import (
	"context"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ClusterDeploymentReconciler) ReconcileClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1alpha1.ClusterDeployment,
	config *ClusterDeploymentConfig,
) error {
	log := log.FromContext(ctx)

	if config.ClusterLabels["k0rdent.mirantis.com/kof-cluster-role"] == "child" {
		childClusterDeployment := clusterDeployment

		labelName := "k0rdent.mirantis.com/kof-regional-cluster-name"
		regionalClusterName, ok := config.ClusterLabels[labelName]
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

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kof-cluster-config-" + childClusterDeployment.Name,
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
				"regional_domain": regionalDomain,
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
		}

		log.Info(
			"Created or updated child cluster ConfigMap",
			"name", configMap.Name,
			"regional_domain", regionalDomain,
		)
	}

	return nil
}
