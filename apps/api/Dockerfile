# syntax=docker/dockerfile:1
# API de Soundboard: binario estático de Go en una imagen scratch. Se construye desde la raíz:
#   docker build -f apps/api/Dockerfile -t soundboard-api .
# La compilación corre en la plataforma del host y cruza a la de destino (sin emulación QEMU).

ARG GO_VERSION=1.27

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
# El módulo de la API usa packages/gosig por `replace ../../packages/gosig`: se conserva esa disposición.
WORKDIR /src/apps/api
COPY packages/gosig /src/packages/gosig
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
  && mkdir -p /out/data /out/tmp

# Usuario 1000 (el mismo `node` de la imagen anterior), así el volumen /data existente sigue siendo suyo.
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/api /api
COPY --from=build --chown=1000:1000 /out/data /data
COPY --from=build --chmod=1777 /out/tmp /tmp
USER 1000:1000
ENV ADDR=:8788 DATABASE_PATH=/data/soundboard.db LOG_FORMAT=json
EXPOSE 8788
VOLUME /data
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
CMD ["serve"]
