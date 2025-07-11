#syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang AS builder
ENV GOTOOLCHAIN=auto
ENV CGO_ENABLED=0
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    --mount=type=bind,source=.,target=. \
    go build -o /bin/provider -ldflags="-s -w" -trimpath .

FROM gcr.io/distroless/static-debian12 AS artifact
WORKDIR /app
COPY --from=builder --chown=nonroot:nonroot /bin/provider /bin/secrets-store-csi-driver-provider-sakuracloud
ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-sakuracloud"]
