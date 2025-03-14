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
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	remotesecret "github.com/k0rdent/kof/kof-operator/internal/controller/remote-secret"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const istioReleaseName = "kof-istio"
const IstioRoleLabel = "k0rdent.mirantis.com/istio-role"

// ClusterDeploymentReconciler reconciles a ClusterDeployment object
type ClusterDeploymentReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	RemoteSecretManager *remotesecret.RemoteSecretManager
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

	clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, clusterDeployment); err != nil {
		if errors.IsNotFound(err) {
			if err := r.RemoteSecretManager.TryDelete(ctx, req); err != nil {
				log.Error(err, "failed to delete remote secret")
				return ctrl.Result{}, err
			}

			if err := r.Client.Delete(ctx, &cmv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getCertName(req.Name),
					Namespace: istio.IstioSystemNamespace,
				},
			}); err != nil {
				if errors.IsNotFound(err) {
					log.Info("CA already deleted")
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
			}

			log.Info("CA successfully deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot read clusterDeployment")
		return ctrl.Result{}, err
	}

	if err := r.ReconcileKofClusterRole(ctx, clusterDeployment); err != nil {
		log.Error(err, "cannot reconcile kof-cluster-role label")
		return ctrl.Result{}, err
	}

	if istioRole, ok := clusterDeployment.Labels[IstioRoleLabel]; ok {
		if istioRole != "child" {
			return ctrl.Result{}, nil
		}

		if err := r.RemoteSecretManager.TryCreate(clusterDeployment, ctx, req); err != nil {
			log.Error(err, "failed to create remote secret")
			return ctrl.Result{}, err
		}

		certName := getCertName(clusterDeployment.Name)
		cert := &cmv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certName,
				Namespace: istio.IstioSystemNamespace,
			},
		}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      certName,
			Namespace: istio.IstioSystemNamespace,
		}, cert); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "cannot read certificate", "name", certName, "namespace", istio.IstioSystemNamespace)
				return ctrl.Result{}, err
			}
			cert.Labels = map[string]string{
				"app.kubernetes.io/managed-by": "kof-operator",
			}
			cert.Spec = cmv1.CertificateSpec{
				IsCA:       true,
				CommonName: fmt.Sprintf("%s CA", clusterDeployment.Name),
				Subject: &cmv1.X509Subject{
					Organizations: []string{"Istio"},
				},
				PrivateKey: &cmv1.CertificatePrivateKey{
					Algorithm: "ECDSA",
					Size:      256,
				},
				SecretName: certName,
				IssuerRef: cmmetav1.ObjectReference{
					Name:  fmt.Sprintf("%s-root", istioReleaseName),
					Kind:  "Issuer",
					Group: "cert-manager.io",
				},
			}
			log.Info("Creating Intermediate Istio CA certificate", "certificateName", cert.Name)
			if err := r.Create(ctx, cert); err != nil {
				log.Error(err, "cannot create certificate")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcmv1alpha1.ClusterDeployment{}).
		Complete(r)
}

func getCertName(clusterName string) string {
	return fmt.Sprintf("kof-istio-%s-ca", clusterName)
}
