package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"

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
	clusterNamespace        string
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
		clusterNamespace:        cd.Namespace,
		client:                  client,
		ownerReference:          ownerReference,
	}, nil
}

func (c *ChildClusterRole) Reconcile() error {
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
		return fmt.Errorf("failed to create or update config map: %v", err)
	}

	return nil
}

func (c *ChildClusterRole) GetConfigMap() (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      GetConfigMapName(c.clusterName),
		Namespace: c.clusterNamespace,
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
	crossNamespace := os.Getenv("CROSS_NAMESPACE") == "true"
	regionalClusterDeployment := &kcmv1beta1.ClusterDeployment{}
	regionalClusterName, ok := c.clusterDeployment.Labels[KofRegionalClusterNameLabel]

	if ok {
		regionalClusterNamespace := c.clusterNamespace
		if crossNamespace {
			regionalClusterNamespace, ok = c.clusterDeployment.Labels[KofRegionalClusterNamespaceLabel]
			if !ok {
				err := fmt.Errorf(`"%s" label is required`, KofRegionalClusterNamespaceLabel)
				log.Error(
					err, fmt.Sprintf(`when crossNamespace is true and "%s" label is set`, KofRegionalClusterNameLabel),
					"crossNamespace", crossNamespace,
					KofRegionalClusterNameLabel, regionalClusterName,
					"childClusterDeploymentName", c.clusterName,
				)
				return nil, err
			}
		}

		err := c.client.Get(c.ctx, types.NamespacedName{
			Name:      regionalClusterName,
			Namespace: regionalClusterNamespace,
		}, regionalClusterDeployment)
		if err != nil {
			log.Error(
				err, "cannot get regional ClusterDeployment",
				"regionalClusterName", regionalClusterName,
				"regionalClusterNamespace", regionalClusterNamespace,
			)
			return nil, err
		}
	} else {
		var err error
		if regionalClusterDeployment, err = c.DiscoverRegionalClusterDeploymentByLocation(
			crossNamespace,
		); err != nil {
			log.Error(
				err, "regional ClusterDeployment not found both by label and by location",
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"clusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return nil, err
		}
	}

	return NewRegionalClusterRole(c.ctx, regionalClusterDeployment, c.client)
}

func (c *ChildClusterRole) DiscoverRegionalClusterDeploymentByLocation(
	crossNamespace bool,
) (*kcmv1beta1.ClusterDeployment, error) {
	log := log.FromContext(c.ctx)
	childCloud := getCloud(c.clusterDeployment)

	configMap, err := c.GetConfigMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	regionalClusterDeploymentList := &kcmv1beta1.ClusterDeploymentList{}
	opts := []client.ListOption{client.MatchingLabels{KofClusterRoleLabel: "regional"}}
	if !crossNamespace {
		opts = append(opts, client.InNamespace(c.clusterNamespace))
	}
	if err := c.client.List(c.ctx, regionalClusterDeploymentList, opts...); err != nil {
		log.Error(err, "cannot list regional ClusterDeployments")
		return nil, err
	}

	candidates := []kcmv1beta1.ClusterDeployment{}
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
			if isPreviouslyUsedRegionalCluster(configMap, &regionalClusterDeployment) {
				return &regionalClusterDeployment, nil
			}

			candidates = append(candidates, regionalClusterDeployment)
		}
	}

	if configMap != nil {
		err := fmt.Errorf(
			"previously used regional cluster is not discovered in the same location"+
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
			"childClusterDeploymentNamespace", c.clusterNamespace,
			"oldRegionalClusterName", configMap.Data[RegionalClusterNameKey],
			"oldRegionalClusterNamespace", configMap.Data[RegionalClusterNamespaceKey],
		)
		return nil, err
	}

	if len(candidates) > 0 {
		return &candidates[0], nil
	}

	err = fmt.Errorf(
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
		"childClusterDeploymentNamespace", c.clusterNamespace,
		"crossNamespace", crossNamespace,
	)
	return nil, err
}

func isPreviouslyUsedRegionalCluster(
	configMap *corev1.ConfigMap,
	regionalClusterDeployment *kcmv1beta1.ClusterDeployment,
) bool {
	if configMap == nil {
		return false
	}
	return configMap.Data[RegionalClusterNameKey] == regionalClusterDeployment.Name &&
		(configMap.Data[RegionalClusterNamespaceKey] == "" ||
			configMap.Data[RegionalClusterNamespaceKey] == regionalClusterDeployment.Namespace)
}

func (c *ChildClusterRole) CreateProfile(regionalCD *RegionalClusterRole) error {
	log := log.FromContext(c.ctx)
	remoteSecretName := remotesecret.GetRemoteSecretName(regionalCD.clusterName)

	log.Info("Creating profile")

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:            remotesecret.CopyRemoteSecretProfileName(c.clusterName),
			Namespace:       c.clusterNamespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{c.ownerReference},
		},
		Spec: sveltosv1beta1.Spec{
			ClusterRefs: []corev1.ObjectReference{
				{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       clusterv1.ClusterKind,
					Name:       c.clusterName,
					Namespace:  c.clusterNamespace,
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
		return fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	if configMap == nil {
		if err := c.CreateConfigMap(newConfigData); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %v", err)
		}
		return nil
	}

	if err := c.UpdateConfigMap(configMap, newConfigData); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %v", err)
	}

	return nil
}

func (c *ChildClusterRole) UpdateConfigMap(configMap *corev1.ConfigMap, newConfigData map[string]string) error {
	if reflect.DeepEqual(configMap.Data, newConfigData) {
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
			"configMapNamespace", c.clusterNamespace,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"ConfigMapUpdated",
		"Updated child cluster ConfigMap",
		c.clusterDeployment,
		nil,
		"configMapName", configMap.Name,
		"configMapNamespace", c.clusterNamespace,
	)

	return nil
}

func (c *ChildClusterRole) CreateConfigMap(configData map[string]string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            GetConfigMapName(c.clusterName),
			Namespace:       c.clusterNamespace,
			OwnerReferences: []metav1.OwnerReference{c.ownerReference},
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
		},
		Data: configData,
	}

	if err := utils.CreateIfNotExists(c.ctx, c.client, configMap, "child cluster ConfigMap", []any{
		"configMapName", configMap.Name,
		"configMapNamespace", c.clusterNamespace,
		"configMapData", configData,
	}); err != nil {
		utils.LogEvent(
			c.ctx,
			"ConfigMapCreationFailed",
			"Failed to create child cluster ConfigMap",
			c.clusterDeployment,
			err,
			"configMapName", configMap.Name,
			"configMapNamespace", c.clusterNamespace,
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
		"configMapNamespace", c.clusterNamespace,
		"configMapData", configData,
	)
	return nil
}

func (r *ChildClusterRole) IsIstioCluster() bool {
	_, isIstio := r.clusterDeployment.Labels[IstioRoleLabel]
	return isIstio
}

func GetConfigMapName(clusterName string) string {
	return "kof-cluster-config-" + clusterName
}
