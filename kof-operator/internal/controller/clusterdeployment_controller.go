/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/istio/cert"
	remotesecret "github.com/k0rdent/kof/kof-operator/internal/controller/istio/remote-secret"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const IstioRoleLabel = "k0rdent.mirantis.com/istio-role"

// ClusterDeploymentReconciler reconciles a ClusterDeployment object
type ClusterDeploymentReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	RemoteSecretManager *remotesecret.RemoteSecretManager
	IstioCertManager    *cert.CertManager
}

// +kubebuilder:rbac:groups=k0rdent.mirantis.com,resources=clusterdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k0rdent.mirantis.com,resources=clusterdeployments/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ClusterDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	clusterDeployment := utils.GetClusterDeploymentStub(req.Name, req.Namespace)
	if err := r.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, clusterDeployment); err != nil {
		if errors.IsNotFound(err) {
			if err := r.RemoteSecretManager.TryDelete(ctx, req); err != nil {
				utils.LogEvent(
					ctx,
					"SecretDeletionFailed",
					"Failed to delete remote secret",
					clusterDeployment,
					err,
					"remoteSecretName", remotesecret.GetRemoteSecretName(req.Name),
				)
				return ctrl.Result{}, err
			}

			if err := r.IstioCertManager.TryDelete(ctx, req); err != nil {
				utils.LogEvent(
					ctx,
					"IstioCertDeletionFailed",
					"Failed to delete istio certificate",
					clusterDeployment,
					err,
					"certName", cert.GetCertName(req.Name),
				)
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot read clusterDeployment")
		return ctrl.Result{}, err
	}

	if err := r.ReconcileKofClusterRole(ctx, clusterDeployment); err != nil {
		return ctrl.Result{}, err
	}

	if istioRole, ok := clusterDeployment.Labels[IstioRoleLabel]; ok {
		if istioRole != "child" {
			return ctrl.Result{}, nil
		}

		if err := r.RemoteSecretManager.TryCreate(clusterDeployment, ctx, req); err != nil {
			utils.LogEvent(
				ctx,
				"SecretCreationFailed",
				"Failed to create remote secret",
				clusterDeployment,
				err,
				"remoteSecretName", remotesecret.GetRemoteSecretName(req.Name),
			)
			return ctrl.Result{}, err
		}

		if err := r.IstioCertManager.TryCreate(ctx, clusterDeployment); err != nil {
			utils.LogEvent(
				ctx,
				"IstioCertCreationFailed",
				"Failed to create istio CA certificate",
				clusterDeployment,
				err,
				"certName", cert.GetCertName(req.Name),
			)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcmv1beta1.ClusterDeployment{}).
		Complete(r)
}
