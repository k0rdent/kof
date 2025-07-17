package controller

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            GetRegionalClusterConfigMapName(r.clusterName),
			Namespace:       r.clusterDeployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{*r.ownerReference},
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue, utils.KofGeneratedLabel: "true", KofRegionalClusterConfigMapLabel: r.clusterName},
		},
		Data: configData.ToMap(),
	}
	operation := "Create"
	if err = r.client.Create(r.ctx, cm); err != nil && errors.IsAlreadyExists(err) {
		err = r.client.Update(r.ctx, cm)
		operation = "Update"
	}
	eventName := "RegionalClusterConfigMap" + operation
	if err != nil {
		utils.LogEvent(
			r.ctx,
			eventName+"Failed",
			"Failed to "+operation+" RegionalClusterConfigMap",
			r.clusterDeployment,
			err,
			"configMap", cm.Name,
		)
		return err
	}

	utils.LogEvent(
		r.ctx,
		eventName,
		fmt.Sprintf("%sd RegionalClusterConfigMap", operation),
		r.clusterDeployment,
		nil,
		"configMap", cm.Name,
	)

	return nil
}

func GetRegionalClusterConfigMapName(clusterName string) string {
	return "kof-" + clusterName
}
