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

    The secrets will be mounted as files in the `/mnt/secrets-store` directory inside the container.

## Development

### Prerequisites

- Go
- Docker
- [`aqua`](https://aquaproj.github.io/)

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
