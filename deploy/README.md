# Deployment Manifests

This directory contains Kubernetes manifests for deploying the Sakura Cloud Secrets Store CSI Driver Provider.

## Quick Start

1. **Install the provider using Kustomize:**
   ```bash
   kubectl apply -k deploy/
   ```

2. **Create credentials secret:**
   ```bash
   # Create the secret with your Sakura Cloud API credentials
   kubectl create secret generic sakuracloud-credentials \
     --from-literal=access-token="your-access-token" \
     --from-literal=access-token-secret="your-access-token-secret" \
     -n kube-system
   ```

3. **Use the examples:**
   ```bash
   # Apply the example SecretProviderClass (update the vaultID first)
   kubectl apply -f deploy/examples/secretproviderclass.yaml
   
   # Apply the example pod
   kubectl apply -f deploy/examples/pod.yaml
   ```

## Files

- `kustomization.yaml`: Kustomize configuration file
- `rbac.yaml`: RBAC resources (ServiceAccount, Role, RoleBinding)  
- `daemonset.yaml`: DaemonSet for running the provider on all nodes
- `examples/`: Example configurations for users

## Customization

You can customize the deployment by:

1. **Changing the image tag:**
   ```bash
   cd deploy
   kustomize edit set image ghcr.io/tosuke/secrets-store-csi-driver-provider-sakuracloud:v1.0.0
   ```

2. **Adding resource limits or requests:**
   Create a patch file and reference it in `kustomization.yaml`

3. **Changing the namespace:**
   ```bash
   cd deploy  
   kustomize edit set namespace your-namespace
   ```

## Prerequisites

- Kubernetes cluster with Secrets Store CSI Driver installed
- Sakura Cloud account with Secret Manager enabled