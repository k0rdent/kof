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

	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
)

const PromxySecretNameLabel = "k0rdent.mirantis.com/promxy-secret-name"

type PromxyConfigReloadFunc func() error

// PromxyServerGroupReconciler reconciles a PromxyServerGroup object
type PromxyServerGroupReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	RemoteWriteUrl     string
	PromxyConfigReload PromxyConfigReloadFunc
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

	promxyServerGroupsList := &kofv1beta1.PromxyServerGroupList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, promxyServerGroupsList, opts...); err != nil {
		log.Error(err, "cannot get promxy server group list")
		return ctrl.Result{}, err
	}

	promxyServerGroupsBySecretName := make(map[string][]*kofv1beta1.PromxyServerGroup)

	for _, promxyServerGroup := range promxyServerGroupsList.Items {
		name, ok := promxyServerGroup.Labels[PromxySecretNameLabel]
		if !ok {
			log.Info("Skipping promxyServerGroup that doesn't have secret name label", "promxyServerGroup", promxyServerGroup)
			continue
		}
		groups, ok := promxyServerGroupsBySecretName[name]
		if !ok {
			groups = make([]*kofv1beta1.PromxyServerGroup, 0)
		}
		groups = append(groups, &promxyServerGroup)
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
			basicAuthEnabled := group.Spec.HttpClient.BasicAuth.CredentialsSecretName != ""
			if basicAuthEnabled {
				if err := r.Get(ctx, types.NamespacedName{
					Name:      group.Spec.HttpClient.BasicAuth.CredentialsSecretName,
					Namespace: req.Namespace,
				}, credentialsSecret); err != nil {
					log.Error(err, "cannot read auth credentials secret")
					return ctrl.Result{}, err
				}
			}
			secretTemplateData.ServerGroups = append(secretTemplateData.ServerGroups, &PromxyConfigServerGroup{
				Targets:               group.Spec.Targets,
				PathPrefix:            group.Spec.PathPrefix,
				Scheme:                group.Spec.Scheme,
				DialTimeout:           group.Spec.HttpClient.DialTimeout.Duration.String(),
				TlsInsecureSkipVerify: group.Spec.HttpClient.TLSConfig.InsecureSkipVerify,
				BasicAuthEnabled:      basicAuthEnabled,
				Username:              string(credentialsSecret.Data[group.Spec.HttpClient.BasicAuth.UsernameKey]),
				Password:              string(credentialsSecret.Data[group.Spec.HttpClient.BasicAuth.PasswordKey]),
				ClusterName:           group.Spec.ClusterName,
			})
		}
		data, err := RenderPromxySecretTemplate(secretTemplateData)
		if err != nil {
			log.Error(err, "cannot render promxy secret template")
			return ctrl.Result{}, err
		}
		secret := &coreV1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		}
		err = r.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: req.Namespace,
		}, secret)
		if err != nil && errors.IsNotFound(err) {
			secret.ObjectMeta = metav1.ObjectMeta{
				Name:      name,
				Namespace: req.Namespace,
			}
			setSecretOperatorLabels(secret)
			secret.StringData = map[string]string{
				"config.yaml": data,
			}
			log.Info("Creating promxy config secret", "secretName", name)
			if err := r.Create(ctx, secret); err != nil {
				utils.LogEvent(
					ctx,
					"PromxySecretCreationFailed",
					"Cannot create promxy secret",
					secret,
					err,
					"promxySecretName", secret.Name,
				)
				return ctrl.Result{}, err
			}
			log.Info("Reloading promxy config")
			if err := r.PromxyConfigReload(); err != nil {
				utils.LogEvent(
					ctx,
					"PromxyConfigReloadingFailed",
					"Cannot reload promxy config",
					secret,
					err,
					"promxySecretName", secret.Name,
				)
				return ctrl.Result{}, err
			}
			continue
		}
		if err != nil {
			utils.LogEvent(
				ctx,
				"PromxySecretNotFound",
				"Cannot get promxy secret",
				secret,
				err,
				"promxySecretName", secret.Name,
			)
			return ctrl.Result{}, err
		}
		setSecretOperatorLabels(secret)
		secret.StringData = map[string]string{
			"config.yaml": data,
		}
		log.Info("Updating promxy config secret", "secretName", name)
		if err := r.Update(ctx, secret); err != nil {
			utils.LogEvent(
				ctx,
				"PromxySecretUpdateFailed",
				"Cannot update promxy secret",
				secret,
				err,
				"promxySecretName", secret.Name,
			)
			return ctrl.Result{}, err
		}
		log.Info("Reloading promxy config")
		if err := r.PromxyConfigReload(); err != nil {
			utils.LogEvent(
				ctx,
				"PromxySecretReloadFailed",
				"Cannot reload promxy config",
				secret,
				err,
				"promxySecretName", secret.Name,
			)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func setSecretOperatorLabels(secret *coreV1.Secret) {
	secret.Labels = map[string]string{utils.ManagedByLabel: utils.ManagedByValue}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PromxyServerGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kofv1beta1.PromxyServerGroup{}).
		Complete(r)
}
