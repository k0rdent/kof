package vmuser

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateOptions defines parameters for creating VMUser resources.
type CreateOptions struct {
	// Name is the unique identifier for the VMUser resources (e.g., cluster name).
	Name string
	// Namespace where VMUser resources will be created.
	Namespace string
	// OwnerReference links resources to an owner for garbage collection.
	OwnerReference *metav1.OwnerReference
	// Labels to apply to VMUser, Secret, and MultiClusterService resources.
	Labels map[string]string
	// MCSConfig defines MultiClusterService propagation settings. If nil, MCS will not be created.
	MCSConfig *MCSConfig
	// VMUserConfig contains additional configuration for the VMUserConfig resources.
	VMUserConfig *VMUserConfig
	// ClusterRef is a reference to the cluster associated with the VMUser.
	ClusterRef client.Object
}

// MCSConfig defines MultiClusterService propagation configuration.
type MCSConfig struct {
	// ClusterSelector determines which clusters receive the VMUser credentials.
	ClusterSelector metav1.LabelSelector
}

type VMUserConfig struct {
	// ExtraLabel is an additional label to apply to VMUser resources.
	ExtraLabel *ExtraLabel
	// ExtraFilters are additional filters to apply to VMUser target references.
	ExtraFilters map[string]string
}

type ExtraLabel struct {
	Key   string
	Value string
}

const KofTenantLabel = "k0rdent.mirantis.com/kof-tenant-id"

const (
	UsernameKey = "username"
	PasswordKey = "password"
)

// Password generation parameters
const (
	passwordLength = 16
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

func (m *Manager) Create(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	log.Info("Creating VMUser resources", "name", opts.Name)

	if opts.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := m.createSecret(ctx, opts); err != nil {
		return fmt.Errorf("failed to create secret for VMUser %s: %w", opts.Name, err)
	}

	if err := m.createOrUpdateVMUser(ctx, opts); err != nil {
		return fmt.Errorf("failed to create or update VMUser %s: %w", opts.Name, err)
	}

	if opts.MCSConfig != nil {
		if err := m.createPropagationMCS(ctx, opts); err != nil {
			return fmt.Errorf("failed to create propagation MultiClusterService for VMUser %s: %w", opts.Name, err)
		}
	}

	log.Info("VMUser resources created successfully", "name", opts.Name)
	return nil
}

// Removes the MultiClusterService for the given VMUser.
// Secret and VMUser resources will be deleted automatically via garbage collection
// when their owner reference is removed.
func (m *Manager) Delete(ctx context.Context, name, namespace string) error {
	log := log.FromContext(ctx)
	log.Info("Deleting VMUser resources", "name", name)

	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := m.deleteResource(ctx, &kcmv1beta1.MultiClusterService{}, BuildMCSName(name), namespace); err != nil {
		return fmt.Errorf("failed to delete MultiClusterService for VMUser %s: %w", BuildMCSName(name), err)
	}

	if err := m.deleteResource(ctx, &vmv1beta1.VMUser{}, BuildVMUserName(name), namespace); err != nil {
		return fmt.Errorf("failed to delete VMUser %s: %w", BuildVMUserName(name), err)
	}

	if err := m.deleteResource(ctx, &corev1.Secret{}, BuildSecretName(name), namespace); err != nil {
		return fmt.Errorf("failed to delete secret for VMUser %s: %w", BuildSecretName(name), err)
	}

	log.Info("VMUser resources deletion completed", "name", name)
	return nil
}

func (m *Manager) createSecret(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	secretName := BuildSecretName(opts.Name)

	secret, err := buildCredsSecret(opts)
	if err != nil {
		return fmt.Errorf("failed to build credentials secret: %w", err)
	}

	log.Info("Creating VMUser credentials secret", "name", secretName)
	created, err := utils.EnsureCreated(ctx, m.client, secret)
	if err != nil {
		return err
	}

	if !created {
		log.Info("VMUser credentials secret already exists", "name", secretName)
		return nil
	}

	log.Info("VMUser credentials secret created successfully", "name", secretName)
	return nil
}

func (m *Manager) createPropagationMCS(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	mcsName := BuildMCSName(opts.Name)

	mcs, err := buildPropagationMCS(opts)
	if err != nil {
		return fmt.Errorf("failed to build MultiClusterService: %w", err)
	}

	log.Info("Creating MultiClusterService", "name", mcsName)
	created, err := utils.EnsureCreated(ctx, m.client, mcs)
	if err != nil {
		return err
	}

	if !created {
		log.Info("MultiClusterService already exists", "name", mcsName)
		return nil
	}

	log.Info("MultiClusterService created successfully", "name", mcsName)
	return nil
}

func (m *Manager) createOrUpdateVMUser(ctx context.Context, opts *CreateOptions) error {
	vmUser, err := m.getVMUser(ctx, opts.Name, opts.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get VMUser: %v", err)
	}

	if vmUser == nil {
		if err := m.createVMUser(ctx, opts); err != nil {
			return fmt.Errorf("failed to create VMUser %s: %w", opts.Name, err)
		}
		return nil
	}

	newVMUser := buildVMUser(opts)
	if err := m.updateVMUser(ctx, opts, vmUser, newVMUser); err != nil {
		return fmt.Errorf("failed to update VMUser %s: %w", opts.Name, err)
	}
	return nil
}

func (m *Manager) updateVMUser(ctx context.Context, opts *CreateOptions, existingVMUser, newVMUser *vmv1beta1.VMUser) error {
	if reflect.DeepEqual(existingVMUser.Spec, newVMUser.Spec) {
		return nil
	}

	existingVMUser.Spec = newVMUser.Spec
	if err := m.client.Update(ctx, existingVMUser); err != nil {
		utils.LogEvent(
			ctx,
			"VMUserUpdateFailed",
			"Failed to update VMUser",
			opts.ClusterRef,
			err,
			"name", existingVMUser.Name,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"VMUserUpdateSuccessful",
		"VMUser updated successfully",
		opts.ClusterRef,
		nil,
		"name", existingVMUser.Name,
	)

	return nil
}

func (m *Manager) getVMUser(ctx context.Context, name, namespace string) (*vmv1beta1.VMUser, error) {
	vmUser := &vmv1beta1.VMUser{}
	err := m.client.Get(ctx, client.ObjectKey{Name: BuildVMUserName(name), Namespace: namespace}, vmUser)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return vmUser, nil
}

func (m *Manager) createVMUser(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	vmUserName := BuildVMUserName(opts.Name)
	vmUser := buildVMUser(opts)

	log.Info("Creating VMUser", "name", vmUserName)
	created, err := utils.EnsureCreated(ctx, m.client, vmUser)
	if err != nil {
		utils.LogEvent(
			ctx,
			"VMUserCreationFailed",
			"Failed to create VMUser",
			opts.ClusterRef,
			err,
			"name", vmUserName,
		)
		return err
	}

	if !created {
		log.Info("VMUser already exists", "name", vmUserName)
		return nil
	}

	utils.LogEvent(
		ctx,
		"VMUserCreated",
		"VMUser created successfully",
		opts.ClusterRef,
		nil,
		"name", vmUserName,
	)
	return nil
}

func (m *Manager) deleteResource(ctx context.Context, obj client.Object, name, namespace string) error {
	log := log.FromContext(ctx)
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	obj.SetName(name)
	obj.SetNamespace(namespace)

	log.Info(fmt.Sprintf("Deleting VMUser %s resource", kind), "name", name)
	if err := m.client.Delete(ctx, obj); err != nil {
		if errors.IsNotFound(err) {
			log.Info(fmt.Sprintf("VMUser %s resource already deleted", kind), "name", name)
			return nil
		}
		return err
	}

	log.Info(fmt.Sprintf("VMUser %s resource deleted successfully", kind), "name", name)
	return nil
}

// buildPropagationMCS constructs a MultiClusterService for propagating VMUser resources to managed clusters.
func buildPropagationMCS(opts *CreateOptions) (*kcmv1beta1.MultiClusterService, error) {
	providerConfig, err := json.Marshal(addoncontrollerv1beta1.Spec{
		TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
			{
				Identifier: "vmuser",
				Resource: corev1.ObjectReference{
					APIVersion: vmv1beta1.GroupVersion.String(),
					Kind:       "VMUser",
					Name:       BuildVMUserName(opts.Name),
					Namespace:  opts.Namespace,
				},
			},
			{
				Identifier: "secret",
				Resource: corev1.ObjectReference{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Secret",
					Name:       BuildSecretName(opts.Name),
					Namespace:  opts.Namespace,
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	labels := getMandatoryLabels()
	maps.Copy(labels, opts.Labels)

	return &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name:   BuildMCSName(opts.Name),
			Labels: labels,
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: opts.MCSConfig.ClusterSelector,
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
					Config: &v1.JSON{Raw: providerConfig},
				},
			},
		},
	}, nil
}

// buildCredsSecret constructs a Secret containing VMUser credentials with a generated password.
func buildCredsSecret(opts *CreateOptions) (*corev1.Secret, error) {
	pass, err := utils.GeneratePassword(passwordLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildSecretName(opts.Name),
			Namespace: opts.Namespace,
			Labels:    getMandatoryLabels(),
		},
		Data: map[string][]byte{
			UsernameKey: []byte(opts.Name),
			PasswordKey: []byte(pass),
		},
	}

	if opts.OwnerReference != nil {
		secret.OwnerReferences = []metav1.OwnerReference{*opts.OwnerReference}
	}

	return secret, nil
}

// buildVMUser constructs a VMUser resource with the specified configuration.
func buildVMUser(opts *CreateOptions) *vmv1beta1.VMUser {
	vmUser := &vmv1beta1.VMUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildVMUserName(opts.Name),
			Namespace: opts.Namespace,
			Labels:    getMandatoryLabels(),
		},
		Spec: vmv1beta1.VMUserSpec{
			UserName: &opts.Name,
			PasswordRef: &corev1.SecretKeySelector{
				Key: PasswordKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: BuildSecretName(opts.Name),
				},
			},
			TargetRefs: buildTargetRefs(opts.VMUserConfig),
		},
	}

	if opts.OwnerReference != nil {
		vmUser.OwnerReferences = []metav1.OwnerReference{*opts.OwnerReference}
	}

	return vmUser
}

// Returns the list of target references for VMUser.
func buildTargetRefs(vmUserConfig *VMUserConfig) []vmv1beta1.TargetRef {
	var insertTargetPathSuffix string
	var selectTargetPathSuffix string

	if vmUserConfig != nil {
		if vmUserConfig.ExtraLabel != nil {
			insertTargetPathSuffix = "?extra_label=" + formatVMLabelParam(vmUserConfig.ExtraLabel)
		}
		if len(vmUserConfig.ExtraFilters) > 0 {
			selectTargetPathSuffix = "?extra_filters[]=" + formatVMFilterParams(vmUserConfig.ExtraFilters)
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

func getMandatoryLabels() map[string]string {
	return map[string]string{
		utils.ManagedByLabel:    utils.ManagedByValue,
		utils.KofGeneratedLabel: utils.True,
	}
}

func formatVMLabelParam(extraLabel *ExtraLabel) string {
	return fmt.Sprintf("%s=%s", extraLabel.Key, extraLabel.Value)
}

// formatVMFilterParams formats filter parameters into VictoriaMetrics-specific format: {key1="value1",key2="value2"}
func formatVMFilterParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(params))
	for key, value := range params {
		pairs = append(pairs, fmt.Sprintf("%s=\"%s\"", key, value))
	}

	return fmt.Sprintf("{%s}", strings.Join(pairs, ","))
}

// BuildSecretName returns the secret name for VMUser credentials.
func BuildSecretName(name string) string {
	return "kof-vmuser-creds-" + name
}

// BuildMCSName returns the MultiClusterService name for VMUser propagation.
func BuildMCSName(name string) string {
	return "kof-vmuser-propagation-" + name
}

// BuildVMUserName returns the VMUser resource name.
func BuildVMUserName(name string) string {
	return "kof-vmuser-" + name
}
