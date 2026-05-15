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
	vtStorageConnectionFinalizer = "k0rdent.mirantis.com/vtstorageconnection"
	vtSecretMountPath            = "/etc/vm/secrets"

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

// VTStorageConnectionReconciler reconciles a VTStorageConnection object
type VTStorageConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vtstorageconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vtstorageconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kof.k0rdent.mirantis.com,resources=vtstorageconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.victoriametrics.com,resources=vtclusters,verbs=get;list;watch;update;patch

// Reconcile fetches the VTStorageConnection and updates the referenced VTCluster's
// spec.select to add or remove the target storage node.
func (r *VTStorageConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	conn := new(kofv1beta1.VTStorageConnection)
	if err := r.Get(ctx, req.NamespacedName, conn); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get VTStorageConnection: %w", err)
	}

	vtClusterNS := conn.Spec.VTClusterRef.Namespace
	if vtClusterNS == "" {
		vtClusterNS = conn.Namespace
	}

	vtCluster, err := r.fetchVTCluster(ctx, conn.Spec.VTClusterRef.Name, vtClusterNS)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get VTCluster %s/%s: %w", vtClusterNS, conn.Spec.VTClusterRef.Name, err)
	}

	if !conn.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.handleDeletion(ctx, conn, vtCluster)
	}

	if vtCluster == nil {
		return ctrl.Result{}, fmt.Errorf("VTCluster %s/%s not found", vtClusterNS, conn.Spec.VTClusterRef.Name)
	}

	if !controllerutil.ContainsFinalizer(conn, vtStorageConnectionFinalizer) {
		controllerutil.AddFinalizer(conn, vtStorageConnectionFinalizer)
		if conn.Labels == nil {
			conn.Labels = make(map[string]string)
		}
		// Index by VTCluster name so syncVTCluster can list all related connections efficiently.
		conn.Labels[labels.VtClusterNameLabelKey] = conn.Spec.VTClusterRef.Name
		if err := r.Update(ctx, conn); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	if err := r.syncVTCluster(ctx, vtCluster); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled VTStorageConnection", "vtCluster", conn.Spec.VTClusterRef.Name, "address", conn.Spec.TargetStorageNode.Address)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VTStorageConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kofv1beta1.VTStorageConnection{}).
		Named("vtstorageconnection").
		Complete(r)
}

// handleDeletion syncs the VTCluster (which will exclude the being-deleted connection
// because its DeletionTimestamp is set) and then drops the finalizer.
// If the VTCluster is already gone, the finalizer is dropped immediately.
func (r *VTStorageConnectionReconciler) handleDeletion(ctx context.Context, conn *kofv1beta1.VTStorageConnection, vtCluster *vmv1.VTCluster) error {
	if !controllerutil.ContainsFinalizer(conn, vtStorageConnectionFinalizer) {
		return nil
	}

	if vtCluster != nil {
		if err := r.syncVTCluster(ctx, vtCluster); err != nil {
			return fmt.Errorf("failed to sync VTCluster on deletion: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(conn, vtStorageConnectionFinalizer)
	return r.Update(ctx, conn)
}

// syncVTCluster rebuilds VTCluster.spec.select from all active (non-deleting)
// VTStorageConnections that reference this cluster, preserving any ExtraArgs
// not owned by this controller.
func (r *VTStorageConnectionReconciler) syncVTCluster(ctx context.Context, vtCluster *vmv1.VTCluster) error {
	connList := new(kofv1beta1.VTStorageConnectionList)
	if err := r.List(ctx, connList, client.MatchingLabels{labels.VtClusterNameLabelKey: vtCluster.Name}); err != nil {
		return fmt.Errorf("failed to list VTStorageConnections: %w", err)
	}

	updated := vtCluster.DeepCopy()

	// Preserve ExtraArgs not owned by this controller, then clear the owned keys
	// so they can be rebuilt cleanly below.
	args := maps.Clone(updated.Spec.Select.ExtraArgs)
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

		addresses = append(addresses, node.Address)
		if node.Secret.Name != "" {
			secrets = append(secrets, node.Secret.Name)
		}

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

	if reflect.DeepEqual(updated.Spec.Select.ExtraArgs, args) && reflect.DeepEqual(updated.Spec.Select.Secrets, secrets) {
		return nil
	}

	updated.Spec.Select.ExtraArgs = args
	updated.Spec.Select.Secrets = secrets

	return r.Update(ctx, updated)
}

// fetchVTCluster returns the named VTCluster, or (nil, nil) if it does not exist.
func (r *VTStorageConnectionReconciler) fetchVTCluster(ctx context.Context, name, namespace string) (*vmv1.VTCluster, error) {
	vtCluster := new(vmv1.VTCluster)
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, vtCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return vtCluster, nil
}

// secretKeyPath returns the mount path for secretName/key inside the VTCluster pod.
// Returns "" when key is empty so the caller gets a positional placeholder.
func secretKeyPath(secretName, key string) string {
	if key == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", vtSecretMountPath, secretName, key)
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
