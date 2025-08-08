package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"time"

	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
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
	clusterName      string
	clusterNamespace string
	releaseNamespace string
	ctx              context.Context
	client           client.Client
	configMap        *corev1.ConfigMap
	ownerReference   *metav1.OwnerReference
	configData       *ConfigData
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

	return &RegionalClusterConfigMap{
		clusterName:      clusterName,
		clusterNamespace: clusterNamespace,
		releaseNamespace: releaseNamespace,
		ctx:              ctx,
		client:           client,
		configMap:        cm,
		ownerReference:   ownerReference,
		configData:       configMapData,
	}, nil
}

func (c *RegionalClusterConfigMap) Reconcile() error {
	if err := c.CreateVmRulesConfigMap(); err != nil {
		return fmt.Errorf("failed to create vm rules ConfigMap: %v", err)
	}

	if err := c.UpdateChildConfigMap(); err != nil {
		return fmt.Errorf("failed to update child's ConfigMap: %v", err)
	}

	if err := c.CreateOrUpdatePromxyServerGroup(); err != nil {
		return fmt.Errorf("failed to create or update PromxyServerGroup: %v", err)
	}

	if err := c.CreateOrUpdateGrafanaDatasource(); err != nil {
		return fmt.Errorf("failed to create or update GrafanaDatasource: %v", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) CreateVmRulesConfigMap() error {
	vmRulesConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       c.clusterNamespace,
			Name:            vmRulesConfigMapNamePrefix + c.clusterName,
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
			Labels: map[string]string{
				KofRecordVMRulesClusterNameLabel: c.clusterName,
				utils.ManagedByLabel:             utils.ManagedByValue,
				utils.KofGeneratedLabel:          "true",
			},
		},
	}

	return utils.CreateIfNotExists(c.ctx, c.client, vmRulesConfigMap, "VMRulesConfigMap", []any{
		"configMapName", vmRulesConfigMap.Name,
	})
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
		if regionalCloud != getCloud(&childClusterDeployment) {
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

	if !c.IsIstioCluster() {
		basicAuth := &promxyServerGroup.Spec.HttpClient.BasicAuth
		basicAuth.CredentialsSecretName = KofStorageSecretName
		basicAuth.UsernameKey = "username"
		basicAuth.PasswordKey = "password"
	}

	if err := utils.CreateIfNotExists(c.ctx, c.client, promxyServerGroup, "PromxyServerGroup", []any{
		"promxyServerGroupName", promxyServerGroup.Name,
	}); err != nil {
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

func (c *RegionalClusterConfigMap) CreateOrUpdateGrafanaDatasource() error {
	grafanaDatasource, err := c.GetGrafanaDatasource()
	if err != nil {
		return fmt.Errorf("failed to get grafana datasource: %v", err)
	}

	if grafanaDatasource == nil {
		if err := c.DeleteOldGrafanaDatasource(); err != nil {
			return fmt.Errorf("failed to delete old grafana datasource: %v", err)
		}
		if err := c.CreateGrafanaDataSource(); err != nil {
			return fmt.Errorf("failed to create GrafanaDatasource: %v", err)
		}
		return nil
	}

	if err := c.UpdateGrafanaDatasource(grafanaDatasource); err != nil {
		return fmt.Errorf("failed to update GrafanaDatasource: %v", err)
	}

	return nil
}

func (c *RegionalClusterConfigMap) GetGrafanaDatasource() (*grafanav1beta1.GrafanaDatasource, error) {
	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      c.GetGrafanaDatasourceName(),
		Namespace: c.clusterNamespace,
	}, grafanaDatasource); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return grafanaDatasource, nil
}

// To migrate GrafanaDatasource from releaseNamespace to clusterNamespace for multi-tenancy,
// we need to delete the old GrafanaDatasource in the releaseNamespace first.
func (c *RegionalClusterConfigMap) DeleteOldGrafanaDatasource() error {
	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      c.GetGrafanaDatasourceName(),
		Namespace: c.releaseNamespace,
	}, grafanaDatasource); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := c.client.Delete(c.ctx, grafanaDatasource); err != nil {
		return err
	}
	return nil
}

func (c *RegionalClusterConfigMap) CreateGrafanaDataSource() error {
	isIstio := c.IsIstioCluster()
	logsEndpoint := c.configData.ReadLogsEndpoint

	if utils.IsEmptyString(logsEndpoint) {
		return fmt.Errorf("failed to get log endpoint from configmap '%s'", c.configMap.Name)
	}

	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            c.GetGrafanaDatasourceName(),
			Namespace:       c.clusterNamespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
		},
		Spec: grafanav1beta1.GrafanaDatasourceSpec{
			GrafanaCommonSpec: grafanav1beta1.GrafanaCommonSpec{
				AllowCrossNamespaceImport: true,
				InstanceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"dashboards": "grafana"},
				},
				ResyncPeriod: metav1.Duration{Duration: 5 * time.Minute},
			},
			Datasource: &grafanav1beta1.GrafanaDatasourceInternal{
				Name:      c.clusterName,
				Type:      "victoriametrics-logs-datasource",
				URL:       logsEndpoint,
				Access:    "proxy",
				IsDefault: utils.BoolPtr(false),
				BasicAuth: utils.BoolPtr(!isIstio),
			},
		},
	}

	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	if httpClientConfig != nil {
		grafanaDatasource.Spec.Datasource.JSONData = json.RawMessage(
			fmt.Sprintf(`{"tlsSkipVerify": %t, "timeout": "%d"}`, httpClientConfig.TLSConfig.InsecureSkipVerify, int(httpClientConfig.DialTimeout.Duration.Seconds())),
		)
	}

	if !isIstio {
		grafanaDatasource.Spec.Datasource.BasicAuthUser = "${username}" // Set in `ValuesFrom`.
		grafanaDatasource.Spec.Datasource.SecureJSONData = json.RawMessage(
			`{"basicAuthPassword": "${password}"}`,
		)
		grafanaDatasource.Spec.ValuesFrom = []grafanav1beta1.ValueFrom{
			{
				TargetPath: "basicAuthUser",
				ValueFrom: grafanav1beta1.ValueFromSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: KofStorageSecretName,
						},
						Key: "username",
					},
				},
			},
			{
				TargetPath: "secureJsonData.basicAuthPassword",
				ValueFrom: grafanav1beta1.ValueFromSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: KofStorageSecretName,
						},
						Key: "password",
					},
				},
			},
		}
	}

	if err := utils.CreateIfNotExists(c.ctx, c.client, grafanaDatasource, "GrafanaDatasource", []any{
		"grafanaDatasourceName", grafanaDatasource.Name,
	}); err != nil {
		utils.LogEvent(
			c.ctx,
			"GrafanaDatasourceCreationFailed",
			"Failed to create GrafanaDatasource",
			c.configMap,
			err,
			"grafanaDatasourceName", grafanaDatasource.Name,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"GrafanaDatasourceCreated",
		"GrafanaDatasource is successfully created",
		c.configMap,
		nil,
		"grafanaDatasourceName", grafanaDatasource.Name,
	)

	return nil
}

func (c *RegionalClusterConfigMap) UpdateGrafanaDatasource(grafanaDatasource *grafanav1beta1.GrafanaDatasource) error {
	logsEndpoint := c.configData.ReadLogsEndpoint

	if utils.IsEmptyString(logsEndpoint) {
		return fmt.Errorf("failed to get log endpoint from configmap '%s'", c.configMap.Name)
	}

	httpClientConfig, err := c.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	httpClientConfigRaw := c.httpClientConfigToRawJSON(httpClientConfig)

	if logsEndpoint == grafanaDatasource.Spec.Datasource.URL &&
		string(httpClientConfigRaw) == string(grafanaDatasource.Spec.Datasource.JSONData) {
		return nil
	}

	grafanaDatasource.Spec.Datasource.URL = logsEndpoint
	grafanaDatasource.Spec.Datasource.JSONData = httpClientConfigRaw
	if err := c.client.Update(c.ctx, grafanaDatasource); err != nil {
		utils.LogEvent(
			c.ctx,
			"GrafanaDatasourceUpdateFailed",
			"Failed to update GrafanaDatasource",
			c.configMap,
			err,
			"grafanaDatasourceName", grafanaDatasource.Name,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"GrafanaDatasourceUpdated",
		"GrafanaDatasource is successfully updated",
		c.configMap,
		nil,
		"grafanaDatasourceName", grafanaDatasource.Name,
	)

	return nil
}

func (r *RegionalClusterConfigMap) httpClientConfigToRawJSON(httpClientConfig *kofv1beta1.HTTPClientConfig) json.RawMessage {
	if httpClientConfig == nil {
		return []byte{}
	}

	return json.RawMessage(
		fmt.Sprintf(`{"tlsSkipVerify": %t, "timeout": "%d"}`, httpClientConfig.TLSConfig.InsecureSkipVerify, int(httpClientConfig.DialTimeout.Duration.Seconds())),
	)
}

func (c *RegionalClusterConfigMap) GetPromxyServerGroupName() string {
	return c.clusterName + "-metrics"
}

func (c *RegionalClusterConfigMap) GetGrafanaDatasourceName() string {
	return c.clusterName + "-logs"
}

func (c *RegionalClusterConfigMap) IsIstioCluster() bool {
	return !utils.IsEmptyString(c.configData.IstioRole)
}
