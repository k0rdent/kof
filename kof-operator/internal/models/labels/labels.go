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

	// IstioRoleLabel indicates the Istio service mesh role assigned to a cluster
	// deployment (e.g. "member").
	IstioRoleLabel = "k0rdent.mirantis.com/istio-role"

	// KofClusterRoleLabel specifies the KOF hierarchy role of a cluster deployment.
	// Supported values are "child" and "regional". The controller uses this label
	// to branch into role-specific reconciliation logic and as a selector when
	// listing clusters of a given tier.
	KofClusterRoleLabel = "k0rdent.mirantis.com/kof-cluster-role"

	// KofKcmRegionLabel marks a cluster deployment as belonging to a KCM region.
	KofKcmRegionLabel = "k0rdent.mirantis.com/kcm-region-cluster"

	// ManagedByLabel is the standard Kubernetes ownership label
	// (app.kubernetes.io/managed-by). It is applied to all resources generated
	// by the kof-operator to declare operator ownership and enable filtering of
	// operator-managed objects.
	ManagedByLabel = "app.kubernetes.io/managed-by"
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
