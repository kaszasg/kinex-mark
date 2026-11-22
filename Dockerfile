# syntax=docker/dockerfile:1

# 1. Static web build
FROM --platform=$BUILDPLATFORM node:26-alpine AS web
WORKDIR /src/web
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# 2. Go binary with the embedded web build
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ENV GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/embed.go
COPY --from=web /src/web/build web/build
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/kinex-mark ./cmd/kinex-mark
RUN mkdir -p /out/data

# 3. Binary export only: docker build --target binary --output type=local,dest=dist .
FROM scratch AS binary
COPY --from=server /out/kinex-mark /kinex-mark

# 4. Runtime image: no shell, runs as the nonroot user (uid 65532)
FROM gcr.io/distroless/static-debian13:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.source="https://github.com/kaszasg/kinex-mark" \
      org.opencontainers.image.title="Kinex Mark" \
      org.opencontainers.image.description="KINEXIUM Bookmark Manager" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="$VERSION"
COPY --from=server /out/kinex-mark /app/kinex-mark
COPY --from=server --chown=65532:65532 /out/data /data
ENV APP_ADDR=:8080 APP_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --start-interval=2s --retries=3 \
    CMD ["/app/kinex-mark", "-healthcheck"]
ENTRYPOINT ["/app/kinex-mark"]
