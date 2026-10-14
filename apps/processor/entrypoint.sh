#!/bin/sh
# YouTube cambia seguido; se intenta actualizar yt-dlp en cada arranque sin bloquear si falla.
yt-dlp -U >/dev/null 2>&1 || echo "No se pudo actualizar yt-dlp; se usa la versión instalada"
exec /usr/local/bin/processor serve
