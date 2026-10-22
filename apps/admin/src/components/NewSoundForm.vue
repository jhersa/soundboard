<script setup lang="ts">
import type { AdminSoundDto, JobDto } from "@soundboard/shared";
import { onMounted, reactive, ref, watch } from "vue";
import { api } from "../api";
import { errorMessage } from "../errors";
import SourceField from "./SourceField.vue";

const TIME_PATTERN = "\\d+(:\\d{1,2}){0,2}(\\.\\d+)?";
const TIME_RE = new RegExp(`^${TIME_PATTERN}$`);
// Debe coincidir con maxUploadBytes de la API (apps/api/internal/httpapi/dto.go).
const MAX_UPLOAD_BYTES = 50 << 20;

// En modo reemplazo vive dentro de un diálogo que se monta al abrirse: parte del título del sonido.
const props = defineProps<{ replacing?: AdminSoundDto }>();
const emit = defineEmits<{ created: [job: JobDto]; cancel: [] }>();

const initial = () => ({
  title: props.replacing?.title ?? "",
  audioUrl: "",
  audioFile: null as File | null,
  imageUrl: "",
  imageFile: null as File | null,
  from: "",
  to: "",
  stereo: false,
});
const form = reactive(initial());
const titleEl = ref<HTMLInputElement | null>(null);
const optionsEl = ref<HTMLDetailsElement | null>(null);
const toEl = ref<HTMLInputElement | null>(null);
const status = ref<string | null>(null); // texto del botón mientras se envía
const uploaded = ref<number | null>(null); // avance de la subida en curso, de 0 a 1
const sent = ref<string | null>(null);
const error = ref<string | null>(null);

// Al crear, el foco va al título; al reemplazar (en un diálogo), autofocus lo lleva al audio.
onMounted(() => {
  if (!props.replacing) titleEl.value?.focus();
});

// "1:05.5" → 65.5 segundos.
const toSeconds = (time: string) => time.split(":").reduce((total, part) => total * 60 + Number(part), 0);

watch(
  () => [form.from, form.to],
  () => {
    const [from, to] = [form.from.trim(), form.to.trim()];
    const backwards = TIME_RE.test(from) && TIME_RE.test(to) && toSeconds(to) <= toSeconds(from);
    toEl.value?.setCustomValidity(backwards ? "«Hasta» debe ser mayor que «Desde»." : "");
  },
);

// Si un campo inválido está dentro de «Opciones» plegado, el navegador no puede enfocarlo: se despliega.
function onInvalid(event: Event) {
  if (optionsEl.value?.contains(event.target as Node)) optionsEl.value.open = true;
}

const percent = new Intl.NumberFormat("es", { style: "percent" });

async function resolveSource(kind: string, file: File | null, url: string): Promise<string | undefined> {
  if (!file) return url || undefined;
  if (file.size > MAX_UPLOAD_BYTES) throw new Error(`«${file.name}» pesa más de 50 MB, el máximo para subir.`);
  status.value = `Subiendo ${kind}…`;
  uploaded.value = 0;
  try {
    return await api.upload(file, (fraction) => {
      uploaded.value = fraction;
      status.value = `Subiendo ${kind}… ${percent.format(fraction)}`;
    });
  } finally {
    uploaded.value = null;
  }
}

async function submit() {
  status.value = "Enviando…";
  sent.value = null;
  error.value = null;
  try {
    const url = await resolveSource("audio", form.audioFile, form.audioUrl);
    if (!url) throw new Error("Falta el audio: pega un link o sube un archivo.");
    const imageUrl = await resolveSource("imagen", form.imageFile, form.imageUrl);
    status.value = "Enviando…";
    const job = await api.createJob({
      url,
      title: form.title.trim(),
      imageUrl,
      from: form.from.trim() || undefined,
      to: form.to.trim() || undefined,
      stereo: form.stereo,
      replacesSoundId: props.replacing?.id,
    });
    sent.value = `«${job.title}» está en proceso; síguelo en Procesos recientes.`;
    Object.assign(form, initial());
    emit("created", job);
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    status.value = null;
  }
}
</script>

<template>
  <form :class="{ card: !replacing }" class="sound-form" @submit.prevent="submit" @invalid.capture="onInvalid">
    <label>
      Título
      <input ref="titleEl" v-model="form.title" required maxlength="80" />
    </label>
    <SourceField
      v-model:url="form.audioUrl"
      v-model:file="form.audioFile"
      label="Audio"
      accept="audio/*,video/*"
      :autofocus="!!replacing"
      hint="Obligatorio: link (YouTube, TikTok, Epidemic Sound…) o archivo de audio o video de hasta 50 MB."
    />
    <SourceField
      v-model:url="form.imageUrl"
      v-model:file="form.imageFile"
      label="Imagen o GIF"
      accept="image/*"
      :hint="replacing ? 'Opcional; si no pones otra, se conserva la actual.' : 'Opcional.'"
    />
    <details ref="optionsEl" class="options">
      <summary>Recorte y estéreo <span class="muted">(opcional)</span></summary>
      <div class="row">
        <label>
          Desde
          <input v-model="form.from" placeholder="0:03" inputmode="decimal" :pattern="TIME_PATTERN" aria-describedby="time-hint" />
        </label>
        <label>
          Hasta
          <input ref="toEl" v-model="form.to" placeholder="0:08" inputmode="decimal" :pattern="TIME_PATTERN" aria-describedby="time-hint" />
        </label>
      </div>
      <p id="time-hint" class="muted">
        Segundos (3.5) o minutos:segundos (0:03). Vacío: desde el inicio o hasta el final.
      </p>
      <label class="inline"><input v-model="form.stereo" type="checkbox" /> Conservar estéreo (si no, se mezcla a mono)</label>
    </details>
    <div class="row">
      <button :disabled="!!status">
        {{ status ?? (replacing ? "Procesar y reemplazar" : "Procesar y publicar") }}
      </button>
      <button type="button" class="link" :disabled="!!status" @click="emit('cancel')">
        {{ replacing ? "Cancelar" : "Cerrar" }}
      </button>
    </div>
    <progress v-if="uploaded !== null" class="upload-progress" :value="uploaded" max="1" aria-label="Avance de la subida" />
    <p v-if="sent && !replacing" class="success" role="status">{{ sent }}</p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
  </form>
</template>
