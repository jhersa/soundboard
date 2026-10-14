import { createApp } from "vue";
import App from "./App.vue";
// La identidad visual (colores claro/oscuro, formas y tipografías) es la de brand/; se importa tal cual
// para no copiarla. Se regenera con `pnpm brand:build`.
import "../../../brand/generated/tokens.css";
import "./styles.css";

createApp(App).mount("#root");
