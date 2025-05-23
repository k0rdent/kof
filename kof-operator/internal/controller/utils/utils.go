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
const KofGeneratedLabel = "k0rdent.mirantis.com/kof-generated"

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

// Creates a log line and an `Event` object from the same arguments.
//
// If you pass `nil` instead of `err`,
// then `log.Info` and `record.Event` are used,
// else `log.Error` and `record.Warn` are used.
//
// Example:
//
//	utils.LogEvent(
//		ctx,
//		"ConfigMapUpdateFailed",
//		"Failed to update ConfigMap",
//		clusterDeployment,
//		err,
//		"configMapName", configMap.Name,
//		"key2", "value2",
//		"key3", "value3",
//	)
func LogEvent(
	ctx context.Context,
	reason, message string,
	obj runtime.Object,
	err error,
	keysAndValues ...any,
) {
	log := log.FromContext(ctx)
	recordFunc := record.Event

	if err == nil {
		log.Info(message, keysAndValues...)
	} else {
		log.Error(err, message, keysAndValues...)
		recordFunc = record.Warn
		keysAndValues = append([]any{"err", err}, keysAndValues...)
	}

	parts := make([]string, 0, len(keysAndValues))
	for i, keyOrValue := range keysAndValues {
		if i%2 == 0 { // key
			parts = append(parts, fmt.Sprintf(", %v=", keyOrValue))
		} else { // value
			parts = append(parts, fmt.Sprintf("%#v", keyOrValue))
		}
	}

	recordFunc(
		obj,
		GetEventsAnnotations(obj),
		reason,
		message+strings.Join(parts, ""),
	)
}
