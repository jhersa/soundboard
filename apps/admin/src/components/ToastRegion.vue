<script setup lang="ts">
import { dismiss, hold, release, toasts, type Toast } from "../toasts";
import Icon from "./Icon.vue";

function run(item: Toast) {
  dismiss(item.id);
  item.action?.run();
}
</script>

<template>
  <!-- Región viva siempre presente: los lectores de pantalla anuncian cada aviso al aparecer. -->
  <div class="toasts" role="status" aria-live="polite">
    <div
      v-for="item in toasts"
      :key="item.id"
      class="toast"
      :class="{ error: item.error }"
      @pointerenter="hold(item.id)"
      @pointerleave="release(item.id)"
      @focusin="hold(item.id)"
      @focusout="release(item.id)"
    >
      <p>{{ item.text }}</p>
      <button v-if="item.action" type="button" class="link" @click="run(item)">{{ item.action.label }}</button>
      <button type="button" class="icon-button" aria-label="Cerrar aviso" @click="dismiss(item.id)">
        <Icon name="close" />
      </button>
    </div>
  </div>
</template>
