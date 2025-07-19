# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes Secrets Store CSI Driver provider for Sakura Cloud Secret Manager. It enables Kubernetes pods to fetch secrets from Sakura Cloud's Secret Manager and mount them as files in the filesystem.

## Key Architecture Components

- **main.go**: Entry point that starts a gRPC server implementing the CSI Driver Provider interface
- **server/server.go**: Core gRPC server implementing the `Mount` and `Version` methods for the CSI Driver Provider
- **server/config.go**: Configuration parsing for SecretProviderClass parameters (vaultID, secrets list)
- **version.go**: Version string definition

The provider works by:
1. Receiving mount requests from the Secrets Store CSI Driver
2. Parsing SecretProviderClass configuration to extract vaultID and secret names
3. Using the `sacloud/secretmanager-api-go` client to fetch secrets from Sakura Cloud
4. Returning secret contents as files to be mounted in the pod

## Development Commands

### Building and Testing
```bash
# Run unit tests
go test ./...

# Run end-to-end tests (requires environment variables)
export SAKURACLOUD_ACCESS_TOKEN="your-access-token"
export SAKURACLOUD_ACCESS_TOKEN_SECRET="your-access-token-secret"  
export SAKURACLOUD_VAULT_ID="your-vault-id"
make e2e-test

# Build Docker image
docker build -t secrets-store-csi-driver-provider-sakuracloud:test .

# Install tools via Aqua
aqua install
```

### Linting and Code Quality
```bash
# Run golangci-lint (installed via Aqua)
golangci-lint run
```

## Environment Setup

The project uses [Aqua](https://aquaproj.github.io/) for tool management. Key tools include:
- golangci-lint for linting
- kind for Kubernetes testing
- kubectl, helm for Kubernetes operations
- bats for e2e testing
- usacloud for Sakura Cloud API operations

## Configuration Structure

The provider accepts configuration via SecretProviderClass parameters:
- `vaultID`: Sakura Cloud Secret Manager Vault ID (can be global or per-secret)
- `secrets`: YAML string containing array of secret objects with `name` and optional `vaultID`

## Testing Architecture

- Unit tests: Standard Go tests in `server/` package
- E2E tests: BATS tests in `e2e/sakuracloud.bats` that:
  - Set up a kind cluster
  - Install Secrets Store CSI Driver via Helm
  - Build and load the provider Docker image
  - Create test secrets in Sakura Cloud
  - Verify secret mounting functionality

## Important Files

- `e2e/manifest/`: Kubernetes manifests for testing
- `Dockerfile`: Multi-stage build for the provider binary
- `aqua.yaml`: Tool dependency management
- `.github/copilot-instructions.md`: Contains development workflow and commit conventions

## Commit Conventions

Follow Conventional Commits specification for commit messages and pull request titles.