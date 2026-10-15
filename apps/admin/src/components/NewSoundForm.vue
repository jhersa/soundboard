<script setup lang="ts">
import type { AdminSoundDto } from "@soundboard/shared";
import { reactive, ref, watch } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import SourceField from "./SourceField.vue";

const TIME_PATTERN = "\\d+(:\\d{1,2}){0,2}(\\.\\d+)?";

const props = defineProps<{ replacing: AdminSoundDto | null }>();
const emit = defineEmits<{ created: []; cancel: [] }>();

const initial = () => ({
  title: "",
  audioUrl: "",
  audioFile: null as File | null,
  imageUrl: "",
  imageFile: null as File | null,
  from: "",
  to: "",
  stereo: false,
});
const form = reactive(initial());
const formEl = ref<HTMLFormElement | null>(null);
const busy = ref(false);
const error = ref<string | null>(null);

// Al elegir un sonido para reemplazar se parte de su título; al cancelar se limpia el formulario.
watch(
  () => props.replacing,
  (sound) => {
    Object.assign(form, initial(), { title: sound?.title ?? "" });
    formEl.value?.reset();
    error.value = null;
  },
);

async function resolveSource(file: File | null, url: string): Promise<string | undefined> {
  if (file) return api.upload(file);
  return url || undefined;
}

async function submit() {
  busy.value = true;
  error.value = null;
  try {
    const url = await resolveSource(form.audioFile, form.audioUrl);
    if (!url) throw new Error("Falta el link o archivo del audio");
    await api.createJob({
      url,
      title: form.title.trim(),
      imageUrl: await resolveSource(form.imageFile, form.imageUrl),
      from: form.from.trim() || undefined,
      to: form.to.trim() || undefined,
      stereo: form.stereo,
      replacesSoundId: props.replacing?.id,
    });
    Object.assign(form, initial());
    formEl.value?.reset(); // limpia los <input type="file">
    emit("created");
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section>
    <h2 v-if="replacing">Reemplazar audio de «{{ replacing.title }}»</h2>
    <h2 v-else>Nuevo sonido</h2>
    <form ref="formEl" class="card" @submit.prevent="submit">
      <p v-if="replacing" class="muted">
        El audio nuevo toma el lugar de este sonido en el catálogo y en las bibliotecas de quien lo tenga. Si no
        subes otra imagen, conserva la actual. El audio anterior se borra al terminar.
      </p>
      <label>
        Título
        <input v-model="form.title" required maxlength="80" />
      </label>
      <SourceField
        v-model:url="form.audioUrl"
        v-model:file="form.audioFile"
        label="Audio (YouTube, TikTok, Epidemic Sound…)"
        accept="audio/*,video/*"
        required
      />
      <SourceField
        v-model:url="form.imageUrl"
        v-model:file="form.imageFile"
        :label="replacing ? 'Imagen o GIF (opcional; si no, se conserva la actual)' : 'Imagen o GIF (opcional)'"
        accept="image/*"
      />
      <div class="row">
        <label>
          Desde
          <input v-model="form.from" placeholder="0:03 o 3.5" :pattern="TIME_PATTERN" />
        </label>
        <label>
          Hasta
          <input v-model="form.to" placeholder="0:08" :pattern="TIME_PATTERN" />
        </label>
        <label class="inline"><input v-model="form.stereo" type="checkbox" /> Estéreo</label>
      </div>
      <div class="row">
        <button :disabled="busy">
          {{ busy ? "Enviando…" : replacing ? "Procesar y reemplazar" : "Procesar y publicar" }}
        </button>
        <button v-if="replacing" type="button" class="link" :disabled="busy" @click="emit('cancel')">Cancelar</button>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </section>
</template>
