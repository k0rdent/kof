package v1beta1

import (
	kofv1alpha1 "github.com/k0rdent/kof/kof-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// nolint:dupl
func (beta *PromxyServerGroup) ConvertTo(alphaRaw conversion.Hub) error {
	alpha := alphaRaw.(*kofv1alpha1.PromxyServerGroup)
	alpha.ObjectMeta = beta.ObjectMeta
	alpha.Spec.ClusterName = beta.Spec.ClusterName
	alpha.Spec.Targets = beta.Spec.Targets
	alpha.Spec.PathPrefix = beta.Spec.PathPrefix
	alpha.Spec.Scheme = beta.Spec.Scheme
	alpha.Spec.HttpClient.DialTimeout = beta.Spec.HttpClient.DialTimeout
	alpha.Spec.HttpClient.TLSConfig.InsecureSkipVerify = beta.Spec.HttpClient.TLSConfig.InsecureSkipVerify
	alpha.Spec.HttpClient.BasicAuth.CredentialsSecretName = beta.Spec.HttpClient.BasicAuth.CredentialsSecretName
	alpha.Spec.HttpClient.BasicAuth.UsernameKey = beta.Spec.HttpClient.BasicAuth.UsernameKey
	alpha.Spec.HttpClient.BasicAuth.PasswordKey = beta.Spec.HttpClient.BasicAuth.PasswordKey
	return nil
}

// nolint:dupl
func (beta *PromxyServerGroup) ConvertFrom(alphaRaw conversion.Hub) error {
	alpha := alphaRaw.(*kofv1alpha1.PromxyServerGroup)
	beta.ObjectMeta = alpha.ObjectMeta
	beta.Spec.ClusterName = alpha.Spec.ClusterName
	beta.Spec.Targets = alpha.Spec.Targets
	beta.Spec.PathPrefix = alpha.Spec.PathPrefix
	beta.Spec.Scheme = alpha.Spec.Scheme
	beta.Spec.HttpClient.DialTimeout = alpha.Spec.HttpClient.DialTimeout
	beta.Spec.HttpClient.TLSConfig.InsecureSkipVerify = alpha.Spec.HttpClient.TLSConfig.InsecureSkipVerify
	beta.Spec.HttpClient.BasicAuth.CredentialsSecretName = alpha.Spec.HttpClient.BasicAuth.CredentialsSecretName
	beta.Spec.HttpClient.BasicAuth.UsernameKey = alpha.Spec.HttpClient.BasicAuth.UsernameKey
	beta.Spec.HttpClient.BasicAuth.PasswordKey = alpha.Spec.HttpClient.BasicAuth.PasswordKey
	return nil
}
