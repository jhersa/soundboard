#!/usr/bin/env node
// Descarga un sonido (Epidemic Sound, YouTube u otro sitio soportado por yt-dlp) a assets/
// y lo registra en index.json.
// Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen]

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

const [url, title, imageUrl] = process.argv.slice(2);
if (!url || !title) {
  console.error("Uso: node scripts/download.mjs <url-del-sonido> <titulo> [url-de-imagen]");
  process.exit(1);
}

async function fetchOk(target) {
  const res = await fetch(target, { headers: { "User-Agent": USER_AGENT } });
  if (!res.ok) throw new Error(`HTTP ${res.status} al descargar ${target}`);
  return res;
}

async function downloadFromEpidemic(trackUrl) {
  const trackId = trackUrl.match(/tracks\/([0-9a-f-]{36})/i)?.[1];
  if (!trackId) throw new Error(`No se encontró el id del track en la URL: ${trackUrl}`);

  // La página embebe los datos del track en el payload de Next.js, a veces con comillas escapadas.
  const html = (await (await fetchOk(trackUrl)).text()).replaceAll('\\"', '"');
  const trackStart = html.indexOf(`"kosmosId":"${trackId}"`);
  if (trackStart === -1) throw new Error("No se encontraron los datos del track en la página");
  const trackData = html.slice(trackStart, trackStart + 5000);

  const audioUrl = trackData.match(/"lqMp3Url":"([^"]+)"/)?.[1];
  if (!audioUrl) throw new Error("No se encontró la URL del audio");

  return {
    data: Buffer.from(await (await fetchOk(audioUrl)).arrayBuffer()),
    ext: path.extname(new URL(audioUrl).pathname) || ".mp3",
  };
}

// yt-dlp extrae el audio y ffmpeg lo convierte a mp3 en una carpeta temporal.
async function downloadWithYtDlp(videoUrl) {
  const tmpDir = await mkdtemp(path.join(os.tmpdir(), "soundboard-"));
  try {
    await promisify(execFile)("yt-dlp", [
      "--extract-audio",
      "--audio-format", "mp3",
      "--no-playlist",
      "--quiet",
      "--no-warnings",
      "--output", path.join(tmpDir, "audio.%(ext)s"),
      videoUrl,
    ]);
    const file = (await readdir(tmpDir)).find((name) => name.endsWith(".mp3"));
    if (!file) throw new Error("yt-dlp no generó el archivo de audio");
    return { data: await readFile(path.join(tmpDir, file)), ext: ".mp3" };
  } catch (error) {
    if (error.code === "ENOENT") throw new Error("No se encontró yt-dlp. Instálalo con: brew install yt-dlp ffmpeg");
    throw error;
  } finally {
    await rm(tmpDir, { recursive: true, force: true });
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

const isEpidemic = new URL(url).hostname.endsWith("epidemicsound.com");
const audio = isEpidemic ? await downloadFromEpidemic(url) : await downloadWithYtDlp(url);
// Se descarga antes de escribir nada para no dejar el audio a medias si la imagen falla.
const image = imageUrl ? await downloadImage(imageUrl) : undefined;

const assetId = createHash("sha256").update(audio.data).digest("hex");

await mkdir(ASSETS_DIR, { recursive: true });
const filePath = path.join(ASSETS_DIR, `${assetId}${audio.ext}`);
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
} else {
  index.push({ title, assetId });
}
await writeFile(INDEX_FILE, JSON.stringify(index, null, 4) + "\n");

console.log(`${existing ? "Actualizado" : "Agregado"}: "${title}"`);
console.log(`  assetId: ${assetId}`);
console.log(`  archivo: ${path.relative(ROOT, filePath)}`);
if (imagePath) console.log(`  imagen:  ${path.relative(ROOT, imagePath)}`);
