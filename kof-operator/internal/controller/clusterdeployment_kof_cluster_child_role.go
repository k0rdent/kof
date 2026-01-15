package controller

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/controller/vmuser"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ChildClusterRole struct {
	clusterName             string
	clusterNamespace        string
	isClusterInRegion       bool
	client                  client.Client
	ctx                     context.Context
	clusterDeployment       *kcmv1beta1.ClusterDeployment
	clusterDeploymentConfig *ClusterDeploymentConfig
	ownerReference          *metav1.OwnerReference
	vmUserManager           *vmuser.Manager
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

	isInRegion, err := k8s.CreatedInKCMRegion(ctx, client, cd)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if cluster is in region: %v", err)
	}

	return &ChildClusterRole{
		ctx:                     ctx,
		clusterDeployment:       cd,
		clusterDeploymentConfig: cdConfig,
		clusterName:             cd.Name,
		clusterNamespace:        cd.Namespace,
		client:                  client,
		ownerReference:          ownerReference,
		isClusterInRegion:       isInRegion,
		vmUserManager:           vmuser.NewManager(client),
	}, nil
}

func (c *ChildClusterRole) Reconcile() error {
	regionalConfigMap, err := c.GetRegionalConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get regional cluster: %v", err)
	}

	if err := c.CreateOrUpdateConfigMap(regionalConfigMap.configMap.Data); err != nil {
		return fmt.Errorf("failed to create or update config map: %v", err)
	}

	if err := c.CreateVMUserCredentials(regionalConfigMap.clusterName); err != nil {
		return fmt.Errorf("failed to create VM user credentials: %v", err)
	}

	return nil
}

func (c *ChildClusterRole) CreateVMUserCredentials(regionalClusterName string) error {
	opts := &vmuser.CreateOptions{
		Name:       GetVMUserName(GetConfigMapName(c.clusterName)),
		Namespace:  k8s.KofNamespace,
		ClusterRef: c.clusterDeployment,
		MCSConfig: &vmuser.MCSConfig{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					// MatchLabel is used to select regional cluster where VMUser will be propagated
					"k0rdent.mirantis.com/kof-cluster-name": regionalClusterName,
				},
			},
		},
	}

	if tenantID, ok := c.clusterDeployment.Labels[vmuser.KofTenantLabel]; ok {
		opts.Labels = map[string]string{
			vmuser.KofTenantLabel: tenantID,
		}
		opts.VMUserConfig = &vmuser.VMUserConfig{
			ExtraFilters: map[string]string{"tenantId": tenantID},
			ExtraLabel:   &vmuser.ExtraLabel{Key: "tenantId", Value: tenantID},
		}
	}

	return c.vmUserManager.Create(c.ctx, opts)
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
	var err error
	var regionalClusterConfigMap *corev1.ConfigMap
	log := log.FromContext(c.ctx)

	if regionalClusterName, ok := c.clusterDeployment.Labels[KofRegionalClusterNameLabel]; ok {
		if regionalClusterConfigMap, err = c.DiscoverRegionalClusterCmByLabel(regionalClusterName); err != nil {
			log.Error(
				err, "regional cluster ConfigMap not found by label",
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"regionalClusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return nil, err
		}
	} else {
		if regionalClusterConfigMap, err = c.DiscoverRegionalClusterConfigMapByLocation(); err != nil {
			log.Error(
				err, "regional cluster ConfigMap not found by location",
				"childClusterDeploymentName", c.clusterName,
				"childClusterDeploymentNamespace", c.clusterNamespace,
				"regionalClusterDeploymentLabel", KofRegionalClusterNameLabel,
			)
			return nil, err
		}
	}

	return NewRegionalClusterConfigMap(c.ctx, regionalClusterConfigMap, c.client)
}

func (c *ChildClusterRole) DiscoverRegionalClusterCmByLabel(regionalClusterName string) (*corev1.ConfigMap, error) {
	ok := false
	log := log.FromContext(c.ctx)
	crossNamespace := os.Getenv("CROSS_NAMESPACE") == "true"
	regionalClusterNamespace := c.clusterNamespace
	regionalClusterConfigMap := new(corev1.ConfigMap)

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

	if c.isClusterInRegion {
		isSameRegion, err := k8s.IsClusterInSameKcmRegion(c.ctx, c.client,
			c.clusterName, c.clusterNamespace,
			regionalClusterName, regionalClusterNamespace,
		)

		if err != nil {
			err := fmt.Errorf("failed to determine if clusters are in the same KCM region: %v", err)
			log.Error(
				err, "when regional cluster label is set",
				"crossNamespace", crossNamespace,
				KofRegionalClusterNameLabel, regionalClusterName,
				"childClusterDeploymentName", c.clusterName,
			)
			return nil, err
		}

		if !isSameRegion {
			err := fmt.Errorf("child cluster and regional cluster are not in the same KCM region")
			log.Error(
				err, "when regional cluster label is set",
				"crossNamespace", crossNamespace,
				KofRegionalClusterNameLabel, regionalClusterName,
				"childClusterDeploymentName", c.clusterName,
			)
			return nil, err
		}
	}

	if err := c.client.Get(c.ctx, types.NamespacedName{
		Name:      GetRegionalClusterConfigMapName(regionalClusterName),
		Namespace: regionalClusterNamespace,
	}, regionalClusterConfigMap); err != nil {
		log.Error(
			err, "cannot get regional Configmap",
			"regionalClusterName", regionalClusterName,
			"regionalClusterNamespace", regionalClusterNamespace,
		)
		return nil, err
	}

	return regionalClusterConfigMap, nil
}

func (c *ChildClusterRole) DiscoverRegionalClusterConfigMapByLocation() (*corev1.ConfigMap, error) {
	log := log.FromContext(c.ctx)
	crossNamespace := os.Getenv("CROSS_NAMESPACE") == "true"

	childCloud, err := getCloud(c.ctx, c.client, c.clusterDeployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get child cluster cloud: %v", err)
	}

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

	candidates := make([]*corev1.ConfigMap, 0)
	cdsInSameRegion := make([]*kcmv1beta1.ClusterDeployment, 0)

	if c.isClusterInRegion {
		cdsInSameRegion, err = k8s.GetClusterDeploymentsInSameKcmRegion(c.ctx, c.client, c.clusterDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster deployments in the same kcm region: %v", err)
		}
	}

	for _, regionalClusterConfigmap := range regionalClusterConfigMapList.Items {
		regionalConfigData, err := NewConfigDataFromConfigMap(&regionalClusterConfigmap)
		if err != nil {
			log.Error(err, "failed to create new configmap data",
				"regionalClusterConfigmapName", regionalClusterConfigmap.Name,
				"regionalClusterConfigmapNamespace", regionalClusterConfigmap.Namespace,
			)
			continue
		}

		if c.isClusterInRegion {
			contain := slices.ContainsFunc(cdsInSameRegion, func(cd *kcmv1beta1.ClusterDeployment) bool {
				return cd.Name == regionalConfigData.RegionalClusterName
			})

			if !contain {
				continue
			}
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

func (c *ChildClusterRole) CreateOrUpdateConfigMap(newConfigData map[string]string) error {
	configMap, err := c.GetConfigMap()
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %v", err)
	}

	if configMap == nil {
		if err := c.CreateConfigMap(newConfigData); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %v", err)
		}

		if err := c.CreateConfigMapPropagation(); err != nil {
			return fmt.Errorf("failed to create ConfigMap propagation: %v", err)
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
	log := log.FromContext(c.ctx)

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

	created, err := utils.EnsureCreated(c.ctx, c.client, configMap)
	if err != nil {
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

	if !created {
		log.Info("ConfigMap already created",
			"configMapName", configMap.Name,
			"configMapNamespace", c.clusterNamespace,
			"configMapData", configData,
		)
		return nil
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

// Function creates MultiClusterService to propagate child ConfigMap to the region cluster.
// TODO: Remove this function once KCM implements automatic copying of the required resources to region clusters.
func (c *ChildClusterRole) CreateConfigMapPropagation() error {
	if !c.isClusterInRegion {
		return nil
	}

	mcs := &kcmv1beta1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name: GetChildConfigMapPropagationName(c.clusterName),
			Labels: map[string]string{
				utils.ManagedByLabel: utils.ManagedByValue,
				"cluster-name":       c.clusterName,
				"cluster-namespace":  c.clusterNamespace,
			},
		},
		Spec: kcmv1beta1.MultiClusterServiceSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k0rdent.mirantis.com/kcm-region-cluster": "true",
				},
			},
			ServiceSpec: kcmv1beta1.ServiceSpec{
				Services: []kcmv1beta1.Service{
					{
						Name:      "copy-config",
						Namespace: k8s.DefaultSystemNamespace,
						Template:  "kof-configmap-propagation",
					},
				},
				TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
					{
						Identifier: "ConfigMap",
						Resource: corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       GetConfigMapName(c.clusterName),
							Namespace:  k8s.DefaultSystemNamespace,
						},
					},
				},
			},
		},
	}

	if err := c.client.Create(c.ctx, mcs); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create propagation MCS for '%s' cluster: %v", c.clusterName, err)
	}
	return nil
}

func GetConfigMapName(clusterName string) string {
	return "kof-cluster-config-" + clusterName
}

func GetChildConfigMapPropagationName(clusterName string) string {
	return utils.GetNameHash("kof-child-config-propagation", clusterName)
}

func GetVMUserName(cmName string) string {
	return utils.GetHelmAdler32Checksum(cmName)
}
