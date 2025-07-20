# secrets-store-csi-driver-provider-sakuracloud

This is a provider for the [Kubernetes Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/), allowing you to fetch secrets from [Sakura Cloud Secret Manager](https://manual.sakura.ad.jp/cloud/appliance/secretsmanager/index.html) and mount them into your pods.

## Features

- Fetches secrets from Sakura Cloud Secret Manager.
- Mounts secrets as files into pods.
- Supports specifying `vaultID` and a list of secrets in the `SecretProviderClass`.

## Installation

TODO

## Usage

1.  **Create a `SecretProviderClass`**

    Create a `SecretProviderClass` resource to define which secrets to fetch.

    ```yaml
    apiVersion: secrets-store.csi.x-k8s.io/v1
    kind: SecretProviderClass
    metadata:
      name: my-sakuracloud-secrets
    spec:
      provider: sakuracloud
      parameters:
        vaultID: "your-vault-id" # Specify your Vault ID
        secrets: |
          - name: "my-secret-1"
          - name: "my-secret-2"
    ```

2.  **Mount Secrets in a Pod**

    Reference the `SecretProviderClass` in your pod's volume mounts.

    ```yaml
    apiVersion: v1
    kind: Pod
    metadata:
      name: my-pod
    spec:
      containers:
        - name: my-container
          image: nginx
          volumeMounts:
            - name: secrets-store-inline
              mountPath: "/mnt/secrets-store"
              readOnly: true
      volumes:
        - name: secrets-store-inline
          csi:
            driver: secrets-store.csi.k8s.io
            readOnly: true
            volumeAttributes:
              secretProviderClass: "my-sakuracloud-secrets"
    ```

    The secrets will be mounted as files in the `/mnt/secrets-store` directory inside the container. Each secret will be available as a separate file using either its `name` or the specified `path` from the secret configuration.

## Parameters Reference

The `SecretProviderClass` supports the following parameters:

### Root Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `vaultID` | string | No* | Default Vault ID to use for all secrets. Can be overridden per secret. Required if not specified per secret. |
| `secrets` | YAML string | Yes | YAML-formatted list of secrets to fetch from Sakura Cloud Secret Manager. |

### Secrets Configuration

The `secrets` parameter accepts a YAML string containing a list of secret configurations. Each secret supports the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the secret in Sakura Cloud Secret Manager. |
| `vaultID` | string | No | Vault ID for this specific secret. If not specified, uses the root `vaultID` parameter. |
| `version` | integer | No | Specific version of the secret to fetch. If not specified, fetches the latest version. |
| `path` | string | No | Relative path where the secret should be mounted. If not specified, uses the secret `name`. |

### Path Validation Rules

The `path` parameter has the following constraints:
- Must be a relative path (cannot start with `/`)
- Cannot contain path traversal sequences (`../`)
- Can be empty (use `name` if path is not specified)

## Examples

### Basic Usage

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: basic-secrets
spec:
  provider: sakuracloud
  parameters:
    vaultID: "123456"
    secrets: |
      - name: "database-password"
      - name: "api-key"
```

### Per-Secret Vault ID

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: multi-vault-secrets
spec:
  provider: sakuracloud
  parameters:
    vaultID: "123456"  # Default vault
    secrets: |
      - name: "common-secret"           # Uses default vault 123456
      - name: "special-secret"
        vaultID: "789012"               # Uses different vault
```

### Version-Specific Secrets

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: versioned-secrets
spec:
  provider: sakuracloud
  parameters:
    vaultID: "123456"
    secrets: |
      - name: "config-file"
        version: 2                      # Fetch version 2 specifically
      - name: "current-token"           # Uses latest version
```

### Custom Mount Paths

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: custom-path-secrets
spec:
  provider: sakuracloud
  parameters:
    vaultID: "123456"
    secrets: |
      - name: "db-config"
        path: "config/database.json"    # Mounted as config/database.json
      - name: "ssl-cert"
        path: "certs/server.crt"        # Mounted as certs/server.crt
      - name: "simple-secret"           # Mounted as simple-secret (uses name)
```

### Advanced Configuration

```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: advanced-secrets
spec:
  provider: sakuracloud
  parameters:
    vaultID: "123456"
    secrets: |
      - name: "prod-database-url"
        version: 3
        path: "config/db.url"
      - name: "staging-api-key"
        vaultID: "789012"
        version: 1
        path: "keys/staging.key"
      - name: "shared-certificate"
        path: "ssl/shared.crt"
```

## Development

### Prerequisites

- Go
- Docker
- [Aqua](https://aquaproj.github.io/)

### Running Tests

To run the end-to-end tests, you need to set the following environment variables:

- `SAKURACLOUD_ACCESS_TOKEN`: Your Sakura Cloud API access token.
- `SAKURACLOUD_ACCESS_TOKEN_SECRET`: Your Sakura Cloud API access token secret.
- `SAKURACLOUD_VAULT_ID`: The ID of the Sakura Cloud Secret Manager Vault to use for tests.

```bash
export SAKURACLOUD_ACCESS_TOKEN="your-access-token"
export SAKURACLOUD_ACCESS_TOKEN_SECRET="your-access-token-secret"
export SAKURACLOUD_VAULT_ID="your-vault-id"
make e2e-test
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
