<script setup lang="ts">
import type { AdminSoundDto, JobDto } from "@soundboard/shared";
import { computed, nextTick, onMounted, ref, watch, watchEffect } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import { toast } from "../toasts";
import Icon from "./Icon.vue";
import JobsList from "./JobsList.vue";
import NewSoundForm from "./NewSoundForm.vue";
import SoundsGrid from "./SoundsGrid.vue";

const POLL_MS = 2000;

const jobs = ref<JobDto[]>([]);
const sounds = ref<AdminSoundDto[]>([]);
const error = ref<string | null>(null);
const loaded = ref(false);
// Último job creado desde el panel: se resalta en la lista hasta el siguiente.
const highlight = ref<string | null>(null);

// El formulario va plegado: lo más frecuente es operar el catálogo. Se abre solo si aún no hay sonidos.
const creating = ref(false);
const toggleEl = ref<HTMLButtonElement | null>(null);
watch(loaded, (now) => {
  if (now && sounds.value.length === 0) creating.value = true;
});

async function closeForm() {
  creating.value = false;
  await nextTick();
  toggleEl.value?.focus();
}

function onCreated(job: JobDto) {
  highlight.value = job.id;
  void refresh();
}

async function refresh() {
  try {
    [jobs.value, sounds.value] = await Promise.all([api.jobs(), api.sounds()]);
    loaded.value = true;
    error.value = null;
  } catch (e) {
    error.value = errorMessage(e);
  }
}

onMounted(refresh);

// Mientras haya jobs en curso solo se consulta su estado; los sonidos se recargan al terminar (abajo).
async function refreshJobs() {
  try {
    jobs.value = await api.jobs();
    error.value = null;
  } catch (e) {
    error.value = errorMessage(e);
  }
}
const active = computed(() => jobs.value.some((job) => job.status === "queued" || job.status === "running"));
watchEffect((onCleanup) => {
  if (!active.value) return;
  const timer = setInterval(refreshJobs, POLL_MS);
  onCleanup(() => clearInterval(timer));
});
// Avisa cuando un job que estaba en curso termina o falla (la lista no se anuncia en cada consulta).
watch(jobs, (now, before) => {
  const previous = new Map(before.map((job) => [job.id, job.status]));
  for (const job of now) {
    const was = previous.get(job.id);
    if (was !== "queued" && was !== "running") continue;
    if (job.status === "done") {
      toast(job.replacesSoundId ? `«${job.title}» ya tiene el audio nuevo.` : `«${job.title}» está listo.`);
    } else if (job.status === "failed") {
      toast(`«${job.title}» falló. El detalle está en Procesos recientes.`, { error: true });
    }
  }
});
// Al terminar un job se recargan los sonidos una última vez.
watch(active, (now, before) => {
  if (before && !now) void refresh();
});
</script>

<template>
  <p v-if="error" class="card error" role="alert">{{ error }}</p>
  <section v-if="creating" aria-labelledby="new-heading">
    <h2 id="new-heading">Nuevo sonido</h2>
    <NewSoundForm @created="onCreated" @cancel="closeForm" />
  </section>
  <div v-else class="toolbar">
    <button ref="toggleEl" type="button" @click="creating = true">
      <Icon name="plus" /> Nuevo sonido
    </button>
  </div>
  <JobsList :jobs="jobs" :highlight="highlight" @changed="refresh" />
  <SoundsGrid :sounds="sounds" :jobs="jobs" :loaded="loaded" @changed="refresh" @created="onCreated" />
</template>
