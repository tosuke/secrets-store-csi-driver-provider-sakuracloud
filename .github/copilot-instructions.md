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
- Lint: `golangci-lint run`
- Format: `golangci-lint fmt`

## Repository Structure

- `cmd/`: Contains the main application entrypoint.
- `server/`: Contains the gRPC server implementation.
- `config/`: Contains configuration parsing logic.
- `e2e/`: End-to-end tests.
- `Dockerfile`: Used to build the Docker image.
- `Makefile`: Contains helper commands for development.

## Commit and Pull Request Conventions

- Pull Request titles and commit messages should follow the Conventional Commits format below.
- Each commit should represent a single logical change (e.g., a new feature, a bug fix, a documentation update).

### Conventional Commit Format Explained

Conventional Commits is a specification for writing consistent and meaningful commit messages. The format is:

```
<type>(<scope>): <description>
```

- **type**: Describes the kind of change. Common types include:
    - `feat`: A new feature
    - `fix`: A bug fix
    - `docs`: Documentation only changes
    - `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
    - `refactor`: A code change that neither fixes a bug nor adds a feature
    - `test`: Adding or correcting tests
    - `chore`: Other changes that don't modify src or test files (build process, auxiliary tools, etc)
- **scope**: A short description of the area affected (optional), e.g. `server`, `e2e`.
- **description**: A brief summary of the change.

#### Examples

- `feat(server): add gRPC health check endpoint`
- `fix(e2e): correct manifest path`
- `docs: update README for usage`

Using this format makes commit history easier to read and enables automated changelog generation.
