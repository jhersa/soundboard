<script setup lang="ts">
import type { AdminSoundDto, JobDto } from "@soundboard/shared";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import { toast } from "../toasts";
import AppDialog from "./AppDialog.vue";
import Icon from "./Icon.vue";
import NewSoundForm from "./NewSoundForm.vue";
import SoundCard from "./SoundCard.vue";

const props = defineProps<{ sounds: AdminSoundDto[]; jobs: JobDto[]; loaded: boolean }>();
const emit = defineEmits<{ changed: []; created: [job: JobDto] }>();

// Sin mayúsculas ni acentos: "cancion" encuentra "Canción".
const fold = (text: string) => text.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase();
const seconds = new Intl.NumberFormat("es", { minimumFractionDigits: 1, maximumFractionDigits: 1 });

const filter = ref("");
type StatusFilter = "all" | AdminSoundDto["status"];
const status = ref<StatusFilter>("all");
const searchEl = ref<HTMLInputElement | null>(null);
const headingEl = ref<HTMLElement | null>(null);
const visible = computed(() =>
  props.sounds.filter(
    (sound) =>
      (status.value === "all" || sound.status === status.value) && fold(sound.title).includes(fold(filter.value.trim())),
  ),
);
const hiddenCount = computed(() => props.sounds.filter((s) => s.status === "hidden").length);
const statusOptions = computed(() => [
  { value: "all" as const, label: "Todos", count: props.sounds.length },
  { value: "published" as const, label: "Publicados", count: props.sounds.length - hiddenCount.value },
  { value: "hidden" as const, label: "Ocultos", count: hiddenCount.value },
]);
const countLabel = computed(() =>
  visible.value.length === props.sounds.length ? `${props.sounds.length}` : `${visible.value.length} de ${props.sounds.length}`,
);

// Sonidos con un reemplazo de audio en curso: la tarjeta lo muestra hasta que el job termina.
const replacingIds = computed(
  () =>
    new Set(
      props.jobs
        .filter((job) => job.replacesSoundId && (job.status === "queued" || job.status === "running"))
        .map((job) => job.replacesSoundId),
    ),
);

function clearSearch() {
  filter.value = "";
  searchEl.value?.focus();
}

// «/» enfoca el buscador desde cualquier parte, salvo mientras se escribe o hay un diálogo abierto.
function onKeydown(event: KeyboardEvent) {
  if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
  const target = event.target as HTMLElement;
  if (target.closest("input, textarea, select, [contenteditable]") || document.querySelector("dialog[open]")) return;
  event.preventDefault();
  searchEl.value?.focus();
}

// Un solo reproductor para todo el panel: escuchar un sonido detiene el anterior.
const audio = new Audio();
const playingId = ref<string | null>(null);
const progress = ref(0);
audio.addEventListener("timeupdate", () => {
  progress.value = audio.duration ? audio.currentTime / audio.duration : 0;
});
audio.addEventListener("ended", stop);

function stop() {
  audio.pause();
  playingId.value = null;
  progress.value = 0;
}

function play(sound: AdminSoundDto) {
  if (playingId.value === sound.id) return stop();
  audio.src = sound.audioUrl;
  playingId.value = sound.id;
  progress.value = 0;
  audio.play().catch((error: unknown) => {
    // AbortError: se cambió de sonido o se detuvo antes de empezar; no es un fallo.
    if (error instanceof DOMException && error.name === "AbortError") return;
    stop();
    toast(`No se pudo reproducir «${sound.title}». ${errorMessage(error)}`, { error: true });
  });
}

onMounted(() => document.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => {
  document.removeEventListener("keydown", onKeydown);
  stop();
});

// Ejecuta una acción sobre un sonido; si la API falla, lo dice en vez de fallar en silencio.
async function run(action: () => Promise<unknown>): Promise<boolean> {
  try {
    await action();
    return true;
  } catch (error) {
    toast(errorMessage(error), { error: true });
    return false;
  } finally {
    emit("changed");
  }
}

async function rename(sound: AdminSoundDto, title: string) {
  if (await run(() => api.updateSound(sound.id, { title }))) toast(`Ahora se llama «${title}».`);
}

// Ocultar llega a todas las bibliotecas, así que siempre se confirma con un aviso que permite deshacerlo.
async function setStatus(sound: AdminSoundDto, next: AdminSoundDto["status"], undoable = true) {
  if (!(await run(() => api.updateSound(sound.id, { status: next })))) return;
  const text =
    next === "hidden"
      ? `«${sound.title}» está oculto: ya no aparece en el catálogo ni en las bibliotecas.`
      : `«${sound.title}» está publicado de nuevo.`;
  const previous = sound.status;
  toast(text, undoable ? { action: { label: "Deshacer", run: () => void setStatus(sound, previous, false) } } : {});
}

const toggle = (sound: AdminSoundDto) => setStatus(sound, sound.status === "published" ? "hidden" : "published");

function onReplaceCreated(job: JobDto) {
  emit("created", job);
  toast(`El reemplazo de «${job.title}» está en proceso.`, {
    action: {
      label: "Ver proceso",
      run: () => {
        const heading = document.getElementById("jobs-heading");
        heading?.scrollIntoView({ block: "start" });
        heading?.focus({ preventScroll: true });
      },
    },
  });
}

const deleting = ref<AdminSoundDto | null>(null);
const replacing = ref<AdminSoundDto | null>(null);
const busy = ref(false);

async function confirmDelete(close: () => void) {
  const sound = deleting.value;
  if (!sound) return;
  busy.value = true;
  if (playingId.value === sound.id) stop();
  const ok = await run(() => api.deleteSound(sound.id));
  busy.value = false;
  close();
  if (!ok) return;
  toast(`«${sound.title}» se borró.`);
  // La tarjeta desaparece: el foco va al encabezado de la sección en vez de perderse.
  headingEl.value?.focus();
}

async function hideInstead(close: () => void) {
  const sound = deleting.value;
  if (!sound) return;
  busy.value = true;
  await setStatus(sound, "hidden");
  busy.value = false;
  close();
}
</script>

<template>
  <section aria-labelledby="sounds-heading">
    <div class="section-head">
      <h2 id="sounds-heading" ref="headingEl" tabindex="-1">
        Sonidos <span class="count">{{ countLabel }}</span>
      </h2>
      <div v-if="sounds.length > 0" class="search">
        <Icon name="search" />
        <input
          ref="searchEl"
          v-model="filter"
          type="search"
          placeholder="Buscar sonidos"
          aria-label="Buscar sonidos"
          aria-keyshortcuts="/"
          @keydown.escape="filter = ''"
        />
        <kbd v-if="!filter" class="search-hint" aria-hidden="true">/</kbd>
        <button v-else type="button" class="icon-button" aria-label="Borrar búsqueda" @click="clearSearch">
          <Icon name="close" />
        </button>
      </div>
    </div>
    <div v-if="hiddenCount > 0 || status !== 'all'" class="chips" role="group" aria-label="Filtrar por estado">
      <button
        v-for="option in statusOptions"
        :key="option.value"
        type="button"
        class="chip-toggle"
        :aria-pressed="status === option.value"
        @click="status = option.value"
      >
        {{ option.label }} <span class="count">{{ option.count }}</span>
      </button>
    </div>

    <p v-if="!loaded" class="muted" role="status">Cargando sonidos…</p>
    <p v-else-if="sounds.length === 0" class="muted">Aún no hay sonidos. Crea el primero con «Nuevo sonido», arriba.</p>
    <p v-else-if="visible.length === 0" class="muted" role="status">
      <template v-if="filter.trim()">Ningún sonido coincide con «{{ filter.trim() }}».</template>
      <template v-else>No hay sonidos en este estado.</template>
    </p>

    <div class="grid">
      <SoundCard
        v-for="sound in visible"
        :key="sound.id"
        :sound="sound"
        :playing="playingId === sound.id"
        :progress="playingId === sound.id ? progress : 0"
        :replacing="replacingIds.has(sound.id)"
        @play="play(sound)"
        @rename="rename(sound, $event)"
        @toggle="toggle(sound)"
        @replace="replacing = sound"
        @delete="deleting = sound"
      />
    </div>

    <AppDialog v-if="deleting" :title="`¿Borrar «${deleting.title}»?`" @close="deleting = null">
      <template #default="{ close }">
        <div class="dialog-sound">
          <img v-if="deleting.imageUrl" :src="deleting.imageUrl" alt="" />
          <div v-else class="placeholder"><Icon name="note" /></div>
          <p>
            Se quita del catálogo <strong>y de la biblioteca de cada persona que lo tenga</strong>, junto con su audio
            e imagen. No se puede deshacer.
          </p>
        </div>
        <p v-if="deleting.status === 'published'" class="muted">
          Si quieres sacarlo sin perderlo, ocúltalo: deja de aparecer en el catálogo y en las bibliotecas, y al volver a
          publicarlo regresa a las bibliotecas que lo tenían.
        </p>
        <div class="dialog-actions">
          <button type="button" class="link" :disabled="busy" autofocus @click="close">Cancelar</button>
          <button v-if="deleting.status === 'published'" type="button" class="secondary" :disabled="busy" @click="hideInstead(close)">
            Ocultar en su lugar
          </button>
          <button type="button" class="danger-fill" :disabled="busy" @click="confirmDelete(close)">Borrar para siempre</button>
        </div>
      </template>
    </AppDialog>

    <AppDialog v-if="replacing" :title="`Reemplazar audio de «${replacing.title}»`" @close="(replacing = null), stop()">
      <template #default="{ close }">
        <div class="dialog-sound">
          <button
            type="button"
            class="play inline"
            :class="{ playing: playingId === replacing.id }"
            :aria-label="`${playingId === replacing.id ? 'Detener' : 'Escuchar'} el audio actual`"
            @click="play(replacing)"
          >
            <Icon :name="playingId === replacing.id ? 'stop' : 'play'" />
          </button>
          <p>
            Audio actual: {{ seconds.format(replacing.durationMs / 1000) }} s. El nuevo toma su lugar en el catálogo y en
            las bibliotecas; si no subes otra imagen, se conserva la actual. El audio anterior se borra al terminar.
          </p>
        </div>
        <NewSoundForm :replacing="replacing" @created="close(), onReplaceCreated($event)" @cancel="close" />
      </template>
    </AppDialog>
  </section>
</template>
