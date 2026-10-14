<script setup lang="ts">
import type { UserDto } from "@soundboard/shared";
import { onMounted, ref } from "vue";
import { api, storedToken, storeToken } from "./api";
import Dashboard from "./components/Dashboard.vue";
import LoginView from "./components/LoginView.vue";

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
  <main v-if="checking" class="center">Cargando…</main>
  <LoginView v-else-if="!user" @signed-in="user = $event" />
  <main v-else-if="!user.isAdmin" class="center">
    <p>{{ user.email }} no tiene permisos de administrador.</p>
    <button @click="signOut">Cerrar sesión</button>
  </main>
  <main v-else>
    <header>
      <img class="logo" src="/icon-192.png" alt="" />
      <h1>Soundboard Admin</h1>
      <span class="muted">{{ user.email }}</span>
      <button class="link" @click="signOut">Cerrar sesión</button>
    </header>
    <Dashboard />
  </main>
</template>
