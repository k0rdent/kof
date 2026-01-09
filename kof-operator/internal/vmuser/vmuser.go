package vmuser

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	"github.com/sethvargo/go-password/password"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateOptions defines parameters for creating VMUser resources.
type CreateOptions struct {
	// Name is the unique identifier for the VMUser resources (e.g., tenant ID or cluster name).
	Name string
	// OwnerReference links resources to an owner for garbage collection.
	OwnerReference *metav1.OwnerReference
	// Labels to apply to VMUser, Secret, and MultiClusterService resources.
	Labels map[string]string
	// MCSConfig defines MultiClusterService propagation settings. If nil, MCS will not be created.
	MCSConfig *MCSConfig
	// VMUserConfig contains additional configuration for the VMUserConfig resources. If nil, defaults are used.
	VMUserConfig *VMUserConfig
}

// MCSConfig defines MultiClusterService propagation configuration.
type MCSConfig struct {
	// ClusterSelector determines which clusters receive the VMUser credentials.
	ClusterSelector metav1.LabelSelector
}

type VMUserConfig struct {
	// ExtraLabels are additional labels to apply to VMUser resources.
	ExtraLabels map[string]string
	// ExtraFilters are additional filters to apply to VMUser target references.
	ExtraFilters map[string]string
}

const KofTenantLabel = "k0rdent.mirantis.com/kof-tenant-id"

const (
	usernameKey = "username"
	passwordKey = "password"
)

// Password generation parameters
const (
	passwordLength      = 32
	passwordNumDigits   = 8
	passwordNumSymbol   = 8
	passwordAllowUpper  = true
	passwordAllowRepeat = true
)

// Service URLs and Paths for VictoriaMetrics components
const (
	vlSelectURL = "http://kof-storage-victoria-logs-cluster-vlselect.kof.svc:9471"
	vlInsertURL = "http://kof-storage-victoria-logs-cluster-vlinsert.kof.svc:9481"
	vmSelectURL = "http://vmselect-cluster.kof.svc:8481"
	vmInsertURL = "http://vminsert-cluster.kof.svc:8480"
	vtSelectURL = "http://kof-storage-vt-cluster-vtselect.kof.svc:10471"
	vtInsertURL = "http://kof-storage-vt-cluster-vtinsert.kof.svc:10481"

	vlSelectPath = "/vls/.*"
	vlInsertPath = "/vli/.*"
	vmSelectPath = "/vm/select/.*"
	vmInsertPath = "/vm/insert/.*"
	vtSelectPath = "/vts/.*"
	vtInsertPath = "/vti/.*"
)

type Manager struct {
	client client.Client
}

func NewManager(c client.Client) *Manager {
	return &Manager{client: c}
}

func (m *Manager) Create(ctx context.Context, opts CreateOptions) error {
	log := log.FromContext(ctx)
	log.Info("Creating VMUser resources", "name", opts.Name)

	if opts.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := m.createSecret(ctx, opts.Name, opts.Labels, opts.OwnerReference); err != nil {
		return fmt.Errorf("failed to create secret for VMUser %s: %w", opts.Name, err)
	}

	if err := m.createVMUser(ctx, opts.Name, opts.Labels, opts.OwnerReference, opts.VMUserConfig); err != nil {
		return fmt.Errorf("failed to create VMUser %s: %w", opts.Name, err)
	}

	if opts.MCSConfig != nil {
		if err := m.createPropagationMCS(ctx, opts.Name, opts.Labels, opts.MCSConfig); err != nil {
			return fmt.Errorf("failed to create propagation MultiClusterService for VMUser %s: %w", opts.Name, err)
		}
	}

	log.Info("VMUser resources created successfully", "name", opts.Name)
	return nil
}

// Removes the MultiClusterService for the given VMUser.
// Secret and VMUser resources will be deleted automatically via garbage collection
// when their owner reference is removed.
func (m *Manager) Delete(ctx context.Context, name string) error {
	log := log.FromContext(ctx)
	log.Info("Deleting VMUser resources", "name", name)

	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := m.deletePropagationMCS(ctx, name); err != nil {
		return fmt.Errorf("failed to delete propagation MultiClusterService for VMUser %s: %w", name, err)
	}

	log.Info("VMUser resources deletion completed", "name", name)
	return nil
}

func (m *Manager) deletePropagationMCS(ctx context.Context, name string) error {
	log := log.FromContext(ctx)
	mcsName := buildMCSName(name)

	exists, err := m.isMultiClusterServiceExisting(ctx, mcsName)
	if err != nil {
		return fmt.Errorf("failed to check if MultiClusterService %s exists: %w", mcsName, err)
	}

	if !exists {
		log.Info("VMUser propagation MultiClusterService already deleted", "name", mcsName)
		return nil
	}

	log.Info("Deleting VMUser propagation MultiClusterService", "name", mcsName)
	if err := m.client.Delete(ctx, &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcsName,
			Namespace: k8s.KofNamespace,
		},
	}); err != nil {
		return fmt.Errorf("failed to delete MultiClusterService %s: %w", mcsName, err)
	}

	log.Info("MultiClusterService deleted successfully", "name", mcsName)
	return nil
}

func (m *Manager) createSecret(ctx context.Context, name string, labels map[string]string, ownerReference *metav1.OwnerReference) error {
	log := log.FromContext(ctx)
	secretName := buildSecretName(name)

	exists, err := m.isSecretExisting(ctx, secretName)
	if err != nil {
		return fmt.Errorf("failed to check if secret %s exists: %w", secretName, err)
	}

	if exists {
		log.Info("VMUser credentials secret already exists", "secretName", secretName)
		return nil
	}

	secret, err := buildCredsSecret(name, secretName, labels, ownerReference)
	if err != nil {
		return fmt.Errorf("failed to build credentials secret: %w", err)
	}

	log.Info("Creating VMUser credentials secret", "secretName", secretName)
	if err := m.client.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create credentials secret %s: %w", secretName, err)
	}

	log.Info("VMUser credentials secret created successfully", "secretName", secretName)
	return nil
}

func (m *Manager) createVMUser(ctx context.Context, name string, labels map[string]string, ownerReference *metav1.OwnerReference, vmUserConfig *VMUserConfig) error {
	log := log.FromContext(ctx)
	vmUserName := buildVMUserName(name)

	exists, err := m.isVMUserExisting(ctx, vmUserName)
	if err != nil {
		return fmt.Errorf("failed to check if VMUser %s exists: %w", vmUserName, err)
	}

	if exists {
		log.Info("VMUser already exists", "name", vmUserName)
		return nil
	}

	log.Info("Creating VMUser", "name", vmUserName)
	vmUser := buildVMUser(name, vmUserName, labels, ownerReference, vmUserConfig)
	if err := m.client.Create(ctx, vmUser); err != nil {
		return fmt.Errorf("failed to create VMUser %s: %w", vmUserName, err)
	}

	log.Info("VMUser created successfully", "name", vmUserName)
	return nil
}

func (m *Manager) createPropagationMCS(ctx context.Context, name string, labels map[string]string, mcsConfig *MCSConfig) error {
	log := log.FromContext(ctx)
	mcsName := buildMCSName(name)

	exists, err := m.isMultiClusterServiceExisting(ctx, mcsName)
	if err != nil {
		return fmt.Errorf("failed to check if MultiClusterService %s exists: %w", mcsName, err)
	}

	if exists {
		log.Info("MultiClusterService already exists", "name", mcsName)
		return nil
	}

	log.Info("Creating MultiClusterService", "name", mcsName)
	mcs, err := buildPropagationMCS(name, mcsName, labels, mcsConfig)
	if err != nil {
		return fmt.Errorf("failed to build MultiClusterService: %w", err)
	}

	if err := m.client.Create(ctx, mcs); err != nil {
		return fmt.Errorf("failed to create MultiClusterService %s: %w", mcsName, err)
	}

	log.Info("MultiClusterService created successfully", "name", mcsName)
	return nil
}

func (m *Manager) isMultiClusterServiceExisting(ctx context.Context, mcsName string) (bool, error) {
	return utils.IsResourceExist(ctx, m.client, &kcmv1beta1.MultiClusterService{}, mcsName, "")
}

func (m *Manager) isSecretExisting(ctx context.Context, secretName string) (bool, error) {
	return utils.IsResourceExist(ctx, m.client, &corev1.Secret{}, secretName, k8s.KofNamespace)
}

func (m *Manager) isVMUserExisting(ctx context.Context, name string) (bool, error) {
	return utils.IsResourceExist(ctx, m.client, &vmv1beta1.VMUser{}, name, k8s.KofNamespace)
}

func buildPropagationMCS(name string, mcsName string, labels map[string]string, mcsConfig *MCSConfig) (*kcmv1beta1.MultiClusterService, error) {
	rawConfig, err := json.Marshal(addoncontrollerv1beta1.Spec{
		TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
			{
				Identifier: "vmuser",
				Resource: corev1.ObjectReference{
					APIVersion: vmv1beta1.GroupVersion.String(),
					Kind:       "VMUser",
					Name:       buildVMUserName(name),
					Namespace:  k8s.KofNamespace,
				},
			},
			{
				Identifier: "secret",
				Resource: corev1.ObjectReference{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
					Name:       buildSecretName(name),
					Namespace:  k8s.KofNamespace,
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	mcsLabels := map[string]string{
		utils.ManagedByLabel:    utils.ManagedByValue,
		utils.KofGeneratedLabel: utils.True,
	}
	maps.Copy(mcsLabels, labels)

	return &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name:   mcsName,
			Labels: mcsLabels,
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: mcsConfig.ClusterSelector,
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      "vmuser-propagation",
						Template:  "kof-vmuser-propagation",
						Namespace: k8s.DefaultSystemNamespace,
					},
					{
						Name:      "secret-propagation",
						Template:  "kof-secret-propagation",
						Namespace: k8s.DefaultSystemNamespace,
					},
				},
				Provider: kcmv1beta1.StateManagementProviderConfig{
					Config: &v1.JSON{Raw: rawConfig},
				},
			},
		},
	}, nil
}

func buildCredsSecret(name, secretName string, labels map[string]string, ownerReference *metav1.OwnerReference) (*corev1.Secret, error) {
	pass, err := password.Generate(passwordLength, passwordNumDigits, passwordNumSymbol, passwordAllowUpper, passwordAllowRepeat)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: k8s.KofNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			usernameKey: []byte(name),
			passwordKey: []byte(pass),
		},
	}

	if ownerReference != nil {
		secret.OwnerReferences = []metav1.OwnerReference{*ownerReference}
	}

	return secret, nil
}

func buildVMUser(name, vmUserName string, labels map[string]string, ownerReference *metav1.OwnerReference, vmUserConfig *VMUserConfig) *vmv1beta1.VMUser {
	vmuser := &vmv1beta1.VMUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmUserName,
			Namespace: k8s.KofNamespace,
			Labels:    labels,
		},
		Spec: vmv1beta1.VMUserSpec{
			UserName: &name,
			PasswordRef: &corev1.SecretKeySelector{
				Key: passwordKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: buildSecretName(name),
				},
			},
			TargetRefs: buildTargetRefs(vmUserConfig),
		},
	}

	if ownerReference != nil {
		vmuser.OwnerReferences = []metav1.OwnerReference{*ownerReference}
	}

	return vmuser
}

// Returns the list of target references for VMUser.
func buildTargetRefs(vmUserConfig *VMUserConfig) []vmv1beta1.TargetRef {
	insertTargetPathSuffix := "?extra_label="
	selectTargetPathSuffix := "?extra_filters="

	if vmUserConfig != nil {
		if len(vmUserConfig.ExtraLabels) > 0 {
			insertTargetPathSuffix += encodeQueryParams(vmUserConfig.ExtraLabels)
		}

		if len(vmUserConfig.ExtraFilters) > 0 {
			selectTargetPathSuffix += encodeQueryParams(vmUserConfig.ExtraFilters)
		}
	}

	targets := []struct {
		path       string
		url        string
		pathSuffix string
	}{
		{vlSelectPath, vlSelectURL, selectTargetPathSuffix},
		{vlInsertPath, vlInsertURL, insertTargetPathSuffix},
		{vmSelectPath, vmSelectURL, selectTargetPathSuffix},
		{vmInsertPath, vmInsertURL, insertTargetPathSuffix},
		{vtSelectPath, vtSelectURL, selectTargetPathSuffix},
		{vtInsertPath, vtInsertURL, insertTargetPathSuffix},
	}

	refs := make([]vmv1beta1.TargetRef, 0, len(targets))
	for _, target := range targets {
		refs = append(refs, vmv1beta1.TargetRef{
			Paths: []string{target.path},
			Static: &vmv1beta1.StaticRef{
				URL: target.url,
			},
			URLMapCommon: vmv1beta1.URLMapCommon{
				DropSrcPathPrefixParts: ptr.To(1),
			},
			TargetPathSuffix: target.pathSuffix,
		})
	}

	return refs
}

func encodeQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	q := url.Values{}
	for key, value := range params {
		q.Add(key, value)
	}

	return q.Encode()
}

func buildSecretName(name string) string {
	return "kof-vmuser-creds-" + name
}

func buildMCSName(name string) string {
	return "vmuser-propagation-" + name
}

func buildVMUserName(name string) string {
	return "kof-vmuser-" + name
}
