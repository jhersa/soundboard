# Soundboard

Catálogo de sonidos para una app Android de botones de efectos. Cada sonido tiene un audio, una imagen para su botón y una entrada en `index.json`.

Los sonidos se agregan con un script que los descarga de YouTube, TikTok, Epidemic Sound u otros sitios, los convierte a un formato ligero para Android y actualiza el índice.

## Estructura

```
.
├── assets/             # Audios (.ogg) e imágenes de los botones
├── index.json          # Lista de sonidos que lee la app
└── scripts/
    └── download.mjs    # Script para agregar sonidos
```

### `index.json`

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

## Requisitos

- [Node.js](https://nodejs.org/) 18.3 o superior
- [yt-dlp](https://github.com/yt-dlp/yt-dlp), para YouTube, TikTok y otros sitios
- [ffmpeg](https://ffmpeg.org/), para convertir audio
- [cwebp](https://developers.google.com/speed/webp/docs/cwebp), para convertir imágenes

En macOS:

```bash
brew install node yt-dlp ffmpeg webp
```

## Agregar un sonido

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

## Qué hace el script

**Audio:** lo descarga en su formato original y lo convierte una sola vez con ffmpeg:

- Ogg Vorbis, calidad 4, 44.1 kHz. Android lo reproduce desde las primeras versiones, incluido `SoundPool`.
- Mono por defecto, que reduce el tamaño a la mitad. Con `--stereo` se conservan los dos canales.
- Volumen normalizado a −16 LUFS (EBU R128), para que todos los sonidos suenen parecido aunque vengan de fuentes distintas. Si se usa `--from` o `--to`, el volumen se calcula solo con el fragmento.
- Sin metadatos.

**Imagen:**

- **PNG y JPG** se reducen a un máximo de 384 px en el lado más largo, sin cambiar la proporción ni agrandar las imágenes pequeñas, y se guardan en WebP con calidad 80.
- **GIF, SVG y los demás formatos** se guardan tal cual.

Si algo falla, por ejemplo una imagen inválida o un corte fuera de rango, el script se detiene sin guardar nada.

## Uso en la app Android

- **Reproducción:** usa `SoundPool` para sonidos cortos (`durationMs` de unos 5000 o menos), porque suenan al instante. Para los largos usa `MediaPlayer` o ExoPlayer, que leen el archivo conforme lo reproducen en vez de cargarlo completo en memoria.
- **Imágenes:** la extensión varía entre sonidos. Busca el archivo que empiece con `<assetId>.` y no termine en `.ogg`.
- **Ubicación:** copia los archivos a la carpeta `assets/` de la app, no a `res/raw`. Los nombres de recursos de `res/raw` no pueden empezar con número, y muchos `assetId` sí empiezan con uno.

## Derechos de los sonidos

Los audios e imágenes son propiedad de sus autores. Antes de publicar la app, revisa que tengas permiso para usarlos. En el caso de Epidemic Sound, el script descarga la vista previa pública y sus términos exigen una suscripción activa para usar los sonidos.
