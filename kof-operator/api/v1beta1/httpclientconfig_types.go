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

// HTTPClientConfig defines the HTTP client TLS and BasicAuth config used when
// connecting to a regional cluster's read endpoint (e.g. from
// RegionalClusterConfigMap.GetHttpClientConfig).
type HTTPClientConfig struct {
	// DialTimeout in the string representation (e.g. 1s)
	DialTimeout metav1.Duration `json:"dial_timeout,omitempty"`
	TLSConfig   TLSConfig       `json:"tls_config,omitempty"`
	BasicAuth   BasicAuth       `json:"basic_auth,omitempty"`
}

// BasicAuth holds basic auth credentials reference.
type BasicAuth struct {
	CredentialsSecretName string `json:"credentials_secret_name,omitempty"`
	UsernameKey           string `json:"username_key,omitempty"`
	PasswordKey           string `json:"password_key,omitempty"`
}

// TLSConfig holds TLS verification options.
type TLSConfig struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}
