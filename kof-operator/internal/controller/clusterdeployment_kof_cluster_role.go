package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1alpha1 "github.com/k0rdent/kof/kof-operator/api/v1alpha1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/isito"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Labels:
const KofClusterRoleLabel = "k0rdent.mirantis.com/kof-cluster-role"
const KofRegionalClusterNameLabel = "k0rdent.mirantis.com/kof-regional-cluster-name"

// Annotations:
const KofLogsEndpointAnnotation = "k0rdent.mirantis.com/kof-logs-endpoint"
const KofMetricsEndpointAnnotation = "k0rdent.mirantis.com/kof-metrics-endpoint"
const KofRegionalDomainAnnotation = "k0rdent.mirantis.com/kof-regional-domain"

// ConfigMap data keys:
const RegionalClusterNameKey = "regional_cluster_name"
const RegionalDomainKey = "regional_domain"

// Other:
const KofStorageSecretName = "storage-vmuser-credentials"
const KofIstioSecretTemplate = "kof-istio-secret-template"

func (r *ClusterDeploymentReconciler) ReconcileKofClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1alpha1.ClusterDeployment,
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
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
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
	regionalClusterDeployment := &kcmv1alpha1.ClusterDeployment{}
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

	ownerReference, err := GetOwnerReference(childClusterDeployment, r.Client)
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

		regionalAnnotations := regionalClusterDeploymentConfig.ClusterAnnotations
		regionalDomain, ok := regionalAnnotations[KofRegionalDomainAnnotation]
		if !ok {
			err := fmt.Errorf("regional domain not found")
			log.Error(
				err, "in",
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
				"clusterAnnotation", KofRegionalDomainAnnotation,
			)
			return err
		}

		configData[RegionalDomainKey] = regionalDomain
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            configMapName,
			Namespace:       childClusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
			Labels:          map[string]string{ManagedByLabel: ManagedByValue},
		},
		Data: configData,
	}

	if err := r.createIfNotExists(ctx, configMap, "child cluster ConfigMap", []any{
		"configMapName", configMap.Name,
		"configMapData", configData,
	}); err != nil {
		return err
	}

	return nil
}

func (r *ClusterDeploymentReconciler) createProfile(
	ctx context.Context,
	ownerReference metav1.OwnerReference,
	childClusterDeployment, regionalClusterDeployment *kcmv1alpha1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)
	remoteSecretName := istio.RemoteSecretNameFromClusterName(regionalClusterDeployment.Name)

	log.Info("Creating profile")

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:            istio.CopyRemoteSecretProfileName(childClusterDeployment.Name),
			Namespace:       childClusterDeployment.Namespace,
			Labels:          map[string]string{ManagedByLabel: ManagedByValue},
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
		return err
	}

	return nil
}

func getCloud(clusterDeployment *kcmv1alpha1.ClusterDeployment) string {
	cloud, _, _ := strings.Cut(clusterDeployment.Spec.Template, "-")
	return cloud
}

func (r *ClusterDeploymentReconciler) discoverRegionalClusterDeploymentByLocation(
	ctx context.Context,
	childClusterDeployment *kcmv1alpha1.ClusterDeployment,
) (*kcmv1alpha1.ClusterDeployment, error) {
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

	regionalClusterDeploymentList := &kcmv1alpha1.ClusterDeploymentList{}
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

	return nil, fmt.Errorf(
		"regional ClusterDeployment with matching location is not found, "+
			`please set .metadata.labels["%s"] explicitly`,
		KofRegionalClusterNameLabel,
	)
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

func (r *ClusterDeploymentReconciler) reconcileRegionalClusterRole(
	ctx context.Context,
	regionalClusterDeployment *kcmv1alpha1.ClusterDeployment,
) error {
	log := log.FromContext(ctx)
	regionalClusterName := regionalClusterDeployment.Name

	releaseNamespace, ok := os.LookupEnv("RELEASE_NAMESPACE")
	if !ok {
		return fmt.Errorf("required RELEASE_NAMESPACE env var is not set")
	}

	grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
	grafanaDatasourceName := regionalClusterName + "-logs"
	err := r.Get(ctx, types.NamespacedName{
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

	regionalAnnotations := regionalClusterDeploymentConfig.ClusterAnnotations
	regionalDomain, hasRegionalDomain := regionalAnnotations[KofRegionalDomainAnnotation]
	_, isIstio := regionalClusterDeployment.Labels[IstioRoleLabel]

	logsEndpoint, ok := regionalAnnotations[KofLogsEndpointAnnotation]
	if !ok {
		if isIstio {
			logsEndpoint = fmt.Sprintf(
				"http://%s-logs:9428/insert/opentelemetry/v1/logs",
				regionalClusterName,
			)
		} else if hasRegionalDomain {
			logsEndpoint = fmt.Sprintf("https://vmauth.%s/vls", regionalDomain)
		} else {
			err := fmt.Errorf("neither logs endpoint nor regional domain is set")
			log.Error(
				err, "in",
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
				"logsEndpointAnnotation", KofLogsEndpointAnnotation,
				"regionalDomainAnnotation", KofRegionalDomainAnnotation,
			)
			return err
		}
	}

	metricsEndpoint, ok := regionalAnnotations[KofMetricsEndpointAnnotation]
	if !ok {
		if isIstio {
			metricsEndpoint = fmt.Sprintf(
				"http://%s-vmselect:8481/select/0/prometheus",
				regionalClusterName,
			)
		} else if hasRegionalDomain {
			metricsEndpoint = fmt.Sprintf(
				"https://vmauth.%s/vm/select/0/prometheus",
				regionalDomain,
			)
		} else {
			err := fmt.Errorf("neither metrics endpoint nor regional domain is set")
			log.Error(
				err, "in",
				"regionalClusterDeploymentName", regionalClusterDeployment.Name,
				"metricsEndpointAnnotation", KofMetricsEndpointAnnotation,
				"regionalDomainAnnotation", KofRegionalDomainAnnotation,
			)
			return err
		}
	}

	metricsURL, err := url.Parse(metricsEndpoint)
	if err != nil {
		log.Error(
			err, "cannot parse metrics endpoint",
			"regionalClusterDeploymentName", regionalClusterDeployment.Name,
			"metricsEndpointAnnotation", KofMetricsEndpointAnnotation,
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
				"metricsEndpointAnnotation", KofMetricsEndpointAnnotation,
				"metricsEndpointValue", metricsEndpoint,
			)
			return err
		}
	}

	metricsTarget := fmt.Sprintf("%s:%s", metricsURL.Hostname(), metricsPort)

	promxyServerGroup := &kofv1alpha1.PromxyServerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      regionalClusterName + "-metrics",
			Namespace: releaseNamespace,
			// `OwnerReferences` is N/A because `regionalClusterDeployment` namespace differs.
			Labels: map[string]string{
				ManagedByLabel:        ManagedByValue,
				PromxySecretNameLabel: "kof-mothership-promxy-config",
			},
		},
		Spec: kofv1alpha1.PromxyServerGroupSpec{
			ClusterName: regionalClusterName,
			Scheme:      metricsURL.Scheme,
			Targets:     []string{metricsTarget},
			PathPrefix:  metricsURL.EscapedPath(),
			HttpClient: kofv1alpha1.HTTPClientConfig{
				DialTimeout: metav1.Duration{Duration: 5 * time.Second},
			},
		},
	}
	if !isIstio {
		httpClient := promxyServerGroup.Spec.HttpClient
		httpClient.TLSConfig = kofv1alpha1.TLSConfig{InsecureSkipVerify: true}
		httpClient.BasicAuth = kofv1alpha1.BasicAuth{
			CredentialsSecretName: KofStorageSecretName,
			UsernameKey:           "username",
			PasswordKey:           "password",
		}
	}

	if err := r.createIfNotExists(ctx, promxyServerGroup, "PromxyServerGroup", []any{
		"promxyServerGroupName", promxyServerGroup.Name,
	}); err != nil {
		return err
	}

	grafanaDatasource = &grafanav1beta1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      grafanaDatasourceName,
			Namespace: releaseNamespace,
			// `OwnerReferences` is N/A because `regionalClusterDeployment` namespace differs.
			Labels: map[string]string{ManagedByLabel: ManagedByValue},
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
				IsDefault: BoolPtr(false),
				BasicAuth: BoolPtr(!isIstio),
			},
		},
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
		return err
	}

	return nil
}
