<script setup lang="ts">
import type { UserDto } from "@soundboard/shared";
import { onMounted, ref } from "vue";
import { api, storedToken, storeToken } from "./api";
import Dashboard from "./components/Dashboard.vue";
import LoginView from "./components/LoginView.vue";
import ToastRegion from "./components/ToastRegion.vue";

const user = ref<UserDto | null>(null);
const checking = ref(storedToken() !== null);

onMounted(async () => {
  if (!storedToken()) return;
  try {
    user.value = await api.me();
  } catch {
    storeToken(null);
  } finally {
    checking.value = false;
  }
});

async function signOut() {
  await api.signOut().catch(() => undefined);
  storeToken(null);
  user.value = null;
}
</script>

<template>
  <main v-if="checking" class="center" role="status">Cargando…</main>
  <LoginView v-else-if="!user" @signed-in="user = $event" />
  <main v-else-if="!user.isAdmin" class="center">
    <p>{{ user.email }} no tiene permisos de administrador.</p>
    <button @click="signOut">Cerrar sesión</button>
  </main>
  <template v-else>
    <a class="skip-link" href="#sounds-heading">Ir a sonidos</a>
    <header class="band">
      <h1 class="wordmark">Soundboard Admin</h1>
      <span class="muted">{{ user.email }}</span>
      <button class="link" @click="signOut">Cerrar sesión</button>
    </header>
    <main><Dashboard /></main>
    <ToastRegion />
  </template>
</template>
