# syntax=docker/dockerfile:1.8

FROM node:24.17.0-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.5-alpine AS go-build
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build -trimpath \
      -ldflags="-s -w -X github.com/kejilion/kejilion-panel/internal/version.Version=${VERSION}" \
      -o /out/paneld ./cmd/paneld

FROM scratch
ARG VERSION=dev
LABEL org.opencontainers.image.title="Kejilion Panel" \
      org.opencontainers.image.description="Safe web management plane for kejilion.sh hosts" \
      org.opencontainers.image.source="https://github.com/kejilion/kejilion-panel" \
      org.opencontainers.image.version="${VERSION}"
COPY --from=go-build /out/paneld /paneld
COPY --from=web-build /src/web/dist /app/web
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/paneld"]
