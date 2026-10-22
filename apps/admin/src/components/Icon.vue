<script setup lang="ts">
// Íconos del panel: trazo de 2px en currentColor, cuadrícula de 24. Decorativos: el botón lleva el nombre.
defineProps<{ name: "play" | "stop" | "more" | "search" | "close" | "note" | "plus" }>();
</script>

<template>
  <svg
    class="icon"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path v-if="name === 'play'" d="M8 5.5v13l10.5-6.5z" fill="currentColor" />
    <rect v-else-if="name === 'stop'" x="7" y="7" width="10" height="10" rx="1.5" fill="currentColor" />
    <template v-else-if="name === 'more'">
      <circle cx="5" cy="12" r="1.25" fill="currentColor" />
      <circle cx="12" cy="12" r="1.25" fill="currentColor" />
      <circle cx="19" cy="12" r="1.25" fill="currentColor" />
    </template>
    <template v-else-if="name === 'search'">
      <circle cx="11" cy="11" r="6.5" />
      <path d="m16 16 4.5 4.5" />
    </template>
    <path v-else-if="name === 'close'" d="M6 6l12 12M18 6 6 18" />
    <path v-else-if="name === 'plus'" d="M12 5v14M5 12h14" />
    <template v-else>
      <path d="M9 18V5l11-2v13" />
      <circle cx="6.5" cy="18" r="2.5" />
      <circle cx="17.5" cy="16" r="2.5" />
    </template>
  </svg>
</template>
