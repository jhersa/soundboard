<script setup lang="ts">
// Menú de acciones secundarias de una tarjeta (patrón disclosure: botón con aria-expanded y una lista
// de botones). Se cierra con Escape, al hacer clic fuera o al salir el foco.
import { onBeforeUnmount, ref, watch } from "vue";
import Icon from "./Icon.vue";

export type MenuAction = { label: string; danger?: boolean; run: () => void };

defineProps<{ label: string; actions: MenuAction[] }>();

const open = ref(false);
const root = ref<HTMLElement | null>(null);
const toggle = ref<HTMLButtonElement | null>(null);
const list = ref<HTMLElement | null>(null);

function close(refocus: boolean) {
  open.value = false;
  if (refocus) toggle.value?.focus();
}

// La acción corre con el foco ya devuelto al botón: si abre un <dialog>, al cerrarse regresa aquí.
function choose(action: MenuAction) {
  close(true);
  action.run();
}

function onDocumentClick(event: MouseEvent) {
  if (!root.value?.contains(event.target as Node)) close(false);
}

function onFocusOut(event: FocusEvent) {
  if (!root.value?.contains(event.relatedTarget as Node | null)) close(false);
}

watch(open, (now) => {
  if (now) {
    document.addEventListener("click", onDocumentClick);
    requestAnimationFrame(() => list.value?.querySelector("button")?.focus());
  } else {
    document.removeEventListener("click", onDocumentClick);
  }
});
onBeforeUnmount(() => document.removeEventListener("click", onDocumentClick));
</script>

<template>
  <div ref="root" class="menu" @focusout="onFocusOut" @keydown.escape.stop="close(true)">
    <button
      ref="toggle"
      type="button"
      class="icon-button"
      :aria-label="label"
      :aria-expanded="open"
      @click="open = !open"
    >
      <Icon name="more" />
    </button>
    <ul v-if="open" ref="list" class="menu-list">
      <li v-for="action in actions" :key="action.label">
        <button type="button" :class="{ danger: action.danger }" @click="choose(action)">{{ action.label }}</button>
      </li>
    </ul>
  </div>
</template>
