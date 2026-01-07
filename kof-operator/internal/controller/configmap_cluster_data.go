package controller

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigData struct {
	RegionalClusterName      string
	RegionalClusterNamespace string
	RegionalClusterCloud     string
	IstioRole                string
	RegionalHTTPClientConfig string

	ReadMetricsEndpoint  string
	ReadLogsEndpoint     string
	ReadTracesEndpoint   string
	WriteMetricsEndpoint string
	WriteLogsEndpoint    string
	WriteTracesEndpoint  string

	AWSRegion         string
	AzureLocation     string
	OpenstackRegion   string
	VSphereDatacenter string
}

func NewConfigDataFromClusterDeployment(ctx context.Context, client client.Client, cd *kcmv1beta1.ClusterDeployment) (*ConfigData, error) {
	var err error

	if cd == nil {
		return nil, fmt.Errorf("cluster deployment is nil")
	}

	cdConfig, err := ReadClusterDeploymentConfig(cd.Spec.Config.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster deployment config: %v", err)
	}

	childCloud, err := getCloud(ctx, client, cd)
	if err != nil {
		return nil, fmt.Errorf("failed to get child cluster cloud: %v", err)
	}

	config := &ConfigData{
		RegionalClusterName:      cd.Name,
		RegionalClusterNamespace: cd.Namespace,
		RegionalClusterCloud:     childCloud,
		RegionalHTTPClientConfig: cd.Annotations[KofRegionalHTTPClientConfigAnnotation],

		AWSRegion:         cdConfig.Region,
		AzureLocation:     cdConfig.Location,
		OpenstackRegion:   cdConfig.IdentityRef.Region,
		VSphereDatacenter: cdConfig.VSphere.Datacenter,
	}

	if config.ReadMetricsEndpoint, err = getEndpoint(ctx, ReadMetricsAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	if config.ReadLogsEndpoint, err = getEndpoint(ctx, ReadLogsAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	if config.ReadTracesEndpoint, err = getEndpoint(ctx, ReadTracesAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	if value, isIstio := cd.Labels[IstioRoleLabel]; isIstio {
		config.IstioRole = value
		return config, nil
	}

	if config.WriteMetricsEndpoint, err = getEndpoint(ctx, WriteMetricsAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	if config.WriteLogsEndpoint, err = getEndpoint(ctx, WriteLogsAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	if config.WriteTracesEndpoint, err = getEndpoint(ctx, WriteTracesAnnotation, cd, cdConfig); err != nil {
		return nil, err
	}

	return config, nil
}

func NewConfigDataFromConfigMap(cm *corev1.ConfigMap) (*ConfigData, error) {
	if cm == nil {
		return nil, fmt.Errorf("configmap is nil")
	}

	return &ConfigData{
		RegionalClusterName:      cm.Data[RegionalClusterNameKey],
		RegionalClusterNamespace: cm.Data[RegionalClusterNamespaceKey],
		RegionalClusterCloud:     cm.Data[RegionalClusterCloudKey],
		IstioRole:                cm.Data[RegionalIstioRoleKey],
		RegionalHTTPClientConfig: cm.Data[RegionalKofHTTPConfigKey],

		ReadMetricsEndpoint:  cm.Data[ReadMetricsKey],
		ReadLogsEndpoint:     cm.Data[ReadLogsKey],
		ReadTracesEndpoint:   cm.Data[ReadTracesKey],
		WriteMetricsEndpoint: cm.Data[WriteMetricsKey],
		WriteLogsEndpoint:    cm.Data[WriteLogsKey],
		WriteTracesEndpoint:  cm.Data[WriteTracesKey],

		AWSRegion:         cm.Data[AwsRegionKey],
		AzureLocation:     cm.Data[AzureLocationKey],
		OpenstackRegion:   cm.Data[OpenstackRegionKey],
		VSphereDatacenter: cm.Data[VSphereDatacenterKey],
	}, nil
}

func (c *ConfigData) ToClusterDeploymentConfig() *ClusterDeploymentConfig {
	return &ClusterDeploymentConfig{
		Region:   c.AWSRegion,
		Location: c.AzureLocation,
		IdentityRef: IdentityRef{
			Region: c.OpenstackRegion,
		},
		VSphere: VSphere{
			Datacenter: c.VSphereDatacenter,
		},
	}
}

func (c *ConfigData) ToMap() map[string]string {
	return map[string]string{
		RegionalClusterNameKey:      c.RegionalClusterName,
		RegionalClusterNamespaceKey: c.RegionalClusterNamespace,
		RegionalClusterCloudKey:     c.RegionalClusterCloud,
		RegionalIstioRoleKey:        c.IstioRole,
		RegionalKofHTTPConfigKey:    c.RegionalHTTPClientConfig,

		ReadMetricsKey:  c.ReadMetricsEndpoint,
		ReadLogsKey:     c.ReadLogsEndpoint,
		ReadTracesKey:   c.ReadTracesEndpoint,
		WriteMetricsKey: c.WriteMetricsEndpoint,
		WriteLogsKey:    c.WriteLogsEndpoint,
		WriteTracesKey:  c.WriteTracesEndpoint,

		AwsRegionKey:         c.AWSRegion,
		AzureLocationKey:     c.AzureLocation,
		OpenstackRegionKey:   c.OpenstackRegion,
		VSphereDatacenterKey: c.VSphereDatacenter,
	}
}
