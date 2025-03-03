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

package k0rdentmirantiscom

import (
	"context"
	"encoding/base64"
	"fmt"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	"istio.io/istio/istioctl/pkg/multicluster"
	"istio.io/istio/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterDeploymentReconciler reconciles a ClusterDeployment object
type ClusterDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
			// Put code to handle deletion case
			return ctrl.Result{}, nil
		}
		log.Error(err, "cannot read clusterDeployment")
	}

	if !isClusterDeploymentReady(*clusterDeployment.GetConditions()) {
		return ctrl.Result{}, nil
	}

	if !hasIstioChildRoleLabel(clusterDeployment.Labels) {
		return ctrl.Result{}, nil
	}

	kubeconfigSecret := &corev1.Secret{}
	secretName := getSecretName(req.Name)
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: req.Namespace,
	}, kubeconfigSecret); err != nil {
		log.Error(err, fmt.Sprintf("Unable to fetch Secret '%s'", secretName))
		return ctrl.Result{}, err
	}

	log.Info("Secret found", "name", kubeconfigSecret.Name, "namespace", kubeconfigSecret.Namespace)

	kubeconfigDataRaw, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return ctrl.Result{}, fmt.Errorf("kubeconfig secret does not contain 'value' key")
	}

	kubeconfigData, err := base64.StdEncoding.DecodeString(string(kubeconfigDataRaw))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to decode base64 kubeconfig data: %v", err)
	}

	config, err := clientcmd.NewClientConfigFromBytes(kubeconfigData)
	if err != nil {
		log.Error(err, "failed to create new client config")
		return ctrl.Result{}, nil
	}

	client, err := kube.NewCLIClient(config)
	if err != nil {
		log.Error(err, "failed to create cli client")
	}

	secret, warn, err := istio.CreateRemoteSecret(multicluster.RemoteSecretOptions{
		ClusterName:          req.Name,
		CreateServiceAccount: true,
	}, client)
	if err != nil {
		log.Error(err, "failed to create remote secret")
		return ctrl.Result{}, nil
	}

	if warn != nil {
		log.Info("warning when creating remote secret", "warning", warn)
	}

	secret.Namespace = req.Namespace
	if err := r.Client.Create(ctx, secret); err != nil {
		log.Error(err, "failed to create secret on mothership cluster")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func hasIstioChildRoleLabel(labels map[string]string) bool {
	return labels["k0rdent.mirantis.com/istio-role"] == "child"
}

func getSecretName(clusterName string) string {
	return fmt.Sprintf("%s-kubeconfig", clusterName)
}

func isClusterDeploymentReady(conditions []metav1.Condition) bool {
	for _, condition := range conditions {
		if condition.Type == kcmv1alpha1.ReadyCondition {
			if condition.Status == metav1.ConditionTrue {
				return true
			}
			return false
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcmv1alpha1.MultiClusterService{}).
		Complete(r)
}
