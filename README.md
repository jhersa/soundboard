# Soundboard

Backend de la app Android de botones de efectos (v2): una API con login de Google donde cada usuario guarda su biblioteca de sonidos, una tienda (catálogo) para agregar sonidos al perfil, un CDN para los audios e imágenes y un panel admin donde se pega un link y un contenedor lo convierte y lo publica.

## Arquitectura

```
App Android ──(ID token de Google)──► API (Go) ──► SQLite: usuarios, sesiones, sonidos, bibliotecas, jobs
     │                                      │   ▲   └─► Almacén de medios: sounds/<sha>.ogg, images/<sha>.<ext>
     └──── descarga audios/imágenes ◄── URL pública del almacén (CDN_BASE_URL)
Panel admin (estático) ──► /v1/admin/jobs ──► Processor (Docker: yt-dlp + ffmpeg + cwebp)
                                                 └── devuelve el resultado firmado (HMAC) a la API
```

La API es un binario estático de Go (imagen `scratch` de ~18 MB, ~6 MB de RAM). Guarda los medios según `GCS_BUCKET` (`apps/api/internal/storage`):
- **Con `GCS_BUCKET`:** en Cloud Storage. Es lo que usa producción.
- **Sin `GCS_BUCKET`:** en una carpeta local que la propia API sirve en `/cdn`. Es lo que se usa en desarrollo.

```
apps/api/        API en Go (net/http + SQLite). Migraciones SQL en migrations/
apps/admin/      Panel admin (Vue 3 + Vite)
apps/processor/  Processor en Go (con ffmpeg, cwebp y yt-dlp) que descarga y convierte los sonidos
packages/gosig/  Firma HMAC en Go, compartida por la API y el processor
packages/shared/ Contratos de la API (zod) que usa el panel
scripts/         migrate-v1.mjs (siembra la biblioteca v1, en local o en GCS) y download.mjs (v1)
infra/           Terraform/OpenTofu (GCP + DNS de Cloudflare) y Ansible (VM con Docker Compose)
```

El processor no tiene credenciales del almacén de medios. Le manda el `.ogg` y la imagen a la API, que:
1. calcula el SHA-256;
2. los guarda con `Cache-Control: immutable`;
3. registra el sonido.

## API (`/v1`)

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| POST | `/auth/google` | — | `{idToken}` de Google → `{token, user}` (sesión de 90 días) |
| POST | `/auth/logout` | usuario | Cierra la sesión |
| GET | `/me` | usuario | Perfil e `isAdmin` |
| GET | `/catalog?q=&cursor=&limit=` | opcional | Tienda: sonidos publicados, con `inLibrary` si hay sesión |
| GET | `/me/library` | usuario | Biblioteca ordenada |
| PUT / DELETE | `/me/library/:soundId` | usuario | Agregar / quitar |
| PUT | `/me/library/order` | usuario | `{ids}` en el nuevo orden |
| POST / GET | `/admin/jobs` | admin | Crear job (`url`, `title`, `imageUrl`, `from`, `to`, `stereo`) / listar |
| POST | `/admin/jobs/:id/retry` | admin | Reintentar un job fallido |
| POST | `/admin/uploads` | admin | Subir un archivo cuando yt-dlp no puede bajar el link; devuelve `upload:<key>` |
| GET / PATCH / DELETE | `/admin/sounds[/:id]` | admin | Listar, renombrar, ocultar/publicar, borrar |

Los admins se definen con `ADMIN_EMAILS` (en local, `apps/api/.env`; en producción, `admin_emails` en `infra/ansible/group_vars/all/main.yml`).

## Desarrollo local

Requisitos: Go 1.27+, Node 22+ (solo para el panel), pnpm, y para el processor sin Docker `brew install yt-dlp ffmpeg webp`.

```bash
pnpm install
cp apps/api/.env.example apps/api/.env                     # PROCESSOR_SECRET, GOOGLE_CLIENT_IDS, ADMIN_EMAILS...
pnpm migrate:v1                                            # copia assets/ e index.json a apps/api/.data
pnpm dev:api                                               # http://localhost:8788 (go run ./cmd/api serve)
# o, para que un teléfono en la misma red Wi-Fi llegue a la API:
pnpm dev:api:lan                                           # http://<IP de la Mac>:8788 (ipconfig getifaddr en0)

# processor (otra terminal), en Docker...
docker compose -f apps/processor/compose.yaml up --build       # escucha en :8790
# ...o sin Docker (requiere yt-dlp, ffmpeg y cwebp en la Mac)
cd apps/processor && PORT=8790 PROCESSOR_SECRET=<el mismo> go run ./cmd/processor serve

# panel (otra terminal)
cp apps/admin/.env.example apps/admin/.env.local && pnpm dev:admin
```

En local, la base es `apps/api/.data/soundboard.db` y los medios están en `apps/api/.data/media`. Con `SERVE_MEDIA=true`, la API los sirve en `/cdn/...` y arma las URLs con el host de cada petición, así funcionan tanto el emulador (`10.0.2.2`) como el teléfono por Wi-Fi.

**Tests:** `pnpm test` corre `go test ./...` para la API, el processor y `packages/gosig` (los tests de integración del processor usan ffmpeg y cwebp reales y se saltan si no están instalados).

## Despliegue

### Producción

| | |
|---|---|
| API | https://api.soundboard.homunculos.suave.sh |
| Panel admin | https://admin.soundboard.homunculos.suave.sh |
| Medios (`CDN_BASE_URL`) | https://storage.googleapis.com/soundboard-media-703607 |

```
Cloudflare DNS (suave.sh, sin proxy)
  api.soundboard.homunculos / admin.soundboard.homunculos ─► A 35.232.70.115
                                                               │
VM e2-micro "soundboard" (GCP zenbash, us-central1-a, Debian 12, 1 GB RAM + 2 GB swap)
  /home/compose/soundboard  ── docker compose
   ├─ nginx      :80 → :443, cert wildcard *.soundboard.homunculos.suave.sh
   │   ├─ admin.* → estáticos del panel (admin/)
   │   └─ api.*   → api:8788
   ├─ api        binario Go (scratch); SQLite en el volumen api-data (/data/soundboard.db)
   ├─ processor  binario Go + yt-dlp + ffmpeg (Alpine); solo red interna (API_BASE_URL=http://api:8788)
   └─ certbot    renueva el cert cada 12 h (DNS-01 en Cloudflare)

Cloud Storage
  soundboard-media-703607    sonidos, imágenes y uploads/ (público por key, sin listar; uploads/ se borra al día)
  soundboard-backups-703607  respaldo diario de la SQLite (09:00 UTC, se conserva 30 días)
Artifact Registry
  us-central1-docker.pkg.dev/zenbash/soundboard/{api,processor}:latest
```

**Por qué así:**
- **Certificado propio:** los nombres son de 4.º nivel en `suave.sh`. El certificado gratuito de Cloudflare no los cubre, así que el DNS va sin proxy y nginx sirve el wildcard de Let's Encrypt.
- **Costo:** la VM y los buckets entran en el free tier de GCP. Lo único que puede cobrarse es la IPv4 estática (~3 USD/mes).
- **Builds fuera de la VM:** las imágenes se construyen en la Mac (`linux/amd64`) y la VM solo hace pull, porque con 1 GB de RAM no puede construir.

**Redesplegar después de cambiar código** (desde `infra/ansible`):
```bash
gcloud auth login                     # si la sesión expiró (el push a Artifact Registry la usa)
ansible-playbook playbook.yml --ask-vault-pass --tags images,app
```

**Operar la VM:**
```bash
ssh alchemist@35.232.70.115
cd /home/compose/soundboard
sudo docker compose ps
sudo docker compose logs -f api processor
sudo docker stats                      # la API usa ~6 MB; el processor ~10 MB en reposo
sudo /usr/local/bin/soundboard-backup  # respaldo manual de la base
```

**Configuración y secretos:**
- **Recursos:** GCP y Cloudflare se definen en `infra/terraform/` y se aplican con OpenTofu. El estado es local (`terraform.tfstate`, ignorado en git).
- **Secretos de Ansible:** van cifrados en `infra/ansible/group_vars/all/vault.yml`.
- **Valores en la VM:** Ansible genera `api.env` y `processor.env` en la VM; no se editan a mano.
- **Cloud Storage:** la API escribe en el bucket con la cuenta de servicio de la VM. No hay claves que guardar.

La guía completa (crear desde cero, sembrar la biblioteca v1, restaurar respaldos, costos) está en [`infra/README.md`](infra/README.md).

### Clientes OAuth de Google

En Google Cloud Console → APIs y servicios → Credenciales hay dos clientes:
- **Cliente Web.** Su origen autorizado es la URL del panel. Su Client ID se usa en tres lugares:
  - `GOOGLE_CLIENT_IDS` (API);
  - `VITE_GOOGLE_CLIENT_ID` (panel);
  - `soundboard.googleWebClientId` en `local.properties` (Android).
- **Cliente Android.** Se registra con el paquete `com.cornytuna.soundboard` y el SHA-1 de los keystores de debug y release (`./gradlew signingReport`).

## Biblioteca v1 (legado)

La app v1 lee `index.json` y `assets/` directo de GitHub. Se conservan hasta que la v1 deje de usarse; después se pueden borrar junto con `scripts/download.mjs`.

### Estructura

```
.
├── assets/             # Audios (.ogg) e imágenes de los botones
├── index.json          # Lista de sonidos que lee la app
└── scripts/
    └── download.mjs    # Script para agregar sonidos
```

#### `index.json`

```json
[
    {
        "title": "uwu",
        "assetId": "3ed045c2881c1bdfdd1758879dad553a2b4188bdaa730856a97eca64e3b2291b",
        "durationMs": 1370
    }
]
```

| Campo | Descripción |
|---|---|
| `title` | Nombre que se muestra en el botón. |
| `assetId` | SHA-256 del archivo de audio. Es también el nombre de sus archivos en `assets/`. |
| `durationMs` | Duración del audio en milisegundos. |

Cada sonido tiene dos archivos en `assets/` con el mismo `assetId` como nombre:

- `<assetId>.ogg`: el audio.
- `<assetId>.<ext>`: la imagen. Puede ser `.webp`, `.gif`, `.svg` u otra extensión, según la imagen original.

### Requisitos

- [Node.js](https://nodejs.org/) 18.3 o superior
- [yt-dlp](https://github.com/yt-dlp/yt-dlp), para YouTube, TikTok y otros sitios
- [ffmpeg](https://ffmpeg.org/), para convertir audio
- [cwebp](https://developers.google.com/speed/webp/docs/cwebp), para convertir imágenes

En macOS:

```bash
brew install node yt-dlp ffmpeg webp
```

### Agregar un sonido

```bash
node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen] [opciones]
```

| Argumento | Descripción |
|---|---|
| `url-del-sonido` | Link de YouTube, TikTok, Epidemic Sound o cualquier sitio que soporte yt-dlp. |
| `titulo` | Nombre del sonido en `index.json`. |
| `url-de-imagen` | Opcional. Imagen o GIF para el botón. |
| `--from <tiempo>` | Opcional. Inicio del corte. |
| `--to <tiempo>` | Opcional. Fin del corte. |
| `--stereo` | Opcional. Guarda el audio en estéreo en lugar de mono. |

Los tiempos se pueden escribir en segundos (`3`, `3.5`) o en formato de reloj (`1:23`, `1:02:03.5`).

### Qué hace el script

**Audio:** lo descarga en su formato original y lo convierte una sola vez con ffmpeg:

- Ogg Vorbis, calidad 4, 44.1 kHz. Android lo reproduce desde las primeras versiones, incluido `SoundPool`.
- Mono por defecto, que reduce el tamaño a la mitad. Con `--stereo` se conservan los dos canales.
- Volumen normalizado a −16 LUFS (EBU R128), para que todos los sonidos suenen parecido aunque vengan de fuentes distintas. Si se usa `--from` o `--to`, el volumen se calcula solo con el fragmento.
- Sin metadatos.

**Imagen:**

- **PNG y JPG** se reducen a un máximo de 384 px en el lado más largo, sin cambiar la proporción ni agrandar las imágenes pequeñas, y se guardan en WebP con calidad 80.
- **GIF, SVG y los demás formatos** se guardan tal cual.

Si algo falla, por ejemplo una imagen inválida o un corte fuera de rango, el script se detiene sin guardar nada.

### Uso en la app Android

- **Reproducción:** usa `SoundPool` para sonidos cortos (`durationMs` de unos 5000 o menos), porque suenan al instante. Para los largos usa `MediaPlayer` o ExoPlayer, que leen el archivo conforme lo reproducen en vez de cargarlo completo en memoria.
- **Imágenes:** la extensión varía entre sonidos. Busca el archivo que empiece con `<assetId>.` y no termine en `.ogg`.
- **Ubicación:** copia los archivos a la carpeta `assets/` de la app, no a `res/raw`. Los nombres de recursos de `res/raw` no pueden empezar con número, y muchos `assetId` sí empiezan con uno.

### Derechos de los sonidos

Los audios e imágenes son propiedad de sus autores. Antes de publicar la app, revisa que tengas permiso para usarlos. En el caso de Epidemic Sound, el script descarga la vista previa pública y sus términos exigen una suscripción activa para usar los sonidos.
