import { reactive } from "vue";

// Avisos breves del panel (en vez de alert()): confirman una acción, ofrecen deshacerla o cuentan un error.
export type Toast = {
  id: number;
  text: string;
  error?: boolean;
  action?: { label: string; run: () => void };
};

export const toasts = reactive<Toast[]>([]);

const timers = new Map<number, ReturnType<typeof setTimeout>>();
let nextId = 1;

// Con acción o error duran más; mientras el puntero o el foco están encima, no se van (hold/release).
const duration = (toast: Toast) => (toast.error ? 10_000 : toast.action ? 7_000 : 4_000);

export function toast(text: string, options: Omit<Toast, "id" | "text"> = {}) {
  const item = { id: nextId++, text, ...options };
  toasts.push(item);
  release(item.id);
}

export function dismiss(id: number) {
  hold(id);
  const index = toasts.findIndex((item) => item.id === id);
  if (index >= 0) toasts.splice(index, 1);
}

export function hold(id: number) {
  clearTimeout(timers.get(id));
  timers.delete(id);
}

export function release(id: number) {
  const item = toasts.find((t) => t.id === id);
  if (!item) return;
  hold(id);
  timers.set(id, setTimeout(() => dismiss(id), duration(item)));
}
