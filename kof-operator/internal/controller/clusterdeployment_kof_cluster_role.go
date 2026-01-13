package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/cloud"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const prefix = "k0rdent.mirantis.com/"

// Labels:
const KofClusterRoleLabel = prefix + "kof-cluster-role"
const KofRegionalClusterNameLabel = prefix + "kof-regional-cluster-name"
const KofRegionalClusterNamespaceLabel = prefix + "kof-regional-cluster-namespace"

// Annotations:
const KofRegionalDomainAnnotation = prefix + "kof-regional-domain"
const KofRegionalHTTPClientConfigAnnotation = prefix + "kof-http-config"
const WriteMetricsAnnotation = prefix + "kof-write-metrics-endpoint"
const ReadMetricsAnnotation = prefix + "kof-read-metrics-endpoint"
const WriteLogsAnnotation = prefix + "kof-write-logs-endpoint"
const ReadLogsAnnotation = prefix + "kof-read-logs-endpoint"
const WriteTracesAnnotation = prefix + "kof-write-traces-endpoint"
const ReadTracesAnnotation = prefix + "kof-read-traces-endpoint"

// Endpoints for Sprintf:
var defaultEndpoints = map[string]string{
	WriteMetricsAnnotation: "https://vmauth.%s/vm/insert/0/prometheus/api/v1/write",
	ReadMetricsAnnotation:  "https://vmauth.%s/vm/select/0/prometheus",
	WriteLogsAnnotation:    "https://vmauth.%s/vli/insert/opentelemetry/v1/logs",
	ReadLogsAnnotation:     "https://vmauth.%s/vls",
	WriteTracesAnnotation:  "https://vmauth.%s/vti/insert/opentelemetry/v1/traces",
	ReadTracesAnnotation:   "https://vmauth.%s/vts/select/jaeger",
}
var istioEndpoints = map[string]string{
	ReadLogsAnnotation:    "http://%s-logs-select:9471",
	ReadMetricsAnnotation: "http://%s-vmselect:8481/select/0/prometheus",
	ReadTracesAnnotation:  "http://%s-traces-select:10471/select/jaeger",
}

// Child cluster ConfigMap data keys:
const RegionalClusterNameKey = "regional_cluster_name"
const RegionalClusterNamespaceKey = "regional_cluster_namespace"
const RegionalClusterCloudKey = "regional_cluster_cloud"
const RegionalIstioRoleKey = "istio_role"
const RegionalKofHTTPConfigKey = "kof_http_config"
const ReadMetricsKey = "read_metrics_endpoint"
const ReadLogsKey = "read_logs_endpoint"
const ReadTracesKey = "read_traces_endpoint"
const WriteMetricsKey = "write_metrics_endpoint"
const WriteLogsKey = "write_logs_endpoint"
const WriteTracesKey = "write_traces_endpoint"
const AwsRegionKey = "aws_region"
const AzureLocationKey = "azure_location"
const OpenstackRegionKey = "openstack_region"
const VSphereDatacenterKey = "vsphere_datacenter"

const KofRoleChild = "child"
const KofRoleRegional = "regional"

var defaultDialTimeout = metav1.Duration{Duration: time.Second * 5}

func (r *ClusterDeploymentReconciler) ReconcileKofClusterRole(
	ctx context.Context,
	clusterDeployment *kcmv1beta1.ClusterDeployment,
) error {
	role := clusterDeployment.Labels[KofClusterRoleLabel]
	switch role {
	case KofRoleChild:
		childClusterRole, err := NewChildClusterRole(ctx, clusterDeployment, r.Client)
		if err != nil {
			return err
		}
		return childClusterRole.Reconcile()
	case KofRoleRegional:
		regionalClusterRole, err := NewRegionalClusterRole(ctx, clusterDeployment, r.Client)
		if err != nil {
			return err
		}
		return regionalClusterRole.Reconcile()
	}
	return nil
}

func getCloud(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (string, error) {
	clusterTemplate, err := k8s.GetClusterTemplate(ctx, client, cd.Spec.Template, cd.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster template: %w", err)
	}

	for _, provider := range clusterTemplate.Status.Providers {
		prefix, cloudName, found := strings.Cut(provider, "-")
		if !found {
			continue
		}

		if prefix != "infrastructure" {
			continue
		}

		if cloud.IsValidName(cloudName) {
			return cloudName, nil
		}
	}

	return "", fmt.Errorf("no valid cloud provider found in ClusterTemplate %s", cd.Spec.Template)
}

func locationIsTheSame(cloudName string, c1, c2 *ClusterDeploymentConfig) bool {
	switch cloudName {
	case cloud.Adopted:
		return false
	case cloud.AWS:
		return c1.Region == c2.Region
	case cloud.Azure:
		return c1.Location == c2.Location
	case cloud.Docker:
		return true
	case cloud.OpenStack:
		return c1.IdentityRef.Region == c2.IdentityRef.Region
	case cloud.Remote:
		return false
	case cloud.VSphere:
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
