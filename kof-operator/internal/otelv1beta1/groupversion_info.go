// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Vendored from github.com/open-telemetry/opentelemetry-operator/apis/v1beta1
// to avoid dependency on an outdated sigs.k8s.io/controller-runtime version.

// Package otelv1beta1 contains API Schema definitions for the OpenTelemetry v1beta1 API group.
// This package is vendored from github.com/open-telemetry/opentelemetry-operator/apis/v1beta1
// and must not be processed by controller-gen — the CRD is owned by the upstream operator.
// +kubebuilder:skip
// +kubebuilder:object:generate=false
package otelv1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "opentelemetry.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&OpenTelemetryCollector{},
		&OpenTelemetryCollectorList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
