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
	"maps"
	"reflect"
	"strconv"
	"strings"

	vmv1 "github.com/VictoriaMetrics/operator/api/operator/v1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	vmStorageConnectionFinalizer = "k0rdent.mirantis.com/vmstorageconnection"
	vmSecretMountPath            = "/etc/vm/secrets"

	storageNodeArg                   = "storageNode"
	storageNodeUsernameFileArg       = "storageNode.usernameFile"
	storageNodePasswordFileArg       = "storageNode.passwordFile"
	storageNodeTLSArg                = "storageNode.tls"
	storageNodeTLSInsecureSkipVerify = "storageNode.tlsInsecureSkipVerify"
)

// storageNodeOwnedArgs are the ExtraArgs keys managed exclusively by this controller.
// They are rebuilt from scratch on every sync and cleared when no connections remain.
var storageNodeOwnedArgs = []string{
	storageNodeArg,
	storageNodeUsernameFileArg,
	storageNodePasswordFileArg,
	storageNodeTLSArg,
	storageNodeTLSInsecureSkipVerify,
}

// storageCluster is the common interface for cluster types managed by VMStorageConnection.
type storageCluster interface {
	client.Object
	// clusterKind returns the resource kind string (e.g. "VTCluster" or "VLCluster").
	clusterKind() string
	// storageExtraArgs returns the ExtraArgs map used for storageNode configuration.
	storageExtraArgs() map[string]string
	// storageSecrets returns the secrets list used for storageNode credentials.
	storageSecrets() []string
	// applyStorageConfig writes the computed args and secrets into the cluster spec.
	applyStorageConfig(args map[string]string, secrets []string)
	// deepCopy returns a deep copy wrapped in the same adapter type.
	deepCopy() storageCluster
	// object returns the underlying Kubernetes resource for API operations.
	object() client.Object
}

// vtClusterAdapter wraps *vmv1.VTCluster to implement storageCluster.
type vtClusterAdapter struct{ *vmv1.VTCluster }

func (a *vtClusterAdapter) clusterKind() string { return "VTCluster" }

func (a *vtClusterAdapter) storageExtraArgs() map[string]string {
	if a.Spec.Select == nil {
		return nil
	}
	return a.Spec.Select.ExtraArgs
}
func (a *vtClusterAdapter) storageSecrets() []string {
	if a.Spec.Select == nil {
		return nil
	}
	return a.Spec.Select.Secrets
}

func (a *vtClusterAdapter) applyStorageConfig(args map[string]string, secrets []string) {
	if a.Spec.Select == nil {
		a.Spec.Select = &vmv1.VTSelect{}
	}
	a.Spec.Select.ExtraArgs = args
	a.Spec.Select.Secrets = secrets
}
func (a *vtClusterAdapter) deepCopy() storageCluster {
	return &vtClusterAdapter{a.DeepCopy()}
}
func (a *vtClusterAdapter) object() client.Object { return a.VTCluster }

// vlClusterAdapter wraps *vmv1.VLCluster to implement storageCluster.
type vlClusterAdapter struct{ *vmv1.VLCluster }

func (a *vlClusterAdapter) clusterKind() string { return "VLCluster" }

func (a *vlClusterAdapter) storageExtraArgs() map[string]string {
	if a.Spec.VLSelect == nil {
		return nil
	}
	return a.Spec.VLSelect.ExtraArgs
}

func (a *vlClusterAdapter) storageSecrets() []string {
	if a.Spec.VLSelect == nil {
		return nil
	}
	return a.Spec.VLSelect.Secrets
}

func (a *vlClusterAdapter) applyStorageConfig(args map[string]string, secrets []string) {
	if a.Spec.VLSelect == nil {
		a.Spec.VLSelect = &vmv1.VLSelect{}
	}
	a.Spec.VLSelect.ExtraArgs = args
	a.Spec.VLSelect.Secrets = secrets
}
func (a *vlClusterAdapter) deepCopy() storageCluster {
	return &vlClusterAdapter{a.DeepCopy()}
}
func (a *vlClusterAdapter) object() client.Object { return a.VLCluster }

// VMStorageConnectionReconciler reconciles a VMStorageConnection object
type VMStorageConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vmstorageconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vmstorageconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vmstorageconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.victoriametrics.com,resources=vtclusters,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.victoriametrics.com,resources=vlclusters,verbs=get;list;watch;update;patch

// Reconcile fetches the VMStorageConnection and configures the referenced VTCluster or VLCluster
// resource with the target storage node address.
func (r *VMStorageConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	conn := new(kofv1beta1.VMStorageConnection)
	if err := r.Get(ctx, req.NamespacedName, conn); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get VMStorageConnection: %w", err)
	}

	clusterNS := conn.Spec.ClusterRef.Namespace
	if clusterNS == "" {
		clusterNS = conn.Namespace
	}

	switch conn.Spec.ClusterRef.Kind {
	case "VTCluster":
		return r.reconcileCluster(ctx, conn, clusterNS, r.fetchVTCluster)
	case "VLCluster":
		return r.reconcileCluster(ctx, conn, clusterNS, r.fetchVLCluster)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported cluster kind: %q", conn.Spec.ClusterRef.Kind)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VMStorageConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kofv1beta1.VMStorageConnection{}).
		Named("vmstorageconnection").
		Complete(r)
}

// reconcileCluster handles reconciling a VMStorageConnection for any supported cluster kind.
func (r *VMStorageConnectionReconciler) reconcileCluster(
	ctx context.Context,
	conn *kofv1beta1.VMStorageConnection,
	clusterNS string,
	fetch func(context.Context, string, string) (storageCluster, error),
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cluster, err := fetch(ctx, conn.Spec.ClusterRef.Name, clusterNS)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get %s %s/%s: %w", conn.Spec.ClusterRef.Kind, clusterNS, conn.Spec.ClusterRef.Name, err)
	}

	if !conn.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.handleClusterDeletion(ctx, conn, cluster)
	}

	if cluster == nil {
		return ctrl.Result{}, fmt.Errorf("%s %s/%s not found", conn.Spec.ClusterRef.Kind, clusterNS, conn.Spec.ClusterRef.Name)
	}

	if !controllerutil.ContainsFinalizer(conn, vmStorageConnectionFinalizer) {
		controllerutil.AddFinalizer(conn, vmStorageConnectionFinalizer)
		if conn.Labels == nil {
			conn.Labels = make(map[string]string)
		}
		conn.Labels[labels.ClusterNameLabelKey] = conn.Spec.ClusterRef.Name
		conn.Labels[labels.ClusterKindLabelKey] = conn.Spec.ClusterRef.Kind
		if err := r.Update(ctx, conn); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	if err := r.syncCluster(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled VMStorageConnection", "kind", conn.Spec.ClusterRef.Kind, "cluster", conn.Spec.ClusterRef.Name, "address", conn.Spec.TargetStorageNode.Address)
	return ctrl.Result{}, nil
}

// handleClusterDeletion syncs the cluster (excluding the being-deleted connection)
// and then drops the finalizer. If the cluster is already gone, drops the finalizer immediately.
func (r *VMStorageConnectionReconciler) handleClusterDeletion(ctx context.Context, conn *kofv1beta1.VMStorageConnection, cluster storageCluster) error {
	if !controllerutil.ContainsFinalizer(conn, vmStorageConnectionFinalizer) {
		return nil
	}

	if cluster != nil {
		if err := r.syncCluster(ctx, cluster); err != nil {
			return fmt.Errorf("failed to sync %s on deletion: %w", cluster.clusterKind(), err)
		}
	}

	controllerutil.RemoveFinalizer(conn, vmStorageConnectionFinalizer)
	return r.Update(ctx, conn)
}

// buildStorageNodeConfig lists all active VMStorageConnections for the given cluster
// and rebuilds the storageNode ExtraArgs and Secrets from scratch.
func (r *VMStorageConnectionReconciler) buildStorageNodeConfig(ctx context.Context, clusterName, clusterKind string, existingArgs map[string]string) (map[string]string, []string, error) {
	connList := new(kofv1beta1.VMStorageConnectionList)
	if err := r.List(ctx, connList, client.MatchingLabels{
		labels.ClusterNameLabelKey: clusterName,
		labels.ClusterKindLabelKey: clusterKind,
	}); err != nil {
		return nil, nil, fmt.Errorf("failed to list VMStorageConnections: %w", err)
	}

	args := maps.Clone(existingArgs)
	if args == nil {
		args = make(map[string]string, len(storageNodeOwnedArgs))
	}

	for _, key := range storageNodeOwnedArgs {
		delete(args, key)
	}

	var (
		addresses     = make([]string, 0, len(connList.Items))
		secrets       = make([]string, 0, len(connList.Items))
		usernameFiles = make([]string, 0, len(connList.Items))
		passwordFiles = make([]string, 0, len(connList.Items))
		tlsEnabled    = make([]string, 0, len(connList.Items))
		tlsInsecure   = make([]string, 0, len(connList.Items))
	)

	for _, conn := range connList.Items {
		if !conn.DeletionTimestamp.IsZero() {
			continue
		}

		node := conn.Spec.TargetStorageNode
		if node.Secret.Name != "" {
			secrets = append(secrets, node.Secret.Name)
		}

		addresses = append(addresses, node.Address)
		usernameFiles = append(usernameFiles, secretKeyPath(node.Secret.Name, node.Secret.UsernameKey))
		passwordFiles = append(passwordFiles, secretKeyPath(node.Secret.Name, node.Secret.PasswordKey))
		tlsEnabled = append(tlsEnabled, strconv.FormatBool(node.TLSConfig.Enabled))
		tlsInsecure = append(tlsInsecure, strconv.FormatBool(node.TLSConfig.InsecureSkipVerify))
	}

	setArg(args, storageNodeArg, addresses)
	setArg(args, storageNodeUsernameFileArg, usernameFiles)
	setArg(args, storageNodePasswordFileArg, passwordFiles)
	setArg(args, storageNodeTLSArg, tlsEnabled)
	setArg(args, storageNodeTLSInsecureSkipVerify, tlsInsecure)

	return args, secrets, nil
}

// syncCluster rebuilds a cluster's storageNode ExtraArgs and Secrets from all active
// VMStorageConnections referencing it, preserving unmanaged ExtraArgs.
func (r *VMStorageConnectionReconciler) syncCluster(ctx context.Context, cluster storageCluster) error {
	updated := cluster.deepCopy()

	args, secrets, err := r.buildStorageNodeConfig(ctx, cluster.GetName(), cluster.clusterKind(), updated.storageExtraArgs())
	if err != nil {
		return err
	}

	if reflect.DeepEqual(updated.storageExtraArgs(), args) && reflect.DeepEqual(updated.storageSecrets(), secrets) {
		return nil
	}

	updated.applyStorageConfig(args, secrets)
	return r.Update(ctx, updated.object())
}

// fetchVTCluster returns the named VTCluster as a storageCluster, or (nil, nil) if not found.
func (r *VMStorageConnectionReconciler) fetchVTCluster(ctx context.Context, name, namespace string) (storageCluster, error) {
	obj := new(vmv1.VTCluster)
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &vtClusterAdapter{obj}, nil
}

// fetchVLCluster returns the named VLCluster as a storageCluster, or (nil, nil) if not found.
func (r *VMStorageConnectionReconciler) fetchVLCluster(ctx context.Context, name, namespace string) (storageCluster, error) {
	obj := new(vmv1.VLCluster)
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &vlClusterAdapter{obj}, nil
}

// secretKeyPath returns the mount path for secretName/key inside the pod.
// Returns "" when key is empty so the caller gets a positional placeholder.
func secretKeyPath(secretName, key string) string {
	if key == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", vmSecretMountPath, secretName, key)
}

// setArg sets key in args to the comma-joined values when at least one value is
// non-empty, preserving positional correspondence for mixed empty/non-empty slices.
// If all values are empty (or the slice is nil) the key is left unset.
func setArg(args map[string]string, key string, values []string) {
	for _, v := range values {
		if v != "" {
			args[key] = strings.Join(values, ",")
			return
		}
	}
}
