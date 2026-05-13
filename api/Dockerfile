# syntax=docker/dockerfile:1.7
# Multi-stage build for the packcalc API. The final image is distroless,
# runs as non-root, and contains a single statically-linked Go binary
# with the OpenAPI spec and Swagger UI embedded — no runtime filesystem
# dependencies and nothing to mount.

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Cache module downloads independently of source changes. The `id=` is
# required by older BuildKit frontends (e.g. Railway's Metal builder);
# modern Docker accepts either form.
COPY go.mod go.sum ./
RUN --mount=type=cache,id=packcalc-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=packcalc-gobuild,target=/root/.cache/go-build \
    go mod download

# Copy the rest of the source and build.
COPY . .
RUN --mount=type=cache,id=packcalc-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=packcalc-gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Distroless static, non-root. Bottom is roughly 2.5 MB on its own; the
# total image weight is dominated by the Go binary.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
