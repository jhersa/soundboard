import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

// Los favicons y el icono del logo salen de brand/assets/web (se sirven en la raíz: /favicon-32.png,
// /icon-192.png...), así no se copian al panel.
export default defineConfig({ plugins: [vue()], publicDir: "../../brand/assets/web" });
