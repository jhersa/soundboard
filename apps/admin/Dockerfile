# syntax=docker/dockerfile:1
# Build estático del panel. No genera una imagen para correr: se exporta dist/ con
#   docker buildx build -f apps/admin/Dockerfile --output type=local,dest=out/admin \
#     --build-arg VITE_API_URL=https://api.example --build-arg VITE_GOOGLE_CLIENT_ID=... .
# VITE_DEV_TOKEN no se pasa nunca: solo se lee en modo dev.

ARG NODE_VERSION=24

FROM node:${NODE_VERSION}-slim AS build
WORKDIR /repo
RUN npm install -g pnpm@11.3.0
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml tsconfig.base.json ./
COPY packages/shared/package.json packages/shared/
COPY apps/admin/package.json apps/admin/
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter @soundboard/admin...
COPY packages/shared packages/shared
COPY apps/admin apps/admin
ARG VITE_API_URL
ARG VITE_GOOGLE_CLIENT_ID
RUN test -n "$VITE_API_URL" && test -n "$VITE_GOOGLE_CLIENT_ID" \
  && pnpm --filter @soundboard/admin build

FROM scratch AS dist
COPY --from=build /repo/apps/admin/dist /
