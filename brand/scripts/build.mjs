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
const lista = (fuentes) => fuentes.map((f) => (/\s/.test(f) && !f.startsWith("var(") ? `"${f}"` : f)).join(", ");
const sinMeta = (o) => Object.fromEntries(Object.entries(o).filter(([k]) => !k.startsWith("$")));
const paleta = t.marca.paleta;

// ---------- CSS ----------
const lineas = (o, sangria = "  ") => Object.entries(o).map(([k, v]) => `${sangria}${k}: ${v};`).join("\n");
const roles = (o) => Object.fromEntries(Object.entries(sinMeta(o)).map(([rol, v]) => [`--color-${kebab(rol)}`, v]));
const colores = (modo) => ({ ...roles(t.colores[modo]), ...roles(t.extensionWeb[modo]), ...t.efectosWeb[modo] });

const css = `/* Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
   El claro es el tema de entrada (sin prefers-color-scheme); el oscuro se activa
   con data-theme="dark" en <html>. Display: importa @fontsource-variable/bricolage-grotesque. */
:root {
  color-scheme: light;

  /* Paleta (igual en claro y oscuro) */
${lineas(Object.fromEntries(Object.entries(paleta).map(([k, v]) => [`--palette-${k}`, v])))}

  /* Formas */
${Object.entries(t.formas).map(([k, v]) => `  --radius-${k}: ${px(v)};`).join("\n")}

  /* Tipografía */
  --font-sans: ${lista(t.tipografia.sans)};
  --font-mono: ${lista(t.tipografia.mono)};
  --font-display: ${lista(t.tipografia.display)};
${Object.entries(t.tipografia.estilos)
  .map(([k, e]) => `  --type-${kebab(k)}: ${e.peso} ${e.tamanoCss ?? px(e.tamano)}/${e.interlineado} var(--font-${e.familia});
  --type-${kebab(k)}-spacing: ${e.espaciadoEm}em;`)
  .join("\n")}

  /* Fijos en ambos temas: reproductor y texto sobre imágenes */
${lineas(sinMeta(t.efectosWeb.comun))}

  /* Colores: claro por defecto */
${lineas(colores("claro"))}
}

:root[data-theme="dark"] {
  color-scheme: dark;
${lineas(colores("oscuro"))}
}

/* Franja petróleo del claro: pon class="band" en la barra y la cabecera. */
.band {
  background: var(--color-background);
  color: var(--color-on-background);
}
:root:not([data-theme="dark"]) .band {
  color-scheme: dark;
${lineas({ ...roles(t.franja), ...t.efectosWeb.franja })}
}
`;

// ---------- Kotlin ----------
const rolesKt = Object.keys(sinMeta(t.colores.oscuro));
const nombreKt = (rol, modo) => `Brand${mayus(rol)}${modo === "oscuro" ? "Dark" : "Light"}`;

const colorKt = `package ${paquete}

import androidx.compose.ui.graphics.Color

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
${Object.entries(paleta).map(([k, v]) => `val Brand${mayus(k)} = ${kt(v)}`).join("\n")}
${["claro", "oscuro"]
  .map((m) => "\n" + rolesKt.map((r) => `val ${nombreKt(r, m)} = ${kt(t.colores[m][r])}`).join("\n"))
  .join("")}
`;

const esquema = (modo, fn) =>
  `private val ${modo === "oscuro" ? "Dark" : "Light"}ColorScheme = ${fn}(\n` +
  rolesKt.map((r) => `    ${r} = ${nombreKt(r, modo)}`).join(",\n") +
  "\n)";

const themeKt = `package ${paquete}

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
${esquema("claro", "lightColorScheme")}

${esquema("oscuro", "darkColorScheme")}

private val BrandShapes = Shapes(
    small = RoundedCornerShape(${t.formas.small}.dp),
    medium = RoundedCornerShape(${t.formas.medium}.dp),
    large = RoundedCornerShape(${t.formas.large}.dp)
)

// El claro es el de entrada; el oscuro lo elige la persona (no sigue al sistema).
@Composable
fun ${t.marca.nombre}Theme(
    darkTheme: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        shapes = BrandShapes,
        content = content
    )
}
`;

// Bricolage no viene en Android: agrégala en res/font y cambia DisplayFont.
const famKt = { sans: "FontFamily.Default", mono: "FontFamily.Monospace", display: "DisplayFont" };
const pesoKt = { 400: "Normal", 600: "SemiBold", 700: "Bold", 800: "ExtraBold" };
const typeKt = `package ${paquete}

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
// Display: Bricolage Grotesque. Agrégala en res/font y reemplaza FontFamily.Default.
val DisplayFont: FontFamily = FontFamily.Default

val Typography = Typography(
${Object.values(t.tipografia.estilos)
  .filter((e) => e.material)
  .map((e) => `    ${e.material} = TextStyle(
        fontFamily = ${famKt[e.familia]},
        fontWeight = FontWeight.${pesoKt[e.peso]},
        fontSize = ${e.tamano}.sp,
        lineHeight = ${e.interlineado}.em,
        letterSpacing = ${e.espaciadoEm}.em
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
