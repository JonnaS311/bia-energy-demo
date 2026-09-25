# Spec: AI Energy Management Platform — Frontend

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8, heredado del maestro) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> Spec hijo de `spec-energy-plataforma-maestro-v1.md` (en adelante, "el maestro"). Los contratos de API que consume este frontend están en la sección 8.2 del maestro y se referencian aquí por número (8.2.1 … 8.2.10). Este documento no redefine ningún contrato; si un campo no aparece en el maestro, no existe.

## 1. Contexto y problema

Resumen (detalle en maestro §1): los datos de 12 medidores eléctricos viven en CSV y nadie puede responder en minutos qué medidor se desvía, si es real o explicable, y qué hacer. El backend (maestro §8.2) expone medidores, lecturas, anomalías y un análisis de IA bajo demanda. Este spec define la única interfaz de usuario: una aplicación web que debe verse y comportarse como un producto SaaS de Energy Management y permitir al evaluador recorrer `Login → Dashboard → Medidores → Detalle → Anomalías IA → Investigación → Acción` en ≤ 10 minutos (maestro §2, rúbrica Frontend/UX = 20 puntos).

## 2. Objetivo y métricas de éxito

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Guion de demo completo sin errores visibles ni errores en consola del navegador | No existe UI | 0 errores en los 8 pasos de §12b | Ensayo de demo previo a la entrega |
| Tiempo desde login hasta ver `M-109` como primera anomalía en `/anomalies` (incluye ejecutar el análisis) | — | ≤ 3 minutos | Ensayo de demo |
| Carga inicial (navegación fría a `/` con sesión válida hasta dashboard interactivo) | — | ≤ 3 s en red local | Ensayo de demo, pestaña Network de Chrome |
| Las 7 secciones de la pantalla de Investigación aparecen con los títulos literales de RF-F-08 | — | 7 de 7 | Test Vitest de RF-F-08 |
| `tsc --noEmit` y `eslint` | — | 0 errores | Antes de cada commit |

## 3. Glosario

Los términos de dominio (baseline, variación, spike, anomalía vigente, severidad, confianza, evidencia, análisis, pipeline, estado de medidor, JWT, LLM) están definidos en el maestro §3 y se usan aquí con el mismo significado. Términos propios de este spec:

| Término | Definición |
|---|---|
| Sesión | Existencia de un token JWT válido en `localStorage` bajo la clave `emp.token`. Sin token o con 401 del backend no hay sesión. |
| Ruta protegida | Cualquier ruta distinta de `/login`; requiere sesión. |
| Query | Petición de datos gestionada por TanStack Query, identificada por una clave (`queryKey`). |
| Invalidar | Marcar una query como obsoleta para que TanStack Query la vuelva a pedir al backend. |
| Polling | Repetir `GET /ai/analysis/:id` (8.2.7) cada 1.000 ms hasta que `status` sea `COMPLETED` o `FAILED`. |
| Chip de análisis | Elemento del header que muestra el estado del último análisis (RF-F-02). |
| Stepper | Lista vertical de los 7 pasos del pipeline con un icono de estado por paso (RF-F-09). |
| Skeleton | Bloques grises animados con las mismas dimensiones del contenido que reemplazan, mostrados mientras una query está cargando. |
| Badge | Etiqueta de texto de 12 px con fondo de color semántico y esquinas redondeadas de 4 px. |
| Toast | Notificación flotante en la esquina inferior derecha que desaparece a los 6 s o al pulsar su botón. |
| Etiqueta en español | Texto visible que reemplaza un valor de enum de la API según las tablas de RF-F-11. |

## 4. Actores y casos de uso

Actores según maestro §4: Analista de operaciones (único rol, credenciales demo) y Evaluador de la prueba (usa el rol de analista). Las 7 historias de usuario del maestro §4 son las que este frontend implementa; cada pantalla de la sección 6 indica qué historia cubre.

## 5. Alcance y no-alcance

**Incluye:**
- Aplicación SPA en `frontend/` con las 6 rutas de RF-F-01 y la página 404.
- Layout SaaS con sidebar, header, botón "Run AI Analysis" y chip de estado del análisis.
- Pantallas: Login, Dashboard, Medidores, Detalle de medidor, Anomalías IA, Investigación (detalle de anomalía).
- Ejecución del análisis con stepper de progreso y polling.
- Cambio de estado de una anomalía (`PATCH /anomalies/:id/status`, 8.2.9).
- Gráficas: consumo diario total, consumo horario con banda baseline y marcadores de eventos, V/I/PF, comparación diaria con ventana sombreada.
- Formato de números y fechas es-CO.
- Tests unitarios con Vitest + Testing Library; `Dockerfile` de producción.

**NO incluye (explícito):**
- NO diseño responsive por debajo de 1280 px de ancho (maestro RNF-11).
- NO tema claro ni selector de tema: la UI es únicamente oscura (decisión del usuario 2026-09-25 al adoptar la paleta de bia.app; ver RF-F-02).
- NO internacionalización: la UI es solo en español; NO selector de idioma.
- NO edición de medidores, lecturas ni eventos; NO carga de CSV desde la UI.
- NO registro de usuarios, NO recuperación de contraseña, NO gestión de perfil.
- NO WebSockets ni Server-Sent Events: el progreso del análisis se obtiene por polling.
- NO exportación a PDF/CSV/Excel de tablas o gráficas.
- NO paginación en tablas (≤ 12 filas en medidores y anomalías; maestro §5).
- NO conversión de zona horaria: las fechas se muestran en UTC tal como llegan (maestro SUPUESTO-04).
- NO Storybook ni catálogo de componentes.

## 6. Requisitos funcionales

### RF-F-01 — Stack, estructura y cliente API
**Prioridad:** DEBE
**Descripción:** El frontend DEBE construirse con React 18, TypeScript 5, Vite 5, React Router 6, TanStack Query 5, Recharts 2 y Tailwind CSS 3, con esta estructura de carpetas:

```
frontend/
  Dockerfile
  index.html
  package.json
  tsconfig.json
  vite.config.ts
  tailwind.config.ts
  src/
    app/          # App.tsx, router.tsx, providers.tsx (QueryClientProvider, AuthProvider), layout/
    pages/        # LoginPage, DashboardPage, MetersPage, MeterDetailPage, AnomaliesPage, AnomalyDetailPage, NotFoundPage
    components/   # KpiTile, StatusBadge, SeverityBadge, TypeBadge, ConfidenceBar, DataTable, EmptyState, ErrorState, Skeleton, Toast, AnalysisStepper, charts/
    api/          # client.ts, meters.ts, anomalies.ts, analysis.ts, dashboard.ts, auth.ts
    hooks/        # useAuth, useRunAnalysis, useAnalysisPolling, useDebounce
    lib/          # format.ts (números/fechas), labels.ts (enum → español), colors.ts
    types/        # api.ts (tipos escritos a mano desde maestro §8.2)
```

El cliente HTTP `src/api/client.ts` DEBE:
- Usar como base URL la variable `VITE_API_URL`; SI no está definida, ENTONCES usar `http://localhost:8080`.
- Añadir la cabecera `Authorization: Bearer <token>` en toda petición CUANDO exista `localStorage["emp.token"]`.
- CUANDO cualquier respuesta sea 401, borrar `localStorage["emp.token"]`, limpiar la caché de TanStack Query y redirigir a `/login` (una sola vez aunque fallen varias peticiones en paralelo).
- Convertir cualquier respuesta con `error.code` (formato maestro §8.2) en un objeto `ApiError { status: number; code: string; message: string; details: unknown }` que lanzan las funciones de `src/api/*.ts`.

`src/types/api.ts` DEBE contener un tipo TypeScript por cada respuesta de 8.2.1 a 8.2.10 con los nombres de campo exactos del maestro (por ejemplo `MeterListItem`, `MeterDetail`, `ReadingsResponse`, `AnomalyListItem`, `AnomalyDetail`, `AnalysisStatus`, `DashboardSummary`, `LoginResponse`, `HealthResponse`) y los enums `MeterStatus`, `AnomalyType`, `Severity`, `AnomalyStatus`, `AnalysisStatusValue`, `StepStatus`.

`frontend/Dockerfile` DEBE tener dos etapas: `node:20-alpine` que ejecuta `npm ci && npm run build`, y `caddy:2-alpine` que sirve `dist/` (copiado a `/srv`) en el puerto 5173 con `try_files {path} /index.html` (fallback SPA) definido en `frontend/Caddyfile`, con `auto_https off` y cabecera `Cache-Control: public, max-age=31536000, immutable` para `/assets/*`. `VITE_API_URL` se inyecta como `ARG` en la etapa de build.

**Criterios de aceptación:**
- DADO el repositorio recién clonado, CUANDO se ejecuta `npm ci && npm run build` en `frontend/`, ENTONCES el comando termina con código 0 y existe `frontend/dist/index.html`.
- DADO un token en `localStorage["emp.token"]`, CUANDO el cliente hace `GET /meters`, ENTONCES la petición incluye la cabecera `Authorization: Bearer <ese token>`.
- DADO una sesión activa, CUANDO el backend responde 401 a cualquier petición, ENTONCES `localStorage["emp.token"]` queda vacío y la URL del navegador pasa a `/login` en ≤ 500 ms.
- DADO la imagen construida con `docker build`, CUANDO se abre `http://localhost:5173/anomalies` directamente, ENTONCES Caddy devuelve `index.html` (HTTP 200) y no 404.

### RF-F-02 — Rutas, guardas y layout SaaS
**Prioridad:** DEBE
**Descripción:** El router DEBE definir exactamente estas rutas:

| Ruta | Página | Protegida | Título en header |
|---|---|---|---|
| `/login` | LoginPage | No | — (sin layout) |
| `/` | DashboardPage | Sí | "Dashboard" |
| `/meters` | MetersPage | Sí | "Medidores" |
| `/meters/:meterId` | MeterDetailPage | Sí | "Medidor {meterId}" |
| `/anomalies` | AnomaliesPage | Sí | "Anomalías IA" |
| `/anomalies/:id` | AnomalyDetailPage | Sí | "Investigación · {meter_id}" |
| cualquier otra | NotFoundPage | Sí | "Página no encontrada" |

CUANDO se navega a una ruta protegida sin sesión, el sistema DEBE redirigir a `/login` conservando la ruta original en el parámetro `?next=` y, tras un login exitoso, DEBE navegar a esa ruta (o a `/` si no hay `next`). CUANDO se navega a `/login` con sesión activa, el sistema DEBE redirigir a `/`.

La página 404 DEBE mostrar el texto "Página no encontrada" y un enlace "Volver al dashboard" hacia `/`.

Toda ruta protegida DEBE renderizarse dentro del layout con estas medidas y textos literales:

```
┌────────────┬──────────────────────────────────────────────────────────────┐
│ Energy     │  {Título de página}      [● Completado hace 2 min] [Run AI Analysis] │
│ Management │──────────────────────────────────────────────────────────────│
│            │                                                              │
│ ▸ Dashboard│   contenido (máx. 1440 px, padding 32 px)                    │
│   Medidores│                                                              │
│   Anomalías│                                                              │
│   IA       │                                                              │
│            │                                                              │
│ ──────────│                                                              │
│ Analista   │                                                              │
│ Demo       │                                                              │
│ Cerrar     │                                                              │
│ sesión     │                                                              │
└────────────┴──────────────────────────────────────────────────────────────┘
```

- Sidebar: ancho fijo 240 px, altura 100 vh, posición fija a la izquierda, fondo `#0A0A0A`, borde derecho 1 px `#27272A`, texto `#E4E4E7`. Arriba el texto "Energy Management" (16 px, peso 600) con un icono de rayo en menta. Debajo tres enlaces en este orden: "Dashboard" (`/`), "Medidores" (`/meters`), "Anomalías IA" (`/anomalies`); el enlace de la ruta activa (coincidencia por prefijo, `/` solo exacta) tiene fondo `#1A1A1A` y una barra izquierda de 3 px color `#08DDBC`. Abajo: el `user.name` de la respuesta de login (8.2.1) y el botón "Cerrar sesión", que borra `localStorage["emp.token"]`, limpia la caché de queries y navega a `/login`.
- Header: altura 64 px, fondo `#0F0F0F`, borde inferior 1 px `#27272A`. A la izquierda el título de página (20 px, peso 600). A la derecha, en este orden: el chip de análisis y el botón "Run AI Analysis" (botón primario: fondo `#08DDBC`, texto `#0A0A0A`, alto 40 px, esquinas 6 px; hover `#17FFDB`).
- Chip de análisis: alimentado por `GET /dashboard/summary` (8.2.8) campo `last_analysis`, y por el polling activo si lo hay. Textos literales según estado:

| Condición | Texto del chip | Color del punto |
|---|---|---|
| `last_analysis` es `null` y no hay polling | "Sin análisis" | `#A1A1AA` |
| polling activo con `status` `QUEUED` | "En curso · En cola" | `#3B82F6` |
| polling activo con `status` `RUNNING` | "En curso · {label del step RUNNING} {n}/7", donde n es la posición 1-7 del step con `status=RUNNING` (ej. "En curso · Detección 3/7") | `#3B82F6` |
| `last_analysis.status` = `COMPLETED` | "Completado hace {tiempo relativo}" (formato de §RF-F-12: "hace 2 min", "hace 1 h", "hace 3 d") | `#08DDBC` |
| `last_analysis.status` = `FAILED` | "Falló" | `#EF4444` |
| `last_analysis.status` = `QUEUED` o `RUNNING` y no hay polling local (otro cliente lo inició) | "En curso" | `#3B82F6` |
| La última query de `["dashboard"]` falló por error de red (CB-F-02) | "Sin conexión" | `#EF4444` |

- Contenido: ancho máximo 1440 px centrado, padding 32 px, fondo `#0F0F0F`.
- Tipografía Inter (Google Fonts, pesos 400-700), tamaño base 14 px, numerales tabulares (`font-variant-numeric: tabular-nums`) en toda columna de datos. Tema oscuro únicamente.
- Tokens de superficie (paleta de bia.app, fuente de verdad `src/lib/colors.ts` y `tailwind.config.ts`): app `#0A0A0A`, contenido `#0F0F0F`, panel `#141414`, elevado `#1A1A1A`, hover `#1F1F1F`, borde `#27272A`, borde fuerte `#3F3F46`; texto principal `#FFFFFF`, secundario `#A1A1AA`, terciario `#71717A`; acento menta `#08DDBC` (hover `#17FFDB`) para acción primaria, navegación activa, selección y estado normal; violeta `#8B5CF6`; azul `#3B82F6`. Paneles planos con borde de 1 px y esquinas de 8 px, sin gradientes; solo el elemento activo o crítico recibe halo (`0 0 0 1px` + `0 0 16px` del color al 25 %). Selección de texto, caret, scrollbar y anillo de foco temados en menta.
- Colores semánticos fijos definidos en `src/lib/colors.ts` y usados por todos los badges y gráficas: `OK` = `#08DDBC`, `ALERT` = `#F59E0B`, `CRITICAL` = `#EF4444`; severidad `HIGH` = `#EF4444`, `MEDIUM` = `#F59E0B`, `LOW` = `#A1A1AA`. Todo badge lleva un glifo (icono lucide de 12 px) además del color: el estado es una marca, no solo un tono.

**Criterios de aceptación:**
- DADO un navegador sin token, CUANDO se abre `/meters/M-109`, ENTONCES la URL pasa a `/login?next=%2Fmeters%2FM-109` y, tras iniciar sesión, la URL pasa a `/meters/M-109`.
- DADO una sesión activa, CUANDO se abre `/ruta-inexistente`, ENTONCES se muestra "Página no encontrada" dentro del layout y el enlace "Volver al dashboard" lleva a `/`.
- DADO `last_analysis = null`, CUANDO se renderiza cualquier ruta protegida, ENTONCES el chip muestra "Sin análisis".
- DADO un análisis en polling cuyo tercer step tiene `status=RUNNING`, CUANDO se renderiza el header, ENTONCES el chip muestra "En curso · Detección 3/7".
- DADO la ruta `/anomalies`, CUANDO se renderiza el sidebar, ENTONCES el enlace "Anomalías IA" tiene la barra izquierda de 3 px y los otros dos no.

### RF-F-03 — Login
**Prioridad:** DEBE
**Descripción:** `/login` DEBE mostrar, centrado vertical y horizontalmente sobre fondo `#0F0F0F`, un panel de 400 px de ancho con:

```
┌──────────────────────────────────────┐
│  Energy Management                   │
│  Inicia sesión para continuar        │
│                                      │
│  Correo electrónico                  │
│  [ analista@energy.local           ] │
│  Contraseña                          │
│  [ ••••••••                        ] │
│                                      │
│  [        Iniciar sesión          ]  │
│  (mensaje de error aquí)             │
└──────────────────────────────────────┘
```

- Campo "Correo electrónico" (`type=email`, `placeholder="analista@energy.local"`, valor inicial vacío) y campo "Contraseña" (`type=password`, `placeholder="Demo1234!"`, valor inicial vacío). Los placeholders son ayuda visual y NO se envían si el campo está vacío.
- Botón "Iniciar sesión" deshabilitado MIENTRAS algún campo esté vacío o la petición esté en curso; MIENTRAS la petición está en curso el texto del botón es "Iniciando…".
- CUANDO se envía el formulario, el sistema DEBE llamar a `POST /auth/login` (8.2.1). SI la respuesta es 200, ENTONCES guarda `token` en `localStorage["emp.token"]`, guarda `user` en memoria (AuthProvider) y en `localStorage["emp.user"]`, y navega a `?next` o `/`. SI la respuesta es 401 `INVALID_CREDENTIALS`, ENTONCES muestra bajo el botón el texto "Credenciales inválidas" en `#EF4444`. SI la petición falla por red, ENTONCES muestra "No se pudo conectar con el servidor".
- La tecla Enter en cualquier campo envía el formulario.
- SI la URL contiene `?expired=1` (puesto por el cliente API en CB-F-01), ENTONCES se muestra sobre el formulario el texto "Tu sesión expiró. Inicia sesión de nuevo." en `#F59E0B`.

**Criterios de aceptación:**
- DADO ambos campos vacíos, CUANDO se renderiza la página, ENTONCES el botón "Iniciar sesión" está deshabilitado y ninguno de los dos inputs tiene valor.
- DADO email `analista@energy.local` y contraseña `Demo1234!`, CUANDO se pulsa "Iniciar sesión" y el backend responde 200, ENTONCES `localStorage["emp.token"]` contiene el token de la respuesta y la URL es `/`.
- DADO credenciales incorrectas, CUANDO el backend responde 401, ENTONCES se muestra el texto "Credenciales inválidas" y la URL sigue siendo `/login`.

### RF-F-04 — Dashboard
**Prioridad:** DEBE
**Descripción:** `/` DEBE consumir `GET /dashboard/summary` (8.2.8) con `queryKey = ["dashboard"]` y renderizar:

```
┌───────────┬───────────┬───────────┬───────────┬───────────┬───────────┐
│ Medidores │ Consumo   │ Anomalías │ Alta      │ Confianza │ Último    │
│ 12        │ 155.250,8 │ IA        │ prioridad │ IA        │ análisis  │
│           │ kWh       │ 4         │ 2         │ 93 %      │ 25 sep    │
│           │           │           │           │           │ 2026 15:02│
│           │           │           │           │           │ Completado│
└───────────┴───────────┴───────────┴───────────┴───────────┴───────────┘
┌──────────────────────────────────────────┬───────────────────────────────┐
│ Consumo diario total (kWh)               │ Top anomalías                 │
│ ▇▇▇▇▇▇▇▇▇▇▇▇▇▇  (14 barras)              │ 1 M-109 Anomalía real  Alta   │
│                                          │ 2 M-112 Calidad de datos Alta │
│                                          │ 3 M-104 Anomalía explicable … │
│                                          │ [Ver todas las anomalías →]   │
└──────────────────────────────────────────┴───────────────────────────────┘
```

Tabla de alimentación de cada elemento:

| Elemento | Campo de 8.2.8 | Formato |
|---|---|---|
| Tile "Medidores" | `meters_total` | entero |
| Tile "Consumo" | `total_consumption_kwh` | kWh, 1 decimal, sufijo " kWh" |
| Tile "Anomalías IA" | `anomalies_total` | entero |
| Tile "Alta prioridad" | `high_priority_total` | entero; SI > 0, ENTONCES el número va en `#EF4444` |
| Tile "Confianza IA" | `ai_confidence_avg` | porcentaje sin decimales (0.93 → "93 %"); SI es `null`, ENTONCES "—" |
| Tile "Último análisis" | `last_analysis.finished_at` y `last_analysis.status` | fecha `dd MMM yyyy HH:mm` en la primera línea y la etiqueta del estado en la segunda ("Completado", "Falló", "En curso" para `RUNNING`/`QUEUED`); SI `finished_at` es `null`, ENTONCES la primera línea es "—"; SI `last_analysis` es `null`, ENTONCES "—" y "Sin análisis" |
| Gráfica "Consumo diario total (kWh)" | `daily_consumption[]` | barras verticales, eje X con día (`dd MMM`), eje Y en kWh, tooltip con fecha completa y valor con 1 decimal, color de barra `#08DDBC` |
| Tabla "Top anomalías" | `top_anomalies[]` (máximo 3) | columnas: posición (1-3), `meter_id` como enlace a `/anomalies/{id}`, etiqueta de `type` (RF-F-11), badge de `severity`, etiqueta de confianza (RF-F-11) |
| Enlace "Ver todas las anomalías →" | — | navega a `/anomalies` |

Los 6 tiles DEBEN tener el mismo ancho (grid de 6 columnas con separación 16 px), fondo `#141414`, borde 1 px `#27272A`, esquinas 8 px, título en 12 px `#A1A1AA` y valor en 28 px peso 600 con numerales proporcionales.

**Estado de la red (fila añadida por la dirección de diseño "Panel de operador", 2026-09-25).** Entre los tiles y las gráficas, el dashboard DEBE mostrar el panel "Estado de la red": consume `GET /meters?sort=severity&order=desc` (8.2.2) con `queryKey = ["meters", { sort: "severity", order: "desc" }]` y renderiza los 12 medidores como celdas en una rejilla de 12 columnas (separación 8 px), en el orden recibido (críticos primero). Cada celda es un enlace a `/meters/{meter_id}` y muestra `meter_id` (12 px, peso 600), el glifo de estado, `current_consumption_kwh` (14 px, peso 600, tabular) y `variation_pct` con signo y el color de RF-F-05. Borde y halo según estado: `CRITICAL` borde `#EF4444` con halo rojo, `ALERT` borde `#F59E0B` con halo ámbar, `OK` borde `#27272A` sin halo. En la cabecera del panel, a la derecha: "{n} críticos · {n} en alerta · {n} normales" desde `meters_by_status` de 8.2.8. CUANDO un análisis termina `COMPLETED` y el estado de una celda cambia, la celda DEBE encenderse con la animación `cell-ignite` (900 ms) en secuencia, con 120 ms entre celdas, y luego reposar. El panel lleva un párrafo `sr-only` que describe cuántos medidores hay en cada estado y cuáles. Estados: skeleton de 12 celdas mientras carga; error → `ErrorState` con "No se pudieron cargar los medidores".

Estados:
- Loading: skeleton de 6 tiles + 2 paneles.
- Empty (`anomalies_total = 0` y `last_analysis = null`): el panel "Top anomalías" muestra el texto "Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías." y los tiles "Anomalías IA", "Alta prioridad" muestran 0 y "Confianza IA" muestra "—". La gráfica de consumo diario se muestra igualmente (no depende del análisis).
- Error: el contenido se reemplaza por el texto "No se pudo cargar el dashboard" y un botón "Reintentar" que vuelve a ejecutar la query.

**Criterios de aceptación:**
- DADO la respuesta de ejemplo de 8.2.8, CUANDO se renderiza `/`, ENTONCES el tile "Consumo" muestra "155.250,8 kWh", el tile "Confianza IA" muestra "93 %", el tile "Alta prioridad" muestra "2" en rojo y la tabla "Top anomalías" tiene 3 filas cuya primera es `M-109`.
- DADO `last_analysis = null` y `anomalies_total = 0`, CUANDO se renderiza `/`, ENTONCES el panel "Top anomalías" muestra exactamente "Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías." y la gráfica de barras tiene 14 barras.
- DADO que la query falla con error de red, CUANDO se renderiza `/`, ENTONCES aparece "No se pudo cargar el dashboard" y el botón "Reintentar", y al pulsarlo se vuelve a llamar a `GET /dashboard/summary`.

### RF-F-05 — Medidores
**Prioridad:** DEBE
**Descripción:** `/meters` DEBE consumir `GET /meters` (8.2.2) con `queryKey = ["meters", { status, q, sort, order }]` y renderizar:

```
[ Todos ] [ Normales ] [ Alertas ] [ Críticas ]        [🔍 Buscar por ID de medidor ]
┌──────────┬─────────────────┬──────────────┬───────────┬──────────────┐
│ Medidor ↕│ Consumo (kWh) ↕ │ Variación ↕  │ Estado    │ Anomalía     │
├──────────┼─────────────────┼──────────────┼───────────┼──────────────┤
│ M-109    │ 2.207,6         │ +110,5 %     │ CRÍTICO   │ Alta         │
│ M-112    │ 662,7           │ +0,1 %       │ ALERTA    │ Alta         │
│ M-104    │ 1.725,6         │ +47,5 %      │ ALERTA    │ Media        │
│ M-101    │ 728,8           │ −0,1 %       │ OK        │ —            │
└──────────┴─────────────────┴──────────────┴───────────┴──────────────┘
```

- Filtro segmentado con cuatro botones en este orden y con estos textos: "Todos" (sin parámetro `status`), "Normales" (`status=OK`), "Alertas" (`status=ALERT`), "Críticas" (`status=CRITICAL`). El botón activo tiene fondo `#08DDBC` y texto `#0A0A0A`. El filtro se refleja en la URL como `?status=`.
- Campo de búsqueda con `placeholder="Buscar por ID de medidor"`; su valor se envía como `q` tras un debounce de 300 ms y se refleja en la URL como `?q=`.
- Columnas y datos:

| Columna | Campo de 8.2.2 | Formato / regla |
|---|---|---|
| "Medidor" | `meter_id` | texto, peso 600; debajo en 12 px `#A1A1AA` el `name` |
| "Consumo (kWh)" | `current_consumption_kwh` | 1 decimal |
| "Variación" | `variation_pct` | 1 decimal con signo explícito (`+`/`−`); color según el valor absoluto: `#EF4444` si `|v| ≥ 20`, `#F59E0B` si `10 ≤ |v| < 20`, `#FFFFFF` si `|v| < 10`; SI es `null`, ENTONCES "—" |
| "Estado" | `status` | badge con etiqueta "OK" / "ALERTA" / "CRÍTICO" y color de RF-F-02 |
| "Anomalía" | `anomaly.severity` | badge con etiqueta "Alta" / "Media" / "Baja"; SI `anomaly` es `null`, ENTONCES "—" |

- Ordenamiento: las cabeceras "Medidor", "Consumo (kWh)", "Variación" y "Estado" son clicables. "Estado" envía `sort=severity`; "Consumo (kWh)" envía `sort=consumption`; "Variación" envía `sort=variation`; "Medidor" ordena en cliente por `meter_id`. El primer clic aplica `order=desc`, el segundo `order=asc`, el tercero vuelve al orden por defecto (`sort=severity&order=desc`). La cabecera activa muestra un indicador "▲" o "▼"; se refleja en la URL como `?sort=&order=`.
- Cada fila es clicable completa (cursor pointer, hover fondo `#1F1F1F`) y navega a `/meters/{meter_id}`.
- Debajo de la tabla: "{total} medidores" (ej. "12 medidores").

Estados:
- Loading: skeleton de 12 filas.
- Empty: SI `items` está vacío, ENTONCES una fila única con "Ningún medidor coincide con los filtros" y, si hay filtro o búsqueda activos, un botón "Limpiar filtros" que vuelve a "Todos" y borra `q`.
- Error: "No se pudieron cargar los medidores" + botón "Reintentar".

**Criterios de aceptación:**
- DADO la respuesta de ejemplo de 8.2.2, CUANDO se renderiza `/meters`, ENTONCES la fila `M-109` muestra "2.207,6", "+110,5 %" en `#EF4444`, badge "CRÍTICO" y badge "Alta"; la fila `M-101` muestra "−0,1 %" y "—" en la columna Anomalía.
- DADO la pestaña "Críticas" pulsada, CUANDO se resuelve la query, ENTONCES la petición al backend incluye `status=CRITICAL` y la URL incluye `?status=CRITICAL`.
- DADO el usuario escribe "10" en la búsqueda, CUANDO pasan 300 ms sin más teclas, ENTONCES se hace exactamente una petición con `q=10`.
- DADO la cabecera "Consumo (kWh)" pulsada una vez, CUANDO se resuelve la query, ENTONCES la petición incluye `sort=consumption&order=desc` y la cabecera muestra "▼".
- DADO una respuesta con `items = []`, CUANDO se renderiza la tabla, ENTONCES aparece "Ningún medidor coincide con los filtros".

### RF-F-06 — Detalle de medidor
**Prioridad:** DEBE
**Descripción:** `/meters/:meterId` DEBE consumir `GET /meters/:meterId` (8.2.3) con `queryKey = ["meter", meterId]` y `GET /meters/:meterId/readings` (8.2.4) con `queryKey = ["readings", meterId, granularity]`, y renderizar:

```
← Medidores
┌────────────────┬────────────────┬────────────────┬────────────────┐
│ Consumo actual │ Baseline       │ Variación      │ Estado         │
│ 2.207,6 kWh    │ 1.048,8 kWh/día│ +110,5 %       │ CRÍTICO        │
└────────────────┴────────────────┴────────────────┴────────────────┘
┌───────────────────────────────────────────────────────────────────┐
│ Consumo                                       [ Hora ] [ Día ]    │
│  ~~~~ línea de consumo, banda gris = baseline media ± 2σ ~~~~     │
│  │ marcador de evento con tooltip                                 │
└───────────────────────────────────────────────────────────────────┘
┌───────────────────┬───────────────────┬───────────────────┐
│ Voltaje (V)       │ Corriente (A)     │ Factor de potencia│
│ ~~~ + línea ref.  │ ~~~ + línea ref.  │ ~~~ + línea ref.  │
└───────────────────┴───────────────────┴───────────────────┘
┌──────────────────────────────┬────────────────────────────────────┐
│ Anomalía vigente             │ Eventos del medidor                │
│ Anomalía real · Alta · 0,98  │ 12 sep 2026 14:00 · UNKNOWN        │
│ [Ver investigación →]        │ No operational event reported      │
└──────────────────────────────┴────────────────────────────────────┘
```

- Enlace "← Medidores" arriba que navega a `/meters`.
- 4 cards (mismo estilo que los tiles del dashboard):

| Card | Campo de 8.2.3 | Formato |
|---|---|---|
| "Consumo actual" | `current_consumption_kwh` | kWh 1 decimal; subtítulo "último día ({period.end} formateado `dd MMM`)" |
| "Baseline" | `baseline_daily_kwh` | kWh 1 decimal + " kWh/día"; subtítulo "promedio de los primeros 7 días" |
| "Variación" | `variation_pct` | igual regla de color que RF-F-05 |
| "Estado" | `status` | badge de estado |

- Gráfica "Consumo": selector segmentado "Hora" / "Día" (default "Hora") que fija `granularity`. Con `granularity=hour`: línea de `items[].consumption_kwh` sobre `timestamp`; banda rellena (`#CBD5E1` al 40 %) entre `mean_kwh − 2·std_kwh` y `mean_kwh + 2·std_kwh` del `baseline_profile` según la hora de cada punto; los puntos con `is_outlier = true` se dibujan como círculo `#EF4444` de radio 4 px. Con `granularity=day`: barras de `items[].consumption_kwh` por `date`, con línea horizontal de referencia en `baseline_daily_kwh` (línea discontinua `#A1A1AA`) y tooltip que incluye `deviation_pct` con signo. En ambos modos, cada elemento de `events[]` se dibuja como línea vertical discontinua `#F59E0B` en su `timestamp` con tooltip "{type} · {description}".
- Tres gráficas pequeñas (alto 180 px) "Voltaje (V)", "Corriente (A)", "Factor de potencia": línea de `voltage_v` / `current_a` / `power_factor` de los `items` horarios (siempre con `granularity=hour`, independientemente del selector), con línea horizontal discontinua en `electrical.*.baseline_avg`. La de voltaje DEBE incluir además dos líneas de referencia en 209 y 231 (banda aceptable del maestro §9) en `#EF4444` al 50 %.
- Panel "Anomalía vigente": SI `anomaly` no es `null`, ENTONCES muestra la etiqueta de tipo, badge de severidad, confianza con 2 decimales y un botón "Ver investigación →" hacia `/anomalies/{anomaly.id}`. SI `anomaly` es `null`, ENTONCES muestra "Sin anomalía vigente".
- Panel "Eventos del medidor": lista de `events[]` con fecha `dd MMM yyyy HH:mm`, `type` y `description`; SI está vacía, ENTONCES "Sin eventos registrados".

Estados:
- Loading: skeleton de 4 cards + 1 gráfica grande + 3 pequeñas.
- Error 404 `METER_NOT_FOUND`: texto "El medidor {meterId} no existe" y enlace "Volver a medidores".
- Otro error: "No se pudo cargar el medidor" + "Reintentar".

**Criterios de aceptación:**
- DADO la respuesta de ejemplo de 8.2.3 para `M-109`, CUANDO se renderiza la página, ENTONCES las cards muestran "2.207,6 kWh", "1.048,8 kWh/día", "+110,5 %" y badge "CRÍTICO", y el panel "Anomalía vigente" muestra "Anomalía real", badge "Alta", "0,98" y el botón "Ver investigación →".
- DADO `granularity=hour` y un `item` con `is_outlier = true`, CUANDO se renderiza la gráfica de consumo, ENTONCES ese punto se dibuja con un círculo rojo y los demás no.
- DADO el selector "Día" pulsado, CUANDO se resuelve la query, ENTONCES la petición incluye `granularity=day` y la gráfica muestra 14 barras con una línea de referencia en `baseline_daily_kwh`.
- DADO un `events[]` con un elemento en `2026-09-12T14:00:00Z`, CUANDO se pasa el puntero por la línea vertical de ese evento, ENTONCES el tooltip muestra "UNKNOWN · No operational event reported".
- DADO `meterId = M-999`, CUANDO el backend responde 404, ENTONCES se muestra "El medidor M-999 no existe".

### RF-F-07 — Anomalías IA
**Prioridad:** DEBE
**Descripción:** `/anomalies` DEBE consumir `GET /anomalies` (8.2.5) con `queryKey = ["anomalies", { type, severity, status }]` y renderizar:

```
Tipo [Todos ▾]  Severidad [Todas ▾]  Estado [Todos ▾]
┌───┬─────────┬─────────────────────┬──────────┬───────────────┬──────────────────────────────┬───────────────┐
│ # │ Medidor │ Tipo                │ Severidad│ Confianza     │ Acción recomendada           │ Estado        │
├───┼─────────┼─────────────────────┼──────────┼───────────────┼──────────────────────────────┼───────────────┤
│ 1 │ M-109   │ Anomalía real       │ Alta     │ 98 % ▓▓▓▓▓▓▓░ │ Investigar el medidor y la … │ Abierta       │
│ 2 │ M-112   │ Calidad de datos    │ Alta     │ 98 % ▓▓▓▓▓▓▓░ │ …                            │ Abierta       │
│ 3 │ M-104   │ Anomalía explicable │ Media    │ 95 % ▓▓▓▓▓▓▓░ │ …                            │ Abierta       │
│ 4 │ M-106   │ Falso positivo      │ Baja     │ 80 % ▓▓▓▓▓▓░░ │ …                            │ Abierta       │
└───┴─────────┴─────────────────────┴──────────┴───────────────┴──────────────────────────────┴───────────────┘
```

- Tres selectores desplegables: "Tipo" (opciones "Todos" + las 4 etiquetas de tipo), "Severidad" ("Todas", "Alta", "Media", "Baja"), "Estado" ("Todos", "Abierta", "En investigación", "Resuelta", "Descartada"). Cada selección envía el valor de enum correspondiente como parámetro y se refleja en la URL.
- Columnas:

| Columna | Campo de 8.2.5 | Formato |
|---|---|---|
| "#" | `priority_rank` | entero |
| "Medidor" | `meter_id` | texto peso 600 |
| "Tipo" | `type` | badge de tipo (RF-F-11) |
| "Severidad" | `severity` | badge de severidad |
| "Confianza" | `confidence` | "{porcentaje sin decimales} %" seguido de una barra horizontal de 80 px cuyo relleno es `confidence × 100 %` en `#08DDBC`; título (tooltip) con la etiqueta de confianza de RF-F-11 |
| "Acción recomendada" | `recommended_action` | texto truncado a una línea con elipsis; tooltip con el texto completo |
| "Estado" | `status` | etiqueta "Abierta" / "En investigación" / "Resuelta" / "Descartada" |

- Las filas se muestran en el orden recibido (`priority_rank` asc) y NO se reordenan en cliente. Cada fila es clicable y navega a `/anomalies/{id}`.

Estados:
- Loading: skeleton de 4 filas.
- Empty sin análisis (`analysis_id` es `null` o `items = []` sin filtros): "Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías."
- Empty con filtros (`items = []` y algún filtro activo): "Ninguna anomalía coincide con los filtros" + botón "Limpiar filtros".
- Error: "No se pudieron cargar las anomalías" + "Reintentar".

**Criterios de aceptación:**
- DADO la respuesta de ejemplo de 8.2.5, CUANDO se renderiza `/anomalies`, ENTONCES hay 4 filas en el orden `M-109, M-112, M-104, M-106`, la primera muestra "Anomalía real", badge "Alta", "98 %" y la barra rellena al 98 %.
- DADO el selector "Tipo" en "Calidad de datos", CUANDO se resuelve la query, ENTONCES la petición incluye `type=DATA_QUALITY` y la tabla muestra solo `M-112`.
- DADO `items = []` sin filtros activos, CUANDO se renderiza, ENTONCES aparece exactamente "Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías."

### RF-F-08 — Investigación (detalle de anomalía)
**Prioridad:** DEBE
**Descripción:** `/anomalies/:id` DEBE consumir `GET /anomalies/:id` (8.2.6) con `queryKey = ["anomaly", id]` y `GET /meters/:meterId/readings?granularity=day` (8.2.4) con el `meter_id` de la anomalía, y renderizar las 7 secciones en este orden, con estos títulos literales como encabezados `<h2>`:

```
← Anomalías IA
M-109 · [Anomalía real] [Alta] · Confianza 0,98 · Ventana 12 sep – 14 sep 2026 · [Explicación generada por IA]

1. Qué encontró la IA
   {reason}

2. Variables que cambiaron
   ┌────────────┬────────────┬────────────┬────────────┐
   │ Consumo    │ Voltaje    │ Corriente  │ Factor pot.│
   │ 2.207,6 kWh│ 216,4 V    │ 420,5 A    │ 0,744      │
   │ +110,5 %   │ −1,6 %     │ +109,7 %   │ −20,8 %    │
   │ base 1.048,8│ base 219,9│ base 200,5 │ base 0,939 │
   └────────────┴────────────┴────────────┴────────────┘

3. Comparación contra baseline
   (gráfica de barras diarias con línea de baseline y ventana sombreada)

4. Eventos relacionados
   {related_event} | "Sin eventos relacionados"

5. Severidad y confianza
   Severidad: [Alta]   Confianza: 0,98 (Alta) ▓▓▓▓▓▓▓░
   Estado: Abierta

6. Acción recomendada
   {recommended_action}
   [Marcar en investigación] [Marcar resuelta] [Descartar]

7. Evidencia
   ┌───┬────────────────────┬───────────┬──────────┬──────────┬────────────────────────┬──────────────────────────┐
   │ # │ Métrica            │ Observado │ Baseline │ Delta    │ Ventana                │ Detalle                  │
   └───┴────────────────────┴───────────┴──────────┴──────────┴────────────────────────┴──────────────────────────┘
```

Reglas por sección:

| Sección | Campo(s) de 8.2.6 | Regla |
|---|---|---|
| Encabezado | `meter_id`, `type`, `severity`, `confidence`, `window_start`, `window_end`, `explanation_source` | `meter_id` como enlace a `/meters/{meter_id}`; badges de tipo y severidad; "Confianza {2 decimales}"; "Ventana {dd MMM} – {dd MMM yyyy}"; badge gris con texto "Explicación generada por IA" si `explanation_source = "llm"` o "Explicación por plantilla" si `"template"` |
| "Qué encontró la IA" | `reason` | párrafo; debajo, lista `<ul>` con `explanation_points[]` |
| "Variables que cambiaron" | `comparison` | 4 mini-cards en este orden y con estos títulos: "Consumo" (`consumption_kwh`, unidad " kWh"), "Voltaje" (`voltage_v`, " V"), "Corriente" (`current_a`, " A"), "Factor de potencia" (`power_factor`, sin unidad, 3 decimales); cada card muestra `observed` (valor grande), `delta_pct` con signo y color (misma regla por valor absoluto de RF-F-05), y "base {baseline}" en 12 px |
| "Comparación contra baseline" | `readings` con `granularity=day`, `window_start`, `window_end` | barras de `consumption_kwh` por `date`; línea horizontal discontinua en el `baseline_daily_kwh` del medidor (tomado de `comparison.consumption_kwh.baseline`); área sombreada `#EF4444` al 14 % cubriendo los días entre `window_start` y `window_end` inclusive; tooltip con `deviation_pct` |
| "Eventos relacionados" | `related_event` | SI no es `null`, ENTONCES card con fecha `dd MMM yyyy HH:mm`, `type` y `description`; SI es `null`, ENTONCES el texto "Sin eventos relacionados" |
| "Severidad y confianza" | `severity`, `confidence`, `status` | badge de severidad; "Confianza {2 decimales} ({etiqueta RF-F-11})" con barra de 120 px; "Estado: {etiqueta de estado}" |
| "Acción recomendada" | `recommended_action`, `status` | párrafo en 16 px peso 500; tres botones: "Marcar en investigación" (envía `INVESTIGATING`), "Marcar resuelta" (envía `RESOLVED`), "Descartar" (envía `DISMISSED`) mediante `PATCH /anomalies/:id/status` (8.2.9). El botón cuyo estado coincide con el `status` actual está deshabilitado. CUANDO la petición responde 200, el sistema DEBE actualizar la query `["anomaly", id]` con la respuesta, invalidar `["anomalies"]` y `["dashboard"]`, y mostrar el toast "Estado actualizado a {etiqueta}". SI responde error, ENTONCES toast "No se pudo actualizar el estado" con botón "Reintentar" |
| "Evidencia" | `evidence[]` | tabla ordenada por `position` con columnas "#", "Métrica" (etiqueta de `metric` según RF-F-11; si no está mapeada, el valor literal), "Observado" (`observed` formateado con `unit`; SI `null`, "—"), "Baseline" (ídem), "Delta" (`delta_pct` con signo y 1 decimal; SI `null`, "—"), "Ventana" (`window` literal), "Detalle" (`detail`) |

Enlace "← Anomalías IA" arriba hacia `/anomalies`.

Estados:
- Loading: skeleton del encabezado + 7 bloques.
- Error 404 `ANOMALY_NOT_FOUND`: "La anomalía no existe" y enlace "Volver a anomalías".
- Anomalía supersedida (`superseded = true` en la respuesta, o el `analysis_id` de la anomalía difiere del `last_analysis.id` del dashboard): banner ámbar arriba del encabezado con el texto "Esta anomalía pertenece a un análisis anterior" y enlace "Ver anomalías vigentes" hacia `/anomalies`. Los botones de acción quedan deshabilitados.

**Criterios de aceptación:**
- DADO la respuesta de ejemplo de 8.2.6 para `M-109`, CUANDO se renderiza la página, ENTONCES existen exactamente 7 encabezados `<h2>` con los textos "Qué encontró la IA", "Variables que cambiaron", "Comparación contra baseline", "Eventos relacionados", "Severidad y confianza", "Acción recomendada", "Evidencia", en ese orden.
- DADO el mismo ejemplo, CUANDO se renderiza "Variables que cambiaron", ENTONCES la card "Corriente" muestra "420,5 A", "+109,7 %" y "base 200,5", y la card "Factor de potencia" muestra "0,744", "−20,8 %" y "base 0,939".
- DADO `related_event = null`, CUANDO se renderiza "Eventos relacionados", ENTONCES aparece exactamente "Sin eventos relacionados".
- DADO `explanation_source = "template"`, CUANDO se renderiza el encabezado, ENTONCES aparece el badge "Explicación por plantilla".
- DADO `status = "OPEN"`, CUANDO se pulsa "Marcar en investigación" y el backend responde 200 con `status = "INVESTIGATING"`, ENTONCES el texto "Estado: En investigación" se muestra, el botón "Marcar en investigación" queda deshabilitado y aparece el toast "Estado actualizado a En investigación".
- DADO 5 elementos en `evidence[]`, CUANDO se renderiza "Evidencia", ENTONCES la tabla tiene 5 filas de datos ordenadas por `position` y la fila con `observed = null` muestra "—".
- DADO una anomalía con `superseded = true`, CUANDO se abre su URL, ENTONCES aparece el banner "Esta anomalía pertenece a un análisis anterior" y los tres botones de acción están deshabilitados.

### RF-F-09 — Ejecutar análisis de IA
**Prioridad:** DEBE
**Descripción:** CUANDO el analista pulsa "Run AI Analysis" en el header, el sistema DEBE llamar a `POST /ai/analyze` (8.2.7) y abrir un panel modal (ancho 480 px, centrado, fondo oscurecido al 40 %) con:

```
┌──────────────────────────────────────────────┐
│ Análisis de IA                          [✕]  │
│                                              │
│ ✔ Lecturas         4.032 lecturas de 12 med. │
│ ✔ Baseline         Baseline calculado …      │
│ ◌ Detección        (en curso)                │
│ ○ Correlación                                │
│ ○ Eventos                                    │
│ ○ Explicación                                │
│ ○ Recomendación                              │
│                                              │
│ Analizando… 12 s                             │
└──────────────────────────────────────────────┘
```

- Título "Análisis de IA". Stepper con los 7 pasos en el orden y con los `label` del maestro 8.2.7 ("Lecturas", "Baseline", "Detección", "Correlación", "Eventos", "Explicación", "Recomendación"). Icono por `steps[i].status`: `PENDING` = círculo vacío gris; `RUNNING` = círculo con animación de giro azul; `COMPLETED` = check verde; `FAILED` = cruz roja. A la derecha de cada paso, `steps[i].detail` cuando no es `null`.
- Polling: el hook `useAnalysisPolling(analysisId)` DEBE llamar a `GET /ai/analysis/:id` cada 1.000 ms MIENTRAS `status` sea `QUEUED` o `RUNNING`, y detenerse CUANDO sea `COMPLETED` o `FAILED`. El `analysis_id` DEBE guardarse en `sessionStorage["emp.analysisId"]` al iniciar y borrarse al terminar (en cualquier estado final).
- Pie del modal MIENTRAS está en curso: "Analizando… {segundos transcurridos} s". Botón "✕" cierra el modal pero NO detiene el polling (el chip del header sigue mostrando el progreso).
- CUANDO `status = COMPLETED`, el pie DEBE mostrar el texto "{summary.anomalies_detected} anomalías detectadas · {summary.high_priority} requieren atención prioritaria" (ej. "4 anomalías detectadas · 2 requieren atención prioritaria"; con 1 se usa la misma forma plural) y dos botones: "Ver anomalías" (navega a `/anomalies` y cierra el modal) y "Cerrar". En ese mismo momento el sistema DEBE invalidar las queries `["dashboard"]`, `["meters"]`, `["meter"]`, `["anomalies"]` y `["anomaly"]`.
- CUANDO `status = FAILED`, el pie DEBE mostrar "El análisis falló: {error_message}" y un botón "Reintentar" que vuelve a llamar a `POST /ai/analyze`.
- SI el polling supera 180 s sin estado final, ENTONCES el pie muestra "El análisis está tardando más de lo esperado" y los botones "Seguir esperando" (reinicia el contador de 180 s) y "Cerrar".
- SI `POST /ai/analyze` responde 409 `ANALYSIS_IN_PROGRESS`, ENTONCES el sistema DEBE mostrar el toast "Ya hay un análisis en curso", tomar el `analysis_id` de `error.details.analysis_id` (maestro CB-M03) y abrir el modal adjuntándose al polling de ese análisis.
- El botón "Run AI Analysis" DEBE estar deshabilitado (opacidad 50 %, cursor not-allowed, texto "Analizando…") MIENTRAS haya un polling activo o una petición `POST /ai/analyze` en curso.
- CUANDO la aplicación carga y existe `sessionStorage["emp.analysisId"]`, el sistema DEBE reanudar el polling de ese id sin abrir el modal; SI el backend responde 404 `ANALYSIS_NOT_FOUND`, ENTONCES borra la clave y no muestra nada.

**Criterios de aceptación:**
- DADO el header, CUANDO se pulsa "Run AI Analysis" y el backend responde 202 con `analysis_id`, ENTONCES se abre el modal "Análisis de IA" con 7 pasos y `sessionStorage["emp.analysisId"]` contiene ese id.
- DADO un polling activo, CUANDO pasan 1.000 ms (timers falsos), ENTONCES se ha hecho exactamente una petición más a `GET /ai/analysis/{id}`.
- DADO la respuesta de 8.2.7 con el tercer paso `RUNNING`, CUANDO se renderiza el modal, ENTONCES "Lecturas" y "Baseline" tienen check verde, "Detección" tiene el icono de giro y los otros 4 el círculo vacío.
- DADO `status = COMPLETED` con `summary = { anomalies_detected: 4, high_priority: 2 }`, CUANDO se actualiza el modal, ENTONCES el pie muestra exactamente "4 anomalías detectadas · 2 requieren atención prioritaria", aparece "Ver anomalías", el polling se detiene y `sessionStorage["emp.analysisId"]` no existe.
- DADO `status = FAILED` con `error_message = "step DETECTION: baseline is zero"`, CUANDO se actualiza el modal, ENTONCES el pie muestra "El análisis falló: step DETECTION: baseline is zero" y el botón "Reintentar".
- DADO un polling activo, CUANDO se pulsa "Run AI Analysis" de nuevo, ENTONCES no ocurre nada porque el botón está deshabilitado.
- DADO `POST /ai/analyze` responde 409, CUANDO se procesa la respuesta, ENTONCES aparece el toast "Ya hay un análisis en curso" y el modal se abre con el `analysis_id` del error.
- DADO `sessionStorage["emp.analysisId"]` con un id válido, CUANDO se recarga la página, ENTONCES el chip del header muestra "En curso · …" en ≤ 2 s sin que el usuario pulse nada.

### RF-F-10 — Gráficas: reglas comunes
**Prioridad:** DEBE
**Descripción:** Toda gráfica DEBE implementarse con Recharts 2 dentro de `src/components/charts/` y cumplir: `ResponsiveContainer` al 100 % de ancho; alto 320 px para gráficas principales y 180 px para las pequeñas; ejes con etiquetas en 12 px `#A1A1AA`; líneas de cuadrícula horizontales `#27272A`; tooltip con fondo blanco, borde 1 px `#27272A`, valores formateados con las reglas de RF-F-12; leyenda solo cuando hay más de una serie. El implementador DEBE cargar el skill `dataviz` antes de escribir el primer componente de gráfica y aplicar su método para color y marcas. SI una gráfica no recibe datos (`items = []`), ENTONCES DEBE mostrar en su lugar el texto "Sin datos para mostrar" centrado en la misma altura.

**Criterios de aceptación:**
- DADO cualquier gráfica con `items = []`, CUANDO se renderiza, ENTONCES muestra "Sin datos para mostrar" y no lanza errores en consola.
- DADO la gráfica de consumo diario del dashboard, CUANDO se pasa el puntero por una barra, ENTONCES el tooltip muestra la fecha como `dd MMM yyyy` y el valor como "{n} kWh" con 1 decimal.

### RF-F-11 — Etiquetas en español para enums
**Prioridad:** DEBE
**Descripción:** `src/lib/labels.ts` DEBE exportar las funciones `typeLabel`, `severityLabel`, `statusLabel` (medidor), `anomalyStatusLabel`, `confidenceLabel`, `analysisStatusLabel`, `metricLabel` con estas tablas exactas. Ningún valor de enum de la API DEBE mostrarse en crudo al usuario, salvo `event.type` y `evidence.metric` no mapeados.

| `type` | Etiqueta | Color de badge |
|---|---|---|
| `REAL_ANOMALY` | "Anomalía real" | `#EF4444` |
| `EXPLAINABLE_ANOMALY` | "Anomalía explicable" | `#F59E0B` |
| `FALSE_POSITIVE` | "Falso positivo" | `#A1A1AA` |
| `DATA_QUALITY` | "Calidad de datos" | `#8B5CF6` |

Estados de anomalía (`anomalies.status`) también llevan color y glifo: `OPEN` `#08DDBC`, `INVESTIGATING` `#3B82F6`, `RESOLVED` `#A1A1AA`, `DISMISSED` `#71717A`.

| `severity` | Etiqueta |
|---|---|
| `HIGH` | "Alta" |
| `MEDIUM` | "Media" |
| `LOW` | "Baja" |

| `meters.status` | Etiqueta |
|---|---|
| `OK` | "OK" |
| `ALERT` | "ALERTA" |
| `CRITICAL` | "CRÍTICO" |

| `anomalies.status` | Etiqueta |
|---|---|
| `OPEN` | "Abierta" |
| `INVESTIGATING` | "En investigación" |
| `RESOLVED` | "Resuelta" |
| `DISMISSED` | "Descartada" |

| `confidence` | Etiqueta |
|---|---|
| ≥ 0,90 | "Alta" |
| 0,75 – 0,89 | "Media/Alta" |
| < 0,75 | "Media" |

| `analysis.status` | Etiqueta |
|---|---|
| `QUEUED` | "En cola" |
| `RUNNING` | "En curso" |
| `COMPLETED` | "Completado" |
| `FAILED` | "Falló" |

| `evidence.metric` | Etiqueta |
|---|---|
| `DAILY_CONSUMPTION` | "Consumo diario" |
| `PERSISTENCE_DAYS` | "Días consecutivos" |
| `HOURLY_OUTLIERS` | "Lecturas fuera de patrón horario" |
| `CURRENT_A` | "Corriente" |
| `POWER_FACTOR` | "Factor de potencia" |
| `VOLTAGE_OUT_OF_RANGE` | "Voltaje fuera de rango" |
| `PHYSICAL_RATIO` | "Ratio físico kWh/(V·I·PF)" |
| `RELATED_EVENT` | "Evento relacionado" |
| `INSUFFICIENT_DATA` | "Datos insuficientes" |
| cualquier otro | el valor literal |

Los 9 valores anteriores son el enum completo de `evidence.metric` definido en `spec-energy-motor-anomalias-ia-v1.md` RF-E-14. SI un análisis devuelve un valor no listado, ENTONCES se muestra el literal y se registra `console.warn("metric sin etiqueta")`.

**Criterios de aceptación:**
- DADO `typeLabel("DATA_QUALITY")`, CUANDO se evalúa, ENTONCES devuelve "Calidad de datos".
- DADO `confidenceLabel(0.80)`, CUANDO se evalúa, ENTONCES devuelve "Media/Alta"; `confidenceLabel(0.90)` devuelve "Alta"; `confidenceLabel(0.74)` devuelve "Media".
- DADO `metricLabel("UNKNOWN_METRIC")`, CUANDO se evalúa, ENTONCES devuelve "UNKNOWN_METRIC".

### RF-F-12 — Formato de números y fechas
**Prioridad:** DEBE
**Descripción:** `src/lib/format.ts` DEBE exportar y toda la UI DEBE usar exclusivamente:

| Función | Regla | Ejemplo |
|---|---|---|
| `formatKwh(n)` | separador de miles ".", decimal ",", 1 decimal | `155250.8` → "155.250,8" |
| `formatPct(n)` | 1 decimal, signo explícito "+" o "−" (U+2212), sufijo " %"; `null` → "—" | `110.5` → "+110,5 %"; `-0.1` → "−0,1 %"; `0` → "+0,0 %" |
| `formatPctInt(n)` | porcentaje entero desde fracción, sin signo | `0.93` → "93 %" |
| `formatConfidence(n)` | 2 decimales con "," | `0.98` → "0,98" |
| `formatNumber(n, decimals)` | miles ".", decimal ",", `decimals` decimales | `420.5, 1` → "420,5"; `0.744, 3` → "0,744" |
| `formatDateTime(iso)` | `dd MMM yyyy HH:mm` en UTC, mes en español en minúscula de 3 letras sin punto ("ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic") | `2026-09-25T15:02:11Z` → "25 sep 2026 15:02" |
| `formatDate(isoDate)` | `dd MMM yyyy` | `2026-09-14` → "14 sep 2026" |
| `formatDayShort(isoDate)` | `dd MMM` | `2026-09-14` → "14 sep" |
| `formatRelative(iso, now)` | "hace {n} s" si < 60 s; "hace {n} min" si < 60 min; "hace {n} h" si < 24 h; "hace {n} d" en otro caso | diferencia 120 s → "hace 2 min" |

Las fechas NO se convierten a la zona local del navegador (maestro SUPUESTO-04): se usa `Date.UTC` / `getUTC*`.

**Criterios de aceptación:**
- DADO `formatKwh(155250.8)`, CUANDO se evalúa, ENTONCES devuelve "155.250,8".
- DADO `formatPct(-0.1)`, CUANDO se evalúa, ENTONCES devuelve "−0,1 %" con el signo U+2212.
- DADO `formatDateTime("2026-09-25T15:02:11Z")` en un navegador con zona horaria America/Bogota, CUANDO se evalúa, ENTONCES devuelve "25 sep 2026 15:02".

### RF-F-13 — Accesibilidad mínima
**Prioridad:** DEBE
**Descripción:** Todo texto sobre su fondo DEBE cumplir contraste WCAG AA (≥ 4,5:1 para texto normal, ≥ 3:1 para texto ≥ 18 px o badges). Todo elemento interactivo DEBE tener un indicador de foco visible (anillo de 2 px `#08DDBC`). Toda tabla DEBE usar `<table>` con `<th scope="col">` en cabeceras. Todo botón que solo tenga icono (por ejemplo "✕") DEBE tener `aria-label` en español. Las gráficas DEBEN tener `role="img"` y `aria-label` que describa la serie (ej. "Consumo horario de M-109 con banda baseline").

**Criterios de aceptación:**
- DADO cualquier página, CUANDO se ejecuta la auditoría de accesibilidad de Lighthouse, ENTONCES no hay errores de contraste ni de nombres accesibles faltantes.
- DADO la tabla de medidores, CUANDO se inspecciona el DOM, ENTONCES cada cabecera es `<th scope="col">`.

### RF-F-14 — Componentes de estado reutilizables
**Prioridad:** DEBERÍA
**Descripción:** `Skeleton`, `EmptyState` (icono + texto + botón opcional), `ErrorState` (texto + botón "Reintentar") y `Toast` DEBERÍAN ser componentes únicos reutilizados por todas las páginas para que los textos de error y vacío sean los literales definidos en RF-F-04 a RF-F-09. `ErrorState` DEBE recibir el `ApiError` y, SI `code = "UNAUTHORIZED"`, ENTONCES no renderizar nada (el cliente ya redirige). SI el error es de red (sin respuesta HTTP), ENTONCES el texto DEBE ser "No se pudo conectar con el servidor" en lugar del texto específico de la página.

**Criterios de aceptación:**
- DADO una query que falla sin respuesta HTTP, CUANDO se renderiza cualquier página protegida, ENTONCES aparece "No se pudo conectar con el servidor" y el botón "Reintentar".

## 7. Casos borde y manejo de errores

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-F-01 | El token expira a mitad de sesión (cualquier petición responde 401) | El cliente borra `emp.token` y `emp.user`, limpia la caché de queries y redirige a `/login?expired=1&next={ruta actual}`; el login muestra "Tu sesión expiró. Inicia sesión de nuevo." (RF-F-03). Solo una redirección aunque fallen varias peticiones a la vez. |
| CB-F-02 | La API no responde (error de red, `fetch` rechazado) | La página muestra "No se pudo conectar con el servidor" con botón "Reintentar"; el chip del header muestra "Sin conexión" en `#EF4444`; TanStack Query reintenta 2 veces con 1 s de espera antes de mostrar el error. |
| CB-F-03 | El análisis termina `FAILED` | Modal con "El análisis falló: {error_message}" y botón "Reintentar"; toast con el mismo texto si el modal está cerrado; chip "Falló"; las listas siguen mostrando las anomalías del análisis anterior (el backend no las supersede, maestro CB-M04). |
| CB-F-04 | Medidor sin anomalía vigente (`anomaly = null`) | En `/meters` la columna Anomalía muestra "—"; en el detalle el panel "Anomalía vigente" muestra "Sin anomalía vigente" sin botón. |
| CB-F-05 | Se abre `/anomalies/{id}` de una anomalía de un análisis anterior (`superseded = true`) | Banner "Esta anomalía pertenece a un análisis anterior" con enlace "Ver anomalías vigentes"; botones de acción deshabilitados; el resto de secciones se muestra con los datos históricos. |
| CB-F-06 | Lista vacía tras aplicar filtros o búsqueda | Medidores: "Ningún medidor coincide con los filtros" + "Limpiar filtros". Anomalías: "Ninguna anomalía coincide con los filtros" + "Limpiar filtros". |
| CB-F-07 | Doble clic o clics repetidos en "Run AI Analysis" | El botón se deshabilita en el primer clic (estado local `isStarting`) y permanece deshabilitado MIENTRAS haya polling; solo se envía un `POST`. |
| CB-F-08 | Recarga de página mientras el análisis está en curso | Al montar `App`, se lee `sessionStorage["emp.analysisId"]`; si existe, se reanuda el polling y el chip muestra el progreso; el modal no se abre automáticamente. |
| CB-F-09 | `POST /ai/analyze` responde 409 | Toast "Ya hay un análisis en curso"; el modal se abre adjunto al `analysis_id` devuelto en `error.details.analysis_id`. SI `details` no trae el id, ENTONCES solo se muestra el toast y se invalida `["dashboard"]` para refrescar el chip. |
| CB-F-10 | `GET /ai/analysis/:id` responde 404 durante el polling | Se detiene el polling, se borra `sessionStorage["emp.analysisId"]`, toast "El análisis ya no existe". |
| CB-F-11 | `variation_pct` o `ai_confidence_avg` es `null` | Se muestra "—" (RF-F-05, RF-F-04). |
| CB-F-12 | `readings` devuelve `baseline_profile` vacío (medidor con datos insuficientes, maestro CB-M11) | La gráfica de consumo se dibuja sin banda baseline y muestra bajo el título el texto "Sin baseline: datos insuficientes". |
| CB-F-13 | `related_event` es `null` pero `evidence[]` contiene un ítem `RELATED_EVENT` | "Eventos relacionados" muestra "Sin eventos relacionados"; el ítem de evidencia se muestra en la tabla de evidencia tal cual (el texto de `detail` explica la ausencia). |
| CB-F-14 | El usuario navega a `/login` con sesión activa | Redirección inmediata a `/`. |
| CB-F-15 | `explanation_points` está vacío | La lista bajo "Qué encontró la IA" no se renderiza; solo el párrafo `reason`. |
| CB-F-16 | Anchura de ventana < 1280 px | La app no adapta el layout; aparece una barra superior fija con "Esta aplicación está diseñada para pantallas de al menos 1280 px de ancho". |
| CB-F-17 | `VITE_API_URL` apunta a un origen no permitido por CORS del backend | El navegador bloquea la petición; se trata como CB-F-02 ("No se pudo conectar con el servidor"). El README indica que `CORS_ALLOWED_ORIGINS` del backend debe incluir el origen del frontend. |

## 8. Contratos de datos e interfaces

Este frontend NO define contratos propios de datos: consume exactamente los de maestro §8.2 (8.2.1 login, 8.2.2 lista de medidores, 8.2.3 detalle de medidor, 8.2.4 lecturas, 8.2.5 lista de anomalías, 8.2.6 detalle de anomalía, 8.2.7 análisis, 8.2.8 resumen de dashboard, 8.2.9 cambio de estado, 8.2.10 health). Los tipos de `src/types/api.ts` DEBEN reflejarlos campo por campo.

Contratos internos del frontend:

**Almacenamiento del navegador**

| Clave | Almacén | Contenido | Ejemplo |
|---|---|---|---|
| `emp.token` | `localStorage` | JWT en texto | `eyJhbGciOiJIUzI1NiJ9...` |
| `emp.user` | `localStorage` | JSON `{ email, name }` de 8.2.1 | `{"email":"analista@energy.local","name":"Analista Demo"}` |
| `emp.analysisId` | `sessionStorage` | UUID del análisis en curso | `b7a4c1d2-3e4f-4a5b-8c6d-7e8f9a0b1c2d` |

**Query keys de TanStack Query**

| Clave | Endpoint | `staleTime` |
|---|---|---|
| `["dashboard"]` | 8.2.8 | 30 s |
| `["meters", { status, q, sort, order }]` | 8.2.2 | 30 s |
| `["meter", meterId]` | 8.2.3 | 30 s |
| `["readings", meterId, granularity]` | 8.2.4 | 5 min |
| `["anomalies", { type, severity, status }]` | 8.2.5 | 30 s |
| `["anomaly", id]` | 8.2.6 | 30 s |
| `["analysis", id]` | 8.2.7 | 0 (polling con `refetchInterval: 1000`) |

**Parámetros de URL de páginas**

| Página | Parámetros | Valores |
|---|---|---|
| `/meters` | `status`, `q`, `sort`, `order` | los de 8.2.2 |
| `/anomalies` | `type`, `severity`, `status` | los de 8.2.5 |
| `/login` | `next` | ruta codificada con `encodeURIComponent` |

**Variables de entorno de build**

| Variable | Default | Uso |
|---|---|---|
| `VITE_API_URL` | `http://localhost:8080` | Base URL del backend |

Ejemplo de `ApiError` producido por el cliente ante la respuesta de error del maestro:

```json
{ "status": 404, "code": "METER_NOT_FOUND", "message": "No existe el medidor M-999", "details": null }
```

## 9. Restricciones y decisiones tomadas

Las restricciones de stack (React 18 + TypeScript + Vite + React Router + TanStack Query + Recharts + Tailwind), puertos (frontend 5173, API 8080), idioma (UI en español, enums en inglés), zona horaria (UTC) y prohibición de `expected_results.csv` están en el maestro §9 y aplican íntegramente. Decisiones adicionales propias del frontend:

| Decisión/Restricción | Justificación |
|---|---|
| Token JWT en `localStorage` (no cookie httpOnly) | La API usa cabecera `Authorization: Bearer` (maestro §8.2) y no emite cookies; un solo usuario demo hace aceptable el riesgo de XSS para un MVP local. |
| Polling cada 1.000 ms en lugar de WebSockets/SSE | El análisis dura ≤ 60 s (maestro RNF-02); 60 peticiones ligeras son más simples que mantener una conexión persistente en Go y en el cliente. |
| `analysis_id` en `sessionStorage` y no en `localStorage` | Sobrevive a la recarga (CB-F-08) pero no a cerrar la pestaña, evitando reanudar polling de análisis viejos días después. |
| Ordenamiento de medidores en servidor, de anomalías en servidor sin reordenar en cliente | El `priority_rank` es la decisión de la IA (maestro RF-M02); reordenar en cliente contradiría la priorización que se evalúa. |
| Filtros y orden reflejados en la URL | Permite compartir/recargar la vista durante la demo sin perder el estado. |
| Caddy 2 en la imagen de producción en lugar de `vite preview` o nginx | `vite preview` no está pensado para servir en contenedores; Caddy aporta fallback SPA, compresión y cache de estáticos con un `Caddyfile` de 15 líneas y sin sintaxis de módulos; decisión del usuario (2026-09-25) de preferir Caddy sobre nginx. |
| Color de tipo `DATA_QUALITY` = `#8B5CF6` (violeta de la paleta bia) | Los tres colores semánticos cubren estados y severidades; calidad de datos necesita un color que no se confunda con "crítico" ni "alerta". |
| Paleta de bia.app y tema oscuro único; acento menta `#08DDBC` como "normal" y como acción | Pinned por el usuario (2026-09-25). En una consola de operación lo normal es silencioso y solo lo crítico o lo activo brilla; unificar "sano" y "acción" en el color de marca evita un segundo verde que compita con el menta. Validado con el validador de paleta de dataviz: contraste ≥ 3:1 de los 5 colores semánticos sobre `#0F0F0F`; las gráficas son de una sola serie. |
| Fuente Inter desde Google Fonts (`preconnect` + `display=swap`) | Cierra [PENDIENTE-F-02]: el bundle inicial gzip es de ≈ 197 KB, muy por debajo de RNF-F-01, y la referencia bia.app carga Inter de la misma forma. Si la demo se hace sin red, PUEDE empaquetarse en `public/fonts/` sin cambiar este spec. |
| Mocks MSW en el navegador (`VITE_USE_MOCKS=true`) con `src/mocks/data/dataset.json` generado desde los CSV reales | Decisión del usuario (2026-09-25) para desarrollar y demostrar el frontend antes de que exista el backend. Los mismos handlers se usan en los tests; el dataset se regenera con `npm run mock:dataset`. El modo mock NUNCA se activa en la imagen de producción. |
| Iconografía `lucide-react` en trazo 1,5 px | Un solo sistema de iconos; los badges y celdas usan glifos además de color (accesibilidad y consola). |

## 10. Requisitos no funcionales

Los RNF del maestro §10 que aplican al frontend son RNF-04 (carga inicial ≤ 3 s), RNF-08 (ningún secreto en el repositorio), RNF-11 (Chrome y Edge últimas 2 versiones, ancho mínimo 1280 px) y RNF-12 (`npm run lint` y `tsc --noEmit` sin errores). Adicionales:

| # | Requisito | Métrica |
|---|---|---|
| RNF-F-01 | Tamaño del bundle inicial | ≤ 600 KB gzip para el chunk de entrada + vendors, medido con `npm run build` (salida de Vite) |
| RNF-F-02 | Rendimiento Lighthouse | Puntuación Performance ≥ 80 en `/` con sesión activa, Chrome en modo incógnito, servido desde la imagen Docker |
| RNF-F-03 | Tiempo de respuesta de interacción | Cambiar filtro, orden o pestaña actualiza la tabla en ≤ 500 ms tras la respuesta del backend (P95 ≤ 300 ms según maestro RNF-03) |
| RNF-F-04 | Errores en consola | 0 errores y 0 advertencias de React en la consola del navegador durante el guion de demo |
| RNF-F-05 | Calidad estática | `tsc --noEmit` con `strict: true`; `eslint` con `@typescript-eslint/recommended` y `react-hooks/recommended` sin errores |
| RNF-F-06 | Cobertura de tests | ≥ 60 % de líneas en `src/lib/` y `src/hooks/`; cada página tiene al menos 1 test de render con datos mock |
| RNF-F-07 | Polling acotado | Máximo 1 petición por segundo a `GET /ai/analysis/:id` y ninguna después de un estado final |
| RNF-F-08 | Sin secretos | El repositorio no contiene tokens ni contraseñas fuera de los placeholders del login y de `.env.example` |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| Resueltos en maestro v1.0 | Los antiguos SUPUESTO-F-01 (campo del `analysis_id` en el 409), SUPUESTO-F-02 y SUPUESTO-F-03 (`superseded` y `analysis_id` en anomalías) y PENDIENTE-F-01 (enum de `evidence.metric`) quedaron fijados en el maestro §7 CB-M03, §8.2.5, §8.2.6 y en el motor RF-E-14. Ya no son supuestos: el frontend DEBE usar `error.details.analysis_id`, `anomaly.superseded` y `anomaly.analysis_id` tal como los define el maestro. | — | — |
| Resuelto (antes PENDIENTE-F-02) | Inter se carga desde Google Fonts; bundle inicial ≈ 197 KB gzip tras el primer build (§9). | — | — |
| [DECISIÓN ABIERTA-F-01] | Test E2E con Playwright del guion de demo: hacerlo (más confianza, +1 día) vs. solo recorrido manual. Criterio: si el tiempo restante antes de la fecha de entrega (maestro PENDIENTE-01) es ≥ 2 días, se hace. | Jonnathan Sotelo | Antes de la entrega |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

Herramientas: Vitest 1.x, `@testing-library/react`, `@testing-library/user-event`, `msw` (Mock Service Worker) para simular el backend con las respuestas de ejemplo del maestro §8.2 en `src/test/fixtures/`.

| Requisito | Verificación | Archivo |
|---|---|---|
| RF-F-01 | Test del cliente: cabecera Authorization presente con token; 401 borra token y navega a `/login`; error del backend se convierte en `ApiError` | `src/api/client.test.ts` |
| RF-F-02 | Test de rutas: sin token redirige a `/login?next=`; con token `/login` redirige a `/`; ruta desconocida muestra "Página no encontrada"; chip muestra los 5 textos según fixture | `src/app/router.test.tsx`, `src/app/layout/Header.test.tsx` |
| RF-F-03 | Test de login: botón deshabilitado con campos vacíos; 200 guarda token y navega; 401 muestra "Credenciales inválidas" | `src/pages/LoginPage.test.tsx` |
| RF-F-04 | Test de render con fixture 8.2.8: textos de los 6 tiles, 3 filas de top anomalías; fixture vacío muestra el texto de empty; error muestra "Reintentar" | `src/pages/DashboardPage.test.tsx` |
| RF-F-05 | Test de tabla: formato de fila `M-109`; pestaña "Críticas" envía `status=CRITICAL`; debounce de 300 ms con timers falsos; orden por cabecera; empty con filtros | `src/pages/MetersPage.test.tsx` |
| RF-F-06 | Test de detalle: 4 cards; panel de anomalía vigente; 404 muestra "El medidor M-999 no existe"; selector Día envía `granularity=day` | `src/pages/MeterDetailPage.test.tsx` |
| RF-F-07 | Test de lista: orden `M-109, M-112, M-104, M-106`; filtro tipo envía `type=DATA_QUALITY`; empty sin análisis | `src/pages/AnomaliesPage.test.tsx` |
| RF-F-08 | Test de investigación: 7 `<h2>` en orden; mini-cards de comparación; "Sin eventos relacionados"; badge de origen; PATCH actualiza estado y muestra toast; banner de supersedida | `src/pages/AnomalyDetailPage.test.tsx` |
| RF-F-09 | Test del hook de polling con timers falsos: 1 petición por segundo; se detiene en COMPLETED/FAILED; `sessionStorage` se escribe y borra; 409 abre modal con id del error; recarga reanuda polling | `src/hooks/useAnalysisPolling.test.ts`, `src/components/AnalysisStepper.test.tsx` |
| RF-F-10 | Test de cada componente de gráfica con `items = []` muestra "Sin datos para mostrar" | `src/components/charts/*.test.tsx` |
| RF-F-11 | Test unitario de cada función de `labels.ts` con todos los valores de las tablas | `src/lib/labels.test.ts` |
| RF-F-12 | Test unitario de cada función de `format.ts` con los ejemplos de la tabla, ejecutado con `TZ=America/Bogota` | `src/lib/format.test.ts` |
| RF-F-13 | Auditoría Lighthouse de accesibilidad en `/`, `/meters`, `/anomalies/{id}` | Ensayo de demo |
| RF-F-14 | Test de `ErrorState` con error de red y con `UNAUTHORIZED` | `src/components/ErrorState.test.tsx` |
| CB-F-01…17 | Cada caso borde tiene al menos un test en el archivo de la página correspondiente, nombrado `CB-F-NN` | Archivos anteriores |
| RNF-F-01 | Leer la salida de `npm run build` | CI local |
| RNF-F-02, RNF-F-04 | Lighthouse y consola durante el guion de demo | Ensayo de demo |
| RNF-F-05, RNF-F-06 | `npm run lint`, `tsc --noEmit`, `vitest run --coverage` | CI local |
| Opcional (DECISIÓN ABIERTA-F-01) | Playwright: `e2e/demo.spec.ts` que recorre los 8 pasos de 12b contra `docker compose` | `frontend/e2e/` |

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, autor de la prueba técnica y responsable de la entrega.
- **Qué criterios valida:** RF-F-02, RF-F-03, RF-F-04, RF-F-05, RF-F-06, RF-F-07, RF-F-08, RF-F-09 y RF-F-12, ejecutando el guion de demo siguiente sobre `docker compose up` con el dataset entregado y `DEEPSEEK_API_KEY` configurada (y una segunda pasada sin ella para comprobar el badge "Explicación por plantilla").
- **Nivel de UAT:** Profundo → UAT formal completo: el firmante ejecuta el guion de demo, valida cada criterio listado con resultado pasa/falla y firma el acta de aceptación antes de la entrega.

Guion de demo (lo que debe verse en pantalla en cada paso):

| Paso | Acción | Lo que debe verse |
|---|---|---|
| 1 | Abrir `http://localhost:5173/` | Redirección a `/login`; panel "Energy Management" con campos vacíos y placeholders `analista@energy.local` / `Demo1234!` |
| 2 | Iniciar sesión con las credenciales demo | Dashboard con sidebar "Energy Management"; tile "Medidores" = 12; tile "Consumo" = "155.250,8 kWh"; si aún no hay análisis: "Anomalías IA" = 0, "Confianza IA" = "—", chip "Sin análisis", panel "Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías."; gráfica con 14 barras |
| 3 | Ir a "Medidores", pulsar "Críticas" | Antes del análisis: "Ningún medidor coincide con los filtros". Pulsar "Todos": 12 filas, `M-109` con "2.207,6" y "+110,5 %" |
| 4 | Abrir `M-109` | Cards "2.207,6 kWh", "1.048,8 kWh/día", "+110,5 %"; gráfica horaria con banda baseline y subida visible desde el 12 sep; corriente por encima de la línea de referencia y PF por debajo desde el 12 sep; evento UNKNOWN del 12 sep 14:00 marcado |
| 5 | Pulsar "Run AI Analysis" | Modal "Análisis de IA" con 7 pasos que van completándose; chip "En curso · {paso} n/7"; al terminar: "4 anomalías detectadas · 2 requieren atención prioritaria" y botón "Ver anomalías" |
| 6 | Pulsar "Ver anomalías" | Tabla con 4 filas en orden `M-109` (Anomalía real, Alta, 98 %), `M-112` (Calidad de datos, Alta, 98 %), `M-104` (Anomalía explicable, Media, 95 %), `M-106` (Falso positivo, Baja, 80 %) |
| 7 | Abrir `M-109` | Encabezado con "Anomalía real", "Alta", "Confianza 0,98", "Ventana 12 sep – 14 sep 2026", badge "Explicación generada por IA"; las 7 secciones con sus títulos; "Variables que cambiaron" con Corriente "420,5 A / +109,7 %" y Factor de potencia "0,744 / −20,8 %"; "Sin eventos relacionados"; tabla de evidencia con 5 filas |
| 8 | Pulsar "Marcar en investigación" | Toast "Estado actualizado a En investigación"; "Estado: En investigación"; botón deshabilitado; volver a "Anomalías IA" muestra `M-109` con estado "En investigación"; volver al Dashboard muestra "Anomalías IA" = 4, "Alta prioridad" = 2 en rojo, "Confianza IA" = "93 %", chip "Completado hace {n} min", y en "Medidores" `M-109` con badge "CRÍTICO", `M-104` y `M-112` con "ALERTA", `M-106` con "OK" y badge de anomalía "Baja" |
