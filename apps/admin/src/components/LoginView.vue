<script setup lang="ts">
import type { UserDto } from "@soundboard/shared";
import { onMounted, onUnmounted, ref } from "vue";
import { api, storeToken } from "../api";
import { errorMessage } from "../errors";

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as string;

const emit = defineEmits<{ signedIn: [user: UserDto] }>();
const buttonEl = ref<HTMLDivElement | null>(null);
const error = ref<string | null>(null);
let timer: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  // El script de Google Identity Services carga async; se espera a que exista window.google.
  timer = setInterval(() => {
    if (!window.google || !buttonEl.value) return;
    clearInterval(timer);
    window.google.accounts.id.initialize({
      client_id: GOOGLE_CLIENT_ID,
      callback: async ({ credential }) => {
        try {
          const { token, user } = await api.signIn(credential);
          storeToken(token);
          emit("signedIn", user);
        } catch (e) {
          error.value = errorMessage(e);
        }
      },
    });
    window.google.accounts.id.renderButton(buttonEl.value, {
      // Sigue el tema del sistema, igual que el resto del panel.
      theme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "filled_black" : "outline",
      size: "large",
      locale: "es",
    });
  }, 100);
});

onUnmounted(() => clearInterval(timer));
</script>

<template>
  <main class="center">
    <img class="logo" src="/icon-192.png" alt="" />
    <h1>Soundboard Admin</h1>
    <div ref="buttonEl" />
    <p v-if="error" class="error">{{ error }}</p>
  </main>
</template>
