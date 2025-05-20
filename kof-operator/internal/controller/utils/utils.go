package utils

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func BoolPtr(value bool) *bool {
	// `*bool` fields may point to `true`, `false`, or be `nil`.
	// Direct `&true` is an error.
	return &value
}

func GetEventsAnnotations(obj runtime.Object) map[string]string {
	var generation string

	metaObj, ok := obj.(metav1.Object)
	if !ok {
		metaObj = &metav1.ObjectMeta{}
	}

	if metaObj.GetGeneration() == 0 {
		generation = "nil"
	} else {
		generation = strconv.Itoa(int(metaObj.GetGeneration()))
	}

	return map[string]string{
		"generation": generation,
	}
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

func HandleError(ctx context.Context, reason, message string, obj runtime.Object, err error, logKeysAndValues ...any) {
	log := log.FromContext(ctx)
	log.Error(err, message, logKeysAndValues...)

	formattedKeysValues := make([]string, 0, len(logKeysAndValues))
	for i, value := range logKeysAndValues {
		if i%2 == 1 {
			formattedKeysValues = append(formattedKeysValues, fmt.Sprintf("%v", value))
		} else {
			formattedKeysValues = append(formattedKeysValues, fmt.Sprintf(", %v=", value))
		}
	}

	record.Warn(
		obj,
		GetEventsAnnotations(obj),
		reason,
		fmt.Sprintf("%s: %v, %s", message, err, strings.Join(formattedKeysValues, "")),
	)
}
