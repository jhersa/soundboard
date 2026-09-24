#!/usr/bin/env node
// Descarga un sound effect de Epidemic Sound a assets/ y lo registra en index.json.
// Uso: node scripts/download.mjs <url-del-track> <titulo> [url-de-imagen]

import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

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
  console.error("Uso: node scripts/download.mjs <url-del-track> <titulo> [url-de-imagen]");
  process.exit(1);
}

const trackId = url.match(/tracks\/([0-9a-f-]{36})/i)?.[1];
if (!trackId) {
  console.error(`No se encontró el id del track en la URL: ${url}`);
  process.exit(1);
}

async function fetchOk(target) {
  const res = await fetch(target, { headers: { "User-Agent": USER_AGENT } });
  if (!res.ok) throw new Error(`HTTP ${res.status} al descargar ${target}`);
  return res;
}

// La página embebe los datos del track en el payload de Next.js, a veces con comillas escapadas.
const html = (await (await fetchOk(url)).text()).replaceAll('\\"', '"');
const trackStart = html.indexOf(`"kosmosId":"${trackId}"`);
if (trackStart === -1) throw new Error("No se encontraron los datos del track en la página");
const trackData = html.slice(trackStart, trackStart + 5000);

const audioUrl = trackData.match(/"lqMp3Url":"([^"]+)"/)?.[1];
if (!audioUrl) throw new Error("No se encontró la URL del audio");

const audio = Buffer.from(await (await fetchOk(audioUrl)).arrayBuffer());
const assetId = createHash("sha256").update(audio).digest("hex");
const ext = path.extname(new URL(audioUrl).pathname) || ".mp3";

// Se descarga antes de escribir nada para no dejar el audio a medias si la imagen falla.
let image;
if (imageUrl) {
  const res = await fetchOk(imageUrl);
  const contentType = res.headers.get("content-type")?.split(";")[0].trim().toLowerCase() ?? "";
  if (!contentType.startsWith("image/")) {
    throw new Error(`La URL no es una imagen (content-type: ${contentType || "desconocido"})`);
  }
  const imageExt = IMAGE_EXTENSIONS[contentType] ?? (path.extname(new URL(imageUrl).pathname) || ".img");
  image = { data: Buffer.from(await res.arrayBuffer()), ext: imageExt };
}

await mkdir(ASSETS_DIR, { recursive: true });
const filePath = path.join(ASSETS_DIR, `${assetId}${ext}`);
await writeFile(filePath, audio);

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
