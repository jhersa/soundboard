<script setup lang="ts">
import { reactive, ref } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import SourceField from "./SourceField.vue";

const TIME_PATTERN = "\\d+(:\\d{1,2}){0,2}(\\.\\d+)?";

const emit = defineEmits<{ created: [] }>();

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
    <h2>Nuevo sonido</h2>
    <form ref="formEl" class="card" @submit.prevent="submit">
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
        label="Imagen o GIF (opcional)"
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
      <button :disabled="busy">{{ busy ? "Enviando…" : "Procesar y publicar" }}</button>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </section>
</template>
