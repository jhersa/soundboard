import { createApp } from "vue";
import App from "./App.vue";
// Bricolage Grotesque (--font-display) es la letra de los títulos de la marca.
import "@fontsource-variable/bricolage-grotesque";
// La identidad visual (colores claro/oscuro, formas y tipografías) es la de brand/; se importa tal cual
// para no copiarla. Se regenera con `pnpm brand:build`.
import "../../../brand/generated/tokens.css";
import "./styles.css";

createApp(App).mount("#root");
