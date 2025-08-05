#syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang AS base
ENV GOTOOLCHAIN=auto
ENV CGO_ENABLED=0
# renovate: datasource=go depName=github.com/google/go-licenses
ARG GOLICENSES_VERSION=1.6.0
RUN --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/google/go-licenses@v${GOLICENSES_VERSION}
WORKDIR /app
RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

FROM --platform=$BUILDPLATFORM base AS builder
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
    --mount=type=bind,source=.,target=. \
    go build -o /bin/provider -ldflags="-s -w" -trimpath ./cmd

FROM --platform=$BUILDPLATFORM base AS licenses
RUN --mount=type=bind,source=.,target=. \
    go-licenses save ./cmd --save_path /licenses --force
RUN --mount=type=bind,source=.,target=. \
    mkdir -p /licenses/go && \
    curl -Lq -o /licenses/go/LICENSE https://raw.githubusercontent.com/golang/go/refs/tags/$(go env GOVERSION)/LICENSE

FROM gcr.io/distroless/static-debian12 AS artifact
WORKDIR /app
COPY --from=builder --chown=nonroot:nonroot /bin/provider /bin/secrets-store-csi-driver-provider-sakuracloud
COPY --from=licenses /licenses /licenses
ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-sakuracloud"]
