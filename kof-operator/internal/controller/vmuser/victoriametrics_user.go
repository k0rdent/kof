package vmuser

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"github.com/k0rdent/kof/kof-operator/internal/crypto"
	"github.com/k0rdent/kof/kof-operator/internal/env"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	// ClusterName is the name of the cluster associated with the VMUser,
	// added as a label to allow efficient secret lookup by cluster.
	ClusterName string
	// ExtraLabels to apply to VMUser, Secret, and MultiClusterService resources.
	ExtraLabels map[string]string
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
	// DependsOn lists dependency identifiers for this MultiClusterService, such as other
	// MultiClusterService names, and is used to influence dependency/reconciliation ordering.
	DependsOn []string
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
	vlSelectURL      = "http://kof-storage-victoria-logs-cluster-vlselect.kof.svc:9471"
	vlInsertURL      = "http://kof-storage-victoria-logs-cluster-vlinsert.kof.svc:9481"
	vlAuditSelectURL = "http://vlselect-audit-logs.kof.svc:9471"
	vlAuditInsertURL = "http://vlinsert-audit-logs.kof.svc:9481"
	vmSelectURL      = "http://vmselect-cluster.kof.svc:8481"
	vmInsertURL      = "http://vminsert-cluster.kof.svc:8480"
	vtSelectURL      = "http://kof-storage-vt-cluster-vtselect.kof.svc:10471"
	vtInsertURL      = "http://kof-storage-vt-cluster-vtinsert.kof.svc:10481"

	vlSelectPath      = "/vls/.*"
	vlInsertPath      = "/vli/.*"
	vlAuditSelectPath = "/vlas/.*"
	vlAuditInsertPath = "/vlai/.*"
	vmSelectPath      = "/vm/select/.*"
	vmInsertPath      = "/vm/insert/.*"
	vtSelectPath      = "/vts/.*"
	vtInsertPath      = "/vti/.*"
)

const (
	EventTypeVMUserCreationOrUpdateFailed = "VMUserCreationOrUpdateFailed"
	EventTypeVMUserCreated                = "VMUserCreated"
	EventTypeVMUserUpdated                = "VMUserUpdated"

	EventTypeVMUserMCSCreationOrUpdateFailed = "VMUserMCSCreationOrUpdateFailed"
	EventTypeVMUserMCSCreated                = "VMUserMCSCreated"
	EventTypeVMUserMCSUpdated                = "VMUserMCSUpdated"
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
		if err := m.createOrUpdatePropagationMCS(ctx, opts); err != nil {
			return fmt.Errorf("failed to create propagation MultiClusterService for VMUser %s: %w", opts.Name, err)
		}
	}

	log.Info("VMUser resources created successfully", "name", opts.Name)
	return nil
}

// Removes VMUser resources for the given VMUser.
// Secret and VMUser resources may also be deleted automatically by garbage collection
// when their owner reference is removed, if it was set.
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
		return fmt.Errorf("failed to delete secret for VMUser %s/%s: %w", BuildSecretName(name), namespace, err)
	}

	if err := m.deleteResource(ctx, &corev1.Secret{}, BuildSecretName(name), k8s.KofNamespace); err != nil {
		return fmt.Errorf("failed to delete secret for VMUser %s/%s: %w", BuildSecretName(name), k8s.KofNamespace, err)
	}

	log.Info("VMUser resources deletion completed", "name", name)
	return nil
}

// Reconcile ensures that the VMUser and related resources are created or updated according to the provided options.
func (m *Manager) createSecret(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	secretName := BuildSecretName(opts.Name)
	log.Info("Creating or updating VMUser credentials secret", "name", secretName, "namespace", opts.Namespace)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: opts.Namespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, m.client, secret, func() error {
		// Only generate a new password when creating; preserve the existing one on update.
		if len(secret.Data[PasswordKey]) == 0 {
			pass, err := crypto.GeneratePassword(passwordLength)
			if err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}
			secret.Data = map[string][]byte{
				UsernameKey: []byte(opts.Name),
				PasswordKey: []byte(pass),
			}
		}

		secret.Labels = getLabels(opts.ExtraLabels)
		secret.OwnerReferences = getOwnerReferences(opts.OwnerReference)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create or update credentials secret: %w", err)
	}

	kofSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: k8s.KofNamespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, m.client, kofSecret, func() error {
		kofSecret.Labels = secret.Labels
		kofSecret.Data = secret.Data
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create or update secret in kof namespace: %w", err)
	}
	return nil
}

// CreateOrUpdateVMUser creates or updates the VMUser resource based on the provided options, including setting up target references with query arguments derived from VMUserConfig.
func (m *Manager) createOrUpdatePropagationMCS(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	log.Info("Creating or updating MultiClusterService for VMUser propagation", "name", BuildMCSName(opts.Name))

	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: BuildMCSName(opts.Name),
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, m.client, mcs, func() error {
		mcs.Labels = getLabels(opts.ExtraLabels)
		mcs.OwnerReferences = getOwnerReferences(opts.OwnerReference)
		mcs.Spec = kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: opts.MCSConfig.ClusterSelector,
			DependsOn:       opts.MCSConfig.DependsOn,
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      BuildVMUserName(opts.Name),
						Template:  env.GetPropagationTemplateName(),
						Namespace: opts.Namespace,
						Values:    "propagation:\n  enabled: true\n  data: |\n{{ removeField \"vmuser\" \"metadata.ownerReferences\" | nindent 14 }}\n",
						HelmOptions: &kcmv1beta1.ServiceHelmOptions{
							InstallOptions: &addoncontrollerv1beta1.HelmInstallOptions{
								TakeOwnership: true,
							},
							UpgradeOptions: &addoncontrollerv1beta1.HelmUpgradeOptions{
								TakeOwnership: true,
							},
						},
					},
					{
						Name:      BuildSecretName(opts.Name),
						Template:  env.GetPropagationTemplateName(),
						Namespace: opts.Namespace,
						Values:    "propagation:\n  enabled: true\n  data: |\n{{ removeField \"secret\" \"metadata.ownerReferences\" | nindent 14 }}\n",
						HelmOptions: &kcmv1beta1.ServiceHelmOptions{
							InstallOptions: &addoncontrollerv1beta1.HelmInstallOptions{
								TakeOwnership: true,
							},
							UpgradeOptions: &addoncontrollerv1beta1.HelmUpgradeOptions{
								TakeOwnership: true,
							},
						},
					},
				},
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
			},
		}
		return nil
	})

	logEvent := func(reason, message string, err error) {
		record.LogEvent(ctx, reason, message, opts.ClusterRef, err, "VMUser propagation MultiClusterService name", mcs.Name)
	}

	if err != nil {
		logEvent(EventTypeVMUserMCSCreationOrUpdateFailed, "Failed to create or update VMUser propagation MultiClusterService", err)
		return fmt.Errorf("failed to create or update VMUser propagation MultiClusterService: %w", err)
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logEvent(EventTypeVMUserMCSCreated, "VMUser propagation MultiClusterService created successfully", nil)
	case controllerutil.OperationResultUpdated:
		logEvent(EventTypeVMUserMCSUpdated, "VMUser propagation MultiClusterService updated successfully", nil)
	case controllerutil.OperationResultNone:
		log.Info("VMUser propagation MultiClusterService already up to date", "name", mcs.Name)
	}

	return nil
}

// createOrUpdateVMUser creates or updates the VMUser resource based on the provided options, including setting up target references with query arguments derived from VMUserConfig.
func (m *Manager) createOrUpdateVMUser(ctx context.Context, opts *CreateOptions) error {
	log := log.FromContext(ctx)
	log.Info("Creating or updating VMUser", "name", BuildVMUserName(opts.Name))

	vmUser := &vmv1beta1.VMUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildVMUserName(opts.Name),
			Namespace: opts.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, m.client, vmUser, func() error {
		vmUser.OwnerReferences = getOwnerReferences(opts.OwnerReference)
		vmUser.Labels = getLabels(opts.ExtraLabels)
		vmUser.Spec = vmv1beta1.VMUserSpec{
			UserName: &opts.Name,
			PasswordRef: &corev1.SecretKeySelector{
				Key: PasswordKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: BuildSecretName(opts.Name),
				},
			},
			TargetRefs: buildTargetRefs(opts.VMUserConfig),
		}
		return nil
	})

	logEvent := func(reason, message string, err error) {
		record.LogEvent(ctx, reason, message, opts.ClusterRef, err, "name", vmUser.Name)
	}

	if err != nil {
		logEvent(EventTypeVMUserCreationOrUpdateFailed, "Failed to create or update VMUser", err)
		return fmt.Errorf("failed to create or update VMUser: %w", err)
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logEvent(EventTypeVMUserCreated, "VMUser created successfully", nil)
	case controllerutil.OperationResultUpdated:
		logEvent(EventTypeVMUserUpdated, "VMUser updated successfully", nil)
	case controllerutil.OperationResultNone:
		log.Info("VMUser already up to date", "name", vmUser.Name)
	}

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

func getLabels(extraLabels map[string]string) map[string]string {
	labels := map[string]string{
		labels.ManagedByLabel: k8s.ManagedByValue,
	}
	maps.Copy(labels, extraLabels)
	return labels
}

func getOwnerReferences(ownerRef *metav1.OwnerReference) []metav1.OwnerReference {
	if ownerRef == nil {
		return nil
	}
	return []metav1.OwnerReference{*ownerRef}
}

// queryArgs holds the per-component query arguments derived from VMUserConfig.
type queryArgs struct {
	vlSelectArgs []vmv1beta1.QueryArg
	vlInsertArgs []vmv1beta1.QueryArg
	vmInsertArgs []vmv1beta1.QueryArg
	vmSelectArgs []vmv1beta1.QueryArg
	vtSelectArgs []vmv1beta1.QueryArg
	vtInsertArgs []vmv1beta1.QueryArg
}

// buildQueryArgs derives per-component query arguments from the VMUserConfig.
func buildQueryArgs(cfg *VMUserConfig) queryArgs {
	var args queryArgs
	if cfg == nil {
		return args
	}
	args.applyExtraLabel(cfg.ExtraLabel)
	args.applyExtraFilters(cfg.ExtraFilters)
	return args
}

// applyExtraLabel appends insert-side label-tagging query args for each storage backend.
func (a *queryArgs) applyExtraLabel(label *ExtraLabel) {
	if label == nil {
		return
	}
	labelParam := formatLabelParam(label.Key, label.Value)
	a.vmInsertArgs = append(a.vmInsertArgs, vmv1beta1.QueryArg{Name: "extra_label", Values: []string{labelParam}})
	a.vlInsertArgs = append(a.vlInsertArgs, vmv1beta1.QueryArg{Name: "extra_fields", Values: []string{labelParam}})
	a.vtInsertArgs = append(a.vtInsertArgs, vmv1beta1.QueryArg{Name: "extra_fields", Values: []string{formatLabelParam("resource_attr:"+label.Key, label.Value)}})
}

// applyExtraFilters appends select-side filter query args for each storage backend.
// VM filters are combined into a single MetricsQL selector to ensure correct AND semantics.
func (a *queryArgs) applyExtraFilters(filters map[string]string) {
	if len(filters) == 0 {
		return
	}

	vlValues := make([]string, 0, len(filters))
	vmValues := make([]string, 0, len(filters))
	vtValues := make([]string, 0, len(filters))

	for key, value := range filters {
		vlValues = append(vlValues, formatFilterParams(key, value))
		vtValues = append(vtValues, formatFilterParams("resource_attr:"+key, value))
		vmValues = append(vmValues, fmt.Sprintf(`%s="%s"`, key, value))
	}

	sort.Strings(vlValues)
	sort.Strings(vtValues)
	sort.Strings(vmValues)

	a.vlSelectArgs = append(a.vlSelectArgs, vmv1beta1.QueryArg{Name: "extra_filters", Values: vlValues})
	a.vtSelectArgs = append(a.vtSelectArgs, vmv1beta1.QueryArg{Name: "extra_filters", Values: vtValues})
	a.vmSelectArgs = append(a.vmSelectArgs, vmv1beta1.QueryArg{Name: "extra_filters", Values: []string{"{" + strings.Join(vmValues, ",") + "}"}})
}

// buildTargetRefs returns the list of target references for VMUser.
func buildTargetRefs(vmUserConfig *VMUserConfig) []vmv1beta1.TargetRef {
	a := buildQueryArgs(vmUserConfig)

	targets := []struct {
		path string
		url  string
		args []vmv1beta1.QueryArg
	}{
		{path: vlAuditSelectPath, url: vlAuditSelectURL, args: a.vlSelectArgs},
		{path: vlAuditInsertPath, url: vlAuditInsertURL, args: a.vlInsertArgs},
		{path: vlSelectPath, url: vlSelectURL, args: a.vlSelectArgs},
		{path: vlInsertPath, url: vlInsertURL, args: a.vlInsertArgs},
		{path: vmSelectPath, url: vmSelectURL, args: a.vmSelectArgs},
		{path: vmInsertPath, url: vmInsertURL, args: a.vmInsertArgs},
		{path: vtSelectPath, url: vtSelectURL, args: a.vtSelectArgs},
		{path: vtInsertPath, url: vtInsertURL, args: a.vtInsertArgs},
	}

	refs := make([]vmv1beta1.TargetRef, 0, len(targets))
	for _, t := range targets {
		refs = append(refs, vmv1beta1.TargetRef{
			Paths:  []string{t.path},
			Static: &vmv1beta1.StaticRef{URL: t.url},
			URLMapCommon: vmv1beta1.URLMapCommon{
				DropSrcPathPrefixParts: ptr.To(1),
			},
			QueryArgs: t.args,
		})
	}
	return refs
}

func formatLabelParam(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatFilterParams(key, value string) string {
	return fmt.Sprintf("\"%s\":=\"%s\"", key, value)
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
