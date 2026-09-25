---
name: Energy Management
description: Consola de centro de control de energía; fondo casi negro, paneles planos y menta reservada a la acción y al estado normal.
colors:
  app: "#0a0a0a"
  canvas: "#0f0f0f"
  panel: "#141414"
  raised: "#1a1a1a"
  hover: "#1f1f1f"
  line: "#27272a"
  line-strong: "#3f3f46"
  ink: "#ffffff"
  ink-2: "#a1a1aa"
  ink-3: "#71717a"
  mint: "#08ddbc"
  mint-bright: "#17ffdb"
  mint-deep: "#06c4a7"
  amber: "#f59e0b"
  red: "#ef4444"
  violet: "#8b5cf6"
  blue: "#3b82f6"
typography:
  display:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "28px"
    fontWeight: 600
    lineHeight: 1.25
    letterSpacing: "normal"
    fontFeature: "tnum"
  headline:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "20px"
    fontWeight: 600
    lineHeight: "28px"
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "16px"
    fontWeight: 600
    lineHeight: "24px"
    letterSpacing: "normal"
  body:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: "20px"
    letterSpacing: "normal"
  label:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    lineHeight: "16px"
    letterSpacing: "0.06em"
  caption:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: "16px"
    letterSpacing: "normal"
rounded:
  badge: "4px"
  base: "6px"
  md: "8px"
  lg: "10px"
  full: "9999px"
spacing:
  "1": "4px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "20px"
  "6": "24px"
  "8": "32px"
components:
  button-primary:
    backgroundColor: "{colors.mint}"
    textColor: "{colors.app}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0 16px"
    height: "40px"
  button-primary-hover:
    backgroundColor: "{colors.mint-bright}"
  button-primary-active:
    backgroundColor: "{colors.mint-deep}"
  button-secondary:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0 16px"
    height: "40px"
  button-secondary-hover:
    backgroundColor: "{colors.hover}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0 16px"
    height: "40px"
  button-ghost-hover:
    backgroundColor: "{colors.hover}"
    textColor: "{colors.ink}"
  field:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0 12px"
    height: "40px"
  panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
  panel-title:
    textColor: "{colors.ink-2}"
    typography: "{typography.label}"
    padding: "8px 16px"
    height: "44px"
  kpi-tile:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.display}"
    rounded: "{rounded.md}"
    padding: "12px 16px"
    height: "104px"
  badge:
    typography: "{typography.caption}"
    rounded: "{rounded.badge}"
    padding: "0 6px"
    height: "20px"
  segmented-active:
    backgroundColor: "{colors.mint}"
    textColor: "{colors.app}"
    typography: "{typography.caption}"
    rounded: "{rounded.base}"
    padding: "0 12px"
    height: "28px"
  segmented-inactive:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    typography: "{typography.caption}"
    rounded: "{rounded.base}"
    padding: "0 12px"
    height: "28px"
  nav-item:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0 12px"
    height: "40px"
  nav-item-active:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
  chip-analysis:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink-2}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: "0 12px"
    height: "32px"
  network-cell:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.md}"
    padding: "8px 10px"
  toast:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "12px 16px"
    width: "360px"
  modal:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    width: "560px"
---

# Design System: Energy Management

## Overview

**Creative North Star: "El panel de operador"**

Energy Management es una consola de centro de control, no un dashboard de tarjetas iguales. La pantalla vigila 12 puntos de una red eléctrica y su primera fila útil es una franja densa de celdas de estado con las críticas primero: la prioridad se lee antes que cualquier número. Todo lo demás (tiles KPI, tablas, gráficas, investigación) es la sala de control alrededor de esa franja: superficies planas, bordes de 1 px, texto blanco sobre casi negro y una sola luz para lo que importa.

El material es plano y oscuro. No hay gradientes, ni vidrio, ni sombras en reposo: la profundidad se construye con tres tonos de superficie (`canvas`, `panel`, `raised`) y un borde hairline. La menta es la única voz de acción y de "todo en orden"; el ámbar y el rojo marcan alerta y crítico; el violeta señala calidad de datos. El estado nunca es solo un color: siempre lleva un glifo lucide de un solo trazo (1,5 px) al lado del texto o del número.

La tipografía es una sola familia, Inter, con base 14 px y una escala corta (12 / 14 / 16 / 20 / 28). Cada columna de datos usa numerales tabulares para que las cifras se alineen y se lean vivas al cambiar. El movimiento es escaso: transiciones de 150 ms para hover y foco, y una única coreografía firma, el encendido secuencial de las celdas cuando termina un análisis.

**Key Characteristics:**
- Fondo casi negro en dos tonos (`app` para la barra lateral, `canvas` para el contenido) y paneles planos `panel` con borde 1 px `line`.
- Menta reservada a acción primaria, navegación activa, selección y estado normal; halo solo para lo crítico o lo que acaba de cambiar.
- Estado como glifo más color: badges de 12 px con fondo del color al 15 % e icono lucide de 12 px.
- Inter única, base 14 px, numerales tabulares (`.tnum`) en toda cifra.
- Densidad de operación: filas de tabla de 44 px, celdas de red de 76 px, tiles KPI de 104 px, rejillas con separación de 16 px.
- Superficies del navegador temadas: selección, caret, scrollbar y anillo de foco en menta.

## Colors

Paleta oscura de un solo acento: una escala de grises casi neutros (matiz zinc) para superficies y texto, menta como acento primario y cuatro colores semánticos que solo aparecen cuando el dato lo exige.

### Primary
- **Menta** (`mint`): botón primario (texto `app`), enlace activo de navegación (barra de 3 px e icono), opción activa del filtro segmentado, serie de las gráficas, barra de confianza, punto de estado "normal" en la franja, badge OK / Abierta, selección de texto (al 35 %), caret, `accent-color` y anillo de foco. Es la única voz de acción y de "todo en orden".
- **Menta brillante** (`mint-bright`): hover del botón primario y de los enlaces menta (`Ver todas las anomalías →`).
- **Menta profunda** (`mint-deep`): estado `:active` del botón primario. Solo ahí.

### Secondary
- **Ámbar** (`amber`): alerta. Borde de la celda en estado ALERT, badge Alerta / Media / Explicable, aviso de sesión expirada y de anomalía superada (borde al 40 %, fondo al 10 %), línea de eventos en las gráficas, banner de viewport < 1280 px (fondo sólido, texto `app`).
- **Rojo** (`red`): crítico. Único estado que recibe halo en reposo (`glow-red`), badge Crítico / Alta / Anomalía real, tile "Alta prioridad" cuando > 0, variaciones ≥ 20 %, puntos atípicos y ventana de la anomalía en las gráficas (al 14 %), errores de red, límites eléctricos (al 50 %, discontinuo).

### Tertiary
- **Violeta** (`violet`): tipo de anomalía "Calidad de datos". No aparece en ningún otro lugar.
- **Azul** (`blue`): estado transitorio "En curso": punto del chip de análisis, icono girando del stepper, badge En investigación. Nunca acción.

### Neutral
- **Negro app** (`app`): fondo de `<html>`, barra lateral, texto sobre menta y sobre ámbar.
- **Lienzo** (`canvas`): fondo del contenido, del login y del header (al 95 % con `backdrop-blur-sm`).
- **Panel** (`panel`): superficie de paneles, tiles, celdas de red, chip de análisis, contenedor del segmentado y tooltip de gráficas.
- **Elevado** (`raised`): campos, botón secundario, enlace activo de navegación, skeletons, toast, modal, caja de evento relacionado, `<option>` de selects.
- **Hover** (`hover`): fondo de hover de filas, botones fantasma, enlaces de navegación y celdas de red; también rejilla de las gráficas.
- **Línea** (`line`): borde de paneles y celdas normales, divisores de tabla y de header, ejes X de las gráficas.
- **Línea fuerte** (`line-strong`): borde de campos, botón secundario, toast y modal; pulgar del scrollbar.
- **Tinta** (`ink`): texto principal, cifras, ids de medidor, valor de tiles.
- **Tinta 2** (`ink-2`): texto secundario, etiquetas de tiles, títulos de panel y cabeceras de tabla, ticks de las gráficas, badge Baja / Falso positivo / Resuelta / Descartada (Descartada usa `ink-3`).
- **Tinta 3** (`ink-3`): placeholders, separadores `·` y `—`, glifo del estado vacío, icono de búsqueda, eje de gráficas. Tono terciario: no se usa para cifras ni texto de lectura.

### Named Rules
**La regla de una sola luz.** Solo lo crítico o lo que acaba de cambiar recibe halo (`0 0 0 1px` del color + `0 0 16px` al 25 %). La alerta es un borde ámbar sobre el plano; lo normal es borde `line` y un glifo menta. Si más de una celda por estado brilla, la franja deja de decir qué atender primero.

**La regla de la menta escasa.** La menta aparece solo en lo que se puede pulsar o en lo que está bien. Nunca es color decorativo, de fondo ni de gráfica secundaria; el chip de análisis se pone menta solo cuando el análisis se completó.

**La regla del tinte al 15 %.** Todo badge se construye con `tint(color, 0.15)` de fondo y el color puro de texto y glifo. No existen badges sólidos ni badges de borde.

## Typography

**Display Font:** Inter (con ui-sans-serif, system-ui, Segoe UI, Roboto)
**Body Font:** Inter (misma familia; pesos 400, 500, 600, 700 cargados desde Google Fonts, 700 sin uso en el build)
**Label/Mono Font:** ninguna; los numerales tabulares (`font-variant-numeric: tabular-nums`, clase `.tnum`) sustituyen a la monoespaciada.

**Character:** Una sola familia sans neutra, pesos 400 a 600, escala corta y sin ornamento. La jerarquía se construye con tamaño, peso y color (`ink` / `ink-2`), no con fuentes distintas. Las cifras son el material protagonista: siempre tabulares, con separadores es-CO y unidad fuera del número cuando es posible.

### Hierarchy
- **Display** (600, 28 px, `leading-tight` 1.25, tabular): valor principal de los tiles KPI. Variante compacta de 20 px (`compact`) para fechas largas. Nunca aparece en texto corrido.
- **Headline** (600, 20 px / 28 px, `tracking-tight`): título de página en el header («Dashboard», «Investigación · M-109»), marca en el login, id del medidor en la cabecera de investigación.
- **Title** (600, 16 px / 24 px): título del modal, títulos de las siete secciones de investigación, marca «Energy Management» en la barra lateral, acción recomendada (500).
- **Body** (400, 14 px / 20 px): base de la aplicación; celdas de tabla, párrafos (máx. 75 ch en la explicación, `leading-relaxed`), campos, botones (600), enlaces de navegación (500).
- **Label** (500, 12 px / 16 px, 0.06em, MAYÚSCULAS, `ink-2`): títulos de panel (`.panel-title`) y cabeceras de tabla (`th`). Es el título real del contenedor, no un kicker sobre otro título.
- **Caption** (400, 12 px / 16 px): etiqueta y subtexto de tiles, variación en celdas, texto del chip, etiquetas de formulario (500), contadores («12 medidores»), ticks de gráfica. Los badges usan este tamaño en 600 con `leading-none`.

### Named Rules
**La regla del numeral tabular.** Toda cifra que pueda cambiar o alinearse en columna lleva `.tnum`: valores KPI, kWh, porcentajes, contadores, fechas en tablas, el contador «Analizando… N s». Un número proporcional en una columna de datos es un defecto.

**La regla de la escala corta.** Solo existen 12, 14, 16, 20 y 28 px. La única excepción registrada es la cabecera de la tabla «Top anomalías» a 11 px por densidad de columna; no se extiende a otras tablas.

## Layout

Aplicación de escritorio de tema oscuro único, pensada para ≥ 1280 px; por debajo, un banner ámbar fijo arriba avisa del ancho mínimo y nada se reordena. Barra lateral fija de 240 px (`app`, borde derecho `line`) con marca de 64 px de alto, tres enlaces de 40 px y bloque de usuario abajo. Header pegajoso de 64 px (`canvas` al 95 % con desenfoque, borde inferior `line`, padding horizontal 32 px) con el título de página a la izquierda y el chip de análisis más el botón primario a la derecha. El contenido va en `<main>` con padding de 32 px y ancho máximo de 1440 px.

Rejillas del dashboard: 6 tiles KPI de igual ancho con separación 16 px; franja de red de 12 columnas con separación 8 px dentro de un panel con padding 12 px; fila de gráficas 2/3 + 1/3 con separación 16 px. El detalle de medidor usa 4 tiles; la investigación apila siete paneles con separación 16 px y una rejilla de 4 columnas divididas por `line` para las variables. Las tablas viven dentro de un `.panel` con `overflow-hidden`; filas de 44 px (padding vertical 12 px, horizontal 16 px). El login es una tarjeta `.panel` de 400 px centrada con padding 32 px.

Ritmo de espaciado observado: 4 / 8 / 12 / 16 / 20 / 24 / 32 px. Separación entre bloques de página 16 px; entre elementos inline 8 px; padding de panel 16 px (12 px en la franja, 20 px en modal y secciones de investigación).

## Elevation & Depth

Sistema plano por capas tonales. En reposo ninguna superficie proyecta sombra: la profundidad se lee por el escalón `canvas` → `panel` → `raised` y por el borde hairline de 1 px (`line` para paneles, `line-strong` para lo interactivo o flotante). La sombra existe en dos únicos papeles: capas que flotan sobre el lienzo (modal, toast, tooltip de gráfica, tarjeta de login) y halo de estado (crítico y encendido). El header usa `backdrop-blur-sm` sobre `canvas` al 95 % para separarse del scroll sin sombra.

### Shadow Vocabulary
- **Panel flotante** (`box-shadow: 0 1px 2px rgba(0, 0, 0, 0.6), 0 8px 24px rgba(0, 0, 0, 0.35)`): modal de análisis, toast, tooltip de gráficas, tarjeta de login. Nunca en paneles de contenido.
- **Halo crítico** (`box-shadow: 0 0 0 1px #ef4444, 0 0 16px rgba(239, 68, 68, 0.28)`): celda de red en estado CRITICAL, en reposo.
- **Halo de encendido** (pico `0 0 0 1px <color>, 0 0 28px <color al 55 %>`; reposo `0 0 16px <color al 25 %>`, o 0 si el estado es OK): animación `cell-ignite` de 900 ms al completarse un análisis, en las celdas cuyo estado cambió, escalonadas 120 ms.
- **Pulso del chip** (`box-shadow: 0 0 8px <color>`): punto de 8 px del chip de análisis mientras hay polling.

### Named Rules
**La regla del plano.** Los paneles de contenido, tiles, tablas y celdas normales no llevan sombra. Si una superficie necesita destacarse, cambia de tono (`raised`) o de borde (`line-strong`), no de elevación.

**La regla del foco en menta.** El foco de teclado es un `outline` de 2 px menta con desplazamiento 2 px y radio 4 px; los campos, además, cambian el borde a menta. No hay anillos de foco de otro color.

## Shapes

Esquinas apenas suavizadas y una jerarquía de radios ligada al tamaño del contenedor: 4 px para badges y el anillo de foco, 6 px para controles (botones, campos, segmentado, enlaces de navegación, skeletons), 8 px para paneles, tiles, celdas de red, tooltip y toast, 10 px para el modal. Las píldoras (`full`) se reservan para el chip de análisis, la barra de confianza y los puntos de estado de 8 px. Las barras de las gráficas redondean solo la parte superior (4 px). Todo contenedor lleva borde de 1 px; los bordes gruesos no existen, salvo la barra menta de 3 px del enlace activo (radio derecho 6 px) y el pulgar del scrollbar (10 px, borde `canvas` de 3 px, radio 8 px). Sin recortes diagonales, sin formas orgánicas.

## Components

### Buttons
Compactos, de altura fija, con icono opcional de 16 px a la izquierda y texto 14 px semibold. Cambian solo de color, en 150 ms `ease-out`.
- **Shape:** esquinas suaves (6 px), altura 40 px en chrome (header, login) y 36 px dentro de paneles (`h-9`), padding horizontal 16 px.
- **Primary:** fondo menta con texto `app`; hover `mint-bright`; active `mint-deep`. Un botón primario por vista: «Run AI Analysis» en el header, «Iniciar sesión», «Ver anomalías», «Marcar en investigación».
- **Secondary:** fondo `raised`, borde `line-strong`, texto `ink`; hover fondo `hover` y borde `ink-3`. Reintentar, Seguir esperando, Marcar resuelta, Descartar, Limpiar filtros.
- **Ghost:** sin fondo, texto `ink-2`; hover fondo `hover` y texto `ink`. Cerrar.
- **Disabled:** opacidad 50 % y `cursor-not-allowed`, sin cambio de color.

### Chips
- **Analysis chip:** píldora de 32 px, fondo `panel`, borde `line`, texto 12 px 500 `ink-2`, punto de 8 px cuyo color es el estado (`mint` completado, `blue` en curso con pulso, `red` fallo o sin conexión, `ink-2` sin análisis). Es un botón que abre el modal.
- **Badges:** 20 px de alto, radio 4 px, texto 12 px 600 `leading-none`, padding 6 px, fondo `tint(color, 0.15)` y texto del color. Llevan glifo lucide de 12 px cuando representan un estado (Check / AlertTriangle / Octagon para medidores; CircleDot / Search / Check / Ban para anomalías; Octagon / AlertTriangle / ShieldCheck / Database para tipos). El badge de severidad no lleva glifo.

### Cards / Containers
- **Panel (`.panel`):** radio 8 px, fondo `panel`, borde 1 px `line`, sin sombra. Header opcional de 44 px con título en Label (`.panel-title`) y acciones a la derecha, separado por borde `line`; cuerpo con padding 16 px (12 px para la franja, 0 para tablas).
- **KPI tile:** `.panel` de 104 px mínimo, padding 12 / 16 px, etiqueta Caption `ink-2` arriba y valor Display abajo; subtexto Caption opcional. El valor toma color semántico solo cuando el dato lo exige (rojo en «Alta prioridad» > 0, `variationColor` en «Variación»).
- **Sección de investigación:** `.panel` con padding 20 / 16 px y título Title.
- **Aviso ámbar:** radio 6-8 px, borde `amber` al 40 %, fondo `amber` al 10 %, texto 14 px `amber`.

### Inputs / Fields
- **Style:** altura 40 px (36 px en barras de filtro), radio 6 px, fondo `raised`, borde `line-strong`, texto 14 px `ink`, placeholder `ink-3`, padding horizontal 12 px (36 px a la izquierda con icono de búsqueda de 16 px `ink-3`).
- **Hover / Focus:** hover borde `ink-3`; foco borde `mint` sin outline propio.
- **Segmentado:** contenedor de 36 px, fondo `panel`, borde `line`, padding 4 px; opciones de 28 px, texto 12 px 600; activa fondo menta y texto `app`, inactiva `ink-2` con hover `hover` / `ink`.
- **Labels:** 12 px 500 `ink-2`, 6 px por encima del campo.

### Navigation
- **Sidebar:** fija de 240 px, fondo `app`. Marca con icono Zap en cuadrado 28 px menta al 15 %. Enlaces de 40 px, radio 6 px, texto 14 px 500 con icono de 16 px a 12 px; inactivo `ink-2`, hover fondo `hover` y texto `ink`; activo fondo `raised`, texto `ink`, icono menta y barra vertical de 3 px menta a la izquierda (inset vertical 8 px). Bloque de usuario abajo con nombre 14 px 500, correo Caption y «Cerrar sesión» con icono de 14 px.
- **Header:** 64 px pegajoso, título Headline; nada de migas ni pestañas. Los enlaces de retorno («← Anomalías IA») son texto 14 px `ink-2` con flecha de 14 px, sobre el contenido.

### Tables
`table-base`: 14 px, `border-collapse`, cabeceras Label con padding 10 / 16 px, celdas con padding 12 / 16 px y borde inferior `line` (ninguno en la última fila). Cifras a la derecha y `.tnum`; id de medidor 600 `ink`; filas clicables con hover `hover` en 150 ms; ordenación por botón en la cabecera con indicador. Estados vacío y de error viven dentro de la tabla (`EmptyState` / `ErrorState`: icono de 28 px, texto 14 px, botón secundario de 36 px).

### Franja de estado de la red (componente firma)
Doce enlaces en rejilla de 12 columnas con separación 8 px, ordenados por severidad. Cada celda (radio 8 px, fondo `panel`, padding 8 / 10 px, 76 px de alto) muestra el id en 12 px 600, el glifo de estado de 12 px en el color del estado, los kWh en 14 px 600 tabular y la variación en 12 px tabular coloreada por `variationColor` (rojo ≥ 20 %, ámbar ≥ 10 %, `ink` por debajo). El borde es `line` en OK y el color del estado en ALERT / CRITICAL; solo CRITICAL lleva `glow-red`. Al completarse un análisis, las celdas cuyo estado cambió ejecutan `cell-ignite` (900 ms, `ease-out`) con 120 ms de separación entre ellas y reposan al halo del estado (o al plano si volvieron a OK). Hover: fondo `hover`.

### Modal de análisis
560 px, radio 10 px, fondo `raised`, borde `line-strong`, sombra de panel flotante sobre velo negro al 40 %. Header Title con botón de cierre de 18 px; stepper de 7 pasos con glifos de 16 px (Check menta, LoaderCircle azul girando 1,1 s, XCircle rojo, Circle `ink-3` pendiente), etiqueta de 128 px y detalle Caption tabular; pie con contador vivo o acciones.

### Gráficas (Recharts)
Serie única menta (barras con radio superior 4 px, máx. 24 px; líneas de 2 px), rejilla horizontal hairline `hover`, ejes `line`, ticks 12 px `ink-2`, dominio y ticks limpios (`niceScale`), sin animación de entrada, sin leyenda cuando hay una sola serie. Tooltip propio: `panel` con borde `line`, radio 8 px, sombra flotante, título 12 px 500 y filas etiqueta / valor tabular con punto de color. Eventos como línea discontinua ámbar con etiqueta 11 px; atípicos como círculos rojos de 4 px con borde `panel`; ventana de anomalía como área roja al 14 %; banda baseline gris claro al 16 %. Alturas fijas: 320 px principal, 180 px secundarias. Todas llevan `role="img"` y `aria-label` narrativo.

### Toast
360 px fijo abajo a la derecha (24 px de margen), fondo `raised`, borde `line-strong`, radio 8 px, sombra flotante, texto 14 px, acción en menta 600 y cierre de 16 px; entra con `toast-in` (180 ms, 8 px hacia arriba) y se retira a los 6 s.

## Do's and Don'ts

### Do:
- **Do** construir toda superficie sobre `panel` con borde 1 px `line` y radio 8 px; escalar a `raised` + `line-strong` solo para lo interactivo o flotante.
- **Do** reservar la menta para acción primaria, navegación activa, selección y estado normal; el hover del primario sube a `mint-bright`.
- **Do** acompañar cada estado con su glifo lucide de 12-16 px y trazo 1,5 (`STROKE`), además del color.
- **Do** usar `.tnum` en cualquier cifra alineada o cambiante y formatear en es-CO con la unidad fuera del número cuando el layout lo permita.
- **Do** usar `tint(color, 0.15)` para fondos de badge y `tint(color, 0.25)` para halos en reposo; `0.55` solo en el pico del encendido.
- **Do** limitar las transiciones a 150 ms `ease-out` sobre color y borde; el único movimiento coreografiado es `cell-ignite` y respeta `prefers-reduced-motion`.
- **Do** mantener las alturas 40 px (controles de chrome) y 36 px (controles dentro de paneles), 44 px (filas y headers de panel), 64 px (header y marca).

### Don't:
- **Don't** usar gradientes, vidrio, texturas ni sombras en paneles de contenido; la profundidad es tonal.
- **Don't** dar halo a más de un estado: la alerta es borde ámbar plano; solo lo crítico y lo recién cambiado brillan.
- **Don't** introducir tamaños fuera de 12 / 14 / 16 / 20 / 28 px ni pesos por encima de 600.
- **Don't** usar el verde, el cian ni los grises `#9ca3af` / `#d1d5db` de la referencia de marca: el build los descartó y no forman parte del sistema.
- **Don't** colorear texto de lectura con `ink-3`; es el tono de placeholders, separadores y glifos vacíos.
- **Don't** usar badges sólidos, bordeados o de píldora; la píldora se reserva al chip de análisis, la barra de confianza y los puntos de estado.
- **Don't** añadir un segundo botón primario en la misma vista ni pintar el chip de análisis en menta mientras el análisis está en curso.
