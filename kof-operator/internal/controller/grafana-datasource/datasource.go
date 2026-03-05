package datasource

import (
	"context"

	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GrafanaDatasource struct {
	config Config
	ctx    context.Context
	client client.Client
}

type Option func(*Config)

func New(ctx context.Context, k8sClient client.Client, clusterName, clusterNamespace string, opts ...Option) *GrafanaDatasource {
	config := Config{
		ClusterName:      clusterName,
		ClusterNamespace: clusterNamespace,
		Access:           AccessProxy,
		ResyncPeriod:     DefaultResyncPeriod,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &GrafanaDatasource{
		config: config,
		ctx:    ctx,
		client: k8sClient,
	}
}

func WithType(dsType string) Option {
	return func(c *Config) {
		c.Type = dsType
	}
}
func WithCategory(category string) Option {
	return func(c *Config) {
		c.Category = category
	}
}

func WithURL(url string) Option {
	return func(c *Config) {
		c.URL = url
	}
}

func WithAccess(access string) Option {
	return func(c *Config) {
		c.Access = access
	}
}

func WithOwnerReference(ref metav1.OwnerReference) Option {
	return func(c *Config) {
		c.OwnerReference = ref
	}
}

func WithResyncPeriod(period metav1.Duration) Option {
	return func(c *Config) {
		c.ResyncPeriod = period
	}
}

func WithJSONData(data []byte) Option {
	return func(c *Config) {
		c.JSONData = data
	}
}

func WithSecureJSONData(data []byte) Option {
	return func(c *Config) {
		c.SecureJSONData = data
	}
}

func WithValuesFrom(valuesFrom []grafanav1beta1.ValueFrom) Option {
	return func(c *Config) {
		c.ValuesFrom = valuesFrom
	}
}

func WithDefault(isDefault bool) Option {
	return func(c *Config) {
		c.IsDefault = isDefault
	}
}

func WithBasicAuth(secretName, usernameKey, passwordKey string) Option {
	return func(c *Config) {
		c.BasicAuth = true
		c.BasicAuthUser = "${username}"
		c.SecureJSONData = []byte(`{"basicAuthPassword": "${password}"}`)
		c.ValuesFrom = BuildBasicAuthValuesFrom(secretName, usernameKey, passwordKey)
	}
}

func (d *GrafanaDatasource) Create() error {
	datasource := d.buildDatasource()
	if err := d.client.Create(d.ctx, datasource); err != nil {
		d.errorLogEvent("GrafanaDatasourceCreationFailed", "Failed to create GrafanaDatasource", err)
		return err
	}
	d.infoLogEvent("GrafanaDatasourceCreated", "GrafanaDatasource is successfully created")
	return nil
}

func (d *GrafanaDatasource) Get() (*grafanav1beta1.GrafanaDatasource, error) {
	datasource := new(grafanav1beta1.GrafanaDatasource)
	if err := d.client.Get(d.ctx, d.namespacedName(), datasource); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return datasource, nil
}

func (d *GrafanaDatasource) Update(existing *grafanav1beta1.GrafanaDatasource) error {
	if !d.needsUpdate(existing) {
		return nil
	}

	existing.Spec.Datasource.URL = d.config.URL
	existing.Spec.Datasource.JSONData = d.config.JSONData
	if err := d.client.Update(d.ctx, existing); err != nil {
		d.errorLogEvent("GrafanaDatasourceUpdateFailed", "Failed to update GrafanaDatasource", err)
		return err
	}

	d.infoLogEvent("GrafanaDatasourceUpdated", "GrafanaDatasource is successfully updated")
	return nil
}

func (d *GrafanaDatasource) GetName() string {
	return d.config.ClusterName + "-" + d.config.Category
}

func (d *GrafanaDatasource) namespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      d.GetName(),
		Namespace: d.config.ClusterNamespace,
	}
}

func (d *GrafanaDatasource) needsUpdate(existing *grafanav1beta1.GrafanaDatasource) bool {
	if existing.Spec.Datasource == nil {
		return true
	}
	return d.config.URL != existing.Spec.Datasource.URL ||
		string(d.config.JSONData) != string(existing.Spec.Datasource.JSONData)
}

func (d *GrafanaDatasource) buildDatasource() *grafanav1beta1.GrafanaDatasource {
	return &grafanav1beta1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            d.GetName(),
			Namespace:       d.config.ClusterNamespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{d.config.OwnerReference},
		},
		Spec: grafanav1beta1.GrafanaDatasourceSpec{
			GrafanaCommonSpec: grafanav1beta1.GrafanaCommonSpec{
				AllowCrossNamespaceImport: true,
				InstanceSelector: &metav1.LabelSelector{
					MatchLabels: DefaultGrafanaInstanceSelector,
				},
				ResyncPeriod: d.config.ResyncPeriod,
			},
			Datasource: &grafanav1beta1.GrafanaDatasourceInternal{
				Name:           d.GetName(),
				Type:           d.config.Type,
				Access:         d.config.Access,
				URL:            d.config.URL,
				IsDefault:      &d.config.IsDefault,
				BasicAuth:      &d.config.BasicAuth,
				BasicAuthUser:  d.config.BasicAuthUser,
				JSONData:       d.config.JSONData,
				SecureJSONData: d.config.SecureJSONData,
			},
			ValuesFrom: d.config.ValuesFrom,
		},
	}
}

func (d *GrafanaDatasource) ownerReferenceObject() runtime.Object {
	obj := new(unstructured.Unstructured)
	obj.SetName(d.config.OwnerReference.Name)
	obj.SetNamespace(d.config.ClusterNamespace)
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: d.config.OwnerReference.APIVersion,
		Kind:    d.config.OwnerReference.Kind,
	})
	return obj
}

func (d *GrafanaDatasource) errorLogEvent(eventReason, message string, err error) {
	utils.LogEvent(
		d.ctx,
		eventReason,
		message,
		d.ownerReferenceObject(),
		err,
		"grafanaDatasourceName", d.GetName(),
	)
}

func (d *GrafanaDatasource) infoLogEvent(eventReason, message string) {
	utils.LogEvent(
		d.ctx,
		eventReason,
		message,
		d.ownerReferenceObject(),
		nil,
		"grafanaDatasourceName", d.GetName(),
	)
}
