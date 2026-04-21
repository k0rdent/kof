package record

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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

// LogEvent creates a log line and an `Event` object from the same arguments.
//
// If you pass `nil` instead of `err`,
// then `log.Info` and `record.Event` are used,
// else `log.Error` and `record.Warn` are used.
//
// Example:
//
//	record.LogEvent(
//		ctx,
//		"ConfigMapUpdateFailed",
//		"Failed to update ConfigMap",
//		clusterDeployment,
//		err,
//		"configMapName", configMap.Name,
//	)
func LogEvent(
	ctx context.Context,
	reason, message string,
	obj runtime.Object,
	err error,
	keysAndValues ...any,
) {
	log := log.FromContext(ctx)
	recordFunc := Event

	if err == nil {
		log.Info(message, keysAndValues...)
	} else {
		log.Error(err, message, keysAndValues...)
		recordFunc = Warn
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
