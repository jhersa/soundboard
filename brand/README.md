# Identidad visual (CastBerry)

Paquete portable con mi identidad visual favorita: tokens, tema para Compose, variables CSS e iconos. La referencia es CastBerry (`frontend/src/theme.css`, `style.css` y `DESIGN.md`); este directorio la hace reutilizable en cualquier otro proyecto. `DESIGN.md` es una copia tal cual del sistema de diseño completo (componentes, reglas, qué sí y qué no); se actualiza desde CastBerry, no a mano.

## Estructura

```
brand/
  tokens.json            FUENTE ÚNICA: paleta, roles claro/oscuro, franja, efectos, formas, tipografía
  scripts/build.mjs      tokens.json → generated/ (sin dependencias, solo Node)
  DESIGN.md              sistema de diseño de CastBerry (copia)
  generated/
    tokens.css           variables CSS (claro por defecto, oscuro con data-theme="dark", franja .band)
    android/             Color.kt, Theme.kt, Type.kt para Jetpack Compose
  assets/
    logo/                icon.svg (esquinas redondeadas), icon-maskable.svg (fondo a sangre), icon-1024.png
    web/                 favicon-32, icon-192, icon-512, icon-maskable-512, apple-touch-icon, icon.svg
    android/             capas del launcher: foreground vectorial (la «C» y la baya) + fondo petróleo
```

## Identidad en una mirada

| Elemento | Valor |
|---|---|
| Concepto | «Muro de carátulas»: una sola familia de azules fríos que se retira para que el contenido sea el único color fuerte |
| Paleta fija | Tinta `#061E29`, petróleo `#1D546D`, salvia `#5F9598`, nieve `#F3F4F4` |
| Tema claro (de entrada) | Fondo nieve apagada con salvia `#C7D7D8`; tarjetas y campos en nieve; acción principal petróleo con texto blanco |
| Franja | En claro, barra y cabecera en petróleo (`.band`), donde los roles toman valores de noche |
| Tema oscuro (opcional) | Tinta de arriba abajo; superficies `#0F3242` y `#194A5D`; la nieve es la acción principal, la marca y el foco |
| Lo que ya viste | Salvia (progreso), como en el reproductor |
| Reproductor / pantalla completa | Siempre sobre negro petróleo `#03121A`, fuera del tema |
| Formas | Radios 6 / 8 / 14, píldora 999 |
| Tipografía | Display **Bricolage Grotesque** 700–800 para marca y títulos; cuerpo en la sans del sistema; nunca mayúsculas |
| Marca | Sin logo: el nombre escrito en la display. El icono es una «C» de nieve con una baya salvia sobre petróleo |
| Idioma de la UI | Español |

## Cómo llevarlo a otro proyecto

### 1. Web (cualquier framework)

```bash
cp brand/generated/tokens.css  <proyecto>/src/brand.css
cp brand/assets/web/*          <proyecto>/public/
npm i @fontsource-variable/bricolage-grotesque   # e import '@fontsource-variable/bricolage-grotesque' en el main
```

```html
<link rel="icon" type="image/svg+xml" href="/icon.svg" />
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32.png" />
<link rel="apple-touch-icon" href="/apple-touch-icon.png" />
<meta name="theme-color" content="#1d546d" />
```

```css
body { background: var(--color-background); color: var(--color-on-background); font-family: var(--font-sans); }
h1   { font: var(--type-title-large); letter-spacing: var(--type-title-large-spacing); }
.card   { background: var(--color-surface); border: 1px solid var(--color-outline); border-radius: var(--radius-large); }
input   { border: 1px solid var(--color-outline-strong); }
button  { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-pill); }
:focus-visible { outline: 2px solid var(--color-focus); }
```

El claro es el de entrada aunque el sistema esté en oscuro (no hay `prefers-color-scheme`, a propósito). El oscuro se activa con `data-theme="dark"` en `<html>`; guarda la elección en `localStorage` y aplícala con un script en el `<head>` antes de pintar (ver `src/theme.js` e `index.html` de CastBerry). La barra y la cabecera llevan `class="band"` para la franja petróleo; dentro, los componentes usan los mismos `--color-*`.

### 2. Android (Compose)

```bash
node brand/scripts/build.mjs --package com.tuempresa.app.core.ui.theme
cp brand/generated/android/*.kt  <app>/core/ui/src/main/java/.../theme/
cp brand/assets/android/*.xml    <app>/.../res/drawable/
```

`CastBerryTheme(darkTheme = …)` arranca en claro; pasa `true` cuando la persona elija el oscuro. Bricolage no viene en Android: agrégala en `res/font` y cambia `DisplayFont` en `Type.kt`.

### 3. Otro destino (Tailwind, iOS, Flutter…)

Lee `tokens.json` y genera lo que haga falta; agrega un emisor en `scripts/build.mjs` siguiendo el patrón de CSS/Kotlin.

## Cómo cambiar la identidad

1. Edita `tokens.json` (nunca `generated/`). La paleta de cuatro colores es fija.
2. `node brand/scripts/build.mjs` (o `pnpm brand:build` desde la raíz).
3. Copia los archivos a cada proyecto que los use. `node brand/scripts/build.mjs --check` falla si `generated/` quedó desactualizado.

Si cambia el icono, actualiza `icon.svg` e `icon-maskable.svg` y regenera los PNG (desde `brand/`):

```bash
rsvg-convert -w 32 -h 32 assets/logo/icon.svg -o assets/web/favicon-32.png
rsvg-convert -w 180 -h 180 assets/logo/icon.svg -o assets/web/apple-touch-icon.png
rsvg-convert -w 192 -h 192 assets/logo/icon.svg -o assets/web/icon-192.png
rsvg-convert -w 512 -h 512 assets/logo/icon.svg -o assets/web/icon-512.png
rsvg-convert -w 512 -h 512 assets/logo/icon-maskable.svg -o assets/web/icon-maskable-512.png
rsvg-convert -w 1024 -h 1024 assets/logo/icon.svg -o assets/logo/icon-1024.png
cp assets/logo/icon.svg assets/web/icon.svg
```

## Notas

- **Panel admin de Soundboard:** importa `generated/tokens.css` tal cual y usa `assets/web` como `publicDir`, así que ya tiene esta identidad. Como el tema ya no sigue al sistema, el panel queda en claro.
- **Contraste:** los valores vienen de CastBerry, ya revisados para AA (`on-surface-variant` 5:1 sobre el fondo apagado, `outline-strong` ≥3:1 para bordes de campos, acento de la franja 4.9:1).
- **`color-mix`:** las pistas de progreso y la luz ambiental lo usan; requiere navegadores de 2023 en adelante.
- **Solo web:** `extensionWeb`, `franja` y `efectosWeb` no existen en Compose (Material no define esos roles).
