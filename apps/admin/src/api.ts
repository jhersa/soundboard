import type { AdminSoundDto, CreateJobRequest, JobDto, SignInResponse, UpdateSoundRequest, UserDto } from "@soundboard/shared";

const API_URL = (import.meta.env.VITE_API_URL as string).replace(/\/+$/, "");
const TOKEN_KEY = "soundboard.token";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

// VITE_DEV_TOKEN permite saltar el login de Google en `vite dev`; nunca entra al build de producción.
const DEV_TOKEN = import.meta.env.DEV ? (import.meta.env.VITE_DEV_TOKEN as string | undefined) || null : null;

export function storedToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY) ?? DEV_TOKEN;
}

export function storeToken(token: string | null) {
  if (token) sessionStorage.setItem(TOKEN_KEY, token);
  else sessionStorage.removeItem(TOKEN_KEY);
}

const NETWORK_ERROR = "No se pudo conectar con la API. Revisa tu conexión e inténtalo de nuevo.";

// Sesión vencida o revocada: se descarta y se recarga para volver al login. Solo con un token
// guardado, para no recargar en bucle con VITE_DEV_TOKEN o en el propio login.
function onUnauthorized(status: number) {
  if (status === 401 && sessionStorage.getItem(TOKEN_KEY)) {
    storeToken(null);
    location.reload();
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const token = storedToken();
  if (token) headers.set("authorization", `Bearer ${token}`);
  if (typeof init.body === "string") headers.set("content-type", "application/json");
  const response = await fetch(`${API_URL}/v1${path}`, { ...init, headers }).catch(() => {
    throw new ApiError(0, NETWORK_ERROR);
  });
  if (!response.ok) {
    onUnauthorized(response.status);
    const body = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new ApiError(response.status, body?.error ?? `HTTP ${response.status}`);
  }
  return (response.status === 204 ? undefined : await response.json()) as T;
}

export const api = {
  signIn: (idToken: string) => request<SignInResponse>("/auth/google", { method: "POST", body: JSON.stringify({ idToken }) }),
  signOut: () => request<void>("/auth/logout", { method: "POST" }),
  me: () => request<UserDto>("/me"),
  jobs: () => request<{ items: JobDto[] }>("/admin/jobs").then((page) => page.items),
  createJob: (job: CreateJobRequest) => request<JobDto>("/admin/jobs", { method: "POST", body: JSON.stringify(job) }),
  retryJob: (id: string) => request<JobDto>(`/admin/jobs/${id}/retry`, { method: "POST" }),
  deleteJob: (id: string) => request<void>(`/admin/jobs/${id}`, { method: "DELETE" }),
  // Con XMLHttpRequest y no fetch: fetch no informa el avance de la subida (hasta 50 MB).
  upload: (file: File, onProgress: (fraction: number) => void) =>
    new Promise<string>((resolve, reject) => {
      const form = new FormData();
      form.set("file", file);
      const xhr = new XMLHttpRequest();
      xhr.open("POST", `${API_URL}/v1/admin/uploads`);
      const token = storedToken();
      if (token) xhr.setRequestHeader("authorization", `Bearer ${token}`);
      xhr.responseType = "json";
      xhr.upload.onprogress = (event) => event.lengthComputable && onProgress(event.loaded / event.total);
      xhr.onerror = () => reject(new ApiError(0, NETWORK_ERROR));
      xhr.onload = () => {
        const body = xhr.response as { url?: string; error?: string } | null;
        if (xhr.status >= 200 && xhr.status < 300 && body?.url) return resolve(body.url);
        onUnauthorized(xhr.status);
        reject(new ApiError(xhr.status, body?.error ?? `HTTP ${xhr.status}`));
      };
      xhr.send(form);
    }),
  sounds: () => request<{ items: AdminSoundDto[] }>("/admin/sounds").then((page) => page.items),
  updateSound: (id: string, update: UpdateSoundRequest) =>
    request<AdminSoundDto>(`/admin/sounds/${id}`, { method: "PATCH", body: JSON.stringify(update) }),
  deleteSound: (id: string) => request<void>(`/admin/sounds/${id}`, { method: "DELETE" }),
};
