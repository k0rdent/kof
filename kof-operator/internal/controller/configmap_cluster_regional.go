package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"github.com/k0rdent/kof/kof-operator/internal/controller/vmuser"
	"github.com/k0rdent/kof/kof-operator/internal/env"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	"github.com/k0rdent/kof/kof-operator/internal/names"
	"github.com/k0rdent/kof/kof-operator/internal/strutil"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const httpsScheme = "https"

type MetricsData struct {
	Endpoint string
	Target   string
	Port     string
	*url.URL
}

type RegionalClusterConfigMap struct {
	clusterName        string
	clusterNamespace   string
	releaseNamespace   string
	ctx                context.Context
	client             client.Client
	configMap          *corev1.ConfigMap
	ownerReference     *metav1.OwnerReference
	configData         *ConfigData
	VMUserManager      *vmuser.Manager
	isKcmRegionCluster bool
}

func NewRegionalClusterConfigMap(ctx context.Context, cm *corev1.ConfigMap, client client.Client) (*RegionalClusterConfigMap, error) {
	var ownerReference *metav1.OwnerReference
	var err error

	configMapData, err := NewConfigDataFromConfigMap(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configmap data: %v", err)
	}

	clusterName := configMapData.RegionalClusterName
	clusterNamespace := configMapData.RegionalClusterNamespace

	ownerReference, err = k8s.GetOwnerReference(cm, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner reference: %v", err)
	}

	releaseNamespace, err := env.GetReleaseNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get release namespace: %v", err)
	}

	isKcmRegionCluster := false
	if !isRegionlessConfigMap(cm) {
		isKcmRegionCluster, err = k8s.IsClusterKcmRegion(ctx, client, clusterName, clusterNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed to determine if cluster is KCM region cluster: %v", err)
		}
	}

	return &RegionalClusterConfigMap{
		clusterName:        clusterName,
		clusterNamespace:   clusterNamespace,
		releaseNamespace:   releaseNamespace,
		ctx:                ctx,
		client:             client,
		configMap:          cm,
		ownerReference:     ownerReference,
		configData:         configMapData,
		isKcmRegionCluster: isKcmRegionCluster,
		VMUserManager:      vmuser.NewManager(client),
	}, nil
}

func (c *RegionalClusterConfigMap) Reconcile() error {
	if err := c.CreateVmRulesConfigMap(); err != nil {
		return fmt.Errorf("failed to create vm rules ConfigMap: %v", err)
	}

	if err := c.CreateMcsForVmRulesPropagation(); err != nil {
		return fmt.Errorf("failed to create MCS for VM rules propagation: %v", err)
	}

	if err := c.UpdateChildConfigMap(); err != nil {
		return fmt.Errorf("failed to update child's ConfigMap: %v", err)
	}

	if err := c.CreateVMUser(); err != nil {
		return fmt.Errorf("failed to create VMUser: %v", err)
	}

	if err := c.CreateOrUpdatePromxyServerGroup(); err != nil {
		return fmt.Errorf("failed to create or update Promxy ServerGroup: %v", err)
	}

	if err := c.CreateOrUpdateTracesStorageConnection(); err != nil {
		return fmt.Errorf("failed to create or update TracesStorageConnection: %v", err)
	}

	if err := c.CreateOrUpdateLogsStorageConnection(); err != nil {
		return fmt.Errorf("failed to create or update LogsStorageConnection: %v", err)
	}

	if err := c.CreateOrUpdateAuditLogsStorageConnection(); err != nil {
		return fmt.Errorf("failed to create or update AuditLogsStorageConnection: %v", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) CreateVMUser() error {
	return c.VMUserManager.Create(c.ctx, &vmuser.CreateOptions{
		Name:           GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace),
		Namespace:      c.clusterNamespace,
		ClusterRef:     c.configMap,
		OwnerReference: c.ownerReference,
		ExtraLabels: map[string]string{
			labels.ClusterNameLabel: c.clusterName,
		},
		MCSConfig: &vmuser.MCSConfig{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					labels.ClusterNameLabel: c.clusterName,
				},
			},
			DependsOn: []string{c.GetRegionalMCSName()},
		},
	})
}

func (c *RegionalClusterConfigMap) CreateVmRulesConfigMap() error {
	log := log.FromContext(c.ctx)

	vmRulesConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       c.clusterNamespace,
			Name:            vmRulesConfigMapNamePrefix + c.clusterName,
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
			Labels: map[string]string{
				KofRecordVMRulesClusterNameLabel: c.clusterName,
				labels.ManagedByLabel:            k8s.ManagedByValue,
				labels.KofGeneratedLabel:         strutil.True,
			},
		},
	}

	created, err := k8s.EnsureCreated(c.ctx, c.client, vmRulesConfigMap)
	if err != nil {
		return fmt.Errorf("failed to create VMRulesConfigMap: %v", err)
	}

	if !created {
		log.Info("VMRulesConfigMap already created", "configMapName", vmRulesConfigMap.Name)
	}

	log.Info("VMRulesConfigMap created successfully", "configMapName", vmRulesConfigMap.Name)
	return err
}

// Function copies VM rules configMap to region cluster using MultiClusterService.
// TODO: Remove this function once KCM implements automatic copying of the required resources to region clusters.
func (c *RegionalClusterConfigMap) CreateMcsForVmRulesPropagation() error {
	if !c.isKcmRegionCluster {
		return nil
	}

	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetVmRulesMcsPropagationName(c.configMap.Name),
			Labels: map[string]string{
				labels.ManagedByLabel: k8s.ManagedByValue,
				"cluster-name":        c.clusterName,
				"cluster-namespace":   c.clusterNamespace,
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					labels.KofKcmRegionLabel: strutil.True,
				},
			},
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      names.FNVName("kof-vm-rules", fmt.Sprintf("%s/%s", c.clusterNamespace, c.clusterName)),
						Template:  env.GetPropagationTemplateName(),
						Namespace: k8s.KofNamespace,
						Values:    "propagation:\n  enabled: true\n  data: |\n{{ removeField \"vmRules\" \"metadata.ownerReferences\" | nindent 14 }}\n",
					},
				},
				TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
					{
						Identifier: "vmRules",
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       vmRulesConfigMapNamePrefix + c.clusterName,
							Namespace:  c.clusterNamespace,
						},
					},
				},
			},
		},
	}

	if err := c.client.Create(c.ctx, mcs); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create propagation MCS for '%s' cluster: %v", c.clusterName, err)
	}
	return nil
}

func (c *RegionalClusterConfigMap) UpdateChildConfigMap() error {
	childClustersList, err := c.GetChildClusters()
	if err != nil {
		return fmt.Errorf("failed to get child clusters: %v", err)
	}

	for _, childCluster := range childClustersList {
		configMap, err := childCluster.GetConfigMap()
		if err != nil {
			return fmt.Errorf("failed to get config map: %v", err)
		}

		if configMap == nil {
			continue
		}

		if err := childCluster.UpdateConfigMap(configMap, c.configMap.Data); err != nil {
			return fmt.Errorf("failed to update config map: %v", err)
		}
	}

	return nil
}

func (c *RegionalClusterConfigMap) GetChildClusters() ([]*ChildClusterRole, error) {
	log := log.FromContext(c.ctx)
	regionalCloud := c.configData.RegionalClusterCloud
	childClusterRoleList := make([]*ChildClusterRole, 0)
	opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "child"}}

	childClusterDeploymentsList, err := k8s.GetClusterDeployments(c.ctx, c.client, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterDeployments list: %v", err)
	}

	if isRegionlessConfigMap(c.configMap) {
		for _, childClusterDeployment := range childClusterDeploymentsList.Items {
			var childClusterRole *ChildClusterRole
			childClusterRole, err = NewChildClusterRole(c.ctx, &childClusterDeployment, c.client)
			if err != nil {
				return nil, fmt.Errorf("failed to create child cluster: %v", err)
			}
			childClusterRoleList = append(childClusterRoleList, childClusterRole)
		}
		return childClusterRoleList, nil
	}

	// Clusters in unknown clouds cannot be matched by location,
	// so only explicitly labeled child clusters
	// can be associated with the regional cluster.
	if regionalCloud == "" {
		for _, childClusterDeployment := range childClusterDeploymentsList.Items {
			regionalClusterName := childClusterDeployment.Labels[KofRegionalClusterNameLabel]
			regionalClusterNamespace := childClusterDeployment.Labels[KofRegionalClusterNamespaceLabel]
			if regionalClusterName != c.clusterName || regionalClusterNamespace != c.clusterNamespace {
				continue
			}

			childClusterRole, err := NewChildClusterRole(c.ctx, &childClusterDeployment, c.client)
			if err != nil {
				return nil, fmt.Errorf("failed to create child cluster: %v", err)
			}

			childClusterRoleList = append(childClusterRoleList, childClusterRole)
		}
		return childClusterRoleList, nil
	}

	regionalClusterDeploymentConfig := c.configData.ToClusterDeploymentConfig()

	for _, childClusterDeployment := range childClusterDeploymentsList.Items {
		childCloud, err := getCloud(c.ctx, c.client, &childClusterDeployment)
		if err != nil {
			log.Error(err, "failed to get cloud for child cluster deployment", "childClusterDeployment", childClusterDeployment.Name)
			continue
		}

		if regionalCloud != childCloud {
			continue
		}

		childClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
			childClusterDeployment.Spec.Config.Raw,
		)
		if err != nil {
			continue
		}

		if locationIsTheSame(
			regionalCloud,
			regionalClusterDeploymentConfig,
			childClusterDeploymentConfig,
		) && (childClusterDeployment.Labels[KofRegionalClusterNameLabel] == "" ||
			childClusterDeployment.Labels[KofRegionalClusterNameLabel] == c.clusterName) {
			childClusterRole, err := NewChildClusterRole(c.ctx, &childClusterDeployment, c.client)
			if err != nil {
				return nil, fmt.Errorf("failed to create child cluster: %v", err)
			}

			childClusterRoleList = append(childClusterRoleList, childClusterRole)
		}
	}
	return childClusterRoleList, nil
}

func (c *RegionalClusterConfigMap) CreateOrUpdatePromxyServerGroup() error {
	metrics, err := c.GetMetricsData()
	if err != nil {
		return fmt.Errorf("failed to get metrics data: %v", err)
	}

	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	dialTimeout := DefaultDialTimeout
	if httpClientConfig != nil && httpClientConfig.DialTimeout != (metav1.Duration{}) {
		dialTimeout = httpClientConfig.DialTimeout
	}

	tlsInsecureSkipVerify := false
	if httpClientConfig != nil {
		tlsInsecureSkipVerify = httpClientConfig.TLSConfig.InsecureSkipVerify
	}

	credentialsSecretName := vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace))
	if httpClientConfig != nil && httpClientConfig.BasicAuth.CredentialsSecretName != "" {
		credentialsSecretName = httpClientConfig.BasicAuth.CredentialsSecretName
	}

	promxyServerGroup := &kofv1beta1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPromxyServerGroupName(c.configMap.Name, c.configMap.Namespace),
			Namespace: c.clusterNamespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(c.ctx, c.client, promxyServerGroup, func() error {
		promxyServerGroup.OwnerReferences = []metav1.OwnerReference{*c.ownerReference}
		promxyServerGroup.Labels = map[string]string{
			labels.ManagedByLabel:  k8s.ManagedByValue,
			labels.SecretNameLabel: "kof-mothership-promxy-config",
		}
		promxyServerGroup.Spec = kofv1beta1.PromxyServerGroupSpec{
			ClusterName: c.clusterName,
			Scheme:      metrics.Scheme,
			PathPrefix:  metrics.Path,
			Targets:     []string{metrics.Target},
			HttpClient: kofv1beta1.HTTPClientConfig{
				DialTimeout: dialTimeout,
				TLSConfig: kofv1beta1.TLSConfig{
					InsecureSkipVerify: tlsInsecureSkipVerify,
				},
				BasicAuth: kofv1beta1.BasicAuth{
					CredentialsSecretName: credentialsSecretName,
					UsernameKey:           vmuser.UsernameKey,
					PasswordKey:           vmuser.PasswordKey,
				},
			},
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create or update PromxyServerGroup: %v", err)
	}
	return nil
}

func (c *RegionalClusterConfigMap) GetMetricsData() (*MetricsData, error) {
	log := log.FromContext(c.ctx)

	metricsEndpoint := c.configData.ReadMetricsEndpoint
	metricsURL, err := url.Parse(metricsEndpoint)
	if err != nil {
		log.Error(
			err, "cannot parse metrics endpoint",
			"regionalClusterName", c.clusterName,
			"metricsEndpointAnnotation", ReadMetricsAnnotation,
			"metricsEndpointValue", metricsEndpoint,
		)
		return nil, err
	}

	metricsPort, err := parsePort(metricsURL)
	if err != nil {
		log.Error(
			err, "cannot parse metrics endpoint for port",
			"regionalClusterName", c.clusterName,
			"metricsEndpointAnnotation", ReadMetricsAnnotation,
			"metricsEndpointValue", metricsEndpoint,
		)
		return nil, err
	}

	return &MetricsData{
		Endpoint: metricsEndpoint,
		Port:     metricsPort,
		URL:      metricsURL,
		Target:   fmt.Sprintf("%s:%s", metricsURL.Hostname(), metricsPort),
	}, nil
}

func (c *RegionalClusterConfigMap) GetHttpClientConfig() (*kofv1beta1.HTTPClientConfig, error) {
	var httpClientConfig *kofv1beta1.HTTPClientConfig
	httpConfigJson := c.configData.RegionalHTTPClientConfig

	if httpConfigJson != "" {
		httpClientConfig = &kofv1beta1.HTTPClientConfig{
			DialTimeout: defaultDialTimeout,
		}
		if err := json.Unmarshal([]byte(httpConfigJson), httpClientConfig); err != nil {
			record.LogEvent(
				c.ctx,
				"InvalidRegionalHTTPClientConfigAnnotation",
				"Failed to parse JSON from annotation",
				c.configMap,
				err,
				"annotation", KofRegionalHTTPClientConfigAnnotation,
				"value", httpConfigJson,
			)
			return nil, err
		}
	}
	return httpClientConfig, nil
}

func (c *RegionalClusterConfigMap) IsIstioCluster() bool {
	return c.configData.IstioRole != ""
}

func (c *RegionalClusterConfigMap) GetRegionalMCSName() string {
	if c.IsIstioCluster() {
		return env.GetIstioRegionalMCSName()
	}
	return env.GetRegionalMCSName()
}

func (c *RegionalClusterConfigMap) CreateOrUpdateLogsStorageConnection() error {
	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	tlsInsecureSkipVerify := false
	if httpClientConfig != nil {
		tlsInsecureSkipVerify = httpClientConfig.TLSConfig.InsecureSkipVerify
	}

	logsUrl, err := url.Parse(c.configData.ReadLogsEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse logs endpoint: %v", err)
	}

	vlClusterName := env.GetVLClusterName()
	if vlClusterName == "" {
		log.FromContext(c.ctx).Info("Skipping VMStorageConnection creation because KOF_VL_CLUSTER_NAME is not set")
		return nil
	}

	conn := &kofv1beta1.VMStorageConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetLogsStorageConnectionName(c.configMap.Name, c.configMap.Namespace),
			Namespace: c.clusterNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(c.ctx, c.client, conn, func() error {
		if conn.Labels == nil {
			conn.Labels = map[string]string{}
		}
		conn.Labels[labels.KofGeneratedLabel] = strutil.True
		conn.Labels[labels.ClusterNameLabel] = c.clusterName
		conn.Labels[labels.ManagedByLabel] = k8s.ManagedByValue
		conn.OwnerReferences = []metav1.OwnerReference{*c.ownerReference}
		conn.Spec = kofv1beta1.VMStorageConnectionSpec{
			ClusterRef: kofv1beta1.ClusterRef{
				Kind:      "VLCluster",
				Name:      vlClusterName,
				Namespace: c.releaseNamespace,
			},
			TargetStorageNode: kofv1beta1.TargetStorageNode{
				Address: logsUrl.Host + logsUrl.Path,
				Secret: kofv1beta1.SecretRef{
					Name:        vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace)),
					UsernameKey: vmuser.UsernameKey,
					PasswordKey: vmuser.PasswordKey,
				},
				TLSConfig: kofv1beta1.TLSStorageConfig{
					Enabled:            logsUrl.Scheme == httpsScheme,
					InsecureSkipVerify: tlsInsecureSkipVerify,
				},
			},
		}
		return nil
	})
	return err
}

// CreateOrUpdateTracesStorageConnection creates or updates a VMStorageConnection that registers
// the regional cluster's storage node with the VTCluster named by KOF_VT_CLUSTER_NAME.
// When KOF_VT_CLUSTER_NAME is not set the step is skipped.
// The VMStorageConnection is owned by the regional ConfigMap so it is garbage-collected
// automatically when the regional cluster is removed.
func (c *RegionalClusterConfigMap) CreateOrUpdateTracesStorageConnection() error {
	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	tlsInsecureSkipVerify := false
	if httpClientConfig != nil {
		tlsInsecureSkipVerify = httpClientConfig.TLSConfig.InsecureSkipVerify
	}

	tracesUrl, err := url.Parse(c.configData.ReadTracesEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse traces endpoint: %v", err)
	}

	vtClusterName := env.GetVTClusterName()
	if vtClusterName == "" {
		log.FromContext(c.ctx).Info("Skipping VMStorageConnection creation because KOF_VT_CLUSTER_NAME is not set")
		return nil
	}

	conn := &kofv1beta1.VMStorageConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetTracesStorageConnectionName(c.configMap.Name, c.configMap.Namespace),
			Namespace: c.clusterNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(c.ctx, c.client, conn, func() error {
		if conn.Labels == nil {
			conn.Labels = map[string]string{}
		}
		conn.Labels[labels.KofGeneratedLabel] = strutil.True
		conn.Labels[labels.ClusterNameLabel] = c.clusterName
		conn.Labels[labels.ManagedByLabel] = k8s.ManagedByValue
		conn.OwnerReferences = []metav1.OwnerReference{*c.ownerReference}
		conn.Spec = kofv1beta1.VMStorageConnectionSpec{
			ClusterRef: kofv1beta1.ClusterRef{
				Kind:      "VTCluster",
				Name:      vtClusterName,
				Namespace: c.releaseNamespace,
			},
			TargetStorageNode: kofv1beta1.TargetStorageNode{
				Address: tracesUrl.Host + tracesUrl.Path,
				Secret: kofv1beta1.SecretRef{
					Name:        vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace)),
					UsernameKey: vmuser.UsernameKey,
					PasswordKey: vmuser.PasswordKey,
				},
				TLSConfig: kofv1beta1.TLSStorageConfig{
					Enabled:            tracesUrl.Scheme == httpsScheme,
					InsecureSkipVerify: tlsInsecureSkipVerify,
				},
			},
		}
		return nil
	})
	return err
}

func (c *RegionalClusterConfigMap) CreateOrUpdateAuditLogsStorageConnection() error {
	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	tlsInsecureSkipVerify := false
	if httpClientConfig != nil {
		tlsInsecureSkipVerify = httpClientConfig.TLSConfig.InsecureSkipVerify
	}

	auditLogsUrl, err := url.Parse(c.configData.ReadAuditLogsEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse audit logs endpoint: %v", err)
	}

	vlClusterName := env.GetVLClusterName()
	if vlClusterName == "" {
		log.FromContext(c.ctx).Info("Skipping VMStorageConnection creation because KOF_VL_CLUSTER_NAME is not set")
		return nil
	}

	conn := &kofv1beta1.VMStorageConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetAuditLogsStorageConnectionName(c.configMap.Name, c.configMap.Namespace),
			Namespace: c.clusterNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(c.ctx, c.client, conn, func() error {
		if conn.Labels == nil {
			conn.Labels = map[string]string{}
		}
		conn.Labels[labels.KofGeneratedLabel] = strutil.True
		conn.Labels[labels.ClusterNameLabel] = c.clusterName
		conn.Labels[labels.ManagedByLabel] = k8s.ManagedByValue
		conn.OwnerReferences = []metav1.OwnerReference{*c.ownerReference}
		conn.Spec = kofv1beta1.VMStorageConnectionSpec{
			ClusterRef: kofv1beta1.ClusterRef{
				Kind:      "VLCluster",
				Name:      vlClusterName,
				Namespace: c.releaseNamespace,
			},
			TargetStorageNode: kofv1beta1.TargetStorageNode{
				Address: auditLogsUrl.Host + auditLogsUrl.Path,
				Secret: kofv1beta1.SecretRef{
					Name:        vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace)),
					UsernameKey: vmuser.UsernameKey,
					PasswordKey: vmuser.PasswordKey,
				},
				TLSConfig: kofv1beta1.TLSStorageConfig{
					Enabled:            auditLogsUrl.Scheme == httpsScheme,
					InsecureSkipVerify: tlsInsecureSkipVerify,
				},
			},
		}
		return nil
	})
	return err
}

func GetVmRulesMcsPropagationName(cmName string) string {
	return names.FNVName("kof-vm-rules-propagation", cmName)
}

func GetTracesStorageConnectionName(cmName, cmNamespace string) string {
	return names.FNVName("kof-traces-storage-connection", cmName+"/"+cmNamespace)
}

func GetLogsStorageConnectionName(cmName, cmNamespace string) string {
	return names.FNVName("kof-logs-storage-connection", cmName+"/"+cmNamespace)
}

func GetAuditLogsStorageConnectionName(cmName, cmNamespace string) string {
	return names.FNVName("kof-audit-logs-storage-connection", cmName+"/"+cmNamespace)
}

func GetPromxyServerGroupName(cmName, cmNamespace string) string {
	return names.FNVName("promxy-server-group", cmName+"/"+cmNamespace)
}

// GetVMUserAdminName generates a stable VMUser name for admin credentials derived from
// the ConfigMap name. It uses an Adler-32 hash via GetHelmAdler32Name to mirror Helm's
// `adler32sum` helper, ensuring the resulting name matches Helm template naming
// conventions and remains consistent across reconciles.
func GetVMUserAdminName(cmName, cmNamespace string) string {
	return names.Adler32Name("admin", cmName+"/"+cmNamespace)
}

func parsePort(u *url.URL) (string, error) {
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return "", fmt.Errorf("unknown scheme: %s", u.Scheme)
		}
	}
	return port, nil
}
