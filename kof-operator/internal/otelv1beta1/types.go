// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Vendored from github.com/open-telemetry/opentelemetry-operator/apis/v1beta1
// to avoid dependency on an outdated sigs.k8s.io/controller-runtime version.

package otelv1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelcol;otelcols
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.scale.replicas,selectorpath=.status.scale.selector

// OpenTelemetryCollector is the Schema for the opentelemetrycollectors API.
type OpenTelemetryCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenTelemetryCollectorSpec   `json:"spec,omitempty"`
	Status OpenTelemetryCollectorStatus `json:"status,omitempty"`
}

// Hub exists to allow for conversion.
func (*OpenTelemetryCollector) Hub() {}

// +kubebuilder:object:root=true

// OpenTelemetryCollectorList contains a list of OpenTelemetryCollector.
type OpenTelemetryCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollector `json:"items"`
}

// OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.
type OpenTelemetryCollectorStatus struct {
	// Scale is the OpenTelemetryCollector's scale subresource status.
	// +optional
	Scale ScaleSubresourceStatus `json:"scale,omitempty"`

	// Version of the managed OpenTelemetry Collector (operand)
	// +optional
	Version string `json:"version,omitempty"`

	// Image indicates the container image to use for the OpenTelemetry Collector.
	// +optional
	Image string `json:"image,omitempty"`
}

// ScaleSubresourceStatus defines the observed state of the OpenTelemetryCollector's
// scale subresource.
type ScaleSubresourceStatus struct {
	// The selector used to match the OpenTelemetryCollector's deployment or statefulSet pods.
	// +optional
	Selector string `json:"selector,omitempty"`

	// The total number non-terminated pods targeted by this OpenTelemetryCollector's deployment or statefulSet.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// StatusReplicas is the number of pods targeted by this OpenTelemetryCollector's with a Ready Condition /
	// Total number of non-terminated pods targeted by this OpenTelemetryCollector's (their labels match the selector).
	// Deployment, Daemonset, StatefulSet.
	// +optional
	StatusReplicas string `json:"statusReplicas,omitempty"`
}

// ManagementStateType defines the type for CR management states.
//
// +kubebuilder:validation:Enum=managed;unmanaged
type ManagementStateType string

const (
	// ManagementStateManaged when the OpenTelemetryCollector custom resource should be reconciled by the operator.
	ManagementStateManaged ManagementStateType = "managed"

	// ManagementStateUnmanaged when the OpenTelemetryCollector custom resource should not be reconciled by the operator.
	ManagementStateUnmanaged ManagementStateType = "unmanaged"
)

// Mode represents how the collector should be deployed (deployment vs. daemonset).
// +kubebuilder:validation:Enum=daemonset;deployment;sidecar;statefulset
type Mode string

const (
	// ModeDaemonSet specifies that the collector should be deployed as a Kubernetes DaemonSet.
	ModeDaemonSet Mode = "daemonset"

	// ModeDeployment specifies that the collector should be deployed as a Kubernetes Deployment.
	ModeDeployment Mode = "deployment"

	// ModeSidecar specifies that the collector should be deployed as a sidecar to pods.
	ModeSidecar Mode = "sidecar"

	// ModeStatefulSet specifies that the collector should be deployed as a Kubernetes StatefulSet.
	ModeStatefulSet Mode = "statefulset"
)

// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed.
// +kubebuilder:validation:Enum=automatic;none
type UpgradeStrategy string

const (
	// UpgradeStrategyAutomatic specifies that the operator will automatically apply upgrades to the CR.
	UpgradeStrategyAutomatic UpgradeStrategy = "automatic"

	// UpgradeStrategyNone specifies that the operator will not apply any upgrades to the CR.
	UpgradeStrategyNone UpgradeStrategy = "none"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.
type OpenTelemetryCollectorSpec struct {
	// OpenTelemetryCommonFields are fields that are on all OpenTelemetry CRD workloads.
	OpenTelemetryCommonFields `json:",inline"`
	// StatefulSetCommonFields are fields that are on all OpenTelemetry CRD workloads.
	StatefulSetCommonFields `json:",inline"`
	// Autoscaler specifies the pod autoscaling configuration to use for the workload.
	// +optional
	Autoscaler *AutoscalerSpec `json:"autoscaler,omitempty"`
	// TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.
	// +optional
	TargetAllocator TargetAllocatorEmbedded `json:"targetAllocator,omitempty"`
	// Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar).
	// +optional
	Mode Mode `json:"mode,omitempty"`
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed.
	// +optional
	UpgradeStrategy UpgradeStrategy `json:"upgradeStrategy"`
	// Config is the raw JSON to be used as the collector's configuration.
	// +required
	// +kubebuilder:pruning:PreserveUnknownFields
	Config Config `json:"config"`
	// Ingress is used to specify how OpenTelemetry Collector is exposed.
	// +optional
	Ingress Ingress `json:"ingress,omitempty"`
	// NetworkPolicy defines the network policy to be applied to the OpenTelemetry Collector pods.
	// +optional
	NetworkPolicy NetworkPolicy `json:"networkPolicy,omitempty"`
	// LivenessProbe config for the OpenTelemetry Collector.
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
	// ReadinessProbe config for the OpenTelemetry Collector.
	// +optional
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
	// StartupProbe config for the OpenTelemetry Collector.
	// +optional
	StartupProbe *Probe `json:"startupProbe,omitempty"`
	// ObservabilitySpec defines how telemetry data gets handled.
	// +optional
	Observability ObservabilitySpec `json:"observability,omitempty"`
	// ConfigMaps is a list of ConfigMaps in the same namespace as the OpenTelemetryCollector object.
	ConfigMaps []ConfigMapsSpec `json:"configmaps,omitempty"`
	// DaemonSetUpdateStrategy for DaemonSet mode.
	// +optional
	DaemonSetUpdateStrategy appsv1.DaemonSetUpdateStrategy `json:"daemonSetUpdateStrategy,omitempty"`
	// DeploymentUpdateStrategy for Deployment mode.
	// +optional
	DeploymentUpdateStrategy appsv1.DeploymentStrategy `json:"deploymentUpdateStrategy,omitempty"`
}

// ObservabilitySpec defines how telemetry data gets handled.
type ObservabilitySpec struct {
	// Metrics defines the metrics configuration for operands.
	// +optional
	Metrics MetricsConfigSpec `json:"metrics,omitempty"`
}

// MetricsConfigSpec defines a metrics config.
type MetricsConfigSpec struct {
	// EnableMetrics specifies if ServiceMonitor or PodMonitor should be created.
	// +optional
	EnableMetrics bool `json:"enableMetrics,omitempty"`
	// ExtraLabels are additional labels to be added to the ServiceMonitor.
	// +optional
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
	// DisablePrometheusAnnotations controls the automatic addition of default Prometheus annotations.
	// +optional
	DisablePrometheusAnnotations bool `json:"disablePrometheusAnnotations,omitempty"`
}

// ConfigMapsSpec defines the ConfigMap to mount.
type ConfigMapsSpec struct {
	Name      string `json:"name"`
	MountPath string `json:"mountpath"`
}

// MetricSpec defines a subset of metrics to be defined for the HPA's metric array.
type MetricSpec struct {
	Type autoscalingv2.MetricSourceType  `json:"type"`
	Pods *autoscalingv2.PodsMetricSource `json:"pods,omitempty"`
}

// AutoscalerSpec defines the OpenTelemetryCollector's pod autoscaling specification.
type AutoscalerSpec struct {
	// MinReplicas sets a lower bound to the autoscaling feature.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// MaxReplicas sets an upper bound to the autoscaling feature.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
	// Metrics is meant to provide a customizable way to configure HPA metrics.
	// +optional
	Metrics []MetricSpec `json:"metrics,omitempty"`
	// TargetCPUUtilization sets the target average CPU used across all replicas.
	// +optional
	TargetCPUUtilization *int32 `json:"targetCPUUtilization,omitempty"`
	// TargetMemoryUtilization sets the target average memory utilization across all replicas.
	// +optional
	TargetMemoryUtilization *int32 `json:"targetMemoryUtilization,omitempty"`
}

// PodDisruptionBudgetSpec defines the OpenTelemetryCollector's pod disruption budget specification.
type PodDisruptionBudgetSpec struct {
	// +optional
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

// PortsSpec defines the OpenTelemetryCollector's container/service ports additional specifications.
type PortsSpec struct {
	// +optional
	HostPort int32 `json:"hostPort,omitempty"`

	corev1.ServicePort `json:",inline"`
}

// OpenTelemetryCommonFields are fields shared by all OpenTelemetry CRD workloads.
type OpenTelemetryCommonFields struct {
	// ManagementState defines if the CR should be managed by the operator or not.
	// +kubebuilder:default:=managed
	ManagementState ManagementStateType `json:"managementState,omitempty"`
	// Resources to set on generated pods.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// NodeSelector to schedule generated pods.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Args is the set of arguments to pass to the main container's binary.
	// +optional
	Args map[string]string `json:"args,omitempty"`
	// Replicas is the number of pod instances for the underlying replicaset.
	// +optional
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas,omitempty"`
	// PodDisruptionBudget specifies the pod disruption budget configuration.
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	// SecurityContext configures the container security context.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext configures the pod security context.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// PodAnnotations is the set of annotations attached to the generated pods.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// Image indicates the container image to use for the generated pods.
	// +optional
	Image string `json:"image,omitempty"`
	// ImagePullPolicy indicates the pull policy to be used for retrieving the container image.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// VolumeMounts represents the mount points to use in the underlying deployment(s).
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	// Ports allows a set of ports to be exposed by the underlying v1.Service & v1.ContainerPort.
	// +optional
	Ports []PortsSpec `json:"ports,omitempty"`
	// Env are environment variables to set on the generated pods.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// EnvFrom is a list of sources to populate environment variables on the generated pods.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
	// Tolerations to schedule the generated pods.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Volumes represents which volumes to use in the underlying deployment(s).
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// Affinity specifies the pod scheduling constraints.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Lifecycle defines actions the management system should take in response to container lifecycle events.
	// +optional
	Lifecycle *corev1.Lifecycle `json:"lifecycle,omitempty"`
	// TerminationGracePeriodSeconds is the duration in seconds the pod needs to terminate gracefully.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// TopologySpreadConstraints controls how pods are spread across failure-domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	// HostNetwork indicates if the pod should run in the host networking namespace.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`
	// DNSPolicy defines how a pod's DNS will be configured.
	// +optional
	DNSPolicy *corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// HostPID indicates if the pod should have access to the host process ID namespace.
	// +optional
	HostPID bool `json:"hostPID,omitempty"`
	// ShareProcessNamespace indicates if the pod's containers should share process namespace.
	// +optional
	ShareProcessNamespace bool `json:"shareProcessNamespace,omitempty"`
	// PriorityClassName indicates the pod's priority.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// InitContainers allows injecting initContainers to the generated pod definition.
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// AdditionalContainers allows injecting additional containers into the generated pod definition.
	// +optional
	AdditionalContainers []corev1.Container `json:"additionalContainers,omitempty"`
	// PodDNSConfig defines the DNS parameters of a pod.
	PodDNSConfig corev1.PodDNSConfig `json:"podDnsConfig,omitempty"`
	// IpFamilies represents the IP Family (IPv4 or IPv6).
	// +optional
	IpFamilies []corev1.IPFamily `json:"ipFamilies,omitempty"`
	// IpFamilyPolicy represents the dual-stack-ness requested or required by a Service.
	// +optional
	// +kubebuilder:default:=SingleStack
	IpFamilyPolicy *corev1.IPFamilyPolicy `json:"ipFamilyPolicy,omitempty"`
	// TrafficDistribution specifies how traffic to this service is routed.
	// +optional
	TrafficDistribution *string `json:"trafficDistribution,omitempty"`
}

// StatefulSetCommonFields are fields shared by statefulset-based workloads.
type StatefulSetCommonFields struct {
	// VolumeClaimTemplates will provide stable storage using PersistentVolumes.
	// +optional
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	// PersistentVolumeClaimRetentionPolicy describes the lifecycle of persistent volume claims.
	// +optional
	PersistentVolumeClaimRetentionPolicy *appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy `json:"persistentVolumeClaimRetentionPolicy,omitempty"`
	// ServiceName sets the serviceName of the StatefulSet.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`
}

// Probe defines the OpenTelemetry's pod probe config.
type Probe struct {
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
	// +optional
	SuccessThreshold *int32 `json:"successThreshold,omitempty"`
	// +optional
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// NetworkPolicy defines the configuration for NetworkPolicy.
type NetworkPolicy struct {
	// Enabled enables the NetworkPolicy.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// IngressType represents how a collector should be exposed (ingress vs route).
// +kubebuilder:validation:Enum=ingress;route
type IngressType string

const (
	IngressTypeIngress IngressType = "ingress"
	IngressTypeRoute   IngressType = "route"
)

// TLSRouteTerminationType is used to indicate which tls settings should be used.
// +kubebuilder:validation:Enum=insecure;edge;passthrough;reencrypt
type TLSRouteTerminationType string

const (
	TLSRouteTerminationTypeInsecure    TLSRouteTerminationType = "insecure"
	TLSRouteTerminationTypeEdge        TLSRouteTerminationType = "edge"
	TLSRouteTerminationTypePassthrough TLSRouteTerminationType = "passthrough"
	TLSRouteTerminationTypeReencrypt   TLSRouteTerminationType = "reencrypt"
)

// IngressRuleType defines how the collector receivers will be exposed in the Ingress.
// +kubebuilder:validation:Enum=path;subdomain
type IngressRuleType string

const (
	IngressRuleTypePath      IngressRuleType = "path"
	IngressRuleTypeSubdomain IngressRuleType = "subdomain"
)

// Ingress is used to specify how OpenTelemetry Collector is exposed.
type Ingress struct {
	Type             IngressType               `json:"type,omitempty"`
	RuleType         IngressRuleType           `json:"ruleType,omitempty"`
	Hostname         string                    `json:"hostname,omitempty"`
	Annotations      map[string]string         `json:"annotations,omitempty"`
	TLS              []networkingv1.IngressTLS `json:"tls,omitempty"`
	IngressClassName *string                   `json:"ingressClassName,omitempty"`
	Route            OpenShiftRoute            `json:"route,omitempty"`
}

// OpenShiftRoute defines openshift route specific settings.
type OpenShiftRoute struct {
	Termination TLSRouteTerminationType `json:"termination,omitempty"`
}

// TargetAllocatorAllocationStrategy represent a strategy Target Allocator uses to distribute targets to each collector.
// +kubebuilder:validation:Enum=least-weighted;consistent-hashing;per-node
type TargetAllocatorAllocationStrategy string

// TargetAllocatorFilterStrategy represent a filtering strategy for targets before they are assigned to collectors.
// +kubebuilder:validation:Enum="";relabel-config
type TargetAllocatorFilterStrategy string

const (
	TargetAllocatorAllocationStrategyLeastWeighted     TargetAllocatorAllocationStrategy = "least-weighted"
	TargetAllocatorAllocationStrategyConsistentHashing TargetAllocatorAllocationStrategy = "consistent-hashing"
	TargetAllocatorAllocationStrategyPerNode           TargetAllocatorAllocationStrategy = "per-node"
	TargetAllocatorFilterStrategyRelabelConfig         TargetAllocatorFilterStrategy     = "relabel-config"
)

// TargetAllocatorPrometheusCR configures Prometheus CustomResource handling in the Target Allocator.
type TargetAllocatorPrometheusCR struct {
	// Enabled indicates whether to use PrometheusOperator custom resources as targets.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// AllowNamespaces lists namespaces to scope the interaction of the Target Allocator (allow list).
	// +optional
	AllowNamespaces []string `json:"allowNamespaces,omitempty"`
	// DenyNamespaces lists namespaces to scope the interaction of the Target Allocator (deny list).
	// +optional
	DenyNamespaces []string `json:"denyNamespaces,omitempty"`
	// ScrapeInterval is the default interval between consecutive scrapes.
	// +optional
	ScrapeInterval *metav1.Duration `json:"scrapeInterval,omitempty"`
	// ScrapeClasses to be referenced by PodMonitors and ServiceMonitors.
	// +optional
	ScrapeClasses []AnyConfig `json:"scrapeClasses,omitempty"`
	// PodMonitorSelector selects PodMonitors for target discovery.
	// +optional
	PodMonitorSelector *metav1.LabelSelector `json:"podMonitorSelector,omitempty"`
	// ServiceMonitorSelector selects ServiceMonitors for target discovery.
	// +optional
	ServiceMonitorSelector *metav1.LabelSelector `json:"serviceMonitorSelector,omitempty"`
	// ScrapeConfigSelector selects ScrapeConfigs for target discovery.
	// +optional
	ScrapeConfigSelector *metav1.LabelSelector `json:"scrapeConfigSelector,omitempty"`
	// ProbeSelector selects Probes for target discovery.
	// +optional
	ProbeSelector *metav1.LabelSelector `json:"probeSelector,omitempty"`
}

// TargetAllocatorEmbedded defines the configuration for the Prometheus target allocator,
// embedded in the OpenTelemetryCollector spec.
type TargetAllocatorEmbedded struct {
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	// +kubebuilder:default:=consistent-hashing
	AllocationStrategy TargetAllocatorAllocationStrategy `json:"allocationStrategy,omitempty"`
	// +optional
	// +kubebuilder:default:=relabel-config
	FilterStrategy TargetAllocatorFilterStrategy `json:"filterStrategy,omitempty"`
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// +optional
	Image string `json:"image,omitempty"`
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// +optional
	PrometheusCR TargetAllocatorPrometheusCR `json:"prometheusCR,omitempty"`
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// +optional
	Observability ObservabilitySpec `json:"observability,omitempty"`
	// +optional
	PodDisruptionBudget *PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	// +optional
	CollectorNotReadyGracePeriod *metav1.Duration `json:"collectorNotReadyGracePeriod,omitempty"`
	// +optional
	CollectorTargetReloadInterval *metav1.Duration `json:"collectorTargetReloadInterval,omitempty"`
}

// AnyConfig represent parts of the config.
type AnyConfig struct {
	Object map[string]interface{} `json:"-" yaml:",inline"`
}

// Pipeline is a struct of component type to a list of component IDs.
type Pipeline struct {
	Exporters  []string `json:"exporters" yaml:"exporters"`
	Processors []string `json:"processors,omitempty" yaml:"processors,omitempty"`
	Receivers  []string `json:"receivers" yaml:"receivers"`
}

// Service defines the service pipeline configuration.
type Service struct {
	Extensions []string `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Telemetry *AnyConfig           `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	Pipelines map[string]*Pipeline `json:"pipelines" yaml:"pipelines"`
}

// Config encapsulates collector config.
type Config struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Receivers AnyConfig `json:"receivers" yaml:"receivers"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Exporters AnyConfig `json:"exporters" yaml:"exporters"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Processors *AnyConfig `json:"processors,omitempty" yaml:"processors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Connectors *AnyConfig `json:"connectors,omitempty" yaml:"connectors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Extensions *AnyConfig `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	Service    Service    `json:"service" yaml:"service"`
}
