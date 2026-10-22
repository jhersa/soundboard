<script setup lang="ts">
import type { JobDto } from "@soundboard/shared";
import { computed, onBeforeUnmount, ref } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import { toast } from "../toasts";
import AppDialog from "./AppDialog.vue";

const props = defineProps<{ jobs: JobDto[]; highlight: string | null }>();
const emit = defineEmits<{ changed: [] }>();

const STATUS_LABELS: Record<JobDto["status"], string> = {
  queued: "En cola",
  running: "Procesando",
  done: "Listo",
  failed: "Falló",
};

const VISIBLE = 5;
const showAll = ref(false);
const shown = computed(() => (showAll.value ? props.jobs : props.jobs.slice(0, VISIBLE)));

// Los archivos subidos viajan como `upload:<key>`; la key no le dice nada a quien administra.
const source = (job: JobDto) => (job.sourceUrl.startsWith("upload:") ? "Archivo subido" : job.sourceUrl);

// «hace 3 minutos»: se recalcula cada 30 s.
const now = ref(Date.now());
const clock = setInterval(() => (now.value = Date.now()), 30_000);
onBeforeUnmount(() => clearInterval(clock));
const relative = new Intl.RelativeTimeFormat("es", { numeric: "auto" });
function ago(iso: string): string {
  const seconds = (Date.parse(iso) - now.value) / 1000;
  if (seconds > -60) return "hace un momento";
  if (seconds > -3600) return relative.format(Math.round(seconds / 60), "minute");
  if (seconds > -86_400) return relative.format(Math.round(seconds / 3600), "hour");
  return relative.format(Math.round(seconds / 86_400), "day");
}
const fullDate = new Intl.DateTimeFormat("es", { dateStyle: "medium", timeStyle: "short" });

async function retry(job: JobDto) {
  try {
    await api.retryJob(job.id);
    toast(`«${job.title}» volvió a la cola.`);
  } catch (error) {
    toast(errorMessage(error), { error: true });
  }
  emit("changed");
}

// Un job en cola o procesando sin cambios por más de esto está atascado y ya se puede borrar.
// Debe coincidir con stuckJobAge de la API (apps/api/internal/httpapi/admin.go).
const STUCK_MS = 30 * 60 * 1000;

function deletable(job: JobDto): boolean {
  if (job.status === "done" || job.status === "failed") return true;
  return now.value - Date.parse(job.updatedAt) > STUCK_MS;
}

const deleting = ref<JobDto | null>(null);

async function remove(job: JobDto, close: () => void) {
  try {
    await api.deleteJob(job.id);
  } catch (error) {
    toast(errorMessage(error), { error: true });
  }
  close();
  emit("changed");
}
</script>

<template>
  <section v-if="jobs.length > 0" aria-labelledby="jobs-heading">
    <h2 id="jobs-heading" tabindex="-1">Procesos recientes</h2>
    <ul class="jobs">
      <li v-for="job in shown" :key="job.id" class="job" :class="[job.status, { new: job.id === highlight }]">
        <strong>{{ job.title }}</strong>
        <span class="badge">{{ STATUS_LABELS[job.status] }}</span>
        <span v-if="job.replacesSoundId" class="muted">reemplazo</span>
        <span class="muted ellipsis" :title="source(job)">{{ source(job) }}</span>
        <time class="muted" :datetime="job.updatedAt" :title="fullDate.format(Date.parse(job.updatedAt))">
          {{ ago(job.updatedAt) }}
        </time>
        <pre v-if="job.error" class="error">{{ job.error }}</pre>
        <button v-if="job.status === 'failed'" class="link" :aria-label="`Reintentar ${job.title}`" @click="retry(job)">Reintentar</button>
        <button v-if="deletable(job)" class="link" :aria-label="`Quitar de la lista el proceso de ${job.title}`" @click="deleting = job">
          Quitar
        </button>
      </li>
    </ul>
    <button v-if="jobs.length > VISIBLE" type="button" class="link" :aria-expanded="showAll" @click="showAll = !showAll">
      {{ showAll ? "Ver menos" : `Ver todos (${jobs.length})` }}
    </button>

    <AppDialog v-if="deleting" :title="`¿Quitar el proceso de «${deleting.title}»?`" @close="deleting = null">
      <template #default="{ close }">
        <p v-if="deleting.status === 'done'">Se borra el registro del proceso; el sonido ya publicado no se toca.</p>
        <p v-else>
          Se borra el registro del proceso y los archivos que subiste para él, así que ya no se podrá reintentar.
        </p>
        <div class="dialog-actions">
          <button type="button" class="link" autofocus @click="close">Cancelar</button>
          <button type="button" @click="remove(deleting, close)">Quitar</button>
        </div>
      </template>
    </AppDialog>
  </section>
</template>
