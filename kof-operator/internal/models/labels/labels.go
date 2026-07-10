package labels

const (
	// ClusterNameLabel is applied to ClusterDeployment objects to store the
	// cluster's own name (mirrored from metadata.name). It is added automatically
	// by the controller if absent and is used to identify the cluster by name
	// throughout the system.
	ClusterNameLabel = "k0rdent.mirantis.com/kof-cluster-name"

	// KofGeneratedLabel marks resources (ConfigMaps, Secrets, etc.) that were
	// created automatically by the kof-operator. Resources that lack this label
	// are treated as user-managed and skipped during reconciliation to avoid
	// overwriting manual changes.
	KofGeneratedLabel = "k0rdent.mirantis.com/kof-generated"

	// KofClusterRoleLabel specifies the KOF hierarchy role of a cluster deployment.
	// Supported values are "child" and "regional". The controller uses this label
	// to branch into role-specific reconciliation logic and as a selector when
	// listing clusters of a given tier.
	KofClusterRoleLabel = "k0rdent.mirantis.com/kof-cluster-role"

	// KofVersionLabel gates MultiClusterService cluster selectors during upgrades.
	// When autoUpgrade is disabled, MCS objects include this label so only clusters
	// with a matching version receive updated services.
	KofVersionLabel = "k0rdent.mirantis.com/kof-version"

	// KofKcmRegionLabel marks a cluster deployment as belonging to a KCM region.
	KofKcmRegionLabel = "k0rdent.mirantis.com/kcm-region-cluster"

	// KofTenantLabel is used to associate cluster deployments with a specific tenant in multi-tenant environments.
	KofTenantLabel = "k0rdent.mirantis.com/kof-tenant-id"

	// DefaultTenantID is the tenant label value applied to data collected from clusters
	// that do not have the KofTenantLabel set. It ensures all data is always tagged
	// with a tenant, enabling consistent filtering and multi-tenant isolation.
	DefaultTenantID = "PLATFORM"

	// ClusterNameLabelKey is applied to VMStorageConnection resources to indicate which
	// cluster resource they reference. This allows the VMStorageConnection controller to
	// efficiently list all connections associated with a given cluster.
	ClusterNameLabelKey = "k0rdent.mirantis.com/cluster-name"

	// ClusterKindLabelKey is applied to VMStorageConnection resources alongside
	// ClusterNameLabelKey to indicate the kind of the referenced cluster resource
	// (e.g. "VTCluster" or "VLCluster").
	ClusterKindLabelKey = "k0rdent.mirantis.com/cluster-kind"

	// ManagedByLabel is the standard Kubernetes ownership label
	// (app.kubernetes.io/managed-by). It is applied to all resources generated
	// by the kof-operator to declare operator ownership and enable filtering of
	// operator-managed objects.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	SecretNameLabel = "k0rdent.mirantis.com/secret-name"
)

// HasLabel reports whether the given label key is present in the labels map.
func HasLabel(labelKey string, labels map[string]string) bool {
	_, ok := labels[labelKey]
	return ok
}

// HasClusterNameLabel reports whether the ClusterNameLabel is present in the labels map.
func HasClusterNameLabel(labels map[string]string) bool {
	return HasLabel(ClusterNameLabel, labels)
}
