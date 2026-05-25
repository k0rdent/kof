/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMStorageConnectionSpec defines the desired state of VMStorageConnection.
type VMStorageConnectionSpec struct {
	// ClusterRef references the VictoriaMetrics cluster resource (VTCluster or VLCluster)
	// that this storage connection should configure.
	ClusterRef ClusterRef `json:"cluster_ref"`
	// TargetStorageNode defines the connection details for the storage node.
	TargetStorageNode TargetStorageNode `json:"target_storage_node"`
}

type TargetStorageNode struct {
	// Address of the storage node in the format "host:port".
	Address string `json:"address"`
	// Secrets is a list of Secrets in the same namespace as the Application
	// object, which shall be mounted into the Application container
	// at /etc/vm/secrets/SECRET_NAME folder. The secret should have keys
	// "username" and "password" for the respective credentials.
	Secret SecretRef `json:"secret,omitempty"`
	// TLSConfig defines the TLS settings for connecting to the storage node.
	TLSConfig TLSStorageConfig `json:"tls_config,omitempty"`
}

type TLSStorageConfig struct {
	// Enabled indicates whether TLS should be used when connecting to the storage node.
	Enabled bool `json:"enabled,omitempty"`
	// InsecureSkipVerify indicates whether to skip TLS certificate verification when connecting to the storage node.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

type SecretRef struct {
	// Name of the Secret in the same namespace as the Application object,
	// which shall be mounted into the Application container at /etc/vm/secrets/SECRET_NAME folder.
	Name string `json:"name"`
	// Key in the Secret that contains the password for authentication with the storage node.
	PasswordKey string `json:"password_key"`
	// Key in the Secret that contains the username for authentication with the storage node.
	UsernameKey string `json:"username_key"`
}

// ClusterRef defines the reference to the VictoriaMetrics cluster resource for this storage connection.
type ClusterRef struct {
	// Name of the cluster resource.
	Name string `json:"name"`
	// Namespace of the cluster resource. If not specified, defaults to the same namespace as the VMStorageConnection.
	Namespace string `json:"namespace,omitempty"`
	// Kind is the type of cluster resource to configure. Must be either "VTCluster" or "VLCluster".
	// +kubebuilder:validation:Enum=VTCluster;VLCluster
	Kind string `json:"kind"`
}

// VMStorageConnectionStatus defines the observed state of VMStorageConnection.
type VMStorageConnectionStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VMStorageConnection is the Schema for the vmstorageconnections API.
type VMStorageConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMStorageConnectionSpec   `json:"spec,omitempty"`
	Status VMStorageConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VMStorageConnectionList contains a list of VMStorageConnection.
type VMStorageConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMStorageConnection `json:"items"`
}
