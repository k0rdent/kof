package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const ManagedByLabel = "app.kubernetes.io/managed-by"
const ManagedByValue = "kof-operator"

func GetOwnerReference(owner client.Object, client client.Client) (metav1.OwnerReference, error) {
	gvk := owner.GetObjectKind().GroupVersionKind()

	if gvk.Empty() {
		var err error
		gvk, err = client.GroupVersionKindFor(owner)
		if err != nil {
			return metav1.OwnerReference{}, err
		}
	}

	return metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	}, nil
}

func (r *ClusterDeploymentReconciler) createIfNotExists(
	ctx context.Context,
	object client.Object,
	objectDescription string,
	details []any,
) error {
	log := log.FromContext(ctx)

	// `createOrUpdate` would need to read an old version and merge it with the new version
	// to avoid `metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`.
	// As we have immutable specs for now, we will use `createIfNotExists` instead.

	if err := r.Create(ctx, object); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Found existing "+objectDescription, details...)
			return nil
		}

		log.Error(err, "cannot create "+objectDescription, details...)
		return err
	}

	log.Info("Created "+objectDescription, details...)
	return nil
}

func BoolPtr(value bool) *bool {
	// `*bool` fields may point to `true`, `false`, or be `nil`.
	// Direct `&true` is an error.
	return &value
}
