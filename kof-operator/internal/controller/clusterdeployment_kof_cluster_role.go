package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/istio"
	remotesecret "github.com/k0rdent/kof/kof-operator/internal/controller/istio/remote-secret"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const prefix = "k0rdent.mirantis.com/"

// Labels:
const KofClusterRoleLabel = prefix + "kof-cluster-role"
const KofRegionalClusterNameLabel = prefix + "kof-regional-cluster-name"

// Annotations:
const KofRegionalDomainAnnotation = prefix + "kof-regional-domain"
const KofRegionalHTTPClientConfigAnnotation = prefix + "kof-http-config"
const WriteMetricsAnnotation = prefix + "kof-write-metrics-endpoint"
const ReadMetricsAnnotation = prefix + "kof-read-metrics-endpoint"
const WriteLogsAnnotation = prefix + "kof-write-logs-endpoint"
const ReadLogsAnnotation = prefix + "kof-read-logs-endpoint"
const WriteTracesAnnotation = prefix + "kof-write-traces-endpoint"

// Endpoints for Sprintf:
var defaultEndpoints = map[string]string{
	WriteMetricsAnnotation: "https://vmauth.%s/vm/insert/0/prometheus/api/v1/write",
	ReadMetricsAnnotation:  "https://vmauth.%s/vm/select/0/prometheus",
	WriteLogsAnnotation:    "https://vmauth.%s/vli/insert/opentelemetry/v1/logs",
	ReadLogsAnnotation:     "https://vmauth.%s/vls",
	WriteTracesAnnotation:  "https://jaeger.%s/collector",
}
var istioEndpoints = map[string]string{
	ReadLogsAnnotation:    "http://%s-logs-select:9471",
	ReadMetricsAnnotation: "http://%s-vmselect:8481/select/0/prometheus",
}

// Child cluster ConfigMap data keys:
const RegionalClusterNameKey = "regional_cluster_name"
const ReadMetricsKey = "read_metrics_endpoint"
const WriteMetricsKey = "write_metrics_endpoint"
const WriteLogsKey = "write_logs_endpoint"
const WriteTracesKey = "write_traces_endpoint"

// Other:
const KofStorageSecretName = "storage-vmuser-credentials"
const KofIstioSecretTemplate = "kof-istio-secret-template"

var defaultDialTimeout = metav1.Duration{Duration: time.Second * 5}

func (r *ClusterDeploymentReconciler) ReconcileKofClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1beta1.ClusterDeployment,
) error {
	role := clusterDeployment.Labels[KofClusterRoleLabel]
	if role == "child" {
		return r.reconcileChildClusterRole(ctx, clusterDeployment)
	} else if role == "regional" {
		return r.reconcileRegionalClusterRole(ctx, clusterDeployment)
	}
	return nil
}

func (r *ClusterDeploymentReconciler) reconcileChildClusterRole(
	ctx context.Context,
	childClusterDeployment *kcmv1beta1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)

	configMap := &corev1.ConfigMap{}
	configMapName := "kof-cluster-config-" + childClusterDeployment.Name
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: childClusterDeployment.Namespace,
	}, configMap)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(
			err, "cannot read existing child cluster ConfigMap",
			"configMapName", configMapName,
		)
		return err
	}
	if err == nil {
		// Logging nothing as we have a lot of frequent `status` updates to ignore here.
		// Cannot add `WithEventFilter(predicate.GenerationChangedPredicate{})`
		// to `SetupWithManager` of reconciler shared with istio which needs `status` updates.
		return nil
	}

	regionalClusterName, ok := childClusterDeployment.Labels[KofRegionalClusterNameLabel]
	regionalClusterDeployment := &kcmv1beta1.ClusterDeployment{}
	if ok {
		err := r.Get(ctx, types.NamespacedName{
			Name:      regionalClusterName,
			Namespace: childClusterDeployment.Namespace,
		}, regionalClusterDeployment)
		if err != nil {
			log.Error(
				err, "cannot get regional ClusterDeployment",
				"regionalClusterName", regionalClusterName,
			)
			return err
		}
	} else {
		var err error
		if regionalClusterDeployment, err = r.discoverRegionalClusterDeploymentByLocation(
			ctx,
			childClusterDeployment,
		); err != nil {
			log.Error(
				err, "regional ClusterDeployment not found both by label and by location",
				"childClusterDeploymentName", childClusterDeployment.Name,
				"clusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return err
		}
		regionalClusterName = regionalClusterDeployment.Name
	}

	ownerReference, err := utils.GetOwnerReference(childClusterDeployment, r.Client)
	if err != nil {
		log.Error(
			err, "cannot get owner reference from child ClusterDeployment",
			"childClusterDeploymentName", childClusterDeployment.Name,
		)
		return err
	}

	if _, isIstio := childClusterDeployment.Labels[IstioRoleLabel]; isIstio {
		if err := r.createProfile(
			ctx,
			ownerReference,
			childClusterDeployment,
			regionalClusterDeployment,
		); err != nil {
			log.Error(err, "cannot create profile")
			return err
		}
	}

	configData := map[string]string{RegionalClusterNameKey: regionalClusterName}

	if _, isIstio := regionalClusterDeployment.Labels[IstioRoleLabel]; !isIstio {
		regionalClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
			regionalClusterDeployment.Spec.Config.Raw,
		)
		if err != nil {
			log.Error(
				err, "cannot read regional ClusterDeployment config",
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
			)
			return err
		}

		configData[ReadMetricsKey], err = getEndpoint(ctx, ReadMetricsAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
		if err != nil {
			return err
		}

		configData[WriteMetricsKey], err = getEndpoint(ctx, WriteMetricsAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
		if err != nil {
			return err
		}

		configData[WriteLogsKey], err = getEndpoint(ctx, WriteLogsAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
		if err != nil {
			return err
		}

		configData[WriteTracesKey], err = getEndpoint(ctx, WriteTracesAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
		if err != nil {
			return err
		}
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            configMapName,
			Namespace:       childClusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
		},
		Data: configData,
	}

	if err := r.createIfNotExists(ctx, configMap, "child cluster ConfigMap", []any{
		"configMapName", configMap.Name,
		"configMapData", configData,
	}); err != nil {
		utils.LogEvent(
			ctx,
			"ConfigMapCreationFailed",
			"Failed to create child cluster ConfigMap",
			childClusterDeployment,
			err,
			"configMapName", configMap.Name,
			"configMapData", configData,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"ConfigMapCreated",
		"Created child cluster ConfigMap",
		childClusterDeployment,
		nil,
		"configMapName", configMap.Name,
		"configMapData", configData,
	)
	return nil
}

func (r *ClusterDeploymentReconciler) createProfile(
	ctx context.Context,
	ownerReference metav1.OwnerReference,
	childClusterDeployment, regionalClusterDeployment *kcmv1beta1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)
	remoteSecretName := remotesecret.GetRemoteSecretName(regionalClusterDeployment.Name)

	log.Info("Creating profile")

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:            remotesecret.CopyRemoteSecretProfileName(childClusterDeployment.Name),
			Namespace:       childClusterDeployment.Namespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Spec: sveltosv1beta1.Spec{
			ClusterRefs: []corev1.ObjectReference{
				{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       clusterv1.ClusterKind,
					Name:       childClusterDeployment.Name,
					Namespace:  childClusterDeployment.Namespace,
				},
			},
			TemplateResourceRefs: []sveltosv1beta1.TemplateResourceRef{
				{
					Identifier: "Secret",
					Resource: corev1.ObjectReference{
						APIVersion: corev1.SchemeGroupVersion.Version,
						Kind:       "Secret",
						Name:       remoteSecretName,
						Namespace:  istio.IstioSystemNamespace,
					},
				},
			},
			PolicyRefs: []sveltosv1beta1.PolicyRef{
				{
					Kind:      "ConfigMap",
					Name:      KofIstioSecretTemplate,
					Namespace: istio.IstioSystemNamespace,
				},
			},
		},
	}

	if err := r.createIfNotExists(ctx, profile, "Profile", []any{
		"profileName", profile.Name,
	}); err != nil {
		utils.LogEvent(
			ctx,
			"ProfileCreationFailed",
			"Failed to create Profile",
			regionalClusterDeployment,
			err,
			"profileName", profile.Name,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"ProfileCreated",
		"Copy remote secret Profile is successfully created",
		regionalClusterDeployment,
		nil,
		"profileName", profile.Name,
	)

	return nil
}

func getCloud(clusterDeployment *kcmv1beta1.ClusterDeployment) string {
	cloud, _, _ := strings.Cut(clusterDeployment.Spec.Template, "-")
	return cloud
}

func (r *ClusterDeploymentReconciler) discoverRegionalClusterDeploymentByLocation(
	ctx context.Context,
	childClusterDeployment *kcmv1beta1.ClusterDeployment,
) (*kcmv1beta1.ClusterDeployment, error) {
	log := log.FromContext(ctx)
	childCloud := getCloud(childClusterDeployment)

	childClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
		childClusterDeployment.Spec.Config.Raw,
	)
	if err != nil || childClusterDeploymentConfig == nil {
		log.Error(
			err, "cannot read child ClusterDeployment config",
			"childClusterDeploymentName", childClusterDeployment.Name,
		)
		return nil, err
	}

	regionalClusterDeploymentList := &kcmv1beta1.ClusterDeploymentList{}
	for {
		opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "regional"}}
		if regionalClusterDeploymentList.Continue != "" {
			opts = append(opts, client.Continue(regionalClusterDeploymentList.Continue))
		}

		if err := r.List(ctx, regionalClusterDeploymentList, opts...); err != nil {
			log.Error(err, "cannot list regional ClusterDeployments")
			return nil, err
		}

		for _, regionalClusterDeployment := range regionalClusterDeploymentList.Items {
			if childCloud != getCloud(&regionalClusterDeployment) {
				continue
			}

			regionalClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
				regionalClusterDeployment.Spec.Config.Raw,
			)
			if err != nil {
				continue
			}

			if locationIsTheSame(
				childCloud,
				childClusterDeploymentConfig,
				regionalClusterDeploymentConfig,
			) {
				return &regionalClusterDeployment, nil
			}
		}

		if regionalClusterDeploymentList.Continue == "" {
			break
		}
	}

	err = fmt.Errorf(
		"regional ClusterDeployment with matching location is not found, "+
			`please set .metadata.labels["%s"] explicitly`,
		KofRegionalClusterNameLabel,
	)
	utils.LogEvent(
		ctx,
		"RegionalClusterDiscoveryFailed",
		"Failed to discover regional cluster",
		childClusterDeployment,
		err,
		"childClusterDeploymentName", childClusterDeployment.Name,
	)
	return nil, err
}

func locationIsTheSame(cloud string, c1, c2 *ClusterDeploymentConfig) bool {
	switch cloud {
	case "adopted":
		return false
	case "aws":
		return c1.Region == c2.Region
	case "azure":
		return c1.Location == c2.Location
	case "docker":
		return true
	case "openstack":
		return c1.IdentityRef.Region == c2.IdentityRef.Region
	case "remote":
		return false
	case "vsphere":
		return c1.VSphere.Datacenter == c2.VSphere.Datacenter
	}

	return false
}

func getEndpoint(
	ctx context.Context,
	endpointAnnotation string,
	regionalClusterDeployment *kcmv1beta1.ClusterDeployment,
	regionalClusterDeploymentConfig *ClusterDeploymentConfig,
) (string, error) {
	log := log.FromContext(ctx)
	regionalClusterName := regionalClusterDeployment.Name
	_, isIstio := regionalClusterDeployment.Labels[IstioRoleLabel]
	regionalAnnotations := regionalClusterDeploymentConfig.ClusterAnnotations
	regionalDomain, hasRegionalDomain := regionalAnnotations[KofRegionalDomainAnnotation]

	endpoint, ok := regionalAnnotations[endpointAnnotation]
	if !ok {
		if isIstio {
			endpoint = fmt.Sprintf(istioEndpoints[endpointAnnotation], regionalClusterName)
		} else if hasRegionalDomain {
			endpoint = fmt.Sprintf(defaultEndpoints[endpointAnnotation], regionalDomain)
		} else {
			err := fmt.Errorf("neither endpoint nor regional domain is set")
			log.Error(
				err, "in",
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
				"endpointAnnotation", endpointAnnotation,
				"regionalDomainAnnotation", KofRegionalDomainAnnotation,
			)
			return "", err
		}
	}
	return endpoint, nil
}

func (r *ClusterDeploymentReconciler) reconcileRegionalClusterRole(
	ctx context.Context,
	regionalClusterDeployment *kcmv1beta1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)
	regionalClusterName := regionalClusterDeployment.Name

	releaseNamespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}

	ownerReference, err := utils.GetOwnerReference(regionalClusterDeployment, r.Client)
	if err != nil {
		log.Error(
			err, "cannot get owner reference from regional ClusterDeployment",
			"regionalClusterDeploymentName", regionalClusterDeployment.Name,
		)
		return err
	}
	vmRulesConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       regionalClusterDeployment.Namespace,
			Name:            "kof-record-vmrules-" + regionalClusterName,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
			Labels: map[string]string{
				KofRecordVMRulesClusterNameLabel: regionalClusterName,
				utils.ManagedByLabel:             utils.ManagedByValue,
				utils.KofGeneratedLabel:          "true",
			},
		},
	}
	if err := r.createIfNotExists(ctx, vmRulesConfigMap, "VMRulesConfigMap", []any{
		"configMapName", vmRulesConfigMap.Name,
	}); err != nil {
		return err
	}

	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
	grafanaDatasourceName := regionalClusterName + "-logs"
	err = r.Get(ctx, types.NamespacedName{
		Name:      grafanaDatasourceName,
		Namespace: releaseNamespace,
	}, grafanaDatasource)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(
			err, "cannot read existing GrafanaDatasource",
			"grafanaDatasourceName", grafanaDatasourceName,
		)
		return err
	}
	if err == nil {
		log.Info(
			"Found existing regional objects",
			"grafanaDatasourceName", grafanaDatasourceName,
		)
		return nil
	}

	regionalClusterDeploymentConfig, err := ReadClusterDeploymentConfig(
		regionalClusterDeployment.Spec.Config.Raw,
	)
	if err != nil {
		log.Error(
			err, "cannot read regional ClusterDeployment config",
			"regionalClusterDeploymentName", regionalClusterDeployment.Name,
		)
		return err
	}

	logsEndpoint, err := getEndpoint(ctx, ReadLogsAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
	if err != nil {
		return err
	}

	metricsEndpoint, err := getEndpoint(ctx, ReadMetricsAnnotation, regionalClusterDeployment, regionalClusterDeploymentConfig)
	if err != nil {
		return err
	}

	metricsURL, err := url.Parse(metricsEndpoint)
	if err != nil {
		log.Error(
			err, "cannot parse metrics endpoint",
			"regionalClusterDeploymentName", regionalClusterDeployment.Name,
			"metricsEndpointAnnotation", ReadMetricsAnnotation,
			"metricsEndpointValue", metricsEndpoint,
		)
		return err
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
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
				"metricsEndpointAnnotation", ReadMetricsAnnotation,
				"metricsEndpointValue", metricsEndpoint,
			)
			return err
		}
	}

	metricsTarget := fmt.Sprintf("%s:%s", metricsURL.Hostname(), metricsPort)
	_, isIstio := regionalClusterDeployment.Labels[IstioRoleLabel]

	promxyServerGroup := &kofv1beta1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      regionalClusterName + "-metrics",
			Namespace: releaseNamespace,
			// `OwnerReferences` is N/A because `regionalClusterDeployment` namespace differs.
			Labels: map[string]string{
				utils.ManagedByLabel:  utils.ManagedByValue,
				PromxySecretNameLabel: "kof-mothership-promxy-config",
			},
		},
		Spec: kofv1beta1.PromxyServerGroupSpec{
			ClusterName: regionalClusterName,
			Scheme:      metricsURL.Scheme,
			Targets:     []string{metricsTarget},
			PathPrefix:  metricsURL.EscapedPath(),
			HttpClient: kofv1beta1.HTTPClientConfig{
				DialTimeout: defaultDialTimeout,
			},
		},
	}

	var httpClientConfig *kofv1beta1.HTTPClientConfig
	if httpConfigJson, ok := regionalClusterDeployment.Annotations[KofRegionalHTTPClientConfigAnnotation]; ok {
		httpClientConfig = &kofv1beta1.HTTPClientConfig{
			DialTimeout: defaultDialTimeout,
		}
		if err := json.Unmarshal([]byte(httpConfigJson), httpClientConfig); err != nil {
			utils.LogEvent(
				ctx,
				"InvalidRegionalHTTPClientConfigAnnotation",
				"Failed to parse JSON from annotation",
				regionalClusterDeployment,
				err,
				"annotation", KofRegionalHTTPClientConfigAnnotation,
				"value", httpConfigJson,
			)
			return err
		}
		promxyServerGroup.Spec.HttpClient = *httpClientConfig
	}
	if !isIstio {
		basicAuth := &promxyServerGroup.Spec.HttpClient.BasicAuth
		basicAuth.CredentialsSecretName = KofStorageSecretName
		basicAuth.UsernameKey = "username"
		basicAuth.PasswordKey = "password"
	}

	if err := r.createIfNotExists(ctx, promxyServerGroup, "PromxyServerGroup", []any{
		"promxyServerGroupName", promxyServerGroup.Name,
	}); err != nil {
		utils.LogEvent(
			ctx,
			"PromxySeverGroupCreationFailed",
			"Failed to create PromxyServerGroup",
			regionalClusterDeployment,
			err,
			"promxyServerGroupName", promxyServerGroup.Name,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"PromxyServerGroupCreated",
		"PromxyServerGroup is successfully created",
		regionalClusterDeployment,
		nil,
		"promxyServerGroupName", promxyServerGroup.Name,
	)

	grafanaDatasource = &grafanav1beta1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      grafanaDatasourceName,
			Namespace: releaseNamespace,
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
				Name:      regionalClusterName,
				Type:      "victoriametrics-logs-datasource",
				URL:       logsEndpoint,
				Access:    "proxy",
				IsDefault: utils.BoolPtr(false),
				BasicAuth: utils.BoolPtr(!isIstio),
			},
		},
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

	if err := r.createIfNotExists(ctx, grafanaDatasource, "GrafanaDatasource", []any{
		"grafanaDatasourceName", grafanaDatasource.Name,
	}); err != nil {
		utils.LogEvent(
			ctx,
			"GrafanaDatasourceCreationFailed",
			"Failed to create GrafanaDatasource",
			regionalClusterDeployment,
			err,
			"grafanaDatasourceName", grafanaDatasource.Name,
		)
		return err
	}

	utils.LogEvent(
		ctx,
		"GrafanaDatasourceCreated",
		"GrafanaDatasource is successfully created",
		regionalClusterDeployment,
		nil,
		"grafanaDatasourceName", grafanaDatasource.Name,
	)

	return nil
}

func (r *ClusterDeploymentReconciler) createIfNotExists(
	ctx context.Context,
	object client.Object,
	objectDescription string,
	details []any,
) error {
	log := log.FromContext(ctx)

	// `createOrUpdate` would need to read an old version and merge it with the new version
	// to avoid `metadata.resourceVersion: Invalid value: 0x0: must be specified for an update`.
	// As we have immutable specs for now, we will use `createIfNotExists` instead.

	if err := r.Create(ctx, object); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Found existing "+objectDescription, details...)
			return nil
		}

		log.Error(err, "cannot create "+objectDescription, details...)
		return err
	}

	log.Info("Created "+objectDescription, details...)
	return nil
}
