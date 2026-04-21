package k8s

import (
	"context"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetOwnerReference(owner client.Object, c client.Client) (*metav1.OwnerReference, error) {
	gvk := owner.GetObjectKind().GroupVersionKind()

	if gvk.Empty() {
		var err error
		gvk, err = c.GroupVersionKindFor(owner)
		if err != nil {
			return nil, err
		}
	}

	return &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	}, nil
}

func IsAdopted(cluster *kcmv1beta1.ClusterDeployment) bool {
	return strings.HasPrefix(cluster.Spec.Template, "adopted-")
}

func GetClusterDeploymentStub(name, namespace string) *kcmv1beta1.ClusterDeployment {
	return &kcmv1beta1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k0rdent.mirantis.com/v1beta1",
			Kind:       kcmv1beta1.ClusterDeploymentKind,
		},
	}
}

// EnsureCreated creates the given object if it does not exist.
// Returns (created=true, nil) if the object was created.
// Returns (created=false, nil) if the object already exists.
func EnsureCreated(
	ctx context.Context,
	c client.Client,
	object client.Object,
) (created bool, err error) {
	// `createOrUpdate` would need to read an old version and merge it with the new version
	// to avoid `metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`.
	// As we have immutable specs for now, we will use `EnsureCreated` instead.
	if err := c.Create(ctx, object); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IsResourceExist(ctx context.Context, c client.Client, obj client.Object, name, namespace string) (bool, error) {
	if err := c.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
