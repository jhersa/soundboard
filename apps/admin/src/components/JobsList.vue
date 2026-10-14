<script setup lang="ts">
import type { JobDto } from "@soundboard/shared";
import { api } from "../api";

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
      </li>
    </ul>
  </section>
</template>
