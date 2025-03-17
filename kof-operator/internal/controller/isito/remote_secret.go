// Copyright Istio Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package istio

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"istio.io/istio/istioctl/pkg/multicluster"
	"istio.io/istio/operator/cmd/mesh"
	"istio.io/istio/operator/pkg/component"
	"istio.io/istio/operator/pkg/render"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
	mcluster "istio.io/istio/pkg/kube/multicluster"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	IstioSystemNamespace = "istio-system"
	remoteSecretPrefix   = "istio-remote-secret-"
	configSecretName     = "istio-kubeconfig"
	configSecretKey      = "config"

	clusterNameAnnotationKey = "networking.istio.io/cluster"
)

var (
	errMissingRootCAKey = fmt.Errorf("no %q data found", v1.ServiceAccountRootCAKey)
	errMissingTokenKey  = fmt.Errorf("no %q data found", v1.ServiceAccountTokenKey)

	tokenWaitBackoff = time.Second
)

func CreateRemoteSecret(opt multicluster.RemoteSecretOptions, client kube.CLIClient, ctx context.Context) (*v1.Secret, multicluster.Warning, error) {
	// generate the clusterName if not specified
	if opt.ClusterName == "" {
		uid, err := clusterUID(client.Kube())
		if err != nil {
			return nil, nil, err
		}
		opt.ClusterName = string(uid)
	}

	var secretName string
	switch opt.Type {
	case multicluster.SecretTypeRemote:
		secretName = RemoteSecretNameFromClusterName(opt.ClusterName)
		if opt.ServiceAccountName == "" {
			opt.ServiceAccountName = constants.DefaultServiceAccountName
		}
	case multicluster.SecretTypeConfig:
		secretName = configSecretName
		if opt.ServiceAccountName == "" {
			opt.ServiceAccountName = constants.DefaultConfigServiceAccountName
		}
	default:
		return nil, nil, fmt.Errorf("unsupported type: %v", opt.Type)
	}
	tokenSecret, err := getServiceAccountSecret(client, opt, ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get access token to read resources from local kube-apiserver: %v", err)
	}

	var server string
	var warn multicluster.Warning
	if opt.ServerOverride != "" {
		server = opt.ServerOverride
	} else {
		server, warn, err = getServerFromKubeconfig(client)
		if err != nil {
			return nil, warn, err
		}
	}

	var remoteSecret *v1.Secret
	switch opt.AuthType {
	case multicluster.RemoteSecretAuthTypeBearerToken:
		remoteSecret, err = createRemoteSecretFromTokenAndServer(client, tokenSecret, opt.ClusterName, server, secretName, ctx)
	case multicluster.RemoteSecretAuthTypePlugin:
		authProviderConfig := &api.AuthProviderConfig{
			Name:   opt.AuthPluginName,
			Config: opt.AuthPluginConfig,
		}
		remoteSecret, err = createRemoteSecretFromPlugin(tokenSecret, server, opt.ClusterName, secretName,
			authProviderConfig)
	default:
		err = fmt.Errorf("unsupported authentication type: %v", opt.AuthType)
	}
	if err != nil {
		return nil, warn, err
	}

	remoteSecret.Namespace = opt.Namespace
	return remoteSecret, warn, nil
}

func createRemoteSecretFromPlugin(
	tokenSecret *v1.Secret,
	server, clusterName, secName string,
	authProviderConfig *api.AuthProviderConfig,
) (*v1.Secret, error) {
	caData, ok := tokenSecret.Data[v1.ServiceAccountRootCAKey]
	if !ok {
		return nil, errMissingRootCAKey
	}

	// Create a Kubeconfig to access the remote cluster using the auth provider plugin.
	kubeconfig := createPluginKubeconfig(caData, clusterName, server, authProviderConfig)
	if err := clientcmd.Validate(*kubeconfig); err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %v", err)
	}

	// Encode the Kubeconfig in a secret that can be loaded by Istio to dynamically discover and access the remote cluster.
	return createRemoteServiceAccountSecret(kubeconfig, clusterName, secName)
}

func createBaseKubeconfig(caData []byte, clusterName, server string) *api.Config {
	return &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				CertificateAuthorityData: caData,
				Server:                   server,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{},
		Contexts: map[string]*api.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		CurrentContext: clusterName,
	}
}

func createRemoteServiceAccountSecret(kubeconfig *api.Config, clusterName, secName string) (*v1.Secret, error) { // nolint:interfacer
	var data bytes.Buffer
	if err := latest.Codec.Encode(kubeconfig, &data); err != nil {
		return nil, err
	}
	key := clusterName
	if secName == configSecretName {
		key = configSecretKey
	}
	out := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secName,
			Annotations: map[string]string{
				clusterNameAnnotationKey: clusterName,
			},
			Labels: map[string]string{
				mcluster.MultiClusterSecretLabel: "true",
			},
		},
		Data: map[string][]byte{
			key: data.Bytes(),
		},
	}
	return out, nil
}

func createPluginKubeconfig(caData []byte, clusterName, server string, authProviderConfig *api.AuthProviderConfig) *api.Config {
	c := createBaseKubeconfig(caData, clusterName, server)
	c.AuthInfos[c.CurrentContext] = &api.AuthInfo{
		AuthProvider: authProviderConfig,
	}
	return c
}

func createRemoteSecretFromTokenAndServer(client kube.CLIClient, tokenSecret *v1.Secret, clusterName, server, secName string, ctx context.Context) (*v1.Secret, error) {
	caData, token, err := waitForTokenData(client, tokenSecret, ctx)
	if err != nil {
		return nil, err
	}

	// Create a Kubeconfig to access the remote cluster using the remote service account credentials.
	kubeconfig := createBearerTokenKubeconfig(caData, token, clusterName, server)
	if err := clientcmd.Validate(*kubeconfig); err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %v", err)
	}

	// Encode the Kubeconfig in a secret that can be loaded by Istio to dynamically discover and access the remote cluster.
	return createRemoteServiceAccountSecret(kubeconfig, clusterName, secName)
}

func waitForTokenData(client kube.CLIClient, secret *v1.Secret, ctx context.Context) (ca, token []byte, err error) {
	log := log.FromContext(ctx)

	ca, token, err = tokenDataFromSecret(secret)
	if err == nil {
		return
	}

	log.Info("Waiting for data to be populated", "secret name", secret.Name)
	err = backoff.Retry(
		func() error {
			secret, err = client.Kube().CoreV1().Secrets(secret.Namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			ca, token, err = tokenDataFromSecret(secret)
			return err
		},
		backoff.WithMaxRetries(backoff.NewConstantBackOff(tokenWaitBackoff), 5))
	return
}

func tokenDataFromSecret(tokenSecret *v1.Secret) (ca, token []byte, err error) {
	var ok bool
	ca, ok = tokenSecret.Data[v1.ServiceAccountRootCAKey]
	if !ok {
		err = errMissingRootCAKey
		return
	}
	token, ok = tokenSecret.Data[v1.ServiceAccountTokenKey]
	if !ok {
		err = errMissingTokenKey
		return
	}
	return
}

func getServerFromKubeconfig(client kube.CLIClient) (string, multicluster.Warning, error) {
	restCfg := client.RESTConfig()
	if restCfg == nil {
		return "", nil, fmt.Errorf("failed getting REST config from client")
	}
	server := restCfg.Host
	if strings.Contains(server, "127.0.0.1") || strings.Contains(server, "localhost") {
		return server, fmt.Errorf(
			"server in Kubeconfig is %s. This is likely not reachable from inside the cluster, "+
				"if you're using Kubernetes in Docker, pass --server with the container IP for the API Server",
			server), nil
	}
	return server, nil, nil
}

func CopyRemoteSecretProfileName(childClusterName string) string {
	return fmt.Sprintf("%s-istio-remote-secret", childClusterName)
}

func RemoteSecretNameFromClusterName(clusterName string) string {
	return remoteSecretPrefix + clusterName
}

func getServiceAccountSecret(client kube.CLIClient, opt multicluster.RemoteSecretOptions, ctx context.Context) (*v1.Secret, error) {
	// Create the service account if it doesn't exist.
	serviceAccount, err := getOrCreateServiceAccount(client, opt)
	if err != nil {
		return nil, err
	}

	if !kube.IsAtLeastVersion(client, 24) {
		return legacyGetServiceAccountSecret(serviceAccount, client, opt)
	}
	return getOrCreateServiceAccountSecret(serviceAccount, client, opt, ctx)
}

func legacyGetServiceAccountSecret(
	serviceAccount *v1.ServiceAccount,
	client kube.CLIClient,
	opt multicluster.RemoteSecretOptions,
) (*v1.Secret, error) {
	if len(serviceAccount.Secrets) == 0 {
		return nil, fmt.Errorf("no secret found in the service account: %s", serviceAccount)
	}

	secretName := ""
	secretNamespace := ""
	if opt.SecretName != "" {
		found := false
		for _, secret := range serviceAccount.Secrets {
			if secret.Name == opt.SecretName {
				found = true
				secretName = secret.Name
				secretNamespace = secret.Namespace
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("provided secret does not exist: %s", opt.SecretName)
		}
	} else {
		if len(serviceAccount.Secrets) == 1 {
			secretName = serviceAccount.Secrets[0].Name
			secretNamespace = serviceAccount.Secrets[0].Namespace
		} else {
			return nil, fmt.Errorf("wrong number of secrets (%v) in serviceaccount %s/%s, please use --secret-name to specify one",
				len(serviceAccount.Secrets), opt.Namespace, opt.ServiceAccountName)
		}
	}

	if secretNamespace == "" {
		secretNamespace = opt.Namespace
	}
	return client.Kube().CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
}

func getOrCreateServiceAccountSecret(
	serviceAccount *v1.ServiceAccount,
	client kube.CLIClient,
	opt multicluster.RemoteSecretOptions,
	ctx context.Context,
) (*v1.Secret, error) {
	log := log.FromContext(ctx)

	// manually specified secret, make sure it references the ServiceAccount
	if opt.SecretName != "" {
		secret, err := client.Kube().CoreV1().Secrets(opt.Namespace).Get(ctx, opt.SecretName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("could not get specified secret %s/%s: %v",
				opt.Namespace, opt.SecretName, err)
		}
		if err := secretReferencesServiceAccount(serviceAccount, secret); err != nil {
			return nil, err
		}
		return secret, nil
	}

	// first try to find an existing secret that references the SA
	// TODO will the SA have any reference to secrets anymore, can we avoid this list?
	allSecrets, err := client.Kube().CoreV1().Secrets(opt.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed listing secrets in %s: %v", opt.Namespace, err)
	}
	for _, item := range allSecrets.Items {
		secret := item
		if secretReferencesServiceAccount(serviceAccount, &secret) == nil {
			return &secret, nil
		}
	}

	// finally, create the sa token secret manually
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#manually-create-a-service-account-api-token
	// TODO ephemeral time-based tokens are preferred; we should re-think this
	log.Info("Creating token secret", "service account", serviceAccount.Name)
	secretName := tokenSecretName(serviceAccount.Name)
	return client.Kube().CoreV1().Secrets(opt.Namespace).Create(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Annotations: map[string]string{v1.ServiceAccountNameKey: serviceAccount.Name},
		},
		Type: v1.SecretTypeServiceAccountToken,
	}, metav1.CreateOptions{})
}

func secretReferencesServiceAccount(serviceAccount *v1.ServiceAccount, secret *v1.Secret) error {
	if secret.Type != v1.SecretTypeServiceAccountToken ||
		secret.Annotations[v1.ServiceAccountNameKey] != serviceAccount.Name {
		return fmt.Errorf("secret %s/%s does not reference ServiceAccount %s",
			secret.Namespace, secret.Name, serviceAccount.Name)
	}
	return nil
}

func getOrCreateServiceAccount(client kube.CLIClient, opt multicluster.RemoteSecretOptions) (*v1.ServiceAccount, error) {
	if sa, err := client.Kube().CoreV1().ServiceAccounts(opt.Namespace).Get(
		context.TODO(), opt.ServiceAccountName, metav1.GetOptions{}); err == nil {
		return sa, nil
	} else if !opt.CreateServiceAccount {
		// User chose not to automatically create the service account.
		return nil, fmt.Errorf("failed retrieving service account %s.%s required for creating "+
			"the remote secret (hint: try installing a minimal Istio profile on the cluster first, "+
			"or run with '--create-service-account=true'): %v",
			opt.ServiceAccountName,
			opt.Namespace,
			err)
	}

	if err := createServiceAccount(client, opt); err != nil {
		return nil, err
	}

	// Return the newly created service account.
	sa, err := client.Kube().CoreV1().ServiceAccounts(opt.Namespace).Get(
		context.TODO(), opt.ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed retrieving service account %s.%s after creating it: %v",
			opt.ServiceAccountName, opt.Namespace, err)
	}
	return sa, nil
}

func tokenSecretName(saName string) string {
	return saName + "-istio-remote-secret-token"
}

func createServiceAccount(client kube.CLIClient, opt multicluster.RemoteSecretOptions) error {
	yaml, err := generateServiceAccountYAML(opt)
	if err != nil {
		return err
	}

	// Before we can apply the yaml, we have to ensure the system namespace exists.
	if err := createNamespaceIfNotExist(client, opt.Namespace); err != nil {
		return err
	}

	// Apply the YAML to the cluster.
	return client.ApplyYAMLContents(opt.Namespace, yaml)
}

func generateServiceAccountYAML(opt multicluster.RemoteSecretOptions) (string, error) {
	flags := []string{"installPackagePath=" + opt.ManifestsPath, "values.global.istioNamespace=" + opt.Namespace}
	mfs, _, err := render.GenerateManifest(nil, flags, false, nil, nil)
	if err != nil {
		return "", err
	}
	included := []string{}
	for _, mf := range mfs {
		if mf.Component != component.BaseComponentName && mf.Component != component.PilotComponentName {
			continue
		}
		for _, m := range mf.Manifests {
			if m.GetKind() == "ClusterRole" || m.GetKind() == "ClusterRoleBinding" {
				included = append(included, m.Content)
			}
			if m.GetKind() == "ServiceAccount" && m.GetName() == "istio-reader-service-account" {
				included = append(included, m.Content)
			}
		}
	}

	return strings.Join(included, mesh.YAMLSeparator), nil
}

func createNamespaceIfNotExist(client kube.Client, ns string) error {
	if _, err := client.Kube().CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{}); err != nil {
		if _, err := client.Kube().CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed creating namespace %s: %v", ns, err)
		}
	}
	return nil
}

func createBearerTokenKubeconfig(caData, token []byte, clusterName, server string) *api.Config {
	c := createBaseKubeconfig(caData, clusterName, server)
	c.AuthInfos[c.CurrentContext] = &api.AuthInfo{
		Token: string(token),
	}
	return c
}

func clusterUID(client kubernetes.Interface) (types.UID, error) {
	kubeSystem, err := client.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return kubeSystem.UID, nil
}
