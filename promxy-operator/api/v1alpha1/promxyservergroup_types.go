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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PromxyServerGroupSpec defines the desired state of PromxyServerGroup
type PromxyServerGroupSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	ClusterName string           `json:"cluster_name,omitempty"`
	Targets     []string         `json:"targets,omitempty"`
	PathPrefix  string           `json:"path_prefix,omitempty"`
	Scheme      string           `json:"scheme,omitempty"`
	HttpClient  HTTPClientConfig `json:"http_client,omitempty"`
}

// HTTPClientConfig extends prometheus' HTTPClientConfig
type HTTPClientConfig struct {
	DialTimeout metav1.Duration `json:"dial_timeout,omitempty"`
	TLSConfig   TLSConfig       `json:"tls_config,omitempty"`
	BasicAuth   BasicAuth       `json:"basic_auth,omitempty"`
}

// BasicAuth part of prometheus HTTPClientConfig with json annotation
type BasicAuth struct {
	CredentialsSecretName string `json:"credentials_secret_name,omitempty"`
	UsernameKey           string `json:"username_key,omitempty"`
	PasswordKey           string `json:"password_key,omitempty"`
}

// TLSConfig part of prometheus HTTPClientConfig with json annotation
type TLSConfig struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// PromxyServerGroupStatus defines the observed state of PromxyServerGroup
type PromxyServerGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PromxyServerGroup is the Schema for the promxyservergroups API
type PromxyServerGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PromxyServerGroupSpec   `json:"spec,omitempty"`
	Status PromxyServerGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PromxyServerGroupList contains a list of PromxyServerGroup
type PromxyServerGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromxyServerGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PromxyServerGroup{}, &PromxyServerGroupList{})
}
