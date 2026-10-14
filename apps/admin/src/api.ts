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

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const token = storedToken();
  if (token) headers.set("authorization", `Bearer ${token}`);
  if (typeof init.body === "string") headers.set("content-type", "application/json");
  const response = await fetch(`${API_URL}/v1${path}`, { ...init, headers });
  if (!response.ok) {
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
  upload: (file: File) => {
    const form = new FormData();
    form.set("file", file);
    return request<{ url: string }>("/admin/uploads", { method: "POST", body: form }).then((result) => result.url);
  },
  sounds: () => request<{ items: AdminSoundDto[] }>("/admin/sounds").then((page) => page.items),
  updateSound: (id: string, update: UpdateSoundRequest) =>
    request<AdminSoundDto>(`/admin/sounds/${id}`, { method: "PATCH", body: JSON.stringify(update) }),
  deleteSound: (id: string) => request<void>(`/admin/sounds/${id}`, { method: "DELETE" }),
};
