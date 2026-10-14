// Contratos de la API. El panel admin los usa como tipos; la API y el processor (Go) los implementan
// a mano en apps/api/internal/httpapi y apps/processor, y la app Android los replica con
// kotlinx.serialization. Un cambio aquí debe reflejarse en los tres.
import { z } from "zod";

export const SOUND_ID_PATTERN = /^[0-9a-f]{64}$/;
export const SoundId = z.string().regex(SOUND_ID_PATTERN, "Debe ser un SHA-256 en hexadecimal");

// Segundos (3.5) o formato de reloj (1:23, 1:02:03.5), igual que el script original.
export const TrimTime = z.string().regex(/^\d+(:\d{1,2}){0,2}(\.\d+)?$/, "Usa segundos (3.5) o mm:ss (1:23)");

export const GoogleSignInRequest = z.object({ idToken: z.string().min(1) });
export type GoogleSignInRequest = z.infer<typeof GoogleSignInRequest>;

export interface UserDto {
  id: string;
  email: string;
  name: string | null;
  avatarUrl: string | null;
  isAdmin: boolean;
}

export interface SignInResponse {
  token: string;
  user: UserDto;
}

export interface SoundDto {
  id: string;
  title: string;
  durationMs: number;
  audioUrl: string;
  imageUrl: string | null;
  isGif: boolean;
}

export interface CatalogSoundDto extends SoundDto {
  inLibrary: boolean;
}

export interface CatalogPage {
  items: CatalogSoundDto[];
  nextCursor: string | null;
}

export interface LibraryDto {
  items: SoundDto[];
}

export const LibraryOrderRequest = z.object({ ids: z.array(SoundId).max(500) });
export type LibraryOrderRequest = z.infer<typeof LibraryOrderRequest>;

// Solo http(s) o archivos subidos al panel: evita que yt-dlp lea file:// u otros esquemas locales.
const SourceUrl = z.url({ protocol: /^(https?|upload)$/ });

export const CreateJobRequest = z.object({
  url: SourceUrl,
  title: z.string().trim().min(1).max(80),
  imageUrl: SourceUrl.optional(),
  from: TrimTime.optional(),
  to: TrimTime.optional(),
  stereo: z.boolean().default(false),
});
export type CreateJobRequest = z.infer<typeof CreateJobRequest>;

export type JobStatus = "queued" | "running" | "done" | "failed";

export interface JobDto {
  id: string;
  sourceUrl: string;
  imageUrl: string | null;
  title: string;
  from: string | null;
  to: string | null;
  stereo: boolean;
  status: JobStatus;
  error: string | null;
  soundId: string | null;
  createdAt: string;
  updatedAt: string;
}

// Lo que la API manda al processor (POST /process).
export interface ProcessRequest {
  jobId: string;
  url: string;
  title: string;
  imageUrl?: string;
  from?: string;
  to?: string;
  stereo: boolean;
  // Base de la API a la que el processor reporta (status, result) y de la que baja los uploads.
  apiBaseUrl: string;
}

// Avance de un job que reporta el processor (POST /v1/internal/jobs/:id/status).
export const JobStatusUpdate = z.discriminatedUnion("status", [
  z.object({ status: z.literal("running") }),
  z.object({ status: z.literal("failed"), error: z.string().max(2000) }),
]);
export type JobStatusUpdate = z.infer<typeof JobStatusUpdate>;

// Resultado de un job (POST /v1/internal/jobs/:id/result, multipart): campos `audio` (.ogg),
// `image` (opcional) y `meta` con este JSON. La API calcula el id (SHA-256) a partir del audio.
export const JobResultMeta = z.object({ durationMs: z.number().int().positive() });
export type JobResultMeta = z.infer<typeof JobResultMeta>;

export const AUDIO_CONTENT_TYPE = "audio/ogg";
export const IMAGE_CONTENT_TYPES: Record<string, string> = {
  "image/png": "png",
  "image/jpeg": "jpg",
  "image/gif": "gif",
  "image/webp": "webp",
  "image/svg+xml": "svg",
  "image/avif": "avif",
};

// Archivos subidos desde el panel (cuando yt-dlp no puede descargar un link). Se usan como
// `url` o `imageUrl` de un job con el esquema `upload:<key>`.
export const UPLOAD_SCHEME = "upload:";

export const UpdateSoundRequest = z
  .object({
    title: z.string().trim().min(1).max(80).optional(),
    status: z.enum(["published", "hidden"]).optional(),
  })
  .refine((value) => value.title !== undefined || value.status !== undefined, "Nada que actualizar");
export type UpdateSoundRequest = z.infer<typeof UpdateSoundRequest>;

export interface AdminSoundDto extends SoundDto {
  status: "published" | "hidden";
  createdAt: string;
}

// La firma HMAC entre la API y el processor (cabeceras x-soundboard-signature y x-soundboard-timestamp)
// vive en packages/gosig; ya no hay implementación en TypeScript.
