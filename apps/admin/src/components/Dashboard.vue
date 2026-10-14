<script setup lang="ts">
import type { AdminSoundDto, JobDto } from "@soundboard/shared";
import { computed, onMounted, ref, watch, watchEffect } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import JobsList from "./JobsList.vue";
import NewSoundForm from "./NewSoundForm.vue";
import SoundsGrid from "./SoundsGrid.vue";

const POLL_MS = 2000;

const jobs = ref<JobDto[]>([]);
const sounds = ref<AdminSoundDto[]>([]);
const error = ref<string | null>(null);

async function refresh() {
  try {
    [jobs.value, sounds.value] = await Promise.all([api.jobs(), api.sounds()]);
    error.value = null;
  } catch (e) {
    error.value = errorMessage(e);
  }
}

onMounted(refresh);

// Mientras haya jobs en curso se consulta su estado periódicamente.
const active = computed(() => jobs.value.some((job) => job.status === "queued" || job.status === "running"));
watchEffect((onCleanup) => {
  if (!active.value) return;
  const timer = setInterval(refresh, POLL_MS);
  onCleanup(() => clearInterval(timer));
});
// Al terminar un job se recargan los sonidos una última vez.
watch(active, (now, before) => {
  if (before && !now) void refresh();
});
</script>

<template>
  <p v-if="error" class="error">{{ error }}</p>
  <NewSoundForm @created="refresh" />
  <JobsList :jobs="jobs" @changed="refresh" />
  <SoundsGrid :sounds="sounds" @changed="refresh" />
</template>
