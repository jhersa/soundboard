---
name: CastBerry
description: Muro de carátulas para la videoteca de la casa; azules petróleo de tinta a nieve, donde las carátulas son el único color fuerte.
colors:
  tinta: "#061e29"
  petroleo: "#1d546d"
  salvia: "#5f9598"
  nieve: "#f3f4f4"
  acento-noche: "#8fc2c4"
  superficie-noche: "#0f3242"
  superficie-noche-alta: "#194a5d"
  linea-noche: "#2a5a6d"
  borde-control-noche: "#6f9aa6"
  tinta-noche-tenue: "#b3cdd1"
  error-noche: "#ff8a80"
  nieve-salvia-apagada: "#c7d7d8"
  blanco: "#ffffff"
  superficie-clara-alta: "#e3e9ea"
  linea-clara: "#a3b9bc"
  borde-control-claro: "#5a7480"
  tinta-clara: "#0c2a38"
  tinta-clara-tenue: "#3e5864"
  error-claro: "#b3261e"
  pista-progreso-noche: "rgba(243, 244, 244, 0.1)"
  franja-superficie: "#24607b"
  franja-superficie-alta: "#2a6680"
  franja-tinta-tenue: "#c9dcdf"
  franja-linea: "#3a7590"
  franja-borde-control: "#8fb4bf"
  franja-acento: "#a9d0d2"
  pista-progreso-franja: "rgba(243, 244, 244, 0.18)"
  velo: "rgba(3, 18, 26, 0.94)"
  tinta-sobre-velo-tenue: "#c3d6d8"
  fondo-reproductor: "#03121a"
typography:
  marca:
    fontFamily: "'Bricolage Grotesque Variable', system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "1.6rem"
    fontWeight: 800
    lineHeight: 1
    letterSpacing: "-0.04em"
  display:
    fontFamily: "'Bricolage Grotesque Variable', system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "clamp(2rem, 1.25rem + 3vw, 3.5rem)"
    fontWeight: 800
    lineHeight: 1.02
    letterSpacing: "-0.03em"
  title-large:
    fontFamily: "'Bricolage Grotesque Variable', system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "22px"
    fontWeight: 700
    lineHeight: "28px"
    letterSpacing: "-0.01em"
  title-medium:
    fontFamily: "'Bricolage Grotesque Variable', system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 700
    lineHeight: "1.6rem"
    letterSpacing: "-0.01em"
  meta:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: "1.5rem"
    fontFeature: "tnum"
  body:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: "24px"
    letterSpacing: "0"
  pie-caratula:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif"
    fontSize: "0.95rem"
    fontWeight: 650
    lineHeight: 1.3
rounded:
  pequeno: "6px"
  medio: "8px"
  grande: "14px"
  pildora: "999px"
spacing:
  margen-pagina: "clamp(1rem, 4vw, 3.5rem)"
  hueco-fila: "16px"
  separacion-filas: "36px"
  hueco-cuadricula: "28px 20px"
  hueco-cuadricula-movil: "20px 12px"
components:
  boton-principal-oscuro:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 20px"
    height: "44px"
  boton-principal-claro:
    backgroundColor: "{colors.petroleo}"
    textColor: "{colors.blanco}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 20px"
    height: "44px"
  boton-secundario-oscuro:
    textColor: "{colors.nieve}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 20px"
    height: "44px"
  boton-secundario-oscuro-hover:
    backgroundColor: "{colors.superficie-noche-alta}"
  boton-secundario-claro:
    textColor: "{colors.tinta}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 20px"
    height: "44px"
  boton-secundario-claro-hover:
    backgroundColor: "{colors.superficie-clara-alta}"
  boton-pagina-hover-oscuro:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.pildora}"
    size: "40px"
  boton-pagina-hover-claro:
    backgroundColor: "{colors.petroleo}"
    textColor: "{colors.blanco}"
    rounded: "{rounded.pildora}"
    size: "40px"
  chip-orden:
    textColor: "{colors.nieve}"
    rounded: "{rounded.pildora}"
    padding: "6px 16px"
    height: "40px"
  chip-orden-activo-oscuro:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.pildora}"
    padding: "6px 16px"
    height: "40px"
  chip-orden-activo-franja-clara:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.pildora}"
    padding: "6px 16px"
    height: "40px"
  franja-clara:
    backgroundColor: "{colors.petroleo}"
    textColor: "{colors.nieve}"
  buscador-franja-clara:
    backgroundColor: "{colors.franja-superficie}"
    textColor: "{colors.nieve}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 44px"
    height: "44px"
  buscador-oscuro:
    backgroundColor: "{colors.superficie-noche}"
    textColor: "{colors.nieve}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 44px"
    height: "44px"
  campo-oscuro:
    backgroundColor: "{colors.superficie-noche}"
    textColor: "{colors.nieve}"
    typography: "{typography.body}"
    rounded: "{rounded.medio}"
    padding: "8px 14px"
    height: "44px"
  campo-claro:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta-clara}"
    typography: "{typography.body}"
    rounded: "{rounded.medio}"
    padding: "8px 14px"
    height: "44px"
  caratula-fila:
    rounded: "{rounded.medio}"
    width: "168px"
  caratula-vacia-clara:
    backgroundColor: "{colors.nieve}"
    textColor: "{colors.tinta-clara-tenue}"
    rounded: "{rounded.medio}"
  caratula-vacia-oscura:
    backgroundColor: "{colors.superficie-noche-alta}"
    textColor: "{colors.tinta-noche-tenue}"
    rounded: "{rounded.medio}"
  caratula-pie:
    textColor: "{colors.nieve}"
    typography: "{typography.pie-caratula}"
  progreso-caratula-pista:
    backgroundColor: "{colors.velo}"
    height: "4px"
  progreso-caratula-relleno:
    backgroundColor: "{colors.salvia}"
    height: "4px"
  progreso-franja-pista:
    backgroundColor: "{colors.pista-progreso-franja}"
    rounded: "{rounded.pildora}"
    height: "4px"
    width: "18rem"
  progreso-franja-relleno:
    backgroundColor: "{colors.franja-acento}"
    height: "4px"
  progreso-pagina-pista-oscura:
    backgroundColor: "{colors.pista-progreso-noche}"
    rounded: "{rounded.pildora}"
    height: "4px"
    width: "18rem"
  progreso-pagina-relleno-oscuro:
    backgroundColor: "{colors.salvia}"
    height: "4px"
  meta-oscuro:
    textColor: "{colors.tinta-noche-tenue}"
    typography: "{typography.meta}"
  meta-franja-clara:
    textColor: "{colors.franja-tinta-tenue}"
    typography: "{typography.meta}"
  meta-claro:
    textColor: "{colors.tinta-clara-tenue}"
    typography: "{typography.meta}"
  chip-genero-oscuro:
    backgroundColor: "{colors.superficie-noche-alta}"
    textColor: "{colors.tinta-noche-tenue}"
    rounded: "{rounded.pildora}"
    padding: "3px 12px"
  chip-coleccion-oscuro:
    textColor: "{colors.acento-noche}"
    rounded: "{rounded.pildora}"
    padding: "5px 14px"
  chip-coleccion-claro:
    textColor: "{colors.petroleo}"
    rounded: "{rounded.pildora}"
    padding: "5px 14px"
  tarjeta-sugerencia-oscuro:
    backgroundColor: "{colors.superficie-noche}"
    rounded: "{rounded.grande}"
    padding: "16px"
  aviso-error-oscuro:
    backgroundColor: "{colors.superficie-noche}"
    textColor: "{colors.nieve}"
    rounded: "{rounded.medio}"
    padding: "12px 16px"
  boton-transmitir-conectado-oscuro:
    textColor: "{colors.acento-noche}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "0 10px"
    height: "40px"
  boton-transmitir-conectado-claro:
    textColor: "{colors.petroleo}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "0 10px"
    height: "40px"
  reproductor-control:
    textColor: "{colors.nieve}"
    rounded: "{rounded.pildora}"
    size: "44px"
  reproductor-boton-grande:
    backgroundColor: "{colors.salvia}"
    textColor: "{colors.tinta}"
    rounded: "{rounded.pildora}"
    size: "84px"
  reproductor-dejar-de-transmitir:
    textColor: "{colors.nieve}"
    typography: "{typography.body}"
    rounded: "{rounded.pildora}"
    padding: "8px 16px"
    height: "44px"
---

# Design System: CastBerry

## Overview

**Creative North Star: "Muro de carátulas"**

CastBerry es la pared de películas de la casa vista de noche: un fondo de tinta azul petróleo casi negro, filas de carátulas que se deslizan a lo ancho y, arriba de todo, la última película a medias con su backdrop y el botón para seguir. La interfaz se retira para que las carátulas sean el único color fuerte de la pantalla; la paleta propia (tinta, petróleo, salvia, nieve) es una sola familia de azules fríos que solo marca dónde está la mirada y cuál es la acción.

La app abre en el tema claro, sea cual sea la preferencia del sistema; el oscuro es una elección de quien la usa, a un toque en la barra superior, y se recuerda en ese navegador. Los dos temas conservan los mismos papeles con otros valores. El claro no es blanco: el fondo es una nieve apagada con salvia, y arriba la barra, el buscador y el hero forman una sola franja de petróleo, donde todo toma valores de noche. De noche la nieve es lo que brilla (marca, acción principal y foco); de día, sobre el fondo apagado, ese papel lo toma el petróleo. Lo que ya viste se marca en salvia, como en el reproductor. La densidad es la de un catálogo de streaming en modo de operación: arriba el buscador, luego el hero «Seguir viendo», filas temáticas («Seguir viendo», «Agregadas recientemente», «Colecciones») y al final la cuadrícula «Todas las películas» con sus chips de orden. La profundidad es casi toda tonal: fundidos sobre imágenes y una sombra suave que solo aparece cuando una carátula se levanta.

No hay logo: la marca es la palabra «CastBerry» compuesta en la display. El reproductor y la pantalla de cast viven siempre sobre negro, fuera del tema.

**Key Characteristics:**
- En claro (el tema de entrada), franja de petróleo arriba y fondo de nieve apagada con salvia; en el oscuro que se elige en la barra, tinta de arriba abajo. Las carátulas aportan el color.
- Hero «Seguir viendo» con backdrop a sangre fundido al color de fondo, título en la display, línea de datos en Meta y barra de progreso.
- Fila activa que se levanta 4px y atenúa a las demás a 0.45; carátula con foco o cursor que sube 4px con una sombra suave.
- Una sola acción principal rellena por pantalla: nieve con texto Tinta (oscuro y franja) o petróleo con texto blanco (fondo claro).
- Lo que ya viste va en salvia; en la franja del claro, en el acento de franja.
- Display Bricolage Grotesque en 700–800 para marca y títulos; cuerpo en la sans del sistema, sin mayúsculas en ningún texto.

## Colors

Paleta fija de cuatro azules petróleo (tinta, petróleo, salvia, nieve) sobre neutros teñidos de la misma familia, con dos juegos de valores: el claro, que es el de entrada, y el oscuro, que se activa con `data-theme='dark'`. En claro, la franja superior reasigna los mismos papeles a valores de noche sobre petróleo.

### Primary
- **Nieve** (nieve): en oscuro es la acción principal (con texto Tinta), la marca, el anillo de foco, el chip de orden activo y el hover de los botones de página. En ambos temas es la tinta sobre velo y sobre negro. En claro es la superficie de campos, tarjetas y carátulas vacías sobre el fondo apagado y, dentro de la franja, vuelve a ser acción principal, marca, foco y texto.
- **Petróleo** (petroleo): en claro es el fondo de la franja (barra superior, buscador y hero del catálogo, cabecera de la ficha) y, sobre el fondo apagado, toma el papel de la nieve de noche: acción principal con texto blanco, foco, texto de acento y hover de los botones de página. En oscuro es el resplandor del reproductor y de la luz ambiental del catálogo.

### Secondary
- **Salvia** (salvia): lo que ya viste y lo secundario. Es el relleno de las barras de progreso sobre el velo de las carátulas en ambos temas (5.6:1) y, en oscuro, el del hero y la ficha; al 30% con nieve es el fondo apagado del claro. Además: selección de texto (con texto Tinta), la etiqueta de subtítulo hecho para el mismo archivo, la luz ambiental del catálogo en claro y, en el reproductor, el acento de la barra de posición, el foco, el hover de los controles y el botón grande de reproducir (con texto Tinta).

### Tertiary
- **Acento de Noche** (acento-noche): texto de acento solo en oscuro: título de la fila activa, enlaces, estrella de la calificación, chips de colección, el botón Transmitir conectado y la marca de la pantalla de entrada. Existe porque la salvia no llega a contraste de texto sobre tinta.

### Neutral
- **Tinta** (tinta): fondo de página en oscuro y texto sobre el fondo apagado, la nieve o la salvia en ambos temas.
- **Nieve Salvia Apagada** (nieve-salvia-apagada): el fondo de página en claro, nieve con 30% de salvia, para que el claro no deslumbre. Encima, campos, tarjetas y carátulas vacías van en nieve.
- **Superficie Noche** (superficie-noche) y **Superficie Noche Alta** (superficie-noche-alta): campos y tarjetas en oscuro; la alta es el «estante vacío» de las carátulas sin imagen, los esqueletos de carga, los chips de género, las filas de subtítulos y el hover de los botones secundarios. Están separadas del fondo (≈1.3:1 y ≈1.8:1) para que un estante vacío no se lea como hueco.
- **Superficie Clara Alta** (superficie-clara-alta): en claro, el tono de los esqueletos, los chips de género, las filas de subtítulos y el hover de los botones secundarios. **Blanco** (blanco): solo el texto sobre petróleo fuera de la franja; nunca un fondo.
- **Carátula vacía**: en claro es nieve sobre el fondo apagado, para que se lea como portada y no como hueco; vale también para las miniaturas de las sugerencias de colección. Dentro de la franja (el marcador de carátula de la ficha) es la Franja Superficie, para que el título en tinta tenue de la franja dé 4.9:1. En oscuro ese papel lo hace la Superficie Noche Alta.
- **Franja** (franja-superficie, franja-superficie-alta, franja-tinta-tenue, franja-linea, franja-borde-control, franja-acento, pista-progreso-franja): los valores que toman superficie, superficie alta, tinta tenue (5.8:1), divisor, borde de control, texto de acento y la pista de progreso dentro de la franja petróleo del claro. El acento de franja es también el progreso del hero en claro (5:1).
- **Pista de Progreso Noche** (pista-progreso-noche): el riel de las barras de progreso de página en oscuro (hero y ficha), nieve al 10%. En claro esas barras viven en la franja, sobre su propia pista.
- **Línea Noche** / **Línea Clara**: divisores (borde inferior de la barra superior, separadores de secciones en la ficha, contorno de las tarjetas de sugerencia).
- **Borde de Control Noche** / **Borde de Control Claro**: contorno de campos, chips de orden, botones secundarios e insignia de clasificación; ambos superan 3:1 contra su fondo.
- **Tinta Clara** (tinta-clara): texto principal sobre superficie en claro. **Tinta Noche Tenue** / **Tinta Clara Tenue** (5:1 sobre el fondo apagado): texto secundario (la línea Meta, el año bajo la carátula, pistas, iconos del buscador, el nombre de usuario).
- **Error Noche** / **Error Claro**: errores, borde del aviso de error y el botón de peligro.
- **Velo** (velo): base de texto sobre imágenes y pista de la barra de progreso de las carátulas, igual en ambos temas. **Tinta sobre Velo Tenue** (tinta-sobre-velo-tenue): texto secundario sobre negro o velo.
- **Fondo Reproductor** (fondo-reproductor): base del resplandor de la pantalla de cast y del aviso de reproducción, mezclado con Petróleo.

### Named Rules
**La Regla de las Carátulas al Frente.** Las carátulas y los backdrops son el único color fuerte; la paleta propia se reserva para foco, acento, progreso y la acción principal. Ninguna superficie grande se rellena de petróleo, salvia o nieve fuera de su papel de fondo o de la franja.

**La Regla de la Nieve de Noche.** En oscuro, lo que más brilla es la nieve: acción principal, marca y foco, siempre con texto Tinta. En claro, sobre el fondo apagado, ese papel es del petróleo con texto blanco; dentro de la franja vuelve a ser de la nieve. La salvia nunca hace de acción principal fuera del reproductor.

**La Regla de lo Ya Visto.** Lo que ya viste es salvia: sobre el velo de las carátulas en ambos temas, y sobre la página en oscuro. En claro, las barras de página (hero y ficha) viven dentro de la franja y van en el acento de franja; la salvia no llega a 3:1 contra el fondo apagado. Siempre por `--color-progress` (carátula) y `--color-progress-page` (página), sobre `--color-progress-track`.

**La Regla de la Franja.** En claro, lo de arriba de la página (la barra, el buscador con los chips de orden y el hero del catálogo; en la ficha, la cabecera con la película) es una franja de petróleo continua con la clase `.band`, que reasigna los mismos tokens a valores de noche; debajo, las filas, la cuadrícula y el cuerpo de la ficha van sobre el fondo apagado. Los componentes no cambian: cambia el ámbito. En oscuro la franja es el mismo fondo de la página.

**La Regla del Tema Elegido.** El claro es el tema por defecto en `:root`; el oscuro vive solo bajo `:root[data-theme='dark']` y lo enciende el botón de tema de la barra (se guarda en `localStorage`, clave `castberry-theme`, y se aplica antes de pintar). El sistema operativo no decide el tema: nunca se agregan media queries de `prefers-color-scheme`.

**La Regla de los Tokens.** Componentes y vistas usan solo las variables de tema (`--color-*`, `--palette-*`, `--radius-*`, `--type-*`). Las únicas excepciones con color literal son el reproductor y la pantalla de cast (siempre sobre negro), los velos y sombras sobre imágenes y la «G» de Google del botón de entrada, que conserva sus cuatro colores de marca sobre un disco blanco en ambos temas.

## Typography

**Display Font:** Bricolage Grotesque Variable (auto-hospedada con @fontsource, con la sans del sistema como respaldo)
**Body Font:** system-ui (-apple-system, Segoe UI, Roboto, sans-serif)
**Label/Mono Font:** ui-monospace, solo para la pista de teclado «/» del buscador

**Character:** Una grotesca con carácter, apretada y pesada, lleva la marca y los títulos; la sans del sistema hace todo lo demás sin llamar la atención. Cada paso de la escala cambia tamaño y peso a la vez; el contraste es de peso, no de familias decorativas.

### Hierarchy
- **Marca** (800, 1.6rem en la barra y 2.75rem en el login, interlineado 1, -0.04em): la palabra «CastBerry»; en el color de marca, en la barra y en la pantalla de entrada. Es el logo.
- **Display** (800, clamp de 2rem a 3.5rem, 1.02, -0.03em, con balance de líneas): el título de la película del hero y, un paso más grande (clamp de 2.25rem a 4.5rem), el de la ficha. Las páginas de colección y de sugerencias usan la misma voz (800, -0.03em) a su propia escala.
- **Title Large** (700, 22px/28px, -0.01em): títulos de página y de aviso, nombres de saga en las sugerencias, títulos sobre negro en el reproductor y la pantalla de cast, y el título del marcador de carátula de la ficha.
- **Title Medium** (700, 1.25rem/1.6rem, -0.01em): títulos de fila y de sección («Agregadas recientemente», «Todas las películas», y en la ficha «Colecciones» y «Subtítulos»); 1.125rem en el celular. También el título de una carátula sin imagen, a 1rem.
- **Meta** (600, 1rem/1.5rem, cifras tabulares, tinta tenue): la línea de datos bajo un título, en el hero (año y tiempo restante) y en la ficha (año, duración, clasificación, calificación).
- **Body** (400, 16px/24px): texto corrido, botones (en 600) y campos; la sinopsis va a 1.6 de interlineado con un máximo de 65ch. Los campos nunca bajan de 16px para que iOS no haga zoom.
- **Pie de carátula** (650, 0.95rem, 1.3): título bajo cada carátula, cortado a dos líneas; debajo, el año o el tiempo restante en tinta tenue a 0.85rem con cifras tabulares.

### Named Rules
**La Regla de la Marca Tipográfica.** La marca es texto en la display a 800; no se dibuja ni se sustituye por un icono.

**La Regla Sin Mayúsculas.** Ningún texto va en mayúsculas ni con tracking de etiqueta: ni encabezados de sección, ni metadatos, ni etiquetas sobre un título. Un encabezado es un Title Medium escrito como oración; los datos van en Meta.

## Layout

Página a todo lo ancho con un margen lateral fluido (margen-pagina: 16px en el celular, hasta 56px en escritorio) que comparten la barra, el buscador, el hero, las filas y la cuadrícula. La barra superior es pegajosa, de 64px de alto, con un divisor inferior.

El catálogo sigue la topología de las apps de streaming: arriba, dentro de la franja en claro (con 24px de aire arriba y 28px abajo), la barra de herramientas con el buscador en píldora (hasta 420px de ancho, a todo lo ancho en el celular) y, a su derecha, los chips de orden (se bajan de línea si no caben); luego el hero, las filas horizontales (separadas 36px) y, al final, «Todas las películas» en cuadrícula de columnas de mínimo 150px (104px en el celular). Al buscar desaparecen el hero y las filas, y la página es la cuadrícula de resultados; mientras llega la búsqueda nueva, la anterior se atenúa a 0.55. Sin hero, una luz ambiental se desvanece desde la esquina superior izquierda (petróleo en oscuro, salvia al 22% en claro) para que el catálogo no arranque sobre un plano liso.

El hero mide de 18rem a 28rem de alto: el backdrop va a sangre y se funde con el color de fondo hacia abajo, arriba y la izquierda; encima, abajo a la izquierda, carátula de 132px y la columna de texto (hasta 44rem). En el celular el backdrop queda como banda superior de 14rem, la carátula se oculta y los botones se reparten el ancho.

Cada fila desliza en horizontal con scroll-snap y sin barra visible; el carril tiene relleno vertical para que el anillo de foco y la carátula levantada no se recorten. Con mouse aparecen dos botones de página redondos de 40px junto al título; con touch se desliza. En el celular caben dos carátulas y media, y la tercera asoma para indicar que la fila continúa.

La ficha se parte en dos. La cabecera (franja en claro) lleva el backdrop a sangre, fundido hacia abajo y hacia la izquierda con el fondo de la franja; encima, alineadas abajo y hasta 1120px, la carátula (de 160px a 240px) y la columna del título con Meta, géneros, acciones de reproducir y el progreso. Debajo, sobre el fondo de la página, el cuerpo (sinopsis, créditos, «Colecciones», «Subtítulos» y enlaces) va sangrado hasta la columna del título; las secciones se separan con un divisor fino. Debajo de 700px todo se apila y se centra, y el sangrado desaparece.

Objetivos táctiles de 44px en botones, campos, chips (con puntero grueso) y controles del reproductor.

## Elevation & Depth

El sistema es tonal con una sola elevación por estado. La jerarquía sale de los pasos de superficie (fondo, superficie, superficie alta), de los divisores finos y del movimiento de la fila activa. Sobre imágenes, la profundidad son fundidos: el backdrop del hero y de la ficha se disuelve en el color de fondo de su franja, la barra superior es fondo al 95% con desenfoque de 12px, y los controles del reproductor llevan degradados negros. La sombra aparece solo bajo una carátula que se levanta (foco o cursor) y bajo la carátula del hero y la de la ficha, que siempre flotan sobre su backdrop. Ambas son suaves, difusas y hacia abajo, nunca un desplazamiento duro.

### Shadow Vocabulary
- **Carátula levantada** (`box-shadow: 0 14px 28px -12px rgb(0 0 0 / 0.55)`): carátula con foco o cursor, junto con el desplazamiento de -4px.
- **Carátula del hero** (`box-shadow: 0 18px 36px -16px rgb(0 0 0 / 0.6)`): la carátula de 132px sobre el backdrop del hero y la carátula de la cabecera de la ficha.
- **Filo de carátula** (`box-shadow: inset 0 0 0 1px` tinta principal al 10%): contorno interior permanente que separa del fondo las carátulas claras o sin imagen. Es un borde, no elevación.
- **Anillo de miniatura** (`box-shadow: 0 0 0 2px` color de superficie): separa las miniaturas superpuestas en las sugerencias de colección. Es un contorno, no elevación.

### Named Rules
**La Regla del Velo.** El texto sobre una imagen siempre va encima de un velo casi negro o de un fundido al color de fondo, nunca directo sobre la foto.

**La Regla de la Fila Activa.** La profundidad del muro es movimiento: la fila con foco o cursor sube 4px y su título toma el acento; las carátulas de las demás bajan a 0.45 de opacidad (sus títulos no, para que se sigan leyendo). Dentro de ella, solo la carátula elegida se levanta con sombra.

## Shapes

Esquinas suaves y consistentes: 8px (medio) en carátulas, campos, avisos y filas de subtítulos; 6px (pequeño) en esqueletos de texto, la insignia de clasificación, la pista de teclado y las miniaturas apiladas; 14px (grande) en las tarjetas de sugerencia y los avisos leves del reproductor. Todo lo que es acción, búsqueda o etiqueta (botones, buscador, chips de orden, género y colección, etiquetas de subtítulo, botones de página, controles del reproductor) es píldora, igual que las barras de progreso sobre la página; la de la carátula va recta, pegada a su borde inferior y recortada por sus esquinas. Las carátulas recortan la imagen a 2:3 y llevan un filo interior de 1px; los bordes de 1px visibles son para controles y divisores.

## Components

### Buttons
Píldoras firmes y discretas; la principal es la única con relleno.
- **Shape:** píldora completa, alto mínimo de 44px, texto en 600.
- **Primary:** una por pantalla (Reanudar, Reproducir, Entrar con Google). Nieve con texto Tinta en oscuro y dentro de la franja; petróleo con texto blanco sobre el fondo claro. Hover: brillo +8%. En la ficha, Reproducir y Reanudar van a 1.05rem con más relleno lateral.
- **Secondary:** fondo transparente, borde de control y tinta principal; hover rellena con la superficie alta. En el hero lleva el fondo al 60% para leerse sobre el backdrop.
- **Danger:** como el secundario, con borde y texto en el color de error (borrar una colección).
- **Texto (Salir):** sin borde, en tinta tenue a 0.9rem, 40px de alto (44px en touch); lo menos usado de la barra no compite con Transmitir.
- **Botón de página:** círculo de 40px con borde de control; al pasar el cursor se rellena con el color primario y su texto; deshabilitado a 0.35.
- **Disabled:** 0.6 de opacidad.
- **Entrar con Google:** la acción principal de la pantalla de entrada, a todo el ancho de su columna (hasta 340px), con la «G» de Google de 24px sobre un disco blanco. Queda deshabilitada mientras carga Firebase y dice «Entrando…» mientras se valida la cuenta.

### Chips
- **Orden:** píldoras de 40px (44px con puntero grueso) con borde de control y texto en 600 a 0.9rem: tres a la vista (título, agregadas, calificación) y «Más», un selector con la misma forma y una flecha, que guarda los demás órdenes. El activo (o «Más» con un orden suyo elegido) se rellena con el color primario y su texto: nieve con texto Tinta en ambos temas, porque en claro viven dentro de la franja. El orden va en la URL.
- **Género:** píldora en superficie alta con tinta tenue, 0.85rem en 600, no interactiva.
- **Colección:** píldora con borde y texto de acento, enlazada; hover rellena con la superficie alta; más alta en touch.
- **Clasificación:** insignia de esquina pequeña (6px) con borde de control, a 0.85rem dentro de la línea Meta.
- **Etiquetas de subtítulo:** píldoras a 0.75rem en 600 con borde de control («Para sordos», «Traducción automática»); la de «Coincide con tu archivo» va en salvia con texto Tinta.

### Cards / Containers
- **Tarjeta de sugerencia:** superficie con divisor de 1px, esquinas de 14px y 16px de relleno; a la izquierda, miniaturas de 64px superpuestas con el anillo de miniatura, sobre la carátula vacía si no tienen imagen.
- **Aviso de error:** superficie, borde de 1px en color de error, esquinas de 8px; texto y acción de reintento en línea.
- **Fila de subtítulo:** superficie alta, esquinas de 8px, el idioma en 650 y la acción a la derecha (quitar: botón redondo de 44px que se pinta de error con el cursor).
- **Shadow Strategy:** ninguna en contenedores (ver Elevation & Depth).

### Inputs / Fields
- **Style:** superficie, borde de 1px de control, esquinas de 8px, alto de 44px, texto de 16px; placeholder en tinta tenue; cursor y acento nativo en el color de acento.
- **Buscador:** el mismo campo en píldora (en claro, dentro de la franja: superficie de franja con texto nieve y borde de control de franja), con la lupa a la izquierda en tinta tenue. Vacío, a la derecha muestra la pista «/» (monoespaciada, borde de control, esquinas de 6px; solo con teclado físico y oculta al enfocar); con texto, un botón redondo de 36px para borrar. «/» enfoca el buscador desde cualquier parte del catálogo y Escape lo vacía.
- **Focus:** contorno de 2px en el color de foco (nieve en oscuro, petróleo en claro) separado 1px.
- **Error:** el mensaje va debajo en el color de error; el campo no cambia.

### Navigation
- **Barra superior:** pegajosa, fondo al 95% con desenfoque, divisor inferior; en claro es el arranque de la franja petróleo, con la marca en nieve. Marca a la izquierda en el color de marca; a la derecha el botón Transmitir, el botón de tema, el nombre de usuario en tinta tenue y «Salir» como botón de texto. Del nombre se muestra solo lo que va antes de la @, recortado con elipsis hasta 14rem y con el correo completo en el title; es lo único que cede espacio, y en ≤600px se oculta.
- **Botón de tema:** icono de luna en un círculo de 40px (44px con puntero grueso), sin borde, en tinta tenue; con el cursor se rellena con la superficie alta y el icono pasa a la tinta principal. Su nombre es «Tema oscuro» y está presionado cuando el oscuro está activo: entonces la luna va rellena y en el color de acento. Cambia el tema y el color de la barra del navegador (petróleo en claro, tinta en oscuro).
- **Botón Transmitir:** píldora de 40px (44px en touch) con borde de control y el icono de cast; solo existe si hay un Chromecast en la red. Mientras conecta, el icono parpadea (sin animación con movimiento reducido). Conectado, borde y texto pasan al color de acento, la pantalla del icono se rellena y al lado aparece el nombre del dispositivo en 600 (solo el icono en ≤600px). Con una sesión abierta, la acción principal dice «Reproducir en ‹dispositivo›» o «Reanudar en ‹dispositivo›».
- **Enlaces de texto:** color de acento, subrayado de 1px separado 0.2em; los externos (IMDb, TMDB) subrayan solo el texto y no el icono. El enlace «Datos e imágenes de TMDB» es permanente en la ficha.

### Carátula (componente distintivo)
La única carátula del producto, en dos disposiciones: fila (168px fijos, con scroll-snap) y cuadrícula (la columna que toque). Imagen 2:3 con esquinas de 8px sobre la carátula vacía, con el filo interior de 1px; el título (dos líneas) y el año, el conteo de una colección o el tiempo restante van siempre debajo, nunca sobre la imagen. Con foco o cursor la carátula sube 4px con la sombra de carátula levantada, en 0.28s con salida suave; el foco suma un anillo de 3px separado 3px. Si hay progreso, una barra de 4px en salvia sobre el velo va pegada al borde inferior. Sin imagen, el fondo es la carátula vacía (nieve salvia en claro, superficie alta en oscuro) y el título aparece centrado en la voz de título a 1rem, en tinta tenue. Con movimiento reducido no se desplaza. En el celular su ancho se calcula para que quepan dos y media por fila.

### Barras de progreso
Una barra de 4px que dice cuánto se vio, en tres lugares: al pie de la carátula (recta, salvia sobre velo), en el hero (píldora de hasta 18rem) y en la cabecera de la ficha (píldora de 10rem, junto al tiempo restante y el enlace «Quitar de «Seguir viendo»»). Las dos de página llevan el progreso de página sobre su pista: salvia en oscuro y acento de franja en claro, porque ambas viven en la franja.

### Hero «Seguir viendo»
La película a medias más reciente, dentro de la franja en claro (su backdrop se funde con el petróleo): backdrop, carátula de 132px, título en display, la línea de datos en Meta (año y tiempo restante), la barra de progreso de página y dos acciones: «Reanudar» (principal, abre el reproductor; «Reanudar en ‹dispositivo›» con una sesión de cast abierta) y «Ver ficha» (secundario). Las demás películas a medias siguen en la fila «Seguir viendo», cuyas carátulas abren la ficha.

### Pantalla de entrada
La única forma de entrar es con Google. Columna centrada de hasta 340px: la marca a 2.75rem y debajo el botón Entrar con Google; no hay campos. Los errores van debajo en el color de error, centrados, y nombran la causa y qué hacer.

### Esqueletos de carga
Bloques en el tono de la superficie alta con las mismas proporciones que el contenido (carátula 2:3, líneas de texto), con un pulso suave de opacidad que se desactiva con movimiento reducido.

### Reproductor y pantalla de cast
Siempre sobre negro, en ambos temas; adentro, los botones compartidos toman valores oscuros (acción principal en nieve con texto Tinta, foco en salvia) aunque la app esté en claro. Nunca se usan los controles nativos del navegador.

- **Barra de controles:** la misma en el video local y en el Chromecast. Va abajo, sobre un degradado de negro al 85% a transparente. Arriba, la barra de posición a todo lo ancho: riel de 4px en blanco al 25%, recorrido y perilla de 14px en salvia (la perilla crece a 1.25 con cursor o foco). Con el dedo el área táctil sube a 44px y la perilla a 22px. Debajo, controles circulares de 44px en nieve: play/pausa (con fondo blanco al 12%), retroceder y adelantar 10 s, el tiempo en cifras tabulares y tinta tenue, silenciar y el volumen (se oculta en ≤600px). Al extremo derecho va lo propio de cada modo: pantalla completa en local, «Dejar de transmitir» en cast.
- **Barra superior del reproductor:** volver, subtítulos y cast, circulares de 44px sobre negro, con el título en la voz de título, sobre un degradado negro desde arriba. Lo activo se pinta de salvia.
- **Ocultar en reposo:** en reproducción, barras y cursor se ocultan a los 3 s sin actividad; en pausa nunca.
- **Botón grande de reproducir:** círculo salvia de 84px con el triángulo en Tinta, centrado, solo en pausa.
- **Aviso de reproducción:** panel centrado sobre el resplandor de petróleo (22%) con icono salvia, título en la voz de título y mensaje en tinta tenue, más las acciones. Los avisos leves van en una píldora de esquinas grandes sobre el velo.
- **Pantalla de cast:** resplandor radial de petróleo (38%) sobre el fondo del reproductor que se funde a negro; al centro el icono salvia, «Transmitiendo en» y el nombre del dispositivo. «Dejar de transmitir» es una píldora de 44px con borde blanco al 35% (salvia con cursor).

### Named Rules
**La Regla de una Sola Carátula.** Toda carátula del producto es la misma pieza, en fila o en cuadrícula, con el título debajo. No se crea otra tarjeta de película.

**La Regla de los Mismos Controles.** El video local y el Chromecast comparten la misma barra; solo cambia el botón del extremo. Un control nuevo se agrega a la barra compartida, no a un modo solo.

## Do's and Don'ts

### Do:
- **Do** usar exactamente una acción principal rellena por pantalla: nieve con texto Tinta en oscuro y en la franja, petróleo con texto blanco sobre el fondo claro.
- **Do** poner dentro de `.band` todo lo nuevo que vaya arriba de la página junto a la barra (cromo, buscador, cabecera de una película), para que en claro siga la franja petróleo sin colores propios.
- **Do** pasar el acento, el foco, la marca y el progreso por sus variables (`--color-accent-text`, `--color-focus`, `--color-brand`, `--color-progress`, `--color-progress-page`, `--color-progress-track`, `--color-poster-empty`), no por la paleta directa.
- **Do** poner texto sobre carátulas y backdrops solo encima del velo o de un fundido al color de fondo.
- **Do** mantener las carátulas en 2:3 con esquinas de 8px, el filo de 1px y el título debajo.
- **Do** escribir encabezados como oración en Title Medium y la línea de datos bajo un título en Meta.
- **Do** limitar la sombra a la carátula que se levanta y a las carátulas del hero y de la ficha, siempre suave y hacia abajo.
- **Do** contornear campos, chips y botones secundarios con el borde de control (≥3:1), y dejar las líneas finas para los divisores.
- **Do** cambiar de tema solo con `data-theme` en la raíz; todo valor nuevo de color se define para el claro en `:root` y para el oscuro en `:root[data-theme='dark']`.
- **Do** respetar `prefers-reduced-motion`: sin desplazamiento de fila ni de carátula, sin pulso de esqueletos ni parpadeo del botón Transmitir.
- **Do** dar a cada control táctil al menos 44px.

### Don't:
- **Don't** usar la salvia como acción principal, foco o texto de acento fuera del reproductor; en oscuro el texto de acento es el acento de noche.
- **Don't** poner el progreso en salvia sobre el fondo claro: en claro las barras de página van en la franja, en el acento de franja.
- **Don't** usar blanco puro ni nieve lisa como fondo de página en claro: el fondo es la nieve apagada con salvia, y la nieve queda para superficies encima.
- **Don't** usar mayúsculas, tracking de etiqueta ni etiquetas sobre un título (kicker o eyebrow).
- **Don't** agregar media queries de `prefers-color-scheme` ni dejar que el sistema elija el tema: la app abre en claro y el oscuro es una elección.
- **Don't** usar sombras de tarjeta, de botón o con desplazamiento duro; la profundidad es tonal, por fundido o por el movimiento de la fila activa.
- **Don't** escribir colores fijos en componentes fuera del reproductor, la pantalla de cast, los velos y sombras sobre imágenes y la «G» de Google.
- **Don't** dibujar un logo ni un isotipo: la marca es la palabra «CastBerry» en la display.
- **Don't** rellenar superficies grandes con petróleo, salvia o nieve fuera de su papel de fondo; el color fuerte es de las carátulas.
- **Don't** quitar el enlace «Datos e imágenes de TMDB» de la ficha.
- **Don't** usar los controles nativos del navegador (`<video controls>`) en el reproductor.
