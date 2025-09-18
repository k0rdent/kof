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
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	ownerReference          *metav1.OwnerReference
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
	regionalConfigMap, err := c.GetRegionalConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get regional cluster: %v", err)
	}

	if c.IsIstioCluster() {
		if err := c.CreateProfile(regionalConfigMap, utils.IsAdopted(c.clusterDeployment)); err != nil {
			return fmt.Errorf("failed to create profile: %v", err)
		}
	}

	if err := c.CreateOrUpdateConfigMap(regionalConfigMap.configMap.Data); err != nil {
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

func (c *ChildClusterRole) GetRegionalConfigMap() (*RegionalClusterConfigMap, error) {
	log := log.FromContext(c.ctx)
	crossNamespace := os.Getenv("CROSS_NAMESPACE") == "true"
	regionalClusterConfigMap := &corev1.ConfigMap{}

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
			Name:      GetRegionalClusterConfigMapName(regionalClusterName),
			Namespace: regionalClusterNamespace,
		}, regionalClusterConfigMap)
		if err != nil {
			log.Error(
				err, "cannot get regional regional Configmap",
				"regionalClusterName", regionalClusterName,
				"regionalClusterNamespace", regionalClusterNamespace,
			)
			return nil, err
		}
	} else {
		var err error
		if regionalClusterConfigMap, err = c.DiscoverRegionalClusterConfigMapByLocation(
			crossNamespace,
		); err != nil {
			log.Error(
				err, "regional cluster ConfigMap not found both by label and by location",
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"regionalClusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return nil, err
		}
	}

	return NewRegionalClusterConfigMap(c.ctx, regionalClusterConfigMap, c.client)
}

func (c *ChildClusterRole) DiscoverRegionalClusterConfigMapByLocation(
	crossNamespace bool,
) (*corev1.ConfigMap, error) {
	log := log.FromContext(c.ctx)
	childCloud := getCloud(c.clusterDeployment)

	configMap, err := c.GetConfigMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	regionalClusterConfigMapList := &corev1.ConfigMapList{}

	selector := labels.Set{KofClusterRoleLabel: KofRoleRegional}.AsSelector()
	opts := &client.ListOptions{LabelSelector: selector}
	if !crossNamespace {
		opts.Namespace = c.clusterNamespace
	}
	if err := c.client.List(c.ctx, regionalClusterConfigMapList, opts); err != nil {
		log.Error(err, "cannot list regional cluster ConfigMap")
		return nil, err
	}

	candidates := []*corev1.ConfigMap{}
	for _, regionalClusterConfigmap := range regionalClusterConfigMapList.Items {
		regionalConfigData, err := NewConfigDataFromConfigMap(&regionalClusterConfigmap)
		if err != nil {
			log.Error(err, "failed to create new configmap data",
				"regionalClusterConfigmapName", regionalClusterConfigmap.Name,
				"regionalClusterConfigmapNamespace", regionalClusterConfigmap.Namespace,
			)
			continue
		}

		if childCloud != regionalConfigData.RegionalClusterCloud {
			log.Info("Discovering regional cluster by location of child cluster: cloud doesn't match",
				"childCloud", childCloud,
				"regionalCloud", regionalConfigData.RegionalClusterCloud,
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"regionalClusterConfigmapName", regionalClusterConfigmap.Name,
				"regionalClusterConfigmapNamespace", regionalClusterConfigmap.Namespace,
			)
			continue
		}

		if locationIsTheSame(
			childCloud,
			c.clusterDeploymentConfig,
			regionalConfigData.ToClusterDeploymentConfig(),
		) {
			if isPreviouslyUsedRegionalCluster(configMap, regionalConfigData) {
				return &regionalClusterConfigmap, nil
			}

			candidates = append(candidates, &regionalClusterConfigmap)
		} else {
			log.Info("Discovering regional cluster by location of child cluster: cloud matches, location doesn't match",
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"regionalClusterConfigmapName", regionalClusterConfigmap.Name,
				"regionalClusterConfigmapNamespace", regionalClusterConfigmap.Namespace,
			)
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
			"RegionalClusterConfigMapDiscoveryFailed",
			"Failed to discover regional cluster ConfigMap",
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
		return candidates[0], nil
	}

	err = fmt.Errorf(
		"regional cluster ConfigMap with matching location is not found, "+
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
	childConfigMap *corev1.ConfigMap,
	regionalConfigData *ConfigData,
) bool {
	if childConfigMap == nil {
		return false
	}
	return childConfigMap.Data[RegionalClusterNameKey] == regionalConfigData.RegionalClusterName &&
		(childConfigMap.Data[RegionalClusterNamespaceKey] == "" ||
			childConfigMap.Data[RegionalClusterNamespaceKey] == regionalConfigData.RegionalClusterNamespace)
}

func (c *ChildClusterRole) CreateProfile(regionalCm *RegionalClusterConfigMap, adopted bool) error {
	remoteSecretName := remotesecret.GetRemoteSecretName(regionalCm.clusterName)

	profile := &sveltosv1beta1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:            remotesecret.CopyRemoteSecretProfileName(c.clusterName),
			Namespace:       c.clusterNamespace,
			Labels:          map[string]string{utils.ManagedByLabel: utils.ManagedByValue},
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
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
	if adopted {
		profile.Spec.ClusterRefs = []corev1.ObjectReference{
			{
				APIVersion: libsveltosv1beta1.GroupVersion.String(),
				Kind:       libsveltosv1beta1.SveltosClusterKind,
				Name:       c.clusterName,
				Namespace:  c.clusterNamespace,
			},
		}
	}

	if err := utils.CreateIfNotExists(c.ctx, c.client, profile, "Profile", []any{
		"profileName", profile.Name,
	}); err != nil {
		utils.LogEvent(
			c.ctx,
			"ProfileCreationFailed",
			"Failed to create Profile",
			regionalCm.configMap,
			err,
			"profileName", profile.Name,
		)
		return err
	}

	utils.LogEvent(
		c.ctx,
		"ProfileCreated",
		"Copy remote secret Profile is successfully created",
		regionalCm.configMap,
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
			OwnerReferences: []metav1.OwnerReference{*c.ownerReference},
			Labels: map[string]string{
				utils.ManagedByLabel: utils.ManagedByValue,
				KofClusterRoleLabel:  KofRoleChild,
			},
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
