# syntax=docker/dockerfile:1
# Processor de Soundboard: binario de Go + ffmpeg, cwebp y yt-dlp. Se construye desde la raíz:
#   docker build -f apps/processor/Dockerfile -t soundboard-processor .
# o, para desarrollo local: docker compose -f apps/processor/compose.yaml up --build

ARG GO_VERSION=1.27

# ---- 1. build: compila a la plataforma de destino sin emulación (cross-compile) ----
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
# El módulo usa packages/gosig por `replace ../../packages/gosig`: se conserva esa disposición.
WORKDIR /src/apps/processor
COPY packages/gosig /src/packages/gosig
COPY apps/processor/go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/processor/ ./
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/processor ./cmd/processor

# ---- 2. runtime: Alpine + ffmpeg + cwebp + yt-dlp ----
FROM alpine:3 AS runtime
ARG TARGETARCH
# yt-dlp como binario standalone (PyInstaller para musl) y propiedad del usuario `app`, para que
# `yt-dlp -U` pueda actualizarlo al arrancar. Se usa este build y no el zipapp porque trae curl_cffi,
# que TikTok exige para la impersonación de navegador ("Unexpected response from webpage request").
# curl solo se necesita para bajarlo: se desinstala en el mismo paso.
RUN apk add --no-cache ffmpeg libwebp-tools ca-certificates \
  && apk add --no-cache --virtual .fetch curl \
  && adduser -D -u 1000 app \
  && case "$TARGETARCH" in arm64) asset=yt-dlp_musllinux_aarch64 ;; *) asset=yt-dlp_musllinux ;; esac \
  && curl -fsSL "https://github.com/yt-dlp/yt-dlp/releases/latest/download/$asset" -o /usr/local/bin/yt-dlp \
  && chmod 0755 /usr/local/bin/yt-dlp \
  && chown app:app /usr/local/bin/yt-dlp \
  && apk del .fetch
COPY --from=build /out/processor /usr/local/bin/processor
COPY --chmod=0755 apps/processor/entrypoint.sh /entrypoint.sh
ENV PORT=8787 LOG_FORMAT=json
EXPOSE 8787
USER app
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s CMD ["/usr/local/bin/processor", "healthcheck"]
ENTRYPOINT ["/entrypoint.sh"]
