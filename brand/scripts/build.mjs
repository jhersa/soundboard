#!/usr/bin/env node
// Genera brand/generated/ a partir de brand/tokens.json. Sin dependencias.
//
//   node brand/scripts/build.mjs                          # paquete Kotlin por defecto
//   node brand/scripts/build.mjs --package com.acme.app.core.ui.theme
//   node brand/scripts/build.mjs --check                  # falla si generated/ no está al día

import { readFileSync, writeFileSync, mkdirSync, existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const raiz = join(dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
const check = args.includes("--check");
const iPkg = args.indexOf("--package");
const paquete = iPkg >= 0 ? args[iPkg + 1] : "com.cornytuna.soundboard.core.ui.theme";

const t = JSON.parse(readFileSync(join(raiz, "tokens.json"), "utf8"));

const mayus = (s) => s[0].toUpperCase() + s.slice(1);
const kebab = (s) => s.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
const kt = (hex) => `Color(0xFF${hex.slice(1).toUpperCase()})`;
const px = (n) => `${n}px`;
const familia = (f) => `var(--font-${f})`;
const lista = (fuentes) => fuentes.map((f) => (/\s/.test(f) ? `"${f}"` : f)).join(", ");

// ---------- CSS ----------
function bloqueColores(modo) {
  const roles = { ...t.colores[modo], ...t.extensionWeb[modo] };
  return Object.entries(roles)
    .map(([rol, hex]) => `  --color-${kebab(rol)}: ${hex};`)
    .join("\n");
}

const css = `/* Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano. */
:root {
  color-scheme: light dark;

  /* Marca (iguales en claro y oscuro) */
  --brand-yellow: ${t.marca.amarillo};
  --brand-yellow-dark: ${t.marca.amarilloOscuro};
  --brand-lime: ${t.marca.lima};
  --brand-magenta: ${t.marca.magenta};
  --brand-cyan: ${t.marca.cian};
  --brand-orange: ${t.marca.naranja};
  --brand-red: ${t.marca.rojo};
  --brand-violet-neon: ${t.marca.violetaNeon};
  --brand-splash: ${t.marca.fondoSplash};

  /* Formas */
${Object.entries(t.formas).map(([k, v]) => `  --radius-${k}: ${px(v)};`).join("\n")}

  /* Tipografía */
  --font-sans: ${lista(t.tipografia.sans)};
  --font-mono: ${lista(t.tipografia.mono)};
${Object.entries(t.tipografia.estilos)
  .map(([k, e]) => `  --type-${kebab(k)}: ${e.peso} ${px(e.tamano)}/${px(e.interlineado)} ${familia(e.familia)};
  --type-${kebab(k)}-spacing: ${e.espaciado}px;`)
  .join("\n")}

  /* Colores: claro por defecto */
${bloqueColores("claro")}
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
${bloqueColores("oscuro").replace(/^/gm, "  ")}
  }
}

:root[data-theme="dark"] {
${bloqueColores("oscuro")}
}
`;

// ---------- Kotlin ----------
const rolesKt = Object.keys(t.colores.oscuro);
const nombreKt = (rol, modo) => `Deck${mayus(rol)}${mayus(modo === "oscuro" ? "dark" : "light")}`;

const colorKt = `package ${paquete}

import androidx.compose.ui.graphics.Color

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
val DeckYellow = ${kt(t.marca.amarillo)}
val DeckYellowDark = ${kt(t.marca.amarilloOscuro)}
val DeckLime = ${kt(t.marca.lima)}
val DeckMagenta = ${kt(t.marca.magenta)}
val DeckCyan = ${kt(t.marca.cian)}
val DeckOrange = ${kt(t.marca.naranja)}
val DeckRed = ${kt(t.marca.rojo)}
val DeckVioletNeon = ${kt(t.marca.violetaNeon)}
val DeckSplash = ${kt(t.marca.fondoSplash)}
${["oscuro", "claro"]
  .map((m) => "\n" + rolesKt.map((r) => `val ${nombreKt(r, m)} = ${kt(t.colores[m][r])}`).join("\n"))
  .join("")}
`;

const esquema = (modo, fn) =>
  `private val ${modo === "oscuro" ? "Dark" : "Light"}ColorScheme = ${fn}(\n` +
  rolesKt.map((r) => `    ${r} = ${nombreKt(r, modo)}`).join(",\n") +
  "\n)";

const themeKt = `package ${paquete}

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
${esquema("oscuro", "darkColorScheme")}

${esquema("claro", "lightColorScheme")}

private val DeckShapes = Shapes(
    small = RoundedCornerShape(${t.formas.small}.dp),
    medium = RoundedCornerShape(${t.formas.medium}.dp),
    large = RoundedCornerShape(${t.formas.large}.dp)
)

@Composable
fun SoundboardTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        shapes = DeckShapes,
        content = content
    )
}
`;

const famKt = { sans: "FontFamily.Default", mono: "FontFamily.Monospace" };
const pesoKt = { 400: "Normal", 600: "SemiBold" };
const typeKt = `package ${paquete}

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
val Typography = Typography(
${Object.entries(t.tipografia.estilos)
  .map(([k, e]) => `    ${k} = TextStyle(
        fontFamily = ${famKt[e.familia]},
        fontWeight = FontWeight.${pesoKt[e.peso]},
        fontSize = ${e.tamano}.sp,
        lineHeight = ${e.interlineado}.sp,
        letterSpacing = ${e.espaciado}.sp
    )`)
  .join(",\n")}
)
`;

// ---------- Salida ----------
const salidas = {
  "tokens.css": css,
  "android/Color.kt": colorKt,
  "android/Theme.kt": themeKt,
  "android/Type.kt": typeKt
};

let desactualizado = false;
for (const [ruta, contenido] of Object.entries(salidas)) {
  const destino = join(raiz, "generated", ruta);
  if (check) {
    if (!existsSync(destino) || readFileSync(destino, "utf8") !== contenido) {
      console.error(`Desactualizado: generated/${ruta}`);
      desactualizado = true;
    }
    continue;
  }
  mkdirSync(dirname(destino), { recursive: true });
  writeFileSync(destino, contenido);
  console.log(`generated/${ruta}`);
}
if (check) process.exit(desactualizado ? 1 : 0);
