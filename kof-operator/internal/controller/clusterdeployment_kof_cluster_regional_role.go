package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	vmRulesConfigMapNamePrefix = "kof-record-vmrules-"
)

type RegionalClusterRole struct {
	clusterName             string
	releaseNamespace        string
	client                  client.Client
	ctx                     context.Context
	clusterDeployment       *kcmv1beta1.ClusterDeployment
	clusterDeploymentConfig *ClusterDeploymentConfig
	ownerReference          metav1.OwnerReference
}

type MetricsData struct {
	Endpoint string
	Target   string
	Port     string
	*url.URL
}

func NewRegionalClusterRole(ctx context.Context, cd *kcmv1beta1.ClusterDeployment, client client.Client) (*RegionalClusterRole, error) {
	namespace, err := getReleaseNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get release namespace: %v", err)
	}

	ownerReference, err := utils.GetOwnerReference(cd, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner reference: %v", err)
	}

	cdConfig, err := ReadClusterDeploymentConfig(cd.Spec.Config.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster deployment config: %v", err)
	}

	return &RegionalClusterRole{
		ctx:                     ctx,
		clusterDeployment:       cd,
		clusterDeploymentConfig: cdConfig,
		clusterName:             cd.Name,
		client:                  client,
		releaseNamespace:        namespace,
		ownerReference:          ownerReference,
	}, nil
}

func (r *RegionalClusterRole) Reconcile() error {
	if err := r.CreateVmRulesConfigMap(); err != nil {
		return fmt.Errorf("failed to create vm rules config map: %v", err)
	}

	if err := r.UpdateChildConfigMap(); err != nil {
		return fmt.Errorf("failed to update child's config map: %v", err)
	}

	exists, err := r.IsGrafanaDatasourceExists()
	if err != nil {
		return fmt.Errorf("failed to check if grafana datasource exists: %v", err)
	}

	if exists {
		return nil
	}

	if err := r.CreatePromxyServerGroup(); err != nil {
		return fmt.Errorf("failed to create promxy server group: %v", err)
	}

	if err := r.CreateGrafanaDataSource(); err != nil {
		return fmt.Errorf("failed to create grafana datasource: %v", err)
	}

	return nil
}

func (r *RegionalClusterRole) CreateVmRulesConfigMap() error {
	vmRulesConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       r.clusterDeployment.Namespace,
			Name:            vmRulesConfigMapNamePrefix + r.clusterName,
			OwnerReferences: []metav1.OwnerReference{r.ownerReference},
			Labels: map[string]string{
				KofRecordVMRulesClusterNameLabel: r.clusterName,
				utils.ManagedByLabel:             utils.ManagedByValue,
				utils.KofGeneratedLabel:          "true",
			},
		},
	}

	return utils.CreateIfNotExists(r.ctx, r.client, vmRulesConfigMap, "VMRulesConfigMap", []any{
		"configMapName", vmRulesConfigMap.Name,
	})
}

func (r *RegionalClusterRole) UpdateChildConfigMap() error {
	childClustersList, err := r.GetChildClusters()
	if err != nil {
		return fmt.Errorf("failed to get child clusters: %v", err)
	}

	configData, err := r.GetConfigData()
	if err != nil {
		return fmt.Errorf("failed to get config data: %v", err)
	}

	for _, childCluster := range childClustersList {
		configMap, err := childCluster.GetConfigMap()
		if err != nil {
			return fmt.Errorf("failed to get config map: %v", err)
		}

		if configMap == nil {
			continue
		}

		if err := childCluster.UpdateConfigMap(configMap, configData); err != nil {
			return fmt.Errorf("failed to update config map: %v", err)
		}
	}

	return nil
}

func (r *RegionalClusterRole) GetChildClusters() ([]*ChildClusterRole, error) {
	regionalCloud := getCloud(r.clusterDeployment)
	childClusterRoleList := make([]*ChildClusterRole, 0)

	childClusterDeploymentsList := &kcmv1beta1.ClusterDeploymentList{}
	opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "child"}}

	if err := r.client.List(r.ctx, childClusterDeploymentsList, opts...); err != nil {
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
			r.clusterDeploymentConfig,
			childClusterDeploymentConfig,
		) {
			childClusterRole, err := NewChildClusterRole(r.ctx, &childClusterDeployment, r.client)
			if err != nil {
				return nil, fmt.Errorf("failed to create child cluster: %v", err)
			}

			childClusterRoleList = append(childClusterRoleList, childClusterRole)
		}
	}
	return childClusterRoleList, nil
}

func (r *RegionalClusterRole) CreatePromxyServerGroup() error {
	metrics, err := r.GetMetricsData()
	if err != nil {
		return fmt.Errorf("failed to get metrics data: %v", err)
	}

	promxyServerGroup := &kofv1beta1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.clusterName + "-metrics",
			Namespace: r.releaseNamespace,
			// `OwnerReferences` is N/A because `regionalClusterDeployment` namespace differs.
			Labels: map[string]string{
				utils.ManagedByLabel:  utils.ManagedByValue,
				PromxySecretNameLabel: "kof-mothership-promxy-config",
			},
		},
		Spec: kofv1beta1.PromxyServerGroupSpec{
			ClusterName: r.clusterName,
			Scheme:      metrics.Scheme,
			Targets:     []string{metrics.Target},
			PathPrefix:  metrics.EscapedPath(),
			HttpClient: kofv1beta1.HTTPClientConfig{
				DialTimeout: defaultDialTimeout,
			},
		},
	}

	httpClientConfig, err := r.GetHttpClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get http client config: %v", err)
	}

	if httpClientConfig != nil {
		promxyServerGroup.Spec.HttpClient = *httpClientConfig
	}

	if !r.IsIstioCluster() {
		basicAuth := &promxyServerGroup.Spec.HttpClient.BasicAuth
		basicAuth.CredentialsSecretName = KofStorageSecretName
		basicAuth.UsernameKey = "username"
		basicAuth.PasswordKey = "password"
	}

	if err := utils.CreateIfNotExists(r.ctx, r.client, promxyServerGroup, "PromxyServerGroup", []any{
		"promxyServerGroupName", promxyServerGroup.Name,
	}); err != nil {
		utils.LogEvent(
			r.ctx,
			"PromxySeverGroupCreationFailed",
			"Failed to create PromxyServerGroup",
			r.clusterDeployment,
			err,
			"promxyServerGroupName", promxyServerGroup.Name,
		)
		return err
	}

	utils.LogEvent(
		r.ctx,
		"PromxyServerGroupCreated",
		"PromxyServerGroup is successfully created",
		r.clusterDeployment,
		nil,
		"promxyServerGroupName", promxyServerGroup.Name,
	)

	return nil
}

func (r *RegionalClusterRole) IsGrafanaDatasourceExists() (bool, error) {
	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
	if err := r.client.Get(r.ctx, types.NamespacedName{
		Name:      r.GetGrafanaDatasourceName(),
		Namespace: r.releaseNamespace,
	}, grafanaDatasource); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RegionalClusterRole) CreateGrafanaDataSource() error {
	isIstio := r.IsIstioCluster()
	logsEndpoint, err := getEndpoint(r.ctx, ReadLogsAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return err
	}

	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.GetGrafanaDatasourceName(),
			Namespace: r.releaseNamespace,
			// `OwnerReferences` is N/A because `regionalClusterDeployment` namespace differs.
			Labels: map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
		},
		Spec: grafanav1beta1.GrafanaDatasourceSpec{
			GrafanaCommonSpec: grafanav1beta1.GrafanaCommonSpec{
				InstanceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"dashboards": "grafana"},
				},
				ResyncPeriod: metav1.Duration{Duration: 5 * time.Minute},
			},
			Datasource: &grafanav1beta1.GrafanaDatasourceInternal{
				Name:      r.clusterName,
				Type:      "victoriametrics-logs-datasource",
				URL:       logsEndpoint,
				Access:    "proxy",
				IsDefault: utils.BoolPtr(false),
				BasicAuth: utils.BoolPtr(!isIstio),
			},
		},
	}

	httpClientConfig, err := r.GetHttpClientConfig()
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

	if err := utils.CreateIfNotExists(r.ctx, r.client, grafanaDatasource, "GrafanaDatasource", []any{
		"grafanaDatasourceName", grafanaDatasource.Name,
	}); err != nil {
		utils.LogEvent(
			r.ctx,
			"GrafanaDatasourceCreationFailed",
			"Failed to create GrafanaDatasource",
			r.clusterDeployment,
			err,
			"grafanaDatasourceName", grafanaDatasource.Name,
		)
		return err
	}

	utils.LogEvent(
		r.ctx,
		"GrafanaDatasourceCreated",
		"GrafanaDatasource is successfully created",
		r.clusterDeployment,
		nil,
		"grafanaDatasourceName", grafanaDatasource.Name,
	)

	return nil
}

func (r *RegionalClusterRole) GetHttpClientConfig() (*kofv1beta1.HTTPClientConfig, error) {
	var httpClientConfig *kofv1beta1.HTTPClientConfig
	if httpConfigJson, ok := r.clusterDeployment.Annotations[KofRegionalHTTPClientConfigAnnotation]; ok {
		httpClientConfig = &kofv1beta1.HTTPClientConfig{
			DialTimeout: defaultDialTimeout,
		}
		if err := json.Unmarshal([]byte(httpConfigJson), httpClientConfig); err != nil {
			utils.LogEvent(
				r.ctx,
				"InvalidRegionalHTTPClientConfigAnnotation",
				"Failed to parse JSON from annotation",
				r.clusterDeployment,
				err,
				"annotation", KofRegionalHTTPClientConfigAnnotation,
				"value", httpConfigJson,
			)
			return nil, err
		}
	}
	return httpClientConfig, nil
}

func (r *RegionalClusterRole) GetMetricsData() (*MetricsData, error) {
	log := log.FromContext(r.ctx)

	metricsEndpoint, err := getEndpoint(r.ctx, ReadMetricsAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return nil, err
	}

	metricsURL, err := url.Parse(metricsEndpoint)
	if err != nil {
		log.Error(
			err, "cannot parse metrics endpoint",
			"regionalClusterDeploymentName", r.clusterName,
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
				"regionalClusterDeploymentName", r.clusterName,
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

func (r *RegionalClusterRole) GetConfigData() (map[string]string, error) {
	var err error
	configData := map[string]string{RegionalClusterNameKey: r.clusterName}

	if r.IsIstioCluster() {
		return configData, nil
	}

	configData[ReadMetricsKey], err = getEndpoint(r.ctx, ReadMetricsAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return nil, err
	}

	configData[WriteMetricsKey], err = getEndpoint(r.ctx, WriteMetricsAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return nil, err
	}

	configData[WriteLogsKey], err = getEndpoint(r.ctx, WriteLogsAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return nil, err
	}

	configData[WriteTracesKey], err = getEndpoint(r.ctx, WriteTracesAnnotation, r.clusterDeployment, r.clusterDeploymentConfig)
	if err != nil {
		return nil, err
	}

	return configData, nil
}

func (r *RegionalClusterRole) GetGrafanaDatasourceName() string {
	return r.clusterName + "-logs"
}

func (r *RegionalClusterRole) IsIstioCluster() bool {
	_, isIstio := r.clusterDeployment.Labels[IstioRoleLabel]
	return isIstio
}

func getReleaseNamespace() (string, error) {
	namespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return "", fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}
	if len(namespace) == 0 {
		return "", fmt.Errorf("RELEASE_NAMESPACE env var is set but empty")
	}
	return namespace, nil
}
