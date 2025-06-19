package controller

import (
	"context"
	"fmt"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
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

type ChildClusterRole struct {
	clusterName             string
	namespace               string
	client                  client.Client
	ctx                     context.Context
	clusterDeployment       *kcmv1beta1.ClusterDeployment
	clusterDeploymentConfig *ClusterDeploymentConfig
	ownerReference          metav1.OwnerReference
}

func NewChildClusterRole(ctx context.Context, cd *kcmv1beta1.ClusterDeployment, client client.Client) (*ChildClusterRole, error) {
	ownerReference, err := utils.GetOwnerReference(cd, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner reference: %v", err)
	}

	cdConfig, err := ReadClusterDeploymentConfig(cd.Spec.Config.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster deployment config: %v", err)
	}

	return &ChildClusterRole{
		ctx:                     ctx,
		clusterDeployment:       cd,
		clusterDeploymentConfig: cdConfig,
		clusterName:             cd.Name,
		client:                  client,
		namespace:               cd.Namespace,
		ownerReference:          ownerReference,
	}, nil
}

func (c *ChildClusterRole) Reconcile() error {
	exists, err := c.IsConfigMapExists()
	if err != nil {
		return fmt.Errorf("failed to check if config map exists: %v", err)
	}

	if exists {
		// Logging nothing as we have a lot of frequent `status` updates to ignore here.
		// Cannot add `WithEventFilter(predicate.GenerationChangedPredicate{})`
		// to `SetupWithManager` of reconciler shared with istio which needs `status` updates.
		return nil
	}

	regionalClusterDeployment, err := c.GetRegionalCluster()
	if err != nil {
		return fmt.Errorf("failed to get regional cluster: %v", err)
	}

	if c.IsIstioCluster() {
		if err := c.CreateProfile(regionalClusterDeployment); err != nil {
			return fmt.Errorf("failed to create profile: %v", err)
		}
	}

	configData, err := regionalClusterDeployment.GetConfigData()
	if err != nil {
		return fmt.Errorf("failed to get config data: %v", err)
	}

	if err := c.CreateOrUpdateConfigMap(configData); err != nil {
		return fmt.Errorf("failed to create config map: %v", err)
	}

	return nil
}

func (c *ChildClusterRole) IsConfigMapExists() (bool, error) {
	configMap, err := c.GetConfigMap()
	if err != nil {
		return false, err
	}
	if configMap == nil {
		return false, nil
	}
	return true, nil
}

func (c *ChildClusterRole) GetConfigMap() (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      GetConfigMapName(c.clusterName),
		Namespace: c.namespace,
	}, configMap); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return configMap, nil
}

func (c *ChildClusterRole) GetRegionalCluster() (*RegionalClusterRole, error) {
	log := log.FromContext(c.ctx)
	regionalClusterName := c.GetRegionalClusterName()
	regionalClusterDeployment := &kcmv1beta1.ClusterDeployment{}

	if len(regionalClusterName) > 0 {
		err := c.client.Get(c.ctx, types.NamespacedName{
			Name:      regionalClusterName,
			Namespace: c.namespace,
		}, regionalClusterDeployment)
		if err != nil {
			log.Error(
				err, "cannot get regional ClusterDeployment",
				"regionalClusterName", regionalClusterName,
			)
			return nil, err
		}
	} else {
		var err error
		if regionalClusterDeployment, err = c.DiscoverRegionalClusterDeploymentByLocation(); err != nil {
			log.Error(
				err, "regional ClusterDeployment not found both by label and by location",
				"childClusterDeploymentName", c.clusterName,
				"clusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return nil, err
		}
	}

	return NewRegionalClusterRole(c.ctx, regionalClusterDeployment, c.client)
}

func (c *ChildClusterRole) DiscoverRegionalClusterDeploymentByLocation() (*kcmv1beta1.ClusterDeployment, error) {
	log := log.FromContext(c.ctx)
	childCloud := getCloud(c.clusterDeployment)

	regionalClusterDeploymentList := &kcmv1beta1.ClusterDeploymentList{}
	for {
		opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "regional"}}
		if regionalClusterDeploymentList.Continue != "" {
			opts = append(opts, client.Continue(regionalClusterDeploymentList.Continue))
		}

		if err := c.client.List(c.ctx, regionalClusterDeploymentList, opts...); err != nil {
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
				c.clusterDeploymentConfig,
				regionalClusterDeploymentConfig,
			) {
				return &regionalClusterDeployment, nil
			}
		}

		if regionalClusterDeploymentList.Continue == "" {
			break
		}
	}

	err := fmt.Errorf(
		"regional ClusterDeployment with matching location is not found, "+
			`please set .metadata.labels["%s"] explicitly`,
		KofRegionalClusterNameLabel,
	)
	utils.LogEvent(
		c.ctx,
		"RegionalClusterDiscoveryFailed",
		"Failed to discover regional cluster",
		c.clusterDeployment,
		err,
		"childClusterDeploymentName", c.clusterName,
	)
	return nil, err
}

func (c *ChildClusterRole) CreateProfile(regionalCD *RegionalClusterRole) error {
	log := log.FromContext(c.ctx)
	remoteSecretName := remotesecret.GetRemoteSecretName(regionalCD.clusterName)

	log.Info("Creating profile")

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:            remotesecret.CopyRemoteSecretProfileName(c.clusterName),
			Namespace:       c.namespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{c.ownerReference},
		},
		Spec: sveltosv1beta1.Spec{
			ClusterRefs: []corev1.ObjectReference{
				{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       clusterv1.ClusterKind,
					Name:       c.clusterName,
					Namespace:  c.namespace,
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

	if err := utils.CreateIfNotExists(c.ctx, c.client, profile, "Profile", []any{
		"profileName", profile.Name,
	}); err != nil {
		utils.LogEvent(
			c.ctx,
			"ProfileCreationFailed",
			"Failed to create Profile",
			regionalCD.clusterDeployment,
			err,
			"profileName", profile.Name,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"ProfileCreated",
		"Copy remote secret Profile is successfully created",
		regionalCD.clusterDeployment,
		nil,
		"profileName", profile.Name,
	)

	return nil
}

func (c *ChildClusterRole) CreateOrUpdateConfigMap(newConfigData map[string]string) error {
	configMap, err := c.GetConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get config map: %v", err)
	}

	if configMap == nil {
		if err := c.CreateConfigMap(newConfigData); err != nil {
			return fmt.Errorf("failed to create config map: %v", err)
		}
		return nil
	}

	if err := c.UpdateConfigMap(configMap, newConfigData); err != nil {
		return fmt.Errorf("failed to update config map: %v", err)
	}

	return nil
}

func (c *ChildClusterRole) UpdateConfigMap(configMap *corev1.ConfigMap, newConfigData map[string]string) error {
	needsUpdate := false

	for key, newValue := range newConfigData {
		if oldValue, ok := configMap.Data[key]; !ok || oldValue != newValue {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		return nil
	}

	configMap.Data = newConfigData
	if err := c.client.Update(c.ctx, configMap); err != nil {
		utils.LogEvent(
			c.ctx,
			"ConfigMapUpdateFailed",
			"Failed to update child cluster ConfigMap",
			c.clusterDeployment,
			err,
			"configMapName", configMap.Name,
		)
		return fmt.Errorf("failed to update ConfigMap: %v", err)
	}

	utils.LogEvent(
		c.ctx,
		"ConfigMapUpdated",
		"Updated child cluster ConfigMap",
		c.clusterDeployment,
		nil,
		"configMapName", configMap.Name,
	)

	return nil
}

func (c *ChildClusterRole) CreateConfigMap(configData map[string]string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            GetConfigMapName(c.clusterName),
			Namespace:       c.namespace,
			OwnerReferences: []metav1.OwnerReference{c.ownerReference},
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
		},
		Data: configData,
	}

	if err := utils.CreateIfNotExists(c.ctx, c.client, configMap, "child cluster ConfigMap", []any{
		"configMapName", configMap.Name,
		"configMapData", configData,
	}); err != nil {
		utils.LogEvent(
			c.ctx,
			"ConfigMapCreationFailed",
			"Failed to create child cluster ConfigMap",
			c.clusterDeployment,
			err,
			"configMapName", configMap.Name,
			"configMapData", configData,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"ConfigMapCreated",
		"Created child cluster ConfigMap",
		c.clusterDeployment,
		nil,
		"configMapName", configMap.Name,
		"configMapData", configData,
	)
	return nil
}

func (c *ChildClusterRole) GetRegionalClusterName() string {
	return c.clusterDeployment.Labels[KofRegionalClusterNameLabel]
}

func (r *ChildClusterRole) IsIstioCluster() bool {
	_, isIstio := r.clusterDeployment.Labels[IstioRoleLabel]
	return isIstio
}

func GetConfigMapName(clusterName string) string {
	return "kof-cluster-config-" + clusterName
}
