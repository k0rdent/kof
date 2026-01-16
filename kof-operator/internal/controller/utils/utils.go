package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"hash/fnv"
	"os"
	"strconv"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const ManagedByLabel = "app.kubernetes.io/managed-by"
const ManagedByValue = "kof-operator"
const KofGeneratedLabel = "k0rdent.mirantis.com/kof-generated"
const True = "true"

func GetOwnerReference(owner client.Object, client client.Client) (*metav1.OwnerReference, error) {
	gvk := owner.GetObjectKind().GroupVersionKind()

	if gvk.Empty() {
		var err error
		gvk, err = client.GroupVersionKindFor(owner)
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

func GetReleaseNamespace() (string, error) {
	namespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return "", fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}
	if len(namespace) == 0 {
		return "", fmt.Errorf("RELEASE_NAMESPACE env var is set but empty")
	}
	return namespace, nil
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

// EnsureCreated creates the given object if it does not exist.
// Returns (created=true, nil) if the object was created.
// Returns (created=false, nil) if the object already exists.
func EnsureCreated(
	ctx context.Context,
	client client.Client,
	object client.Object,
) (created bool, err error) {
	// `createOrUpdate` would need to read an old version and merge it with the new version
	// to avoid `metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`.
	// As we have immutable specs for now, we will use `EnsureCreated` instead.
	if err := client.Create(ctx, object); err != nil {
		if errors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IsResourceExist(ctx context.Context, client client.Client, obj client.Object, name, namespace string) (bool, error) {
	if err := client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, obj); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

func IsEmptyString(s string) bool {
	return s == ""
}

func MergeConfig(dst, src any) error {
	srcBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(srcBytes, dst); err != nil {
		return err
	}
	return nil
}

func GetNameHash(prefix, name string) string {
	h := fnv.New32a()
	h.Write([]byte(name))
	return fmt.Sprintf("%s-%x", prefix, h.Sum32())
}

// GetHelmAdler32Name formats `<prefix>-<checksum>` and matches Helm's built-in adler32 helper.
// Useful when we need deterministic names in templates but only adler32 is available.
func GetHelmAdler32Name(prefix, name string) string {
	hash := GetHelmAdler32Checksum(name)
	return fmt.Sprintf("%s-%s", prefix, hash)
}

// GetHelmAdler32Checksum returns the decimal adler32 checksum of the provided name.
// Matches Helm's `adler32sum` helper so that templates stay consistent.
func GetHelmAdler32Checksum(name string) string {
	return fmt.Sprintf("%d", adler32.Checksum([]byte(name)))
}

func GrafanaEnabled() bool {
	return os.Getenv("KOF_GRAFANA_ENABLED") == "true"
}

func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	// Each byte = 2 hex chars
	bytesNeeded := (length + 1) / 2
	b := make([]byte, bytesNeeded)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:length], nil
}
