package filewatcher

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// BaselineStore persists file-content hashes across pod restarts.
type BaselineStore interface {
	Load(ctx context.Context) (map[string]string, error)
	Save(ctx context.Context, hashes map[string]string) error
}

// secretBaselineStore stores file hashes in a Kubernetes Secret.
// Each file path is stored as its own data entry (key = hex-encoded path bytes,
// value = FNV-64a hex digest of the file content).
type secretBaselineStore struct {
	client     client.Client
	secretName string
	namespace  string
}

// NewSecretBaselineStore returns a BaselineStore backed by a Kubernetes Secret.
func NewSecretBaselineStore(c client.Client, secretName, namespace string) BaselineStore {
	return &secretBaselineStore{client: c, secretName: secretName, namespace: namespace}
}

// Load reads the stored hashes from the Secret. Returns an empty map when the
// Secret does not yet exist.
func (s *secretBaselineStore) Load(ctx context.Context) (map[string]string, error) {
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Name:      s.secretName,
		Namespace: s.namespace,
	}

	if err := s.client.Get(ctx, namespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("get baseline secret %s/%s: %w", s.namespace, s.secretName, err)
	}

	hashes := make(map[string]string, len(secret.Data))
	for key, val := range secret.Data {
		pathBytes, decErr := hex.DecodeString(key)
		if decErr != nil {
			continue // skip entries whose key is not a hex-encoded path
		}
		hashes[string(pathBytes)] = string(val)
	}
	return hashes, nil
}

// Save persists hashes to the Secret, creating it if it does not exist.
// Each file path is encoded as a hex string to form a valid k8s data key.
func (s *secretBaselineStore) Save(ctx context.Context, hashes map[string]string) error {
	data := make(map[string][]byte, len(hashes))
	for path, hash := range hashes {
		data[hex.EncodeToString([]byte(path))] = []byte(hash)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.secretName,
			Namespace: s.namespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, s.client, secret, func() error {
		secret.Data = data
		return nil
	}); err != nil {
		return fmt.Errorf("create or update baseline secret: %w", err)
	}

	return nil
}

// hashFile computes the FNV-64a hex digest of the file at path.
// The path comes from the operator's own configuration, not from user input.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("warning: failed to close file %s: %v", path, err)
		}
	}()

	h := fnv.New64a()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return fmt.Sprintf("%016x", h.Sum64()), nil
}

// hashTree computes hashes for all regular files reachable from root
// and stores them in out (keyed by absolute path). When root is a file itself,
// only that file is hashed.
func hashTree(root string, out map[string]string) error {
	info, err := os.Stat(root)
	if errors.Is(err, fs.ErrNotExist) {
		out[root] = ""
		return nil
	}

	if err != nil {
		return fmt.Errorf("stat %s: %w", root, err)
	}

	if !info.IsDir() {
		h, err := hashFile(root)
		if err != nil {
			return err
		}
		out[root] = h
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		h, hashErr := hashFile(path)
		if hashErr != nil {
			return nil
		}
		out[path] = h
		return nil
	})
}
