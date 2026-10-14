# Identidad visual de Soundboard

Paquete portable con la identidad visual: tokens, tema para Compose, variables CSS y logos. La referencia es la app Android (`core/ui/.../theme/`); este directorio la hace reutilizable en cualquier otro proyecto.

## Estructura

```
brand/
  tokens.json            FUENTE ÚNICA: colores (claro/oscuro), formas, tipografía
  scripts/build.mjs      tokens.json → generated/ (sin dependencias, solo Node)
  generated/
    tokens.css           variables CSS (claro/oscuro, + data-theme para forzar)
    android/             Color.kt, Theme.kt, Type.kt para Jetpack Compose
  assets/
    logo/                giraffe.svg (jirafa sola), logo-square/circle.svg + PNG 1024,
                         vinyl-background.png (fondo del ícono)
    web/                 favicon-32, icon-192, icon-512, apple-touch-icon
    android/             capas del launcher tal cual están en la app (foreground vectorial + fondo)
```

## Identidad en una mirada

| Elemento | Valor |
|---|---|
| Concepto | Terminal/hacker oscuro y vibrante (acentos del tema de VS Code «Laetus Black»): jirafa amarilla sobre un vinilo violeta neón |
| Color principal | Amarillo `#FFD600` (oscuro) / `#C7A500` (claro): el de la jirafa |
| Acentos vibrantes | Lima `#B2FF59` (secundario), magenta `#E040FB` (terciario), cian `#40C4FF` (info), naranja `#FFAB40` (aviso), rojo `#FF5252` (error) |
| Fondo oscuro | `#0A0A0A`; superficies `#141414` y `#1E1E1E`; texto `#F8F8F0`; borde `#2E2E2E` |
| Fondo claro | `#F7F7F5`; superficies `#FFFFFF`; en claro los acentos de texto se oscurecen para cumplir contraste AA |
| Splash / fondo del ícono | `#0E0A14`; violeta neón del vinilo `#C04BFF` |
| Formas | Radios 10 / 16 / 22 dp, botones en píldora |
| Tipografía | Sans del sistema para texto; **monoespaciada semibold** para títulos |
| Tema | Sigue al sistema (claro/oscuro); el modo oscuro es el protagonista |
| Idioma de la UI | Español |

Uso del logo: el círculo (`logo-circle`) para avatares y encabezados; el cuadrado (`logo-square`) para íconos que el sistema recorta solo (apple-touch, launcher). La jirafa sola (`giraffe.svg`) es amarilla: úsala sobre fondos oscuros. No recolorees la jirafa ni muevas el vinilo.

## Cómo llevarlo a otro proyecto

### 1. Web (cualquier framework)

```bash
cp brand/generated/tokens.css  <proyecto>/src/brand.css
cp brand/assets/web/*          <proyecto>/public/
```

```html
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32.png" />
<link rel="apple-touch-icon" href="/apple-touch-icon.png" />
```

```css
body { background: var(--color-background); color: var(--color-on-background); font-family: var(--font-sans); }
h1   { font: var(--type-title-large); }
.card   { background: var(--color-surface); border: 1px solid var(--color-outline); border-radius: var(--radius-medium); }
button  { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-pill); }
```

El tema sigue `prefers-color-scheme`; para forzarlo pon `data-theme="light"` o `"dark"` en `<html>`.

### 2. Android (Compose)

```bash
node brand/scripts/build.mjs --package com.tuempresa.app.core.ui.theme
cp brand/generated/android/*.kt  <app>/core/ui/src/main/java/.../theme/
cp brand/assets/android/ic_launcher_foreground.xml  <app>/.../res/drawable/
```

El fondo del launcher es `assets/android/ic_launcher_background-xxxhdpi.png` (432 px); si necesitas más densidades, escálalo desde `assets/logo/vinyl-background.png`. El splash usa `DeckSplash` / `#0E0A14` (ver `values-v31/themes.xml` de la app).

### 3. Otro destino (Tailwind, iOS, Flutter…)

Lee `tokens.json` y genera lo que haga falta; agrega un emisor en `scripts/build.mjs` siguiendo el patrón de CSS/Kotlin.

## Cómo cambiar la identidad

1. Edita `tokens.json` (nunca `generated/`).
2. `node brand/scripts/build.mjs` (o `pnpm brand:build` desde la raíz).
3. Copia los archivos a cada proyecto que los use. `node brand/scripts/build.mjs --check` falla si `generated/` quedó desactualizado.

## Notas y deuda conocida

- **Panel admin:** ya usa esta identidad. `apps/admin/src/main.ts` importa `generated/tokens.css` tal cual (sin copiarlo), `vite.config.ts` usa `assets/web` como `publicDir` y su Dockerfile copia ambos. Para cambiar los colores del panel, edita `tokens.json` y corre `pnpm brand:build`; `apps/admin/src/styles.css` solo usa variables `--color-*`.
- **La app Android aún usa la paleta anterior** (menta/violeta suave, fondo `#0D0E12`). Los tokens ya son los vibrantes; falta copiar `generated/android/*.kt` a `core/ui`. Además, `onPrimary` pasó a negro en el tema claro (antes blanco, 2.4:1; ahora ≈8.8:1 sobre `#C7A500`).
- **Origen de los acentos:** salen del tema Laetus (`10f.laetus`, variante Black). El `outline` `#2E2E2E` y el gris de texto secundario `#A0A09A` no existen en el tema (sus bordes son casi invisibles y su gris de comentarios no llega a AA); se derivaron para que la UI sea legible.
- **Logo SVG con raster:** `logo-square.svg` y `logo-circle.svg` referencian `vinyl-background.png` por ruta relativa; cópialos juntos. El vinilo solo existe como PNG de 432 px (no hay vector), por eso los PNG de 1024 son una ampliación suave del fondo; la jirafa sí es vectorial.
- **Extensión web:** `--color-success` no existe en Android (Material no lo define); se agregó en `extensionWeb` para estados "ok".
