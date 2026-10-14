#!/usr/bin/env node
// Migra la biblioteca v1 (assets/ + index.json) a los medios y la base de la API.
// Uso: node scripts/migrate-v1.mjs [--dry-run] [--gcs <bucket> --sql-out <archivo>]
// Sin --gcs trabaja en local: copia los archivos a apps/api/.data/media (con sus metadatos) y
// aplica el SQL en apps/api/.data/soundboard.db con `go run ./cmd/api sql`.
// Con --gcs sube los archivos a Cloud Storage con `gcloud storage cp` y escribe el SQL en
// --sql-out, para aplicarlo en la VM (infra/ansible/seed.yml).
// Es idempotente: vuelve a subir los archivos y hace upsert de las filas.

import { execFile } from "node:child_process";
import { copyFile, mkdir, mkdtemp, readdir, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs, promisify } from "node:util";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const API_DIR = path.join(ROOT, "apps/api");
const ASSETS_DIR = path.join(ROOT, "assets");
const MEDIA_DIR = path.join(API_DIR, ".data/media");
const IMMUTABLE_CACHE = "public, max-age=31536000, immutable";
const IMAGE_TYPES = { gif: "image/gif", webp: "image/webp", png: "image/png", jpg: "image/jpeg", svg: "image/svg+xml", avif: "image/avif" };

const run = promisify(execFile);
const { values } = parseArgs({
  options: {
    "dry-run": { type: "boolean", default: false },
    gcs: { type: "string" },
    "sql-out": { type: "string" },
  },
});
if (values.gcs && !values["sql-out"]) {
  console.error("Con --gcs hace falta --sql-out: el SQL se aplica en la VM con infra/ansible/seed.yml.");
  process.exit(1);
}

async function exec(command, args, options = {}) {
  if (values["dry-run"]) return console.log(`  [dry-run] ${command} ${args.join(" ")}`);
  await run(command, args, { maxBuffer: 16 * 1024 * 1024, ...options });
}

// Local: mismo formato que storage.FS de la API (archivo + <archivo>.meta.json).
async function upload(key, file, contentType) {
  if (values.gcs) {
    return exec("gcloud", ["storage", "cp", file, `gs://${values.gcs}/${key}`, "--content-type", contentType,
      "--cache-control", IMMUTABLE_CACHE, "--quiet"]);
  }
  if (values["dry-run"]) return console.log(`  [dry-run] copiar ${file} → ${key}`);
  const dest = path.join(MEDIA_DIR, key);
  await mkdir(path.dirname(dest), { recursive: true });
  await copyFile(file, dest);
  await writeFile(`${dest}.meta.json`, JSON.stringify({ contentType, cacheControl: IMMUTABLE_CACHE }));
}

const sql = (value) => (value === null ? "NULL" : typeof value === "number" ? String(value) : `'${String(value).replaceAll("'", "''")}'`);

const index = JSON.parse(await readFile(path.join(ROOT, "index.json"), "utf8"));
const files = await readdir(ASSETS_DIR);
const indexed = new Set(index.map((entry) => entry.assetId));
const orphans = files.filter((name) => name.endsWith(".ogg") && !indexed.has(name.slice(0, -4)));

const destination = values.gcs ? `gs://${values.gcs}` : "apps/api/.data (local)";
console.log(`Migrando ${index.length} sonidos a ${destination}...`);

// Se conserva el orden de index.json: el primero queda como el más antiguo del catálogo.
const base = Date.parse("2025-01-01T00:00:00.000Z");
const statements = [];
for (const [position, entry] of index.entries()) {
  const id = entry.assetId;
  const imageFile = files.find((name) => name.startsWith(`${id}.`) && !name.endsWith(".ogg"));
  const imageExt = imageFile ? path.extname(imageFile).slice(1) : null;
  const audioKey = `sounds/${id}.ogg`;
  const imageKey = imageExt ? `images/${id}.${imageExt}` : null;

  console.log(`- ${entry.title}`);
  await upload(audioKey, path.join(ASSETS_DIR, `${id}.ogg`), "audio/ogg");
  if (imageKey) {
    await upload(imageKey, path.join(ASSETS_DIR, imageFile), IMAGE_TYPES[imageExt] ?? "application/octet-stream");
  }

  const createdAt = new Date(base + position * 1000).toISOString();
  const imageType = imageExt ? IMAGE_TYPES[imageExt] ?? null : null;
  statements.push(
    `INSERT INTO sounds (id, title, duration_ms, audio_key, image_key, image_type, status, created_at) VALUES (` +
      [id, entry.title, entry.durationMs ?? 0, audioKey, imageKey, imageType, "published", createdAt].map(sql).join(", ") +
      `) ON CONFLICT (id) DO UPDATE SET title = excluded.title, duration_ms = excluded.duration_ms, ` +
      `audio_key = excluded.audio_key, image_key = excluded.image_key, image_type = excluded.image_type;`,
  );
}

if (values["sql-out"]) {
  await writeFile(values["sql-out"], statements.join("\n") + "\n");
  console.log(`SQL escrito en ${values["sql-out"]}`);
} else {
  const dir = await mkdtemp(path.join(os.tmpdir(), "soundboard-migrate-"));
  try {
    const sqlFile = path.join(dir, "seed.sql");
    await writeFile(sqlFile, statements.join("\n") + "\n");
    await exec("go", ["run", "./cmd/api", "sql", sqlFile], { cwd: API_DIR });
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

console.log(`Listo: ${statements.length} sonidos.`);
if (orphans.length > 0) {
  console.warn(`Aviso: ${orphans.length} audio(s) sin entrada en index.json no se migraron:`);
  for (const name of orphans) console.warn(`  assets/${name}`);
}
