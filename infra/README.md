# Infraestructura: GCP e2-micro + Cloudflare DNS + Cloud Storage

Publica la API en `https://api.soundboard.homunculos.suave.sh` y el panel en `https://admin.soundboard.homunculos.suave.sh`.

```
Internet ─443─> VM e2-micro (GCP, us-central1) ── docker compose
                 ├─ nginx    cert wildcard Let's Encrypt (DNS-01 en Cloudflare)
                 │   ├─ admin.* → estáticos del panel
                 │   └─ api.*   → api:8788
                 ├─ api        binario Go (scratch); SQLite en el volumen api-data
                 ├─ processor  binario Go + yt-dlp + ffmpeg; solo red interna (API_BASE_URL=http://api:8788)
                 └─ certbot    renueva el wildcard cada 12 h
Medios (sounds/, images/, uploads/) → bucket de Cloud Storage, público por key (sin listar)
Respaldos diarios de la SQLite → bucket privado (30 días)
```

**Por qué DNS sin proxy:** los nombres son de 4.º nivel dentro de `suave.sh`, y el certificado gratuito de Cloudflare solo cubre `*.suave.sh`. Por eso los registros van en nube gris y es nginx quien sirve el certificado de Let's Encrypt.

## Requisitos (en tu Mac)

- OpenTofu (`tofu`) o Terraform, `ansible`, Docker con `buildx` y `gcloud`. Ansible llama a `tofu` por defecto; si usas Terraform, cambia `tf_bin` en `ansible/group_vars/all/main.yml`.
- Colecciones de Ansible: `ansible-galaxy collection install -r ansible/requirements.yml`.
- Credenciales:
  ```bash
  gcloud auth application-default login   # la usa OpenTofu
  gcloud auth login                       # la usa `docker push` a Artifact Registry
  ```
  El token de Cloudflare (`Zone:DNS:Edit` y `Zone:Read` sobre `suave.sh`) va en `terraform.tfvars` como `cloudflare_api_token`, o en `export CLOUDFLARE_API_TOKEN=...`.

## 1. Terraform

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars   # tu IP en ssh_source_ranges y el token de Cloudflare
tofu init
tofu plan
tofu apply
```

Esto crea:
- en GCP: la VM y su cuenta de servicio (con acceso a los buckets y al registry), su IP estática, el firewall, el repositorio de Artifact Registry, el bucket de medios y el bucket de respaldos;
- en Cloudflare: los registros A `api.` y `admin.`.

El estado es local (`terraform.tfstate`, ignorado en git); respáldalo en un lugar seguro.

Si la política de la organización bloquea `allUsers` en el bucket ("Domain restricted sharing" o "Public access prevention"), hay que permitirlo en el proyecto, porque la app descarga los audios directo de `storage.googleapis.com`.

## 2. Ansible

```bash
cd infra/ansible
ansible-galaxy collection install -r requirements.yml
cp group_vars/all/vault.yml.example group_vars/all/vault.yml   # rellena los secretos
ansible-vault encrypt group_vars/all/vault.yml
ssh alchemist@$(tofu -chdir=../terraform output -raw vm_ip) true   # acepta la host key
ansible-playbook playbook.yml --ask-vault-pass
```

El playbook:
1. **images** (local): construye las imágenes `api` y `processor` para `linux/amd64` y las publica en Artifact Registry; también genera el panel estático en `out/`. La e2-micro no tiene RAM para construir. El primer build y el primer push tardan porque buildx emula amd64 y se suben todas las capas; en los siguientes solo suben las capas que cambiaron.
2. **base**:
   - swap de 2 GB;
   - chrony, porque la firma HMAC tolera solo 300 s de desfase;
   - Docker con rotación de logs;
   - usuario `compose` y la carpeta `/home/compose/soundboard`.
3. **certs**: emite `*.soundboard.homunculos.suave.sh` con certbot (DNS-01). Solo lo hace la primera vez.
4. **app**:
   - autentica Docker de la VM con Artifact Registry (cuenta de servicio de la VM);
   - sube el panel;
   - genera `compose.yaml`, `api.env`, `processor.env` y la configuración de nginx;
   - la primera vez tras pasar a Go, migra la base de D1 (ver abajo);
   - levanta todo y espera a que `https://api…/v1/catalog` responda.
5. **backup**: cron diario (09:00 UTC) que sube la base a `gs://soundboard-backups-*/db/` (`api backup -`, con `VACUUM INTO`).

**Secretos del vault:**

| Variable | De dónde sale |
|---|---|
| `vault_processor_secret` | `openssl rand -hex 32` |
| `vault_cloudflare_dns_token` | Token de Cloudflare con `DNS:Edit` en `suave.sh`; lo usa certbot |
| `vault_google_web_client_id` | Client ID OAuth **Web** (el del panel) |
| `vault_google_client_ids` | Audiencias que acepta la API, separadas por coma. Con Credential Manager en Android basta el Web |
| `vault_ytdlp_cookies` | Opcional: contenido de un `cookies.txt` de YouTube |

Para redesplegar solo el código: `ansible-playbook playbook.yml --ask-vault-pass --tags images,app`.

## 3. Biblioteca v1 (opcional, una vez)

```bash
ansible-playbook seed.yml --ask-vault-pass
```

Sube `assets/` al bucket con `gcloud storage cp` (mismas keys `sounds/<sha>.ogg`) y aplica el SQL en la base de la VM (`api sql`). Es idempotente.

## 4. Pasos manuales

- **Google Cloud Console → Credenciales → cliente OAuth Web:** agrega `https://admin.soundboard.homunculos.suave.sh` como origen JavaScript autorizado.
- **App Android:** apunta la URL base a `https://api.soundboard.homunculos.suave.sh`. Los contratos no cambian; las URLs de medios ahora apuntan a `storage.googleapis.com`.

## Operación

```bash
ssh alchemist@$(tofu -chdir=infra/terraform output -raw vm_ip)
cd /home/compose/soundboard
sudo docker compose ps
sudo docker compose logs -f api processor
# Los logs son JSON (una línea por evento): se filtran con jq
sudo docker compose logs --no-log-prefix api | jq -c 'select(.level != "INFO")'   # solo avisos y errores
sudo docker compose logs --no-log-prefix processor | jq -c 'select(.job == "<id>")' # un job
sudo /usr/local/bin/soundboard-backup          # respaldo manual
sudo docker compose exec certbot certbot renew --dry-run
```

- **Restaurar un respaldo:**
  1. `sudo docker compose stop api`
  2. Copia el respaldo al volumen:
     ```bash
     gcloud storage cat gs://soundboard-backups-*/db/<respaldo>.sqlite.gz | gunzip > /tmp/soundboard.db
     sudo docker run --rm -v soundboard_api-data:/data -v /tmp:/in busybox:1.37 \
       sh -c 'rm -f /data/soundboard.db-wal /data/soundboard.db-shm && cp /in/soundboard.db /data/ && chown 1000:1000 /data/soundboard.db'
     ```
  3. `sudo docker compose start api`

### Migración desde la API en TypeScript (wrangler)

La API anterior guardaba la base de D1 en `/data/v3/d1/miniflare-D1DatabaseObject/<hash>.sqlite`. La de Go usa `/data/soundboard.db`, con el mismo esquema y la misma tabla `d1_migrations`.

En el primer despliegue con Go, el rol `app` (`roles/app/tasks/migrate-d1.yml`):
1. detiene la API anterior;
2. copia ese archivo (con su WAL) a `/data/soundboard.db`;
3. levanta la nueva.

El archivo original no se borra, así que para volver atrás basta con desplegar la imagen anterior.

## Problemas conocidos

| Síntoma | Causa y solución |
|---|---|
| `Reauthentication failed` al hacer push | La sesión de `gcloud` expiró: corre `gcloud auth login`. El playbook lo revisa al inicio. |
| `'inventory_dir' is undefined` | Versión vieja de `group_vars`; las rutas usan `playbook_dir`. |
| TikTok: `Unexpected response from webpage request` | yt-dlp necesita `curl_cffi` para hacerse pasar por un navegador. El processor usa el build `yt-dlp_musllinux`, que lo incluye; el zipapp de Python no. Algunos videos además exigen sesión ("Log in for access"): usa cookies o sube el archivo. |
| El build falla con `apt`/`apk` (código 100) o sin espacio | El disco de Docker Desktop está lleno. El playbook borra la caché de más de 7 días y las imágenes ya publicadas; si no basta, amplía el disco en Docker Desktop → Settings → Resources. |
| La API no puede escribir en el bucket (403) | La cuenta de servicio de la VM necesita `roles/storage.objectAdmin` en el bucket de medios (`google_storage_bucket_iam_member.media_vm`). Aplica Terraform antes de desplegar. |

## Costos y límites

- **e2-micro:** entra en el free tier solo en us-central1, us-east1 o us-west1, con 30 GB de disco standard. GCP puede cobrar la **IPv4 externa** (~3 USD/mes) aunque la VM sea gratuita.
- **Memoria:** la API usa ~6 MB, el processor ~10 MB en reposo más el pico de ffmpeg, y nginx ~6 MB. La swap absorbe los picos.
- **Reinicios del processor:** al recibir SIGTERM (`docker compose restart` o un redeploy) termina el job en curso (hasta 30 s) y reporta como fallidos los que seguían en cola, con el mensaje «El processor se reinició…»; se reintentan desde el panel.
- **Artifact Registry:** el free tier cubre 0.5 GB; las imágenes suman ~400 MB comprimidas. Las versiones que pierden el tag `latest` se borran al día.
- **Cloud Storage:** el free tier incluye 5 GB-mes y 100 GB de salida al mes desde Norteamérica. La biblioteca actual pesa ~25 MB.
- **YouTube:** suele bloquear IPs de datacenter. Define `vault_ytdlp_cookies` con un `cookies.txt` exportado del navegador o usa la subida de archivos del panel. Si se vuelve un problema, el processor puede moverse a la Raspberry Pi.
