package controller

import (
	"context"
	"fmt"
	"maps"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vmRulesConfigMapNamePrefix = "kof-record-vmrules-"
)

type RegionalClusterRole struct {
	clusterName       string
	client            client.Client
	ctx               context.Context
	clusterDeployment *kcmv1beta1.ClusterDeployment
	ownerReference    *metav1.OwnerReference
}

func NewRegionalClusterRole(ctx context.Context, cd *kcmv1beta1.ClusterDeployment, client client.Client) (*RegionalClusterRole, error) {
	ownerReference, err := utils.GetOwnerReference(cd, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner reference: %v", err)
	}

	return &RegionalClusterRole{
		clusterName:       cd.Name,
		ctx:               ctx,
		clusterDeployment: cd,
		client:            client,
		ownerReference:    ownerReference,
	}, nil
}

func (r *RegionalClusterRole) Reconcile() error {
	if err := r.CreateOrUpdateRegionalConfigMap(); err != nil {
		return fmt.Errorf("failed to create or update regional cluster ConfigMap: %v", err)
	}

	return nil
}

func (r *RegionalClusterRole) CreateOrUpdateRegionalConfigMap() error {
	configData, err := NewConfigDataFromClusterDeployment(r.ctx, r.clusterDeployment)
	if err != nil {
		return fmt.Errorf("failed to get config data: %v", err)
	}

	cm, err := r.GetConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	if cm == nil {
		if err := r.CreateConfigMap(configData); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %v", err)
		}
		return nil
	}

	if err := r.UpdateConfigMap(cm, configData); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %v", err)
	}

	return nil
}

func (r *RegionalClusterRole) GetConfigMap() (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	if err := r.client.Get(r.ctx, types.NamespacedName{
		Name:      GetRegionalClusterConfigMapName(r.clusterName),
		Namespace: r.clusterDeployment.Namespace,
	}, configMap); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return configMap, nil
}

func (r *RegionalClusterRole) CreateConfigMap(configData *ConfigData) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            GetRegionalClusterConfigMapName(r.clusterName),
			Namespace:       r.clusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{*r.ownerReference},
			Labels: map[string]string{
				utils.ManagedByLabel:    utils.ManagedByValue,
				utils.KofGeneratedLabel: "true",
				KofClusterRoleLabel:     KofRoleRegional,
			},
		},
		Data: configData.ToMap(),
	}

	if err := r.client.Create(r.ctx, cm); err != nil {
		utils.LogEvent(
			r.ctx,
			"ConfigMapCreationFailed",
			"Failed to create regional cluster ConfigMap",
			r.clusterDeployment,
			err,
			"configMapName", cm.Name,
			"configMapNamespace", cm.Namespace,
			"configMapData", configData,
		)
		return err
	}

	utils.LogEvent(
		r.ctx,
		"ConfigMapCreated",
		"Created regional cluster ConfigMap",
		r.clusterDeployment,
		nil,
		"configMapName", cm.Name,
		"configMapNamespace", cm.Namespace,
		"configMapData", configData,
	)

	return nil
}

func (r *RegionalClusterRole) UpdateConfigMap(cm *corev1.ConfigMap, configData *ConfigData) error {
	configDataMap := configData.ToMap()

	if maps.Equal(cm.Data, configDataMap) {
		return nil
	}

	cm.Data = configDataMap
	if err := r.client.Update(r.ctx, cm); err != nil {

		utils.LogEvent(
			r.ctx,
			"ConfigMapUpdateFailed",
			"Failed to update regional cluster ConfigMap",
			r.clusterDeployment,
			err,
			"configMapName", cm.Name,
			"configMapNamespace", cm.Namespace,
		)
		return err
	}

	utils.LogEvent(
		r.ctx,
		"ConfigMapUpdated",
		"Updated regional cluster ConfigMap",
		r.clusterDeployment,
		nil,
		"configMapName", cm.Name,
		"configMapNamespace", cm.Namespace,
	)

	return nil

}

func GetRegionalClusterConfigMapName(clusterName string) string {
	return "kof-" + clusterName
}
