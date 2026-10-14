<script setup lang="ts">
// Un campo acepta un link o un archivo; si hay archivo, tiene prioridad sobre el link.
defineProps<{ label: string; accept: string; required?: boolean }>();
const url = defineModel<string>("url", { required: true });
const file = defineModel<File | null>("file", { required: true });

function onFileChange(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0] ?? null;
}
</script>

<template>
  <fieldset>
    <legend>{{ label }}</legend>
    <input v-model.trim="url" type="url" placeholder="https://…" />
    <span class="muted">o archivo</span>
    <input type="file" :accept="accept" @change="onFileChange" />
    <small v-if="required" class="muted">Obligatorio: link o archivo.</small>
  </fieldset>
</template>
