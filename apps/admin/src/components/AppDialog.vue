<script setup lang="ts">
// Diálogo modal sobre <dialog> nativo: atrapa el foco, cierra con Escape (no con clic en el fondo, para
// no perder lo escrito) y, al cerrarse, el navegador devuelve el foco al elemento que lo abrió.
// Se monta con v-if y se abre al montarse.
import { onMounted, ref } from "vue";

defineProps<{ title: string }>();
const emit = defineEmits<{ close: [] }>();
const dialog = ref<HTMLDialogElement | null>(null);

onMounted(() => dialog.value?.showModal());
</script>

<template>
  <dialog ref="dialog" class="dialog" aria-labelledby="dialog-title" @close="emit('close')">
    <div class="dialog-body">
      <h2 id="dialog-title">{{ title }}</h2>
      <slot :close="() => dialog?.close()" />
    </div>
  </dialog>
</template>
