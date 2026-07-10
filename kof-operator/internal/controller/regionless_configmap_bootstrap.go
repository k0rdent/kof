package controller

import (
	"context"
	"fmt"

	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	"github.com/k0rdent/kof/kof-operator/internal/env"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	"github.com/k0rdent/kof/kof-operator/internal/strutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateOrUpdateRegionlessConfigMap(
	ctx context.Context,
	client client.Client,
	managementClusterName string,
) error {
	if !env.RegionlessEnabled() {
		return nil
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetRegionalClusterConfigMapName(managementClusterName),
			Namespace: k8s.DefaultSystemNamespace,
		},
	}

	configData, err := NewRegionlessConfigData(managementClusterName, k8s.DefaultSystemNamespace)
	if err != nil {
		record.LogEvent(
			ctx,
			"RegionlessConfigMapUpdateFailed",
			"Failed to create or update regionless ConfigMap",
			cm,
			err,
			"configMapName", cm.Name,
			"configMapNamespace", cm.Namespace,
			"managementClusterName", managementClusterName,
		)
		return fmt.Errorf("failed to build regionless config data: %v", err)
	}

	result, err := controllerutil.CreateOrUpdate(ctx, client, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}
		cm.Labels[labels.ManagedByLabel] = k8s.ManagedByValue
		cm.Labels[labels.KofGeneratedLabel] = strutil.True
		cm.Labels[KofClusterRoleLabel] = KofRoleRegional
		cm.Labels[KofRegionlessLabel] = strutil.True
		cm.Data = configData.ToMap()
		return nil
	})
	if err != nil {
		record.LogEvent(
			ctx,
			"RegionlessConfigMapUpdateFailed",
			"Failed to create or update regionless ConfigMap",
			cm,
			err,
			"configMapName", cm.Name,
			"configMapNamespace", cm.Namespace,
			"managementClusterName", managementClusterName,
		)
		return fmt.Errorf("failed to create or update regionless regional ConfigMap: %v", err)
	}

	record.LogEvent(
		ctx,
		"RegionlessConfigMapUpdated",
		"Created or updated regionless ConfigMap",
		cm,
		nil,
		"configMapName", cm.Name,
		"configMapNamespace", cm.Namespace,
		"managementClusterName", managementClusterName,
		"operation", result,
	)

	return nil
}

func isRegionlessConfigMap(cm *corev1.ConfigMap) bool {
	if cm == nil {
		return false
	}
	return cm.Labels[KofRegionlessLabel] == strutil.True
}
