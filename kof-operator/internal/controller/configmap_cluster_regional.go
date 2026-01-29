package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	datasource "github.com/k0rdent/kof/kof-operator/internal/controller/grafana-datasource"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/controller/vmuser"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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

	ownerReference, err = utils.GetOwnerReference(cm, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner reference: %v", err)
	}

	releaseNamespace, err := utils.GetReleaseNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get release namespace: %v", err)
	}

	isKcmRegionCluster, err := k8s.IsClusterKcmRegion(ctx, client, clusterName, clusterNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if cluster is KCM region cluster: %v", err)
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
		return fmt.Errorf("failed to create or update PromxyServerGroup: %v", err)
	}

	if !utils.GrafanaEnabled() {
		return nil
	}

	logsDatasource := c.buildLogsDatasource()
	if err := c.CreateOrUpdateDatasource(logsDatasource); err != nil {
		return fmt.Errorf("failed to create or update GrafanaDatasource: %v", err)
	}

	tracesDatasource := c.buildTracesDatasource()
	if err := c.CreateOrUpdateDatasource(tracesDatasource); err != nil {
		return fmt.Errorf("failed to create or update GrafanaDatasource: %v", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) CreateVMUser() error {
	return c.VMUserManager.Create(c.ctx, &vmuser.CreateOptions{
		Name:           GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace),
		Namespace:      c.clusterNamespace,
		ClusterRef:     c.configMap,
		OwnerReference: c.ownerReference,
		MCSConfig: &vmuser.MCSConfig{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k0rdent.mirantis.com/kof-cluster-name": c.clusterName,
				},
			},
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
				utils.ManagedByLabel:             utils.ManagedByValue,
				utils.KofGeneratedLabel:          utils.True,
			},
		},
	}

	created, err := utils.EnsureCreated(c.ctx, c.client, vmRulesConfigMap)
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
				utils.ManagedByLabel: utils.ManagedByValue,
				"cluster-name":       c.clusterName,
				"cluster-namespace":  c.clusterNamespace,
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k0rdent.mirantis.com/kcm-region-cluster": "true",
				},
			},
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      "copy-vm-rules-configmap",
						Namespace: k8s.DefaultSystemNamespace,
						Template:  "kof-configmap-propagation",
					},
				},
				TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
					{
						Identifier: "ConfigMap",
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

// To migrate PromxyServerGroup from releaseNamespace to clusterNamespace for multi-tenancy,
// we need to delete the old PromxyServerGroup in the releaseNamespace first.
func (c *RegionalClusterConfigMap) DeleteOldPromxyServerGroup() error {
	promxyServerGroup := &kofv1beta1.PromxyServerGroup{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      c.GetPromxyServerGroupName(),
		Namespace: c.releaseNamespace,
	}, promxyServerGroup); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := c.client.Delete(c.ctx, promxyServerGroup); err != nil {
		return err
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

	if utils.IsEmptyString(regionalCloud) {
		return nil, fmt.Errorf("failed to get regional cloud from config map '%s'", c.configMap.Name)
	}

	regionalClusterDeploymentConfig := c.configData.ToClusterDeploymentConfig()
	childClusterRoleList := make([]*ChildClusterRole, 0)
	opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "child"}}
	childClusterDeploymentsList, err := k8s.GetClusterDeployments(c.ctx, c.client, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterDeployments list: %v", err)
	}

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
	promxyServerGroup, err := c.GetPromxyServerGroup()
	if err != nil {
		return fmt.Errorf("failed to get promxy server group: %v", err)
	}

	if promxyServerGroup == nil {
		if err := c.DeleteOldPromxyServerGroup(); err != nil {
			return fmt.Errorf("failed to delete old promxy server group: %v", err)
		}
		if err := c.CreatePromxyServerGroup(); err != nil {
			return fmt.Errorf("failed to create promxy server group: %v", err)
		}
		return nil
	}

	if err := c.UpdatePromxyServerGroup(promxyServerGroup); err != nil {
		return fmt.Errorf("failed to update promxy server group: %v", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) GetPromxyServerGroup() (*kofv1beta1.PromxyServerGroup, error) {
	promxyServerGroup := &kofv1beta1.PromxyServerGroup{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      c.GetPromxyServerGroupName(),
		Namespace: c.clusterNamespace,
	}, promxyServerGroup); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return promxyServerGroup, nil
}

func (c *RegionalClusterConfigMap) CreatePromxyServerGroup() error {
	log := log.FromContext(c.ctx)

	metrics, err := c.GetMetricsData()
	if err != nil {
		return fmt.Errorf("failed to get metrics data: %v", err)
	}

	promxyServerGroup := &kofv1beta1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetPromxyServerGroupName(),
			Namespace: c.clusterNamespace,
			Labels: map[string]string{
				utils.ManagedByLabel:  utils.ManagedByValue,
				PromxySecretNameLabel: "kof-mothership-promxy-config",
			},
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
		},
		Spec: kofv1beta1.PromxyServerGroupSpec{
			ClusterName: c.clusterName,
			Scheme:      metrics.Scheme,
			Targets:     []string{metrics.Target},
			PathPrefix:  metrics.EscapedPath(),
			HttpClient: kofv1beta1.HTTPClientConfig{
				DialTimeout: defaultDialTimeout,
			},
		},
	}

	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	if httpClientConfig != nil {
		promxyServerGroup.Spec.HttpClient = *httpClientConfig
	}

	basicAuth := &promxyServerGroup.Spec.HttpClient.BasicAuth
	basicAuth.CredentialsSecretName = vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace))
	basicAuth.UsernameKey = vmuser.UsernameKey
	basicAuth.PasswordKey = vmuser.PasswordKey

	created, err := utils.EnsureCreated(c.ctx, c.client, promxyServerGroup)
	if err != nil {
		utils.LogEvent(
			c.ctx,
			"PromxySeverGroupCreationFailed",
			"Failed to create PromxyServerGroup",
			c.configMap,
			err,
			"promxyServerGroupName", promxyServerGroup.Name,
		)
		return err
	}

	if !created {
		log.Info("PromxyServerGroup already created", "promxyServerGroupName", promxyServerGroup.Name)
		return nil
	}

	utils.LogEvent(
		c.ctx,
		"PromxyServerGroupCreated",
		"PromxyServerGroup is successfully created",
		c.configMap,
		nil,
		"promxyServerGroupName", promxyServerGroup.Name,
	)

	return nil
}

func (c *RegionalClusterConfigMap) UpdatePromxyServerGroup(promxyServerGroup *kofv1beta1.PromxyServerGroup) error {
	newMetrics, err := c.GetMetricsData()
	if err != nil {
		return fmt.Errorf("failed to get metrics data: %v", err)
	}

	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	if httpClientConfig == nil {
		httpClientConfig = &promxyServerGroup.Spec.HttpClient
	}

	basicAuth := &promxyServerGroup.Spec.HttpClient.BasicAuth
	basicAuth.CredentialsSecretName = vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace))
	basicAuth.UsernameKey = vmuser.UsernameKey
	basicAuth.PasswordKey = vmuser.PasswordKey

	if newMetrics.Scheme == promxyServerGroup.Spec.Scheme &&
		newMetrics.Target == promxyServerGroup.Spec.Targets[0] &&
		newMetrics.EscapedPath() == promxyServerGroup.Spec.PathPrefix &&
		reflect.DeepEqual(httpClientConfig, promxyServerGroup.Spec.HttpClient) {
		return nil
	}

	promxyServerGroup.Spec.Scheme = newMetrics.Scheme
	promxyServerGroup.Spec.Targets = []string{newMetrics.Target}
	promxyServerGroup.Spec.PathPrefix = newMetrics.EscapedPath()
	if err := utils.MergeConfig(&promxyServerGroup.Spec.HttpClient, httpClientConfig); err != nil {
		return err
	}

	if err := c.client.Update(c.ctx, promxyServerGroup); err != nil {
		utils.LogEvent(
			c.ctx,
			"PromxySeverGroupUpdateFailed",
			"Failed to update PromxyServerGroup",
			c.configMap,
			err,
			"promxyServerGroupName", promxyServerGroup.Name,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"PromxyServerGroupUpdated",
		"PromxyServerGroup is successfully updated",
		c.configMap,
		nil,
		"promxyServerGroupName", promxyServerGroup.Name,
	)
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

	metricsPort := metricsURL.Port()
	if metricsPort == "" {
		switch metricsURL.Scheme {
		case "http":
			metricsPort = "80"
		case "https":
			metricsPort = "443"
		default:
			err := fmt.Errorf("cannot detect port of metrics endpoint")
			log.Error(
				err, "in",
				"regionalClusterName", c.clusterName,
				"metricsEndpointAnnotation", ReadMetricsAnnotation,
				"metricsEndpointValue", metricsEndpoint,
			)
			return nil, err
		}
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

	if !utils.IsEmptyString(httpConfigJson) {
		httpClientConfig = &kofv1beta1.HTTPClientConfig{
			DialTimeout: defaultDialTimeout,
		}
		if err := json.Unmarshal([]byte(httpConfigJson), httpClientConfig); err != nil {
			utils.LogEvent(
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

func (c *RegionalClusterConfigMap) CreateOrUpdateDatasource(ds *datasource.GrafanaDatasource) error {
	if !utils.GrafanaEnabled() {
		return fmt.Errorf("grafana is not enabled")
	}

	existingDatasource, err := ds.Get()
	if err != nil {
		return fmt.Errorf("failed to get grafana datasource: %w", err)
	}

	if existingDatasource == nil {
		if err := c.DeleteOldGrafanaDatasource(ds); err != nil {
			return fmt.Errorf("failed to delete old grafana datasource: %w", err)
		}

		if err := ds.Create(); err != nil {
			return fmt.Errorf("failed to create GrafanaDatasource: %w", err)
		}

		return nil
	}

	if err := ds.Update(existingDatasource); err != nil {
		return fmt.Errorf("failed to update GrafanaDatasource: %w", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) buildDatasource(dsType, category, url string) *datasource.GrafanaDatasource {
	opts := []datasource.Option{
		datasource.WithType(dsType),
		datasource.WithCategory(category),
		datasource.WithURL(url),
		datasource.WithOwnerReference(*c.ownerReference),
	}

	if httpClientConfig, err := c.GetHttpClientConfig(); err == nil && httpClientConfig != nil {
		jsonData := datasource.BuildJSONDataWithTimeout(
			httpClientConfig.TLSConfig.InsecureSkipVerify,
			int(httpClientConfig.DialTimeout.Seconds()),
		)
		opts = append(opts, datasource.WithJSONData(jsonData))
	}

	opts = append(opts, datasource.WithBasicAuth(
		vmuser.BuildSecretName(GetVMUserAdminName(c.configMap.Name, c.configMap.Namespace)),
		vmuser.UsernameKey,
		vmuser.PasswordKey,
	),
	)

	return datasource.New(c.ctx, c.client, c.clusterName, c.clusterNamespace, opts...)
}

func (c *RegionalClusterConfigMap) buildTracesDatasource() *datasource.GrafanaDatasource {
	return c.buildDatasource(
		datasource.TypeJaeger,
		datasource.CategoryTraces,
		c.configData.ReadTracesEndpoint,
	)
}

func (c *RegionalClusterConfigMap) buildLogsDatasource() *datasource.GrafanaDatasource {
	return c.buildDatasource(
		datasource.TypeVictoriaLogs,
		datasource.CategoryLogs,
		c.configData.ReadLogsEndpoint,
	)
}

func (c *RegionalClusterConfigMap) DeleteOldGrafanaDatasource(ds *datasource.GrafanaDatasource) error {
	if !utils.GrafanaEnabled() {
		return fmt.Errorf("grafana is not enabled")
	}

	datasource := new(grafanav1beta1.GrafanaDatasource)
	datasource.SetName(ds.GetName())
	datasource.SetNamespace(c.releaseNamespace)

	if err := c.client.Delete(c.ctx, datasource); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete datasource: %w", err)
	}
	return nil
}

func (c *RegionalClusterConfigMap) GetPromxyServerGroupName() string {
	return c.clusterName + "-metrics"
}

func (c *RegionalClusterConfigMap) IsIstioCluster() bool {
	return !utils.IsEmptyString(c.configData.IstioRole)
}

func GetVmRulesMcsPropagationName(cmName string) string {
	return utils.GetNameHash("kof-vm-rules-propagation", cmName)
}

// GetVMUserAdminName generates a stable VMUser name for admin credentials derived from
// the ConfigMap name. It uses an Adler-32 hash via GetHelmAdler32Name to mirror Helm's
// `adler32sum` helper, ensuring the resulting name matches Helm template naming
// conventions and remains consistent across reconciles.
func GetVMUserAdminName(cmName, cmNamespace string) string {
	return utils.GetHelmAdler32Name("admin", cmName+"/"+cmNamespace)
}
