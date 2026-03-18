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
	runtime "k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func (in *AnyConfig) DeepCopyInto(out *AnyConfig) {
	*out = *in
	if in.Object != nil {
		in, out := &in.Object, &out.Object
		*out = make(map[string]interface{}, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

func (in *AnyConfig) DeepCopy() *AnyConfig {
	if in == nil {
		return nil
	}
	out := new(AnyConfig)
	in.DeepCopyInto(out)
	return out
}

func (in *AutoscalerSpec) DeepCopyInto(out *AutoscalerSpec) {
	*out = *in
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.MaxReplicas != nil {
		in, out := &in.MaxReplicas, &out.MaxReplicas
		*out = new(int32)
		**out = **in
	}
	if in.Behavior != nil {
		in, out := &in.Behavior, &out.Behavior
		*out = new(autoscalingv2.HorizontalPodAutoscalerBehavior)
		(*in).DeepCopyInto(*out)
	}
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = make([]MetricSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TargetCPUUtilization != nil {
		in, out := &in.TargetCPUUtilization, &out.TargetCPUUtilization
		*out = new(int32)
		**out = **in
	}
	if in.TargetMemoryUtilization != nil {
		in, out := &in.TargetMemoryUtilization, &out.TargetMemoryUtilization
		*out = new(int32)
		**out = **in
	}
}

func (in *AutoscalerSpec) DeepCopy() *AutoscalerSpec {
	if in == nil {
		return nil
	}
	out := new(AutoscalerSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *Config) DeepCopyInto(out *Config) {
	*out = *in
	in.Receivers.DeepCopyInto(&out.Receivers)
	in.Exporters.DeepCopyInto(&out.Exporters)
	if in.Processors != nil {
		in, out := &in.Processors, &out.Processors
		*out = (*in).DeepCopy()
	}
	if in.Connectors != nil {
		in, out := &in.Connectors, &out.Connectors
		*out = (*in).DeepCopy()
	}
	if in.Extensions != nil {
		in, out := &in.Extensions, &out.Extensions
		*out = (*in).DeepCopy()
	}
	in.Service.DeepCopyInto(&out.Service)
}

func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	in.DeepCopyInto(out)
	return out
}

func (in *ConfigMapsSpec) DeepCopyInto(out *ConfigMapsSpec) { *out = *in }

func (in *ConfigMapsSpec) DeepCopy() *ConfigMapsSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigMapsSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *Ingress) DeepCopyInto(out *Ingress) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = make([]networkingv1.IngressTLS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.IngressClassName != nil {
		in, out := &in.IngressClassName, &out.IngressClassName
		*out = new(string)
		**out = **in
	}
	out.Route = in.Route
}

func (in *Ingress) DeepCopy() *Ingress {
	if in == nil {
		return nil
	}
	out := new(Ingress)
	in.DeepCopyInto(out)
	return out
}

func (in *MetricSpec) DeepCopyInto(out *MetricSpec) {
	*out = *in
	if in.Pods != nil {
		in, out := &in.Pods, &out.Pods
		*out = new(autoscalingv2.PodsMetricSource)
		(*in).DeepCopyInto(*out)
	}
}

func (in *MetricSpec) DeepCopy() *MetricSpec {
	if in == nil {
		return nil
	}
	out := new(MetricSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *MetricsConfigSpec) DeepCopyInto(out *MetricsConfigSpec) {
	*out = *in
	if in.ExtraLabels != nil {
		in, out := &in.ExtraLabels, &out.ExtraLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

func (in *MetricsConfigSpec) DeepCopy() *MetricsConfigSpec {
	if in == nil {
		return nil
	}
	out := new(MetricsConfigSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *NetworkPolicy) DeepCopyInto(out *NetworkPolicy) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

func (in *NetworkPolicy) DeepCopy() *NetworkPolicy {
	if in == nil {
		return nil
	}
	out := new(NetworkPolicy)
	in.DeepCopyInto(out)
	return out
}

func (in *ObservabilitySpec) DeepCopyInto(out *ObservabilitySpec) {
	*out = *in
	in.Metrics.DeepCopyInto(&out.Metrics)
}

func (in *ObservabilitySpec) DeepCopy() *ObservabilitySpec {
	if in == nil {
		return nil
	}
	out := new(ObservabilitySpec)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenShiftRoute) DeepCopyInto(out *OpenShiftRoute) { *out = *in }

func (in *OpenShiftRoute) DeepCopy() *OpenShiftRoute {
	if in == nil {
		return nil
	}
	out := new(OpenShiftRoute)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenTelemetryCollector) DeepCopyInto(out *OpenTelemetryCollector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

func (in *OpenTelemetryCollector) DeepCopy() *OpenTelemetryCollector {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCollector)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenTelemetryCollector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *OpenTelemetryCollectorList) DeepCopyInto(out *OpenTelemetryCollectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OpenTelemetryCollector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *OpenTelemetryCollectorList) DeepCopy() *OpenTelemetryCollectorList {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCollectorList)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenTelemetryCollectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *OpenTelemetryCollectorSpec) DeepCopyInto(out *OpenTelemetryCollectorSpec) {
	*out = *in
	in.OpenTelemetryCommonFields.DeepCopyInto(&out.OpenTelemetryCommonFields)
	in.StatefulSetCommonFields.DeepCopyInto(&out.StatefulSetCommonFields)
	if in.Autoscaler != nil {
		in, out := &in.Autoscaler, &out.Autoscaler
		*out = new(AutoscalerSpec)
		(*in).DeepCopyInto(*out)
	}
	in.TargetAllocator.DeepCopyInto(&out.TargetAllocator)
	in.Config.DeepCopyInto(&out.Config)
	in.Ingress.DeepCopyInto(&out.Ingress)
	in.NetworkPolicy.DeepCopyInto(&out.NetworkPolicy)
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.StartupProbe != nil {
		in, out := &in.StartupProbe, &out.StartupProbe
		*out = new(Probe)
		(*in).DeepCopyInto(*out)
	}
	in.Observability.DeepCopyInto(&out.Observability)
	if in.ConfigMaps != nil {
		in, out := &in.ConfigMaps, &out.ConfigMaps
		*out = make([]ConfigMapsSpec, len(*in))
		copy(*out, *in)
	}
	in.DaemonSetUpdateStrategy.DeepCopyInto(&out.DaemonSetUpdateStrategy)
	in.DeploymentUpdateStrategy.DeepCopyInto(&out.DeploymentUpdateStrategy)
}

func (in *OpenTelemetryCollectorSpec) DeepCopy() *OpenTelemetryCollectorSpec {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCollectorSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenTelemetryCollectorStatus) DeepCopyInto(out *OpenTelemetryCollectorStatus) {
	*out = *in
	out.Scale = in.Scale
}

func (in *OpenTelemetryCollectorStatus) DeepCopy() *OpenTelemetryCollectorStatus {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCollectorStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *OpenTelemetryCommonFields) DeepCopyInto(out *OpenTelemetryCommonFields) { //nolint:gocyclo
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.PodDisruptionBudget != nil {
		in, out := &in.PodDisruptionBudget, &out.PodDisruptionBudget
		*out = new(PodDisruptionBudgetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(corev1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(corev1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]corev1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortsSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]corev1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(corev1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Lifecycle != nil {
		in, out := &in.Lifecycle, &out.Lifecycle
		*out = new(corev1.Lifecycle)
		(*in).DeepCopyInto(*out)
	}
	if in.TerminationGracePeriodSeconds != nil {
		in, out := &in.TerminationGracePeriodSeconds, &out.TerminationGracePeriodSeconds
		*out = new(int64)
		**out = **in
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]corev1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.DNSPolicy != nil {
		in, out := &in.DNSPolicy, &out.DNSPolicy
		*out = new(corev1.DNSPolicy)
		**out = **in
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]corev1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AdditionalContainers != nil {
		in, out := &in.AdditionalContainers, &out.AdditionalContainers
		*out = make([]corev1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.PodDNSConfig.DeepCopyInto(&out.PodDNSConfig)
	if in.IpFamilies != nil {
		in, out := &in.IpFamilies, &out.IpFamilies
		*out = make([]corev1.IPFamily, len(*in))
		copy(*out, *in)
	}
	if in.IpFamilyPolicy != nil {
		in, out := &in.IpFamilyPolicy, &out.IpFamilyPolicy
		*out = new(corev1.IPFamilyPolicy)
		**out = **in
	}
	if in.TrafficDistribution != nil {
		in, out := &in.TrafficDistribution, &out.TrafficDistribution
		*out = new(string)
		**out = **in
	}
}

func (in *OpenTelemetryCommonFields) DeepCopy() *OpenTelemetryCommonFields {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCommonFields)
	in.DeepCopyInto(out)
	return out
}

func (in *Pipeline) DeepCopyInto(out *Pipeline) {
	*out = *in
	if in.Exporters != nil {
		in, out := &in.Exporters, &out.Exporters
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Processors != nil {
		in, out := &in.Processors, &out.Processors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Receivers != nil {
		in, out := &in.Receivers, &out.Receivers
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

func (in *Pipeline) DeepCopy() *Pipeline {
	if in == nil {
		return nil
	}
	out := new(Pipeline)
	in.DeepCopyInto(out)
	return out
}

func (in *PodDisruptionBudgetSpec) DeepCopyInto(out *PodDisruptionBudgetSpec) {
	*out = *in
	if in.MinAvailable != nil {
		in, out := &in.MinAvailable, &out.MinAvailable
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		*out = new(intstr.IntOrString)
		**out = **in
	}
}

func (in *PodDisruptionBudgetSpec) DeepCopy() *PodDisruptionBudgetSpec {
	if in == nil {
		return nil
	}
	out := new(PodDisruptionBudgetSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *PortsSpec) DeepCopyInto(out *PortsSpec) {
	*out = *in
	out.ServicePort = in.ServicePort
}

func (in *PortsSpec) DeepCopy() *PortsSpec {
	if in == nil {
		return nil
	}
	out := new(PortsSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *Probe) DeepCopyInto(out *Probe) {
	*out = *in
	if in.InitialDelaySeconds != nil {
		in, out := &in.InitialDelaySeconds, &out.InitialDelaySeconds
		*out = new(int32)
		**out = **in
	}
	if in.TimeoutSeconds != nil {
		in, out := &in.TimeoutSeconds, &out.TimeoutSeconds
		*out = new(int32)
		**out = **in
	}
	if in.PeriodSeconds != nil {
		in, out := &in.PeriodSeconds, &out.PeriodSeconds
		*out = new(int32)
		**out = **in
	}
	if in.SuccessThreshold != nil {
		in, out := &in.SuccessThreshold, &out.SuccessThreshold
		*out = new(int32)
		**out = **in
	}
	if in.FailureThreshold != nil {
		in, out := &in.FailureThreshold, &out.FailureThreshold
		*out = new(int32)
		**out = **in
	}
	if in.TerminationGracePeriodSeconds != nil {
		in, out := &in.TerminationGracePeriodSeconds, &out.TerminationGracePeriodSeconds
		*out = new(int64)
		**out = **in
	}
}

func (in *Probe) DeepCopy() *Probe {
	if in == nil {
		return nil
	}
	out := new(Probe)
	in.DeepCopyInto(out)
	return out
}

func (in *ScaleSubresourceStatus) DeepCopyInto(out *ScaleSubresourceStatus) { *out = *in }

func (in *ScaleSubresourceStatus) DeepCopy() *ScaleSubresourceStatus {
	if in == nil {
		return nil
	}
	out := new(ScaleSubresourceStatus)
	in.DeepCopyInto(out)
	return out
}

func (in *Service) DeepCopyInto(out *Service) {
	*out = *in
	if in.Extensions != nil {
		in, out := &in.Extensions, &out.Extensions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Telemetry != nil {
		in, out := &in.Telemetry, &out.Telemetry
		*out = (*in).DeepCopy()
	}
	if in.Pipelines != nil {
		in, out := &in.Pipelines, &out.Pipelines
		*out = make(map[string]*Pipeline, len(*in))
		for key, val := range *in {
			var outVal *Pipeline
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(Pipeline)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

func (in *Service) DeepCopy() *Service {
	if in == nil {
		return nil
	}
	out := new(Service)
	in.DeepCopyInto(out)
	return out
}

func (in *StatefulSetCommonFields) DeepCopyInto(out *StatefulSetCommonFields) {
	*out = *in
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]corev1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PersistentVolumeClaimRetentionPolicy != nil {
		in, out := &in.PersistentVolumeClaimRetentionPolicy, &out.PersistentVolumeClaimRetentionPolicy
		*out = new(appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy)
		**out = **in
	}
}

func (in *StatefulSetCommonFields) DeepCopy() *StatefulSetCommonFields {
	if in == nil {
		return nil
	}
	out := new(StatefulSetCommonFields)
	in.DeepCopyInto(out)
	return out
}

func (in *TargetAllocatorEmbedded) DeepCopyInto(out *TargetAllocatorEmbedded) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(corev1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	in.PrometheusCR.DeepCopyInto(&out.PrometheusCR)
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(corev1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(corev1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]corev1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Observability.DeepCopyInto(&out.Observability)
	if in.PodDisruptionBudget != nil {
		in, out := &in.PodDisruptionBudget, &out.PodDisruptionBudget
		*out = new(PodDisruptionBudgetSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CollectorNotReadyGracePeriod != nil {
		in, out := &in.CollectorNotReadyGracePeriod, &out.CollectorNotReadyGracePeriod
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.CollectorTargetReloadInterval != nil {
		in, out := &in.CollectorTargetReloadInterval, &out.CollectorTargetReloadInterval
		*out = new(metav1.Duration)
		**out = **in
	}
}

func (in *TargetAllocatorEmbedded) DeepCopy() *TargetAllocatorEmbedded {
	if in == nil {
		return nil
	}
	out := new(TargetAllocatorEmbedded)
	in.DeepCopyInto(out)
	return out
}

func (in *TargetAllocatorPrometheusCR) DeepCopyInto(out *TargetAllocatorPrometheusCR) {
	*out = *in
	if in.AllowNamespaces != nil {
		in, out := &in.AllowNamespaces, &out.AllowNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DenyNamespaces != nil {
		in, out := &in.DenyNamespaces, &out.DenyNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ScrapeInterval != nil {
		in, out := &in.ScrapeInterval, &out.ScrapeInterval
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.ScrapeClasses != nil {
		in, out := &in.ScrapeClasses, &out.ScrapeClasses
		*out = make([]AnyConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodMonitorSelector != nil {
		in, out := &in.PodMonitorSelector, &out.PodMonitorSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.ServiceMonitorSelector != nil {
		in, out := &in.ServiceMonitorSelector, &out.ServiceMonitorSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.ScrapeConfigSelector != nil {
		in, out := &in.ScrapeConfigSelector, &out.ScrapeConfigSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.ProbeSelector != nil {
		in, out := &in.ProbeSelector, &out.ProbeSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

func (in *TargetAllocatorPrometheusCR) DeepCopy() *TargetAllocatorPrometheusCR {
	if in == nil {
		return nil
	}
	out := new(TargetAllocatorPrometheusCR)
	in.DeepCopyInto(out)
	return out
}
