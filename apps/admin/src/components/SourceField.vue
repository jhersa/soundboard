<script setup lang="ts">
// Un campo acepta un link o un archivo, nunca los dos: al elegir archivo el link se vacía y se oculta.
defineProps<{ label: string; accept: string; hint?: string; autofocus?: boolean }>();
const url = defineModel<string>("url", { required: true });
const file = defineModel<File | null>("file", { required: true });

const megabytes = new Intl.NumberFormat("es", { maximumFractionDigits: 1 });

function onFileChange(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0] ?? null;
  if (file.value) url.value = "";
}
</script>

<template>
  <fieldset>
    <legend>{{ label }}</legend>
    <template v-if="!file">
      <input v-model.trim="url" type="url" placeholder="https://…" aria-label="Link" :autofocus="autofocus" />
      <span class="muted or">o</span>
      <label class="file-button">
        Subir archivo
        <input type="file" class="visually-hidden" :accept="accept" @change="onFileChange" />
      </label>
    </template>
    <template v-else>
      <span class="file-name">{{ file.name }} <span class="muted">· {{ megabytes.format(file.size / 2 ** 20) }} MB</span></span>
      <button type="button" class="link" @click="file = null">Quitar archivo</button>
    </template>
    <small v-if="hint" class="muted">{{ hint }}</small>
  </fieldset>
</template>
