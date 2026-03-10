# Dex SSO

Dex is an Identity Provider (IDP) that can be used to enable Single Sign-On for the KOF applications. It provides centralized authentication by integrating with external identity providers such as Google, GitHub, LDAP, and SAML.

## How to Run Locally

Locally, we use Grafana configured with Dex SSO. If you wish to set it up and run it, please follow these steps:

1. **Deploy the kcm Cluster with a Custom Configuration:**

    The next custom config is added to `config/kind-local.yaml` already
    and is applied on `make kcm-dev-apply`

    ```yaml
    kind: Cluster
    apiVersion: kind.x-k8s.io/v1alpha4
    nodes:
    - role: control-plane
    extraPortMappings:
    - containerPort: 32000
      hostPort: 32000
    - containerPort: 32555
      hostPort: 32555
    ```

2. **Create the Dex Secret File**

    Navigate to the `kof` repository directory in your local environment:

    ```bash
    cd kof
    ```

    Create a `dex.env` file inside the `dev` folder with the following contents:

    ```text
    GOOGLE_CLIENT_ID="<YOUR_GOOGLE_CLIENT_ID>"
    GOOGLE_CLIENT_SECRET="<YOUR_GOOGLE_CLIENT_SECRET>"
    ```

    Replace `<YOUR_GOOGLE_CLIENT_ID>` and `<YOUR_GOOGLE_CLIENT_SECRET>` with the appropriate credentials for the Google OAuth client.

    You can either get these credentials from another developer
    or create your own:
    * Open https://console.cloud.google.com/
    * Create a project, e.g. `dex-test`
    * Configure OAuth screen:
      * Audience: Internal
    * Create OAuth client:
      * App type: Web app
      * Authorized JavaScript origins: `https://dex.example.com:32000`
      * Authorized redirect URIs: `https://dex.example.com:32000/callback`
      * Create, download JSON with creds, copy `client_id` and `client_secret` to `dev/dex.env`

    To test multi-tenancy:
    * Uncomment the "Example OIDC connector configuration for Google with multi-tenancy support"
      in the `charts/kof/values-local.yaml` file.
    * Don't commit it. Don't paste any secrets there.

3. **Add DNS to Your Local Machine**

    Update your local DNS configuration to resolve the Dex domain. Open the `/etc/hosts` file as an admin:

    ```bash
    sudo vim /etc/hosts
    ```

    Add the following line to the end of the file:

    ```text
    127.0.0.1 dex.example.com
    ```

    Save and exit the file.

4. **Deploy kof**

    Follow the KOF setup [guide](https://github.com/k0rdent/kof/blob/main/docs/dev.md#kof) to deploy the KOF on your local cluster.

    To test multi-tenancy:
    * On creation of "Adopted local kind cluster" use the domain from SSO email address, e.g:
      ```bash
      KOF_TENANT_ID=mirantis.com make dev-child-deploy-adopted
      ```
    * You can also add `k0rdent.mirantis.com/kof-tenant-id` label to child `ClusterDeployment` manually.

5. **Access Grafana**

    Set up port-forwarding to access Grafana locally using the following command:

    ```bash
    kubectl port-forward svc/grafana-vm-service 3000:3000 -n kof
    ```

    You should now be able to access Grafana in your browser at `http://localhost:3000`.

    Options:
    * Username and password from [grafana-admin-credentials](https://docs.k0rdent.io/next/admin/kof/kof-grafana/#install-and-enable-grafana):
      full access to all tenants and features.
    * Sign in with Dex, Log in with Email, `git config user.email` and `admin` password:
      access to all tenants, but limited features.
    * Sign in with Dex, Log in with Google:
      access to one tenant, limited features.

**Note:** Without `dex.env` file in `dev` directory, Dex will not start.

## How to Run on a Real Cluster

You can enable and customize Dex's configuration directly within the ClusterDeployment resource. Below is an example configuration for running Dex on a Regional cluster:

```yaml
apiVersion: k0rdent.mirantis.com/v1beta1
kind: ClusterDeployment
metadata:
  name: dex-test
  namespace: kcm-system
  labels:
    k0rdent.mirantis.com/kof-aws-dns-secrets: "true"
    k0rdent.mirantis.com/kof-cluster-role: regional
  annotations: {}
spec:
  template: aws-standalone-cp-1-0-1
  credential: aws-cluster-identity-cred
  config:
    clusterIdentity:
      name: aws-cluster-identity
      namespace: kcm-system
    clusterAnnotations:
      k0rdent.mirantis.com/kof-regional-domain: aws-ue2.kof.example.com
      k0rdent.mirantis.com/kof-cert-email: mail@example.com

      # Use this annotation to modify dex configuration
      k0rdent.mirantis.com/kof-storage-values: |
        dex:
          enabled: true
          config:
            # Add the static clients here
            staticClients:
              - id: grafana-id
                redirectURIs:
                  - "https://grafana.aws-ue2.kof.example.com/login/generic_oauth"
                name: "Grafana"
                secret: grafana-secret

            # Add the connectors here, for example, Google or GitHub
            connectors:
              - type: google
                id: google
                name: Google
                config:
                  clientID: <YOUR_GOOGLE_CLIENT_ID>
                  clientSecret: <YOUR_GOOGLE_CLIENT_SECRET>
                  redirectURI: https://dex.aws-ue2.kof.example.com/callback

        grafana:
          config:
            auth.generic_oauth:
              enabled: "true"
              name: Dex
              scopes: "openid email profile groups offline_access"
              auth_url: https://dex.aws-ue2.kof.example.com/auth
              token_url: https://dex.aws-ue2.kof.example.com/token
              api_url: https://dex.aws-ue2.kof.example.com/userinfo
              client_id: grafana-id
              client_secret: grafana-secret

      region: us-east-2

      controlPlaneNumber: 1
      controlPlane:
        instanceType: t3.large

      workersNumber: 3
      worker:
        instanceType: t3.medium
```

After successfully deploying the cluster, Grafana will support SSO login through Dex. The Dex instance will be accessible at a specific URL. In the example above, the Dex URL is: `https://dex.aws-ue2.kof.example.com`.

## Useful Resources

To learn about the possible configurations for dex, refer to the [Dex Helm Chart Values](https://github.com/dexidp/helm-charts/blob/master/charts/dex/values.yaml).

Explore the allowed dex connectors in the [Dex Connectors Documentation](https://dexidp.io/docs/connectors/).

For general guidance on using and configuring dex, visit the [Dex Documentation](https://dexidp.io/docs/).
