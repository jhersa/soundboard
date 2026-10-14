<script setup lang="ts">
import type { AdminSoundDto } from "@soundboard/shared";
import { computed, ref } from "vue";
import { api } from "../api";

const props = defineProps<{ sounds: AdminSoundDto[] }>();
const emit = defineEmits<{ changed: [] }>();

const filter = ref("");
const visible = computed(() =>
  props.sounds.filter((sound) => sound.title.toLowerCase().includes(filter.value.toLowerCase())),
);

async function rename(sound: AdminSoundDto) {
  const title = prompt("Nuevo título", sound.title)?.trim();
  if (!title || title === sound.title) return;
  await api.updateSound(sound.id, { title });
  emit("changed");
}

async function toggle(sound: AdminSoundDto) {
  await api.updateSound(sound.id, { status: sound.status === "published" ? "hidden" : "published" });
  emit("changed");
}

async function remove(sound: AdminSoundDto) {
  if (!confirm(`¿Borrar "${sound.title}"? Se quitará de todas las bibliotecas.`)) return;
  await api.deleteSound(sound.id);
  emit("changed");
}
</script>

<template>
  <section>
    <h2>
      Sonidos <span class="muted">({{ sounds.length }})</span>
    </h2>
    <input v-model="filter" placeholder="Buscar…" />
    <div class="grid">
      <article v-for="sound in visible" :key="sound.id" class="card sound" :class="sound.status">
        <img v-if="sound.imageUrl" :src="sound.imageUrl" alt="" loading="lazy" />
        <div v-else class="placeholder">♪</div>
        <strong>{{ sound.title }}</strong>
        <span class="muted">
          {{ (sound.durationMs / 1000).toFixed(1) }} s<template v-if="sound.status === 'hidden'"> · oculto</template>
        </span>
        <audio :src="sound.audioUrl" controls preload="none" />
        <div class="actions">
          <button class="link" @click="rename(sound)">Renombrar</button>
          <button class="link" @click="toggle(sound)">{{ sound.status === "published" ? "Ocultar" : "Publicar" }}</button>
          <button class="link danger" @click="remove(sound)">Borrar</button>
        </div>
      </article>
    </div>
  </section>
</template>
