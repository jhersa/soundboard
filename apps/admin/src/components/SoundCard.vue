<script setup lang="ts">
import type { AdminSoundDto } from "@soundboard/shared";
import { computed, nextTick, ref } from "vue";
import ActionsMenu, { type MenuAction } from "./ActionsMenu.vue";
import Icon from "./Icon.vue";

const props = defineProps<{ sound: AdminSoundDto; playing: boolean; progress: number; replacing: boolean }>();
const emit = defineEmits<{ play: []; rename: [title: string]; toggle: []; replace: []; delete: [] }>();

const seconds = new Intl.NumberFormat("es", { minimumFractionDigits: 1, maximumFractionDigits: 1 });
const hidden = computed(() => props.sound.status === "hidden");
const titleId = computed(() => `sound-${props.sound.id}`);

// Carátulas quietas: el primer cuadro se dibuja en un canvas encima de la imagen, y la animación (GIF o
// WebP animado) solo se ve con el puntero o el foco en la tarjeta, o mientras suena. Nunca con
// movimiento reducido.
const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const poster = ref<HTMLCanvasElement | null>(null);
const posterReady = ref(false);
const hovered = ref(false);
const focused = ref(false);
const animate = computed(() => !posterReady.value || (!reduceMotion && (hovered.value || focused.value || props.playing)));

function drawPoster(event: Event) {
  const image = event.target as HTMLImageElement;
  const canvas = poster.value;
  if (!canvas) return;
  canvas.width = image.naturalWidth;
  canvas.height = image.naturalHeight;
  canvas.getContext("2d")?.drawImage(image, 0, 0);
  posterReady.value = true;
}

function onFocusOut(event: FocusEvent) {
  focused.value = (event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null);
}

const titleEl = ref<HTMLElement | null>(null);
const input = ref<HTMLInputElement | null>(null);
const renaming = ref(false);
const draft = ref("");

async function startRename() {
  draft.value = props.sound.title;
  renaming.value = true;
  await nextTick();
  input.value?.select();
}

// Enter o salir del campo guarda; Escape descarta. El foco vuelve al título.
async function finishRename(save: boolean) {
  if (!renaming.value) return;
  renaming.value = false;
  const title = draft.value.trim();
  if (save && title && title !== props.sound.title) emit("rename", title);
  await nextTick();
  titleEl.value?.focus();
}

const actions: MenuAction[] = [
  { label: "Renombrar", run: startRename },
  { label: "Reemplazar audio", run: () => emit("replace") },
  { label: "Borrar…", danger: true, run: () => emit("delete") },
];
</script>

<template>
  <article
    class="card sound"
    :class="sound.status"
    :aria-labelledby="titleId"
    @pointerenter="hovered = true"
    @pointerleave="hovered = false"
    @focusin="focused = true"
    @focusout="onFocusOut"
  >
    <div class="cover">
      <template v-if="sound.imageUrl">
        <img :src="sound.imageUrl" alt="" loading="lazy" decoding="async" @load="drawPoster" />
        <canvas v-show="!animate" ref="poster" class="poster" aria-hidden="true" />
      </template>
      <div v-else class="placeholder"><Icon name="note" /></div>
      <button
        type="button"
        class="play"
        :class="{ playing }"
        :aria-label="`${playing ? 'Detener' : 'Escuchar'} ${sound.title}`"
        @click="emit('play')"
      >
        <Icon :name="playing ? 'stop' : 'play'" />
      </button>
      <div v-if="playing" class="progress" aria-hidden="true">
        <span :style="{ transform: `scaleX(${progress})` }" />
      </div>
    </div>
    <div class="sound-head">
      <input
        v-if="renaming"
        ref="input"
        v-model="draft"
        class="rename"
        maxlength="80"
        :aria-label="`Nuevo título para ${sound.title}`"
        @keydown.enter.prevent="finishRename(true)"
        @keydown.escape.stop.prevent="finishRename(false)"
        @blur="finishRename(true)"
      />
      <h3 v-else :id="titleId" ref="titleEl" class="sound-title" tabindex="-1">{{ sound.title }}</h3>
      <ActionsMenu :label="`Más acciones para ${sound.title}`" :actions="actions" />
    </div>
    <div class="sound-meta">
      <span class="muted">{{ seconds.format(sound.durationMs / 1000) }} s</span>
      <span v-if="replacing" class="chip">Reemplazando…</span>
      <span v-else-if="hidden" class="chip">Oculto</span>
      <button type="button" class="link" :aria-label="`${hidden ? 'Publicar' : 'Ocultar'} ${sound.title}`" @click="emit('toggle')">
        {{ hidden ? "Publicar" : "Ocultar" }}
      </button>
    </div>
  </article>
</template>
