<script setup lang="ts">
import type { JobDto } from "@soundboard/shared";
import { api } from "../api";
import { errorMessage } from "../errors";

defineProps<{ jobs: JobDto[] }>();
const emit = defineEmits<{ changed: [] }>();

const STATUS_LABELS: Record<JobDto["status"], string> = {
  queued: "En cola",
  running: "Procesando",
  done: "Listo",
  failed: "Falló",
};

async function retry(job: JobDto) {
  await api.retryJob(job.id);
  emit("changed");
}

// Un job en cola o procesando sin cambios por más de esto está atascado y ya se puede borrar.
// Debe coincidir con stuckJobAge de la API (apps/api/internal/httpapi/admin.go).
const STUCK_MS = 30 * 60 * 1000;

function deletable(job: JobDto): boolean {
  if (job.status === "done" || job.status === "failed") return true;
  return Date.now() - Date.parse(job.updatedAt) > STUCK_MS;
}

async function remove(job: JobDto) {
  const note = job.status === "done" ? " El sonido ya publicado no se borra." : "";
  if (!confirm(`¿Borrar el job "${job.title}"?${note}`)) return;
  try {
    await api.deleteJob(job.id);
  } catch (error) {
    alert(errorMessage(error));
  }
  emit("changed");
}
</script>

<template>
  <section v-if="jobs.length > 0">
    <h2>Jobs recientes</h2>
    <ul class="jobs">
      <li v-for="job in jobs.slice(0, 10)" :key="job.id" class="job" :class="job.status">
        <strong>{{ job.title }}</strong>
        <span class="badge">{{ STATUS_LABELS[job.status] }}</span>
        <span class="muted ellipsis">{{ job.sourceUrl }}</span>
        <pre v-if="job.error" class="error">{{ job.error }}</pre>
        <button v-if="job.status === 'failed'" class="link" @click="retry(job)">Reintentar</button>
        <button v-if="deletable(job)" class="link danger" @click="remove(job)">Borrar</button>
      </li>
    </ul>
  </section>
</template>
