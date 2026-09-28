#!/usr/bin/env node
// Descarga un sonido (Epidemic Sound, YouTube u otro sitio soportado por yt-dlp), lo convierte a
// Ogg Vorbis con volumen normalizado, lo guarda en assets/ y lo registra en index.json.
// Las imágenes PNG/JPG se reducen y se guardan en WebP; GIF y demás formatos se guardan tal cual.
// Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen] [--stereo] [--from <tiempo>] [--to <tiempo>]
// Los tiempos aceptan segundos (3.5) o formato de reloj (1:23, 1:02:03.5).

import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { mkdir, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs, promisify } from "node:util";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const ASSETS_DIR = path.join(ROOT, "assets");
const INDEX_FILE = path.join(ROOT, "index.json");
const USER_AGENT =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36";

const IMAGE_EXTENSIONS = {
  "image/png": ".png",
  "image/jpeg": ".jpg",
  "image/gif": ".gif",
  "image/webp": ".webp",
  "image/svg+xml": ".svg",
  "image/avif": ".avif",
};

// Ogg Vorbis se reproduce en todas las versiones de Android (minSdk 24), incluido SoundPool.
// loudnorm iguala el volumen percibido entre audios de fuentes distintas (EBU R128, -16 LUFS).
const AUDIO_EXT = ".ogg";
const AUDIO_FILTER = "loudnorm=I=-16:TP=-1.5:LRA=11";
const AUDIO_SAMPLE_RATE = "44100";
const AUDIO_QUALITY = "4";

// Suficiente para un botón de ~120 dp en pantallas xxxhdpi. Solo se reduce, nunca se amplía.
const IMAGE_MAX_SIZE = 384;
const IMAGE_QUALITY = "80";
const RESIZABLE_IMAGE_TYPES = new Set(["image/png", "image/jpeg"]);

const INSTALL_HINTS = {
  "yt-dlp": "brew install yt-dlp",
  ffmpeg: "brew install ffmpeg",
  ffprobe: "brew install ffmpeg",
  cwebp: "brew install webp",
};

const run = promisify(execFile);

const USAGE =
  "Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen] [--stereo] [--from <tiempo>] [--to <tiempo>]";

function fail(message) {
  console.error(message);
  process.exit(1);
}

// Convierte "3.5", "1:23" o "1:02:03.5" a segundos.
function parseTime(value, flag) {
  if (value === undefined) return undefined;
  if (!/^\d+(:\d{1,2}){0,2}(\.\d+)?$/.test(value)) fail(`Tiempo inválido en ${flag}: "${value}". Usa segundos (3.5) o mm:ss (1:23).`);
  return value.split(":").reduce((total, part) => total * 60 + Number(part), 0);
}

let parsed;
try {
  parsed = parseArgs({
    allowPositionals: true,
    options: {
      stereo: { type: "boolean", default: false },
      from: { type: "string" },
      to: { type: "string" },
    },
  });
} catch (error) {
  fail(`${error.message}\n${USAGE}`);
}
const [url, title, imageUrl] = parsed.positionals;
if (!url || !title) fail(USAGE);
const { stereo } = parsed.values;
const trimStart = parseTime(parsed.values.from, "--from");
const trimEnd = parseTime(parsed.values.to, "--to");
if (trimStart !== undefined && trimEnd !== undefined && trimEnd <= trimStart) fail("--to debe ser mayor que --from");

async function fetchOk(target) {
  const res = await fetch(target, { headers: { "User-Agent": USER_AGENT } });
  if (!res.ok) throw new Error(`HTTP ${res.status} al descargar ${target}`);
  return res;
}

async function runTool(command, commandArgs) {
  try {
    return await run(command, commandArgs);
  } catch (error) {
    if (error.code === "ENOENT") throw new Error(`No se encontró ${command}. Instálalo con: ${INSTALL_HINTS[command]}`);
    throw error;
  }
}

async function withTempDir(fn) {
  const dir = await mkdtemp(path.join(os.tmpdir(), "soundboard-"));
  try {
    return await fn(dir);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

async function downloadFromEpidemic(trackUrl, dir) {
  const trackId = trackUrl.match(/tracks\/([0-9a-f-]{36})/i)?.[1];
  if (!trackId) throw new Error(`No se encontró el id del track en la URL: ${trackUrl}`);

  // La página embebe los datos del track en el payload de Next.js, a veces con comillas escapadas.
  const html = (await (await fetchOk(trackUrl)).text()).replaceAll('\\"', '"');
  const trackStart = html.indexOf(`"kosmosId":"${trackId}"`);
  if (trackStart === -1) throw new Error("No se encontraron los datos del track en la página");
  const trackData = html.slice(trackStart, trackStart + 5000);

  const audioUrl = trackData.match(/"lqMp3Url":"([^"]+)"/)?.[1];
  if (!audioUrl) throw new Error("No se encontró la URL del audio");

  const sourcePath = path.join(dir, `source${path.extname(new URL(audioUrl).pathname) || ".mp3"}`);
  await writeFile(sourcePath, Buffer.from(await (await fetchOk(audioUrl)).arrayBuffer()));
  return sourcePath;
}

// Se baja el mejor audio original sin convertir; la única conversión la hace ffmpeg después.
async function downloadWithYtDlp(videoUrl, dir) {
  await runTool("yt-dlp", [
    "--format", "bestaudio/best",
    "--no-playlist",
    "--quiet",
    "--no-warnings",
    "--output", path.join(dir, "source.%(ext)s"),
    videoUrl,
  ]);
  const file = (await readdir(dir)).find((name) => name.startsWith("source."));
  if (!file) throw new Error("yt-dlp no generó el archivo de audio");
  return path.join(dir, file);
}

// atrim va antes de loudnorm para que el volumen se mida solo sobre el fragmento recortado.
function audioFilter() {
  if (trimStart === undefined && trimEnd === undefined) return AUDIO_FILTER;
  const bounds = [trimStart !== undefined && `start=${trimStart}`, trimEnd !== undefined && `end=${trimEnd}`].filter(Boolean);
  return `atrim=${bounds.join(":")},asetpts=PTS-STARTPTS,${AUDIO_FILTER}`;
}

async function probeDuration(filePath) {
  const { stdout } = await runTool("ffprobe", [
    "-v", "error",
    "-show_entries", "format=duration",
    "-of", "csv=p=0",
    filePath,
  ]);
  return Number(stdout.trim());
}

async function convertToOgg(sourcePath, dir) {
  const sourceDuration = await probeDuration(sourcePath);
  if (trimStart !== undefined && trimStart >= sourceDuration) {
    throw new Error(`--from (${trimStart} s) está después del final del audio (${sourceDuration.toFixed(2)} s)`);
  }

  const outputPath = path.join(dir, `output${AUDIO_EXT}`);
  await runTool("ffmpeg", [
    "-hide_banner",
    "-loglevel", "error",
    "-i", sourcePath,
    "-vn",
    "-map_metadata", "-1",
    "-af", audioFilter(),
    "-ac", stereo ? "2" : "1",
    "-ar", AUDIO_SAMPLE_RATE,
    "-c:a", "libvorbis",
    "-q:a", AUDIO_QUALITY,
    outputPath,
  ]);
  return { data: await readFile(outputPath), durationMs: Math.round((await probeDuration(outputPath)) * 1000) };
}

async function downloadAudio(sourceUrl) {
  return withTempDir(async (dir) => {
    const isEpidemic = new URL(sourceUrl).hostname.endsWith("epidemicsound.com");
    const sourcePath = isEpidemic ? await downloadFromEpidemic(sourceUrl, dir) : await downloadWithYtDlp(sourceUrl, dir);
    return convertToOgg(sourcePath, dir);
  });
}

async function convertToWebp(data, sourceExt) {
  return withTempDir(async (dir) => {
    const sourcePath = path.join(dir, `source${sourceExt}`);
    const outputPath = path.join(dir, "output.webp");
    await writeFile(sourcePath, data);

    const { stdout } = await runTool("ffprobe", [
      "-v", "error",
      "-select_streams", "v:0",
      "-show_entries", "stream=width,height",
      "-of", "csv=s=x:p=0",
      sourcePath,
    ]);
    const [width, height] = stdout.trim().split("x").map(Number);
    // cwebp mantiene la proporción cuando una de las dos dimensiones es 0.
    const resize =
      Math.max(width, height) <= IMAGE_MAX_SIZE ? [] : width >= height ? ["-resize", String(IMAGE_MAX_SIZE), "0"] : ["-resize", "0", String(IMAGE_MAX_SIZE)];

    await runTool("cwebp", ["-quiet", "-q", IMAGE_QUALITY, "-metadata", "none", ...resize, sourcePath, "-o", outputPath]);
    return readFile(outputPath);
  });
}

async function downloadImage(target) {
  const res = await fetchOk(target);
  const contentType = res.headers.get("content-type")?.split(";")[0].trim().toLowerCase() ?? "";
  if (!contentType.startsWith("image/")) {
    throw new Error(`La URL no es una imagen (content-type: ${contentType || "desconocido"})`);
  }
  const data = Buffer.from(await res.arrayBuffer());
  const ext = IMAGE_EXTENSIONS[contentType] ?? (path.extname(new URL(target).pathname) || ".img");
  if (!RESIZABLE_IMAGE_TYPES.has(contentType)) return { data, ext };
  return { data: await convertToWebp(data, ext), ext: ".webp" };
}

const audio = await downloadAudio(url);
// Se descarga antes de escribir nada para no dejar el audio a medias si la imagen falla.
const image = imageUrl ? await downloadImage(imageUrl) : undefined;

const assetId = createHash("sha256").update(audio.data).digest("hex");

await mkdir(ASSETS_DIR, { recursive: true });
const filePath = path.join(ASSETS_DIR, `${assetId}${AUDIO_EXT}`);
await writeFile(filePath, audio.data);

let imagePath;
if (image) {
  imagePath = path.join(ASSETS_DIR, `${assetId}${image.ext}`);
  await writeFile(imagePath, image.data);
}

const index = existsSync(INDEX_FILE) ? JSON.parse(await readFile(INDEX_FILE, "utf8")) : [];
const existing = index.find((entry) => entry.assetId === assetId);
if (existing) {
  existing.title = title;
  existing.durationMs = audio.durationMs;
} else {
  index.push({ title, assetId, durationMs: audio.durationMs });
}
await writeFile(INDEX_FILE, JSON.stringify(index, null, 4) + "\n");

console.log(`${existing ? "Actualizado" : "Agregado"}: "${title}"`);
console.log(`  assetId:  ${assetId}`);
console.log(`  duración: ${(audio.durationMs / 1000).toFixed(2)} s`);
console.log(`  archivo:  ${path.relative(ROOT, filePath)} (${Math.round(audio.data.length / 1024)} KB)`);
if (imagePath) console.log(`  imagen:   ${path.relative(ROOT, imagePath)}`);
