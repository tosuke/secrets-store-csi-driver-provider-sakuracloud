# GitHub Copilot Instructions for secrets-store-csi-driver-provider-sakuracloud

## Core Functionality

This project is a provider for the Kubernetes Secrets Store CSI Driver, specifically for Sakura Cloud. Its primary purpose is to fetch secrets from Sakura Cloud's Secret Manager and make them available to Kubernetes pods.

## How it Works

1. A user defines a `SecretProviderClass` Kubernetes resource.
2. The Secrets Store CSI Driver communicates with this provider.
3. The provider's `Mount` function is invoked.
4. It uses the `secretmanager-api-go` library to call the Sakura Cloud Secret Manager API.
5. The secrets are returned to the CSI driver and written to the pod's volume.

## Key Technologies

- Go
- gRPC
- Kubernetes Secrets Store CSI Driver

## Development Flow

- Unit tests: `go test ./...`
- End-to-end tests: `make e2e-test`

## Repository Structure

- `server/`: Contains the gRPC server implementation.
- `e2e/`: End-to-end tests.
- `Dockerfile`: Used to build the Docker image.
- `Makefile`: Contains helper commands for development.

## Commit and Pull Request Conventions

- Pull Request titles and commit messages should follow the Conventional Commits specification.
- Each commit should represent a single logical change (e.g., a new feature, a bug fix, a documentation update).