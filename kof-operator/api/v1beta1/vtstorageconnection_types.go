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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VTStorageConnectionSpec defines the desired state of VTStorageConnection.
type VTStorageConnectionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// VTClusterRef references the VTCluster configuration that this storage connection should use.
	VTClusterRef VTClusterRef `json:"vt_cluster_ref"`
	// TargetStorageNode defines the connection details for the VictoriaMetrics storage node that this VTCluster should connect to.
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

// VTClusterRef defines the reference to the VTCluster configuration for this storage connection.
type VTClusterRef struct {
	// Name of the VTCluster resource that this storage connection should use.
	Name string `json:"name"`
	// Namespace of the VTCluster resource. If not specified, it defaults to the same namespace as the VTStorageConnection.
	Namespace string `json:"namespace,omitempty"`
}

// VTStorageConnectionStatus defines the observed state of VTStorageConnection.
type VTStorageConnectionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VTStorageConnection is the Schema for the vtstorageconnections API.
type VTStorageConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VTStorageConnectionSpec   `json:"spec,omitempty"`
	Status VTStorageConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VTStorageConnectionList contains a list of VTStorageConnection.
type VTStorageConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VTStorageConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VTStorageConnection{}, &VTStorageConnectionList{})
}
