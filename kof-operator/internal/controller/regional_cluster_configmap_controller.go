package controller

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/vmuser"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
type RegionalClusterConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Make controller react to `ConfigMaps` having the expected label only.
func (r *RegionalClusterConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("regional-cluster-config-map").
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			return obj.GetLabels()[KofClusterRoleLabel] == KofRoleRegional
		})).
		Complete(r)
}

// When a ConfigMap with the expected labels is created, updated or deleted,
// create or update the related objects.
func (r *RegionalClusterConfigMapReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	cm := &corev1.ConfigMap{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, cm)
	if errors.IsNotFound(err) {
		return ResourceCleanup(ctx, r.Client, req.Name, req.Namespace)
	}

	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get config map: %v", err)
	}

	clusterConfigMap, err := NewRegionalClusterConfigMap(ctx, cm, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create cluster config map: %v", err)
	}

	if err := clusterConfigMap.Reconcile(); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile cluster config map: %v", err)
	}

	return ctrl.Result{}, nil
}

func ResourceCleanup(ctx context.Context, client client.Client, cmName, namespace string) (ctrl.Result, error) {
	if err := vmuser.NewManager(client).Delete(ctx, cmName, namespace); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete VMUser resources: %v", err)
	}

	if err := CleanupVmRulesMcsPropagation(ctx, client, cmName); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to cleanup VM rules MCS propagation: %v", err)
	}

	return ctrl.Result{}, nil
}

// Function deletes the MultiClusterService created to propagate VM rules to the region cluster.
// TODO: Remove this function once KCM implements automatic copying of the required resources to region clusters.
func CleanupVmRulesMcsPropagation(ctx context.Context, client client.Client, cmName string) error {
	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetVmRulesMcsPropagationName(cmName),
		},
	}

	if err := client.Delete(ctx, mcs); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}
