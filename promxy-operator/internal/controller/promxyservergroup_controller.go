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

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kofv1alpha1 "github.com/k0rdent/kof/promxy-operator/api/v1alpha1"
)

const SecretNameLabel = "k0rdent.mirantis.com/promxy-secret-name"

// PromxyServerGroupReconciler reconciles a PromxyServerGroup object
type PromxyServerGroupReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	RemoteWriteUrl string
}

// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=promxyservergroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PromxyServerGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PromxyServerGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	promxyServerGroupsList := &kofv1alpha1.PromxyServerGroupList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, promxyServerGroupsList, opts...); err != nil {
		log.Error(err, "cannot get promxy server group list")
		return ctrl.Result{}, err
	}

	if len(promxyServerGroupsList.Items) == 0 {
		if err := r.Client.DeleteAllOf(ctx, &coreV1.Secret{}, client.InNamespace(req.NamespacedName.Namespace), client.MatchingLabels{
			"app.kubernetes.io/managed-by": "promxy-operator",
		}); err != nil {
			log.Error(err, "cannot delete promxy secrets created by operator")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	promxyServerGroupsBySecretName := make(map[string][]*kofv1alpha1.PromxyServerGroup)

	for _, promxyServerGroup := range promxyServerGroupsList.Items {
		name, ok := promxyServerGroup.Labels[SecretNameLabel]
		if !ok {
			log.Info("Skipping promxyServerGroup that doesn't have secret name label", "promxyServerGroup", promxyServerGroup)
			continue
		}
		groups, ok := promxyServerGroupsBySecretName[name]
		if !ok {
			groups = make([]*kofv1alpha1.PromxyServerGroup, 0)
			groups = append(groups, &promxyServerGroup)
		}
		promxyServerGroupsBySecretName[name] = groups
	}

	log.Info("Processing promxy server groups", "promxyServerGroupsBySecretName", promxyServerGroupsBySecretName)

	for name, groups := range promxyServerGroupsBySecretName {
		secretTemplateData := &PromxyConfig{
			RemoteWriteUrl: r.RemoteWriteUrl,
			ServerGroups:   make([]*PromxyConfigServerGroup, 0),
		}
		for _, group := range groups {
			credentialsSecret := &coreV1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{
				Name:      group.Spec.HttpClient.BasicAuth.CredentialsSecretName,
				Namespace: req.Namespace,
			}, credentialsSecret); err != nil {
				log.Error(err, "cannot read auth credentials secret")
				return ctrl.Result{}, err
			}
			secretTemplateData.ServerGroups = append(secretTemplateData.ServerGroups, &PromxyConfigServerGroup{
				Targets:               group.Spec.Targets,
				PathPrefix:            group.Spec.PathPrefix,
				Scheme:                group.Spec.Scheme,
				DialTimeout:           group.Spec.HttpClient.DialTimeout.String(),
				TlsInsecureSkipVerify: group.Spec.HttpClient.TLSConfig.InsecureSkipVerify,
				Username:              credentialsSecret.StringData[group.Spec.HttpClient.BasicAuth.UsernameKey],
				Password:              credentialsSecret.StringData[group.Spec.HttpClient.BasicAuth.PasswordKey],
				ClusterName:           group.Name,
			})
		}
		data, err := RenderPromxySecretTemplate(secretTemplateData)
		if err != nil {
			log.Error(err, "cannot render promxy secret template")
			return ctrl.Result{}, err
		}
		secret := &coreV1.Secret{}
		err = r.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: req.Namespace,
		}, secret)
		if err != nil && errors.IsNotFound(err) {
			secret.ObjectMeta = metav1.ObjectMeta{
				Name:      name,
				Namespace: req.Namespace,
			}
			secret.Labels = map[string]string{
				"app.kubernetes.io/managed-by": "promxy-operator",
			}
			secret.Labels[SecretNameLabel] = name
			secret.StringData = map[string]string{
				"config.yaml": data,
			}
			if err := r.Create(ctx, secret); err != nil {
				log.Error(err, "cannot create promxy secret")
				return ctrl.Result{}, err
			}
			continue
		}
		if err != nil {
			log.Error(err, "cannot get promxy secret")
			return ctrl.Result{}, err
		}
		secret.StringData = map[string]string{
			"config.yaml": data,
		}
		if err := r.Update(ctx, secret); err != nil {
			log.Error(err, "cannot update promxy secret")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PromxyServerGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kofv1alpha1.PromxyServerGroup{}).
		Complete(r)
}
