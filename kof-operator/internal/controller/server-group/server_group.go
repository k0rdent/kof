package servergroup

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/controller/vmuser"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	ClusterName           string
	ClusterNamespace      string
	CredentialsSecretName string
	ConfigName            string
	Type                  Type
	Scheme                string
	PathPrefix            string
	DialTimeout           metav1.Duration
	TlsInsecureSkipVerify bool
	Targets               []string
	OwnerReference        metav1.OwnerReference
}

type Type string

type Credentials struct {
	Username string
	Password string
}

const (
	ConfigSecretNameLabel = "k0rdent.mirantis.com/secret-name"
	ServerGroupTypeLabel  = "k0rdent.mirantis.com/server-group-type"
)

var (
	DefaultDialTimeout = metav1.Duration{Duration: 5 * time.Second}
)

const (
	TypeMetrics Type = "metrics"
	TypeLogs    Type = "logs"
)

type ServerGroup struct {
	client client.Client
	config *Config
}

type Option func(*Config)

func NewServerGroup(client client.Client, clusterName, clusterNamespace string, ownerReference metav1.OwnerReference, opts ...Option) *ServerGroup {
	cfg := &Config{
		ClusterName:      clusterName,
		ClusterNamespace: clusterNamespace,
		OwnerReference:   ownerReference,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if utils.IsEmptyString(cfg.Scheme) {
		cfg.Scheme = "http"
	}

	if cfg.DialTimeout == (metav1.Duration{}) {
		cfg.DialTimeout = DefaultDialTimeout
	}

	return &ServerGroup{
		config: cfg,
		client: client,
	}
}

func WithCredentials(name string) Option {
	return func(c *Config) {
		c.CredentialsSecretName = name
	}
}

func WithConfigName(name string) Option {
	return func(c *Config) {
		c.ConfigName = name
	}
}

func WithScheme(scheme string) Option {
	return func(c *Config) {
		c.Scheme = scheme
	}
}

func WithPathPrefix(pathPrefix string) Option {
	return func(c *Config) {
		c.PathPrefix = pathPrefix
	}
}

func WithDialTimeout(dialTimeout metav1.Duration) Option {
	return func(c *Config) {
		c.DialTimeout = dialTimeout
	}
}

func WithTlsInsecureSkipVerify(tlsInsecureSkipVerify bool) Option {
	return func(c *Config) {
		c.TlsInsecureSkipVerify = tlsInsecureSkipVerify
	}
}

func WithTarget(target string) Option {
	return func(c *Config) {
		c.Targets = append(c.Targets, target)
	}
}

func WithTargets(targets []string) Option {
	return func(c *Config) {
		c.Targets = append(c.Targets, targets...)
	}
}

func WithType(serverGroupType Type) Option {
	return func(c *Config) {
		c.Type = serverGroupType
	}
}

func (s *ServerGroup) CreateOrUpdate(ctx context.Context) error {
	serverGroup, err := s.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get server group: %v", err)
	}

	if serverGroup == nil {
		if err := s.Create(ctx); err != nil {
			return fmt.Errorf("failed to create server group: %v", err)
		}
		return nil
	}

	if err := s.Update(ctx, serverGroup); err != nil {
		return fmt.Errorf("failed to update server group: %v", err)
	}
	return nil
}

func (s *ServerGroup) Create(ctx context.Context) error {
	if err := s.client.Create(ctx, &kofv1beta1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.GetName(),
			Namespace: s.config.ClusterNamespace,
			Labels: map[string]string{
				utils.ManagedByLabel:  utils.ManagedByValue,
				ConfigSecretNameLabel: s.config.ConfigName,
				ServerGroupTypeLabel:  string(s.config.Type),
			},
			OwnerReferences: []metav1.OwnerReference{s.config.OwnerReference},
		},
		Spec: kofv1beta1.PromxyServerGroupSpec{
			ClusterName: s.config.ClusterName,
			Scheme:      s.config.Scheme,
			Targets:     s.config.Targets,
			PathPrefix:  s.config.PathPrefix,
			HttpClient: kofv1beta1.HTTPClientConfig{
				DialTimeout: s.config.DialTimeout,
				TLSConfig: kofv1beta1.TLSConfig{
					InsecureSkipVerify: s.config.TlsInsecureSkipVerify,
				},
				BasicAuth: kofv1beta1.BasicAuth{
					CredentialsSecretName: s.config.CredentialsSecretName,
					UsernameKey:           vmuser.UsernameKey,
					PasswordKey:           vmuser.PasswordKey,
				},
			},
		},
	}); err != nil {
		s.logEvent(ctx, "ServerGroupCreationFailed", "Failed to create PromxyServerGroup", err)
		return err
	}

	s.logEvent(ctx, "ServerGroupCreatedSuccessfully", "ServerGroup is successfully created", nil)
	return nil
}

func (s *ServerGroup) Update(ctx context.Context, sg *kofv1beta1.PromxyServerGroup) error {
	var needsUpdate bool

	sort.Strings(s.config.Targets)
	sort.Strings(sg.Spec.Targets)

	if !slices.Equal(s.config.Targets, sg.Spec.Targets) {
		sg.Spec.Targets = s.config.Targets
		needsUpdate = true
	}

	if s.config.Scheme != sg.Spec.Scheme {
		sg.Spec.Scheme = s.config.Scheme
		needsUpdate = true
	}

	if s.config.PathPrefix != sg.Spec.PathPrefix {
		sg.Spec.PathPrefix = s.config.PathPrefix
		needsUpdate = true
	}

	if s.config.DialTimeout != sg.Spec.HttpClient.DialTimeout {
		sg.Spec.HttpClient.DialTimeout = s.config.DialTimeout
		needsUpdate = true
	}

	if s.config.TlsInsecureSkipVerify != sg.Spec.HttpClient.TLSConfig.InsecureSkipVerify {
		sg.Spec.HttpClient.TLSConfig.InsecureSkipVerify = s.config.TlsInsecureSkipVerify
		needsUpdate = true
	}

	if s.config.CredentialsSecretName != sg.Spec.HttpClient.BasicAuth.CredentialsSecretName {
		sg.Spec.HttpClient.BasicAuth.CredentialsSecretName = s.config.CredentialsSecretName
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	if err := s.client.Update(ctx, sg); err != nil {
		s.logEvent(ctx, "ServerGroupUpdateFailed", "Failed to update PromxyServerGroup", err)
		return err
	}

	s.logEvent(ctx, "ServerGroupUpdatedSuccessfully", "ServerGroup is successfully updated", nil)
	return nil
}

func (s *ServerGroup) Get(ctx context.Context) (*kofv1beta1.PromxyServerGroup, error) {
	sg := new(kofv1beta1.PromxyServerGroup)
	if err := s.client.Get(ctx, types.NamespacedName{
		Name:      s.GetName(),
		Namespace: s.config.ClusterNamespace,
	}, sg); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return sg, nil
}

func (s *ServerGroup) GetName() string {
	return fmt.Sprintf("%s-%s", s.config.ClusterName, s.config.Type)
}

func (s *ServerGroup) ownerReferenceObject() runtime.Object {
	obj := new(unstructured.Unstructured)
	obj.SetName(s.config.OwnerReference.Name)
	obj.SetNamespace(s.config.ClusterNamespace)
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: s.config.OwnerReference.APIVersion,
		Kind:    s.config.OwnerReference.Kind,
	})
	return obj
}

func (s *ServerGroup) logEvent(ctx context.Context, eventReason, message string, err error) {
	utils.LogEvent(
		ctx,
		eventReason,
		message,
		s.ownerReferenceObject(),
		err,
		"serverGroupName", s.GetName(),
		"clusterName", s.config.ClusterName,
		"clusterNamespace", s.config.ClusterNamespace,
	)
}
