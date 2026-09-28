#!/usr/bin/env node
// Descarga un sonido (Epidemic Sound, YouTube u otro sitio soportado por yt-dlp), lo convierte a
// Ogg Vorbis con volumen normalizado, lo guarda en assets/ y lo registra en index.json.
// Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen] [--stereo]

import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { mkdir, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

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

const run = promisify(execFile);

const args = process.argv.slice(2);
const stereo = args.includes("--stereo");
const [url, title, imageUrl] = args.filter((arg) => !arg.startsWith("--"));
if (!url || !title) {
  console.error("Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen] [--stereo]");
  process.exit(1);
}

async function fetchOk(target) {
  const res = await fetch(target, { headers: { "User-Agent": USER_AGENT } });
  if (!res.ok) throw new Error(`HTTP ${res.status} al descargar ${target}`);
  return res;
}

async function runTool(command, commandArgs) {
  try {
    return await run(command, commandArgs);
  } catch (error) {
    if (error.code === "ENOENT") throw new Error(`No se encontró ${command}. Instálalo con: brew install yt-dlp ffmpeg`);
    throw error;
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

async function convertToOgg(sourcePath, dir) {
  const outputPath = path.join(dir, `output${AUDIO_EXT}`);
  await runTool("ffmpeg", [
    "-hide_banner",
    "-loglevel", "error",
    "-i", sourcePath,
    "-vn",
    "-map_metadata", "-1",
    "-af", AUDIO_FILTER,
    "-ac", stereo ? "2" : "1",
    "-ar", AUDIO_SAMPLE_RATE,
    "-c:a", "libvorbis",
    "-q:a", AUDIO_QUALITY,
    outputPath,
  ]);
  const { stdout } = await runTool("ffprobe", [
    "-v", "error",
    "-show_entries", "format=duration",
    "-of", "csv=p=0",
    outputPath,
  ]);
  return { data: await readFile(outputPath), durationMs: Math.round(Number(stdout.trim()) * 1000) };
}

async function downloadAudio(sourceUrl) {
  const dir = await mkdtemp(path.join(os.tmpdir(), "soundboard-"));
  try {
    const isEpidemic = new URL(sourceUrl).hostname.endsWith("epidemicsound.com");
    const sourcePath = isEpidemic ? await downloadFromEpidemic(sourceUrl, dir) : await downloadWithYtDlp(sourceUrl, dir);
    return await convertToOgg(sourcePath, dir);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

async function downloadImage(target) {
  const res = await fetchOk(target);
  const contentType = res.headers.get("content-type")?.split(";")[0].trim().toLowerCase() ?? "";
  if (!contentType.startsWith("image/")) {
    throw new Error(`La URL no es una imagen (content-type: ${contentType || "desconocido"})`);
  }
  return {
    data: Buffer.from(await res.arrayBuffer()),
    ext: IMAGE_EXTENSIONS[contentType] ?? (path.extname(new URL(target).pathname) || ".img"),
  };
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
