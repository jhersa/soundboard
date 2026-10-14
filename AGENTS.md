# AGENTS.md — Soundboard backend

Backend de la app Android Soundboard (repo de la app: `AndroidStudioProjects/Soundboard`). Monorepo pnpm con una API en Go, un panel admin en Vue, un processor en Go (con ffmpeg y yt-dlp) que convierte links a audio, y la biblioteca v1 heredada (`assets/` + `index.json`).

Escribe código, comentarios, mensajes de error y documentación en **español**.

## Estructura

```
apps/api/         API en Go (net/http + SQLite con modernc.org/sqlite, sin CGO). Imagen scratch
  cmd/api/        binario: serve | migrate | backup <destino|-> | sql <archivo> | healthcheck
  internal/httpapi/  rutas: auth, catalog, library, admin, internal (las que solo llama el processor)
  internal/storage/  medios: GCS (JSON API + cuenta de servicio) o carpeta local (sirve /cdn)
  internal/google/   verificación del ID token de Google (go-oidc)
  internal/db/       SQLite y migraciones (tabla d1_migrations, heredada de D1)
  migrations/     SQL numerado (0001_init.sql, ...), embebido en el binario
apps/admin/       Panel admin: Vue 3 (<script setup>) + Vite. Componentes en src/components/
                  Estilos solo con variables `--color-*` de brand/ (importa brand/generated/tokens.css); no definas colores en styles.css
apps/processor/   Processor en Go: servidor HTTP + cola de un job a la vez + pipeline que ejecuta yt-dlp, ffmpeg y cwebp
  cmd/processor/  binario: serve | healthcheck
  internal/server/    POST /process (firmado), cola, reporte de status y result a la API
  internal/pipeline/  descarga, ffmpeg (Vorbis + loudnorm), imagen a WebP; ParseTime y AudioFilter
  Dockerfile      multistage: build Go + Alpine con ffmpeg, libwebp-tools y yt-dlp_musllinux
packages/gosig/   Firma HMAC en Go; la usan la API y el processor (módulo soundboard/gosig, por `replace`)
brand/            Identidad visual portable (tokens.json → CSS y Compose, logos). Ver brand/README.md; la referencia es la app Android
packages/shared/  Contratos de la API (esquemas zod + tipos) para el panel; la API y el processor los implementan en Go
scripts/          migrate-v1.mjs (v1 → apps/api/.data o GCS) y download.mjs (script v1, legado)
infra/            Despliegue: terraform/ (GCP + DNS de Cloudflare) y ansible/ (VM con Docker Compose)
assets/, index.json  Biblioteca v1 que la app v1 lee desde GitHub. No modificar ni borrar
```

## Comandos

Requisitos: Go 1.27+, Node 22+ (solo para el panel), pnpm 11. El processor sin Docker requiere `yt-dlp`, `ffmpeg` y `cwebp`.

```bash
pnpm install
pnpm test                                     # todos los tests
pnpm typecheck                                # tsc / vue-tsc / go vet en todos los paquetes
pnpm --filter @soundboard/api test            # solo la API (go test ./...)
pnpm dev:api                                  # API en http://localhost:8788 (go run ./cmd/api serve, lee apps/api/.env)
pnpm dev:api:lan                              # igual, pero escuchando en 0.0.0.0 (teléfono por Wi-Fi)
pnpm dev:admin                                # panel en http://localhost:5173
docker compose -f apps/processor/compose.yaml up --build   # processor en :8790
(cd apps/processor && go run ./cmd/processor serve)         # o sin Docker, con .env (PORT=8790)
pnpm migrate:v1                               # siembra apps/api/.data con la biblioteca v1
```

Antes de dar un cambio por terminado, corre `pnpm typecheck` y `pnpm test`.

## Arquitectura y reglas

- **Contratos:** un cambio a un request o response va en tres lugares:
  1. `packages/shared/src/index.ts`, que usa el panel (y documenta el contrato interno);
  2. los DTOs y validaciones de Go (`apps/api/internal/httpapi/dto.go` y `validate.go`; el processor, en `internal/server/server.go`);
  3. la app Android, que replica los DTOs a mano en `core/data/.../remote/ApiDtos.kt`.

  Un cambio incompatible rompe la app publicada. Agrega campos de forma compatible, sin omitir nulos (en Go, punteros sin `omitempty`), y avisa del cambio en Android.
- **IDs de sonido:** son el SHA-256 del `.ogg` y los calcula la API, nunca el cliente. Los archivos viven en el almacén de medios como `sounds/<sha>.ogg` y `images/<sha>.<ext>`, con `Cache-Control: immutable`. Nunca sobrescribas un key con contenido distinto.
- **Flujo de un job:**
  1. El panel llama a `POST /v1/admin/jobs`.
  2. La API le envía el job al processor (`POST /process`), firmado con HMAC.
  3. El processor reporta su avance a `/v1/internal/jobs/:id/status`.
  4. Al terminar, sube el resultado en multipart a `/v1/internal/jobs/:id/result`.

  El processor **no** tiene credenciales del bucket; solo la API escribe en él.
- **Firma HMAC:** se calcula sobre `${timestamp}.${sha256(body)}`; en los GET se firma la key. Hay una sola implementación: `packages/gosig`, compartida por la API y el processor. No crees otra. Su test fija un vector (secreto, timestamp, cuerpo y firma esperada): si lo cambias, una API y un processor de versiones distintas dejan de entenderse y los jobs en vuelo se rechazan.
- **Archivos subidos desde el panel:** son el respaldo cuando yt-dlp falla. Se guardan en `uploads/` y viajan en el job como URLs `upload:<key>`. La API los borra cuando el job termina bien.
- **Seguridad de URLs:** los jobs solo aceptan `http`, `https` y `upload:` (para que yt-dlp no lea `file://`). Mantén esa restricción.
- **Auth:** el ID token de Google se verifica con `go-oidc` contra `GOOGLE_CLIENT_IDS`. La sesión es un token opaco de 90 días y en la base solo se guarda su hash. Los admins se definen con `ADMIN_EMAILS`.
- **Base (SQLite):**
  - Las llaves foráneas están activas (`foreign_keys=ON` al abrir), y borrar un sonido lo quita de todas las bibliotecas por cascada.
  - Las fechas son cadenas ISO (`strftime('%Y-%m-%dT%H:%M:%fZ')`, `db.TimeFormat` en Go).
  - Un cambio de esquema siempre es una migración nueva; nunca edites una migración ya aplicada.
- **Catálogo:** usa paginación por keyset (`created_at DESC, id DESC`) con un cursor en base64 de `{"createdAt","id"}` (compatible con los cursores que emitía la API en TS).
- **Medios en local:** si `CDN_BASE_URL` está vacío y `SERVE_MEDIA=true`, la API sirve los archivos en `/cdn/...` y arma las URLs con el host de cada petición. Así funcionan tanto el emulador Android (`10.0.2.2`) como el navegador (`localhost`).

## Despliegue (`infra/`)

- **Producción:**
  - VM e2-micro de GCP (1 GB de RAM) con Docker Compose, que corre nginx, certbot, api y processor.
  - La API usa su base en `/data/soundboard.db`, dentro del volumen `api-data`.
  - Los medios van a Cloud Storage con la cuenta de servicio de la VM; no hay claves HMAC.
- **DNS:** los registros de Cloudflare van **sin proxy**. Los nombres `*.soundboard.homunculos.suave.sh` son de 4.º nivel y Universal SSL no los cubre; el cert es un wildcard de Let's Encrypt.
- **Builds:** las imágenes se construyen en la Mac (`linux/amd64`) y se publican en Artifact Registry; la VM hace pull. No construyas en la VM. La infraestructura se aplica con OpenTofu (`tofu`).
- **Memoria:** la API usa ~6 MB en reposo; el pico es un upload del panel (hasta 50 MB en memoria; `mem_limit` 192m).
- **Comandos con efecto real:** `tofu apply`, `ansible-playbook` y `seed.yml` afectan producción. No los ejecutes sin que el usuario lo pida.

## Tests

- **API:** `go test ./...`.
  - `internal/httpapi/helpers_test.go` crea el servidor con SQLite `:memory:`, medios en una carpeta temporal, un verificador de Google falso (el token es `"<sub>|<email>"`) y un processor falso como `http.RoundTripper`.
  - `call()` espera los envíos al processor en segundo plano (`Server.Wait`).
- **Processor:** `go test ./...`.
  - `internal/pipeline` se prueba con un `Runner` falso (los argumentos de ffmpeg, cwebp y yt-dlp quedan fijados por tests) y con ffmpeg y cwebp reales en `integration_test.go`, que se salta si no están en el PATH.
  - `internal/server` usa una API falsa que verifica la firma de cada llamada. `harness.stop()` espera a que la cola se vacíe antes de cerrarla: `Server.Stop` descarta (y reporta como fallidos) los jobs que no empezaron.
- **Panel:** no tiene tests. Verifícalo con `pnpm dev:admin` y `VITE_DEV_TOKEN` en `.env.local`. Ese token solo se lee en modo dev y nunca debe llegar al build de producción.

## Configuración y secretos

- **API:** en local, `apps/api/.env` (copia de `.env.example`, la carga el binario al arrancar). En producción, Ansible genera `api.env` desde `infra/ansible/roles/app/templates/api.env.j2`.
- **Panel:** `apps/admin/.env.local` guarda `VITE_API_URL`, `VITE_GOOGLE_CLIENT_ID` y `VITE_DEV_TOKEN`.
- **Processor:** `PROCESSOR_SECRET`, `PORT` (8787), `API_BASE_URL` (en Docker local: `http://host.docker.internal:8788`) y `YTDLP_COOKIES` (opcional). Sin Docker lee `apps/processor/.env`.
- No hagas commit de `.env*`, `apps/api/.data/` ni `dist/`.

## Precauciones

- En macOS el puerto 5000 lo ocupa el Receptor AirPlay; la API usa 8788.
- Si `apt-get` falla en Docker con "invalid signature", casi siempre es que el disco de Docker Desktop está lleno (revisa con `docker system df`).
- `pnpm-workspace.yaml` no tiene `allowBuilds`: ninguna dependencia necesita scripts de instalación. Si agregas una que sí, apruébala ahí.
- `apps/api`, `apps/processor` y `packages/gosig` tienen un `package.json` mínimo solo para que `pnpm test` y `pnpm typecheck` recorran también el código Go; no tienen dependencias de Node.
