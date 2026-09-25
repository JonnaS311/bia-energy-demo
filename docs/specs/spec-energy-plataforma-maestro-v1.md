# Spec: AI Energy Management Platform — Spec maestro

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8: el cambio crea 4 componentes: ingesta de datos, motor de anomalías + IA, backend API, frontend) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> **Cómo leer este conjunto de specs.** Este documento es la **fuente de verdad transversal**: problema, glosario, alcance, modelo de datos, contratos de API, decisiones de stack y requisitos no funcionales. Los cuatro specs hijos detallan cada módulo y **nunca contradicen** lo escrito aquí. Si un hijo necesita cambiar un contrato, se cambia primero aquí.
>
> Orden de lectura para un implementador (humano o agente):
> 1. Este maestro completo.
> 2. `spec-energy-data-ingesta-v1.md` — carga de CSV, baseline, agregados.
> 3. `spec-energy-motor-anomalias-ia-v1.md` — detección, clasificación, severidad, confianza, explicación.
> 4. `spec-energy-backend-api-v1.md` — endpoints, auth, análisis asíncrono.
> 5. `spec-energy-frontend-v1.md` — pantallas y flujo de demo.
> 6. `spec-energy-pruebas-v1.md` — estrategia transversal de pruebas, oráculo del dataset, CI, smoke test y proceso de UAT.

## 1. Contexto y problema

Una empresa que gestiona 12 medidores eléctricos industriales recibe lecturas horarias de consumo (kWh), voltaje (V), corriente (A) y factor de potencia (PF). Hoy esos datos existen como archivos CSV (4.032 lecturas de 14 días) y una lista de 4 eventos operativos conocidos. Nadie puede responder en menos de una hora preguntas como: ¿qué medidor se salió de su comportamiento esperado?, ¿ese cambio es real, lo explica una operación conocida o es un sensor fallando?, ¿cuál debo investigar primero y por qué?

El costo de no resolverlo es doble: (a) una anomalía real como un medidor que duplica su consumo con caída del factor de potencia pasa desapercibida hasta la factura, y (b) el equipo de operaciones gasta tiempo investigando cambios que un evento planificado ya explicaba (una parada programada, una nueva línea productiva).

Esta iniciativa construye un MVP que convierte esos datos en una decisión operativa: un producto web tipo SaaS donde un analista ve el estado de los medidores, ejecuta un análisis de IA, obtiene una lista priorizada de anomalías con tipo, severidad, confianza, explicación sustentada en evidencia y una acción recomendada. El evaluador de la prueba técnica califica el resultado con la rúbrica de la sección 2.

**Datos de entrada (verificados sobre los archivos reales):**

| Archivo | Filas | Columnas exactas | Notas |
|---|---|---|---|
| `readings.csv` | 4.032 | `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status` | 12 medidores (`M-101` … `M-112`) × 336 horas, del `2026-09-01 00:00:00` al `2026-09-14 23:00:00`. Sin nulos, sin duplicados, `status` siempre `OK`. |
| `events.csv` | 4 | `meter_id,event_timestamp,event_type,description` | Tipos presentes: `OPERATIONAL_CHANGE`, `SCHEDULED_OUTAGE`, `UNKNOWN`, `DATA_QUALITY`. Formato de fecha `2026-09-11 00:00` (sin segundos). |
| `expected_results.csv` | — | — | **Reservado al evaluador. NO existe en el repositorio ni DEBE ser leído, referenciado o inferido por ningún componente.** |

## 2. Objetivo y métricas de éxito

Resultado esperado: que una persona sin contexto identifique en menos de 10 minutos qué medidor requiere atención y entienda por qué, a partir de la aplicación y no de los CSV.

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Rúbrica de IA: detecta `M-109` como anomalía real (30 pts) | 0 | El análisis produce una anomalía `type=REAL_ANOMALY`, `severity=HIGH` para `M-109` | Al ejecutar `POST /ai/analyze` sobre el dataset entregado |
| Rúbrica de IA: prioriza `M-109` (25 pts) | 0 | `M-109` es el primer elemento de `GET /anomalies` con el orden por defecto | Ídem |
| Rúbrica de IA: no trata `M-106` como anomalía real (15 pts) | 0 | `M-106` produce `type=FALSE_POSITIVE`, `severity=LOW` | Ídem |
| Rúbrica de IA: detecta `M-112` como calidad de datos (10 pts) | 0 | `M-112` produce `type=DATA_QUALITY`, `severity=HIGH` | Ídem |
| Rúbrica de IA: explica con evidencia (10 pts) | 0 | Cada anomalía tiene ≥ 3 ítems de evidencia con valor observado, valor baseline y delta | Ídem |
| Rúbrica de IA: recomienda acción coherente (10 pts) | 0 | Cada anomalía tiene `recommended_action` no vacío y consistente con la tabla de acciones del spec del motor | Ídem |
| Falsos positivos en medidores normales | — | 0 anomalías en `M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111` | Ídem |
| Demo end-to-end | — | El recorrido `Login → Dashboard → M-109 → Run AI Analysis → Anomalía → Explicación → Acción` se completa en ≤ 10 minutos sin errores en pantalla | Ensayo de demo previo a la entrega |
| Arranque desde cero | — | `docker compose up --build` deja la app usable en ≤ 3 minutos en una máquina con Docker | Ensayo de demo |

Rúbrica global del PDF (referencia para priorizar esfuerzo): Frontend/UX 20, Backend/API 20, Data/Analytics 20, Detección de anomalías 15, IA y explicabilidad 15, Testing/documentación/calidad 10.

## 3. Glosario

| Término | Definición |
|---|---|
| Medidor (`meter`) | Dispositivo que registra consumo eléctrico de una instalación. Identificado por `meter_id` con formato `M-NNN` (ej. `M-109`). |
| Lectura (`reading`) | Registro horario de un medidor con `consumption_kwh`, `voltage_v`, `current_a`, `power_factor`. |
| Periodo | Rango completo de datos: 14 días, del 2026-09-01 al 2026-09-14 inclusive. |
| Ventana de baseline | Los primeros 7 días del periodo (2026-09-01 a 2026-09-07 inclusive). Se usa para calcular el comportamiento "normal" de cada medidor. |
| Baseline diario | Promedio aritmético de los totales diarios de `consumption_kwh` de un medidor en la ventana de baseline. Ej.: `M-109` = 1.048,8 kWh. |
| Perfil horario baseline | Para cada hora del día (0-23), media y desviación estándar muestral de `consumption_kwh` de un medidor en la ventana de baseline. |
| Consumo actual | Total de `consumption_kwh` del último día completo del periodo (2026-09-14). |
| Variación | `(consumo_actual − baseline_diario) / baseline_diario`, expresada en porcentaje con un decimal (ej. `+110,5 %`). |
| Desviación diaria (`dev`) | Igual que variación pero para un día cualquiera: `(total_del_día − baseline_diario) / baseline_diario`. |
| Spike | Un día con `|dev| ≥ 20 %`. |
| Cambio persistente | Dos o más días consecutivos que son spike. |
| Outlier horario | Lectura cuyo z-score respecto al perfil horario baseline es `≥ 3` en valor absoluto. |
| Ratio físico | `consumption_kwh / (voltage_v × current_a × power_factor / 1000)`. En una lectura horaria coherente queda entre 0,75 y 1,30. Fuera de [0,5; 2,0] es físicamente imposible y señala un problema de datos. |
| Corroboración eléctrica | Para un día spike: la corriente media diaria varía ≥ 20 % respecto a la media de baseline **en la misma dirección** que el consumo, o el PF medio diario cae ≥ 0,10 respecto al baseline. |
| Evento operativo (`event`) | Registro de `events.csv`: algo que la operación sabe que ocurrió en un medidor (parada programada, nueva línea, etc.). |
| Evento relacionado | Evento del mismo medidor con `event_type ≠ UNKNOWN` cuyo `event_timestamp` está a ≤ 24 horas del inicio de la anomalía. |
| Anomalía | Resultado del motor para un medidor en un análisis: tipo, severidad, confianza, razón, acción recomendada y evidencia. Un medidor produce como máximo una anomalía por análisis. |
| `REAL_ANOMALY` | Cambio de consumo sin evento que lo explique y, típicamente, con cambios eléctricos coherentes. Requiere investigación. |
| `EXPLAINABLE_ANOMALY` | Cambio de consumo real pero explicado por un evento `OPERATIONAL_CHANGE`. Requiere validar la operación, no investigar el medidor. |
| `FALSE_POSITIVE` | Cambio de consumo explicado por un evento `SCHEDULED_OUTAGE`. No se escala. |
| `DATA_QUALITY` | Las lecturas eléctricas son inconsistentes entre sí o físicamente imposibles aunque el consumo sea estable. Requiere validar sensor/medidor. |
| Severidad | `HIGH`, `MEDIUM` o `LOW`. Cuánta atención requiere. |
| Confianza | Número en [0,50; 0,98] con dos decimales que expresa cuánta evidencia independiente respalda la clasificación. |
| Evidencia | Ítem estructurado (`metric`, `observed`, `baseline`, `delta`, `window`, `detail`) que sustenta la explicación. |
| Análisis (`analysis`) | Una ejecución del pipeline de IA sobre todos los medidores. Tiene `status` y `steps`. |
| Pipeline | Las 7 etapas del análisis: Lecturas → Baseline → Detección → Correlación → Eventos → Explicación → Recomendación. |
| Estado de medidor | `OK`, `ALERT` o `CRITICAL`. Derivado de la anomalía vigente del medidor (ver sección 8.3). |
| Anomalía vigente | La anomalía del último análisis `COMPLETED`. Las de análisis anteriores quedan `superseded = true`. |
| LLM | Modelo de lenguaje (DeepSeek, modelo `deepseek-flash` = DeepSeek-V4.1-Flash, vía su API compatible con OpenAI) usado solo para redactar `reason`, `recommended_action` y `explanation_points`. Nunca decide tipo, severidad ni confianza. |
| Fallback de explicación | Plantilla de texto determinística por tipo de anomalía que se usa cuando el LLM no está disponible, falla o responde algo inválido. |
| PF | Factor de potencia (`power_factor`), adimensional, entre 0 y 1. |
| JWT | JSON Web Token, token firmado que el frontend envía en `Authorization: Bearer <token>`. |

## 4. Actores y casos de uso

| Actor | Rol y permisos |
|---|---|
| Analista de operaciones | Único rol de la aplicación. Inicia sesión con credenciales demo, ve todo, ejecuta análisis y cambia el estado de una anomalía. |
| Evaluador de la prueba | Persona que levanta la app con Docker, sigue el guion de demo y califica con la rúbrica. Usa el rol de analista. |
| Sistema (job de análisis) | Proceso asíncrono dentro del backend que ejecuta el pipeline cuando el analista lo solicita. |

Historias:

- Como analista quiero ver en un dashboard cuántos medidores hay, el consumo total del periodo, cuántas anomalías detectó la IA, cuántas son de alta prioridad, la confianza agregada y cuándo fue el último análisis, para saber en 10 segundos si algo requiere atención.
- Como analista quiero listar los medidores con consumo, variación, estado y severidad, filtrarlos (todos / normales / alertas / críticas), buscarlos por `meter_id` y ordenarlos por consumo, variación o severidad, para localizar el que más se desvía.
- Como analista quiero abrir un medidor y ver consumo actual, baseline, variación, estado, histórico horario y las series de voltaje, corriente y PF con los eventos marcados, para entender su comportamiento.
- Como analista quiero pulsar "Run AI Analysis" y ver el progreso de las 7 etapas, para saber que la IA está trabajando y cuándo terminó.
- Como analista quiero ver la lista de anomalías ordenada por prioridad con tipo, severidad, confianza y acción, para decidir qué investigar primero.
- Como analista quiero abrir una anomalía y leer qué encontró la IA, qué variables cambiaron, la comparación contra baseline, los eventos relacionados, la severidad y confianza, la acción recomendada y la evidencia, para tomar una decisión sin abrir los CSV.
- Como analista quiero marcar una anomalía como "en investigación", "resuelta" o "descartada", para cerrar el ciclo Investigación → Acción.

## 5. Alcance y no-alcance

**Incluye:**
- Ingesta idempotente de `readings.csv` y `events.csv` a PostgreSQL al arrancar.
- Cálculo de baseline diario, perfil horario y agregados diarios por medidor.
- Motor de anomalías determinístico (reglas + estadística) que produce tipo, severidad, confianza, evidencia y priorización.
- Explicación y recomendación en lenguaje natural generadas por LLM con fallback determinístico.
- API HTTP en Go con los 8 endpoints del PDF más login y cambio de estado de anomalía.
- Frontend SaaS en React con login, dashboard, medidores, detalle, anomalías e investigación.
- `docker-compose` que levanta base de datos, API y frontend con un comando.
- Tests unitarios del motor y de la API, README con instrucciones y guion de demo.

**NO incluye (explícito):**
- NO ingesta en tiempo real, streaming, MQTT ni carga de archivos desde la UI. Los CSV se leen del disco al arrancar.
- NO uso, lectura, copia ni inferencia de `expected_results.csv` en código, tests, prompts, datos de semilla ni documentación técnica. Los casos esperados que aparecen en los specs provienen del PDF de la prueba (sección 9 del PDF), no del archivo reservado.
- NO gestión de usuarios: un único usuario demo sembrado desde variables de entorno. NO registro, NO recuperación de contraseña, NO roles.
- NO multi-tenant, NO organizaciones, NO permisos por medidor.
- NO edición ni creación de medidores, lecturas o eventos desde la UI o la API (solo lectura), salvo el cambio de `status` de una anomalía.
- NO modelos de ML entrenados (Isolation Forest, redes, etc.) ni reentrenamiento. La técnica es reglas + estadística descriptiva.
- NO notificaciones externas (email, Slack, SMS).
- NO despliegue en nube, NO CI/CD obligatorio (PUEDE agregarse un workflow de GitHub Actions, no es requisito).
- NO internacionalización: la UI está en español; los identificadores técnicos (enums, campos) en inglés.
- NO paginación en `GET /meters` (12 filas) ni en `GET /anomalies` (≤ 12 filas). `GET /meters/:meterId/readings` sí acepta rango de fechas.

## 6. Requisitos funcionales

Los requisitos de este maestro son de nivel producto (`RF-M`). Cada uno se detalla en un spec hijo con requisitos propios (`RF-D` data, `RF-E` motor/IA, `RF-B` backend, `RF-F` frontend).

### RF-M01 — Ingesta de datos al arranque
**Prioridad:** DEBE
**Descripción:** CUANDO el backend arranca, el sistema DEBE cargar `readings.csv` y `events.csv` en PostgreSQL de forma idempotente y dejar disponibles los 12 medidores, 4.032 lecturas y 4 eventos. Detalle en `spec-energy-data-ingesta-v1.md`.
**Criterios de aceptación:**
- DADO un contenedor de base de datos vacío, CUANDO el backend arranca con los CSV en `./data/`, ENTONCES `SELECT count(*) FROM readings` devuelve 4032, `SELECT count(*) FROM meters` devuelve 12 y `SELECT count(*) FROM events` devuelve 4.
- DADO un backend que ya cargó los datos, CUANDO arranca por segunda vez, ENTONCES los conteos no cambian y no se registra ningún error.

### RF-M02 — Análisis de IA bajo demanda
**Prioridad:** DEBE
**Descripción:** CUANDO el analista invoca `POST /ai/analyze`, el sistema DEBE ejecutar el pipeline de 7 etapas sobre los 12 medidores, persistir un registro en `analyses` con su progreso, y al terminar dejar en `anomalies` exactamente una fila por medidor anómalo con tipo, severidad, confianza, razón, acción y evidencia. Detalle en `spec-energy-motor-anomalias-ia-v1.md` y `spec-energy-backend-api-v1.md`.
**Criterios de aceptación:**
- DADO el dataset entregado, CUANDO el análisis termina con `status=COMPLETED`, ENTONCES existen exactamente 4 anomalías no supersedidas: `M-109 REAL_ANOMALY HIGH`, `M-112 DATA_QUALITY HIGH`, `M-104 EXPLAINABLE_ANOMALY MEDIUM`, `M-106 FALSE_POSITIVE LOW`, y ninguna para los otros 8 medidores.
- DADO el análisis completado, CUANDO se consulta `GET /anomalies`, ENTONCES el orden es `M-109, M-112, M-104, M-106`.

### RF-M03 — Explicación sustentada en evidencia
**Prioridad:** DEBE
**Descripción:** El sistema DEBE producir para cada anomalía un `reason` de 1 a 3 frases, un `recommended_action` de 1 frase, entre 2 y 6 `explanation_points` y ≥ 3 ítems de evidencia estructurada. El LLM redacta texto a partir de la evidencia; SI el LLM no está disponible o falla, ENTONCES el sistema DEBE usar el fallback de explicación sin cambiar tipo, severidad ni confianza.
**Criterios de aceptación:**
- DADO `DEEPSEEK_API_KEY` sin definir, CUANDO se ejecuta el análisis, ENTONCES las 4 anomalías tienen `reason` y `recommended_action` no vacíos, `explanation_source = "template"` y el análisis termina `COMPLETED`.
- DADO `DEEPSEEK_API_KEY` válida, CUANDO se ejecuta el análisis, ENTONCES `explanation_source = "llm"` en las anomalías y los valores de `type`, `severity` y `confidence` son idénticos a los obtenidos sin API key.

### RF-M04 — Estado de medidores derivado del análisis
**Prioridad:** DEBE
**Descripción:** El sistema DEBE derivar el `status` de cada medidor de su anomalía vigente según la tabla de la sección 8.3. MIENTRAS no exista ningún análisis `COMPLETED`, el sistema DEBE reportar todos los medidores en `OK` con `anomaly = null`.
**Criterios de aceptación:**
- DADO el análisis completado sobre el dataset, CUANDO se consulta `GET /meters`, ENTONCES `M-109` tiene `status=CRITICAL`, `M-104` y `M-112` tienen `status=ALERT`, y los 9 restantes tienen `status=OK`.

### RF-M05 — Dashboard de resumen
**Prioridad:** DEBE
**Descripción:** El sistema DEBE exponer `GET /dashboard/summary` con los 6 KPI de la sección 8.4 y el frontend DEBE mostrarlos en la pantalla inicial.
**Criterios de aceptación:**
- DADO el análisis completado, CUANDO se consulta el resumen, ENTONCES `meters_total=12`, `anomalies_total=4`, `high_priority_total=2`, `ai_confidence_avg=0.93`, `total_consumption_kwh=155250.8` (±0,5) y `last_analysis.status="COMPLETED"`.

### RF-M06 — Navegación de producto
**Prioridad:** DEBE
**Descripción:** El frontend DEBE implementar el flujo `Login → Dashboard → Medidores → Detalle → Anomalías IA → Investigación → Acción` con las pantallas y estados de `spec-energy-frontend-v1.md`, y DEBE verse como un producto SaaS (layout con navegación lateral persistente, encabezado con acción "Run AI Analysis" y estado del último análisis).
**Criterios de aceptación:**
- DADO un navegador sin sesión, CUANDO se abre cualquier ruta distinta de `/login`, ENTONCES se redirige a `/login`.
- DADO una sesión iniciada, CUANDO se sigue el guion de demo de la sección 12b, ENTONCES cada paso carga en ≤ 3 segundos y no aparece ningún error en pantalla ni en consola del navegador.

### RF-M07 — Cierre del ciclo con acción
**Prioridad:** DEBERÍA
**Descripción:** CUANDO el analista cambia el estado de una anomalía mediante `PATCH /anomalies/:id/status`, el sistema DEBE persistir el nuevo estado y reflejarlo en la lista de anomalías y en el detalle.
**Criterios de aceptación:**
- DADO una anomalía `OPEN`, CUANDO se envía `{"status":"INVESTIGATING"}`, ENTONCES la respuesta es 200 con `status="INVESTIGATING"` y `GET /anomalies/:id` devuelve lo mismo.

### RF-M08 — Arranque con un comando
**Prioridad:** DEBE
**Descripción:** El repositorio DEBE incluir `docker-compose.yml` y un `Makefile` (o `README` equivalente) tal que `docker compose up --build` levante base de datos, API y frontend, siembre los datos y deje la app accesible en `http://localhost:5173` con la API en `http://localhost:8080`.
**Criterios de aceptación:**
- DADO una máquina con Docker y sin nada más instalado, CUANDO se ejecuta `docker compose up --build`, ENTONCES en ≤ 3 minutos `GET http://localhost:8080/health` responde 200 y `http://localhost:5173/login` carga.

## 7. Casos borde y manejo de errores

Los casos borde transversales; cada hijo agrega los suyos.

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-M01 | `readings.csv` o `events.csv` no existen en `./data/` al arrancar | El backend registra un error `INGEST_FILE_MISSING` con la ruta absoluta, arranca igualmente (sirve `/health` y `/auth/login`) y `GET /meters` devuelve lista vacía. NO se cae el proceso. |
| CB-M02 | PostgreSQL no está disponible al arrancar | El backend reintenta la conexión cada 2 s hasta 30 intentos; si agota, termina con exit code 1 y log `DB_UNAVAILABLE`. |
| CB-M03 | Se invoca `POST /ai/analyze` mientras hay un análisis `QUEUED` o `RUNNING` | 409 `ANALYSIS_IN_PROGRESS` con `error.details = { "analysis_id": "<uuid en curso>" }`. |
| CB-M04 | El pipeline falla en una etapa (excepción no controlada) | El análisis queda `FAILED`, `steps[i].status="FAILED"`, `error_message` con el mensaje; las anomalías del análisis anterior siguen vigentes (no se supersedan). |
| CB-M05 | `DEEPSEEK_API_KEY` ausente, inválida, timeout (> 20 s), error HTTP, o respuesta que no valida el schema | Se usa el fallback de explicación para esa anomalía, `explanation_source="template"`, se registra `LLM_FALLBACK` con la causa, y el análisis termina `COMPLETED`. |
| CB-M06 | Se consulta `GET /meters`, `GET /anomalies` o `GET /dashboard/summary` sin ningún análisis previo | 200 con `status=OK` y `anomaly=null` en todos los medidores; `anomalies=[]`; en el summary `anomalies_total=0`, `ai_confidence_avg=null`, `last_analysis=null`. |
| CB-M07 | Token JWT ausente, inválido o expirado en cualquier endpoint protegido | 401 `UNAUTHORIZED`; el frontend borra la sesión y redirige a `/login`. |
| CB-M08 | `meter_id` inexistente en `GET /meters/:meterId` | 404 `METER_NOT_FOUND`. |
| CB-M09 | Parámetros de query inválidos (`sort` fuera del enum, fechas no ISO, `from > to`) | 400 `INVALID_QUERY` con `details` indicando el parámetro. |
| CB-M10 | Duplicado `(meter_id, timestamp)` dentro del CSV | La última fila del archivo gana; se incrementa `ingest_stats.duplicates` y se registra en log. |
| CB-M11 | Medidor con menos de 7 días de datos (dataset distinto al entregado) | Baseline = promedio de los días disponibles; si hay < 2 días completos, el medidor se marca `insufficient_data=true` y se excluye de la detección, con evidencia `INSUFFICIENT_DATA`. |
| CB-M12 | Baseline diario = 0 | Variación = `null`; el medidor se excluye de la detección por consumo y se evalúa solo calidad de datos. |

## 8. Contratos de datos e interfaces

### 8.1 Modelo de datos (PostgreSQL 16)

Nombres exactos. Todas las tablas en el schema `public`. Timestamps en UTC (`timestamptz`).

```sql
CREATE TABLE meters (
  id            bigserial PRIMARY KEY,
  meter_id      text NOT NULL UNIQUE,            -- 'M-109'
  name          text NOT NULL,                   -- 'Medidor M-109'
  location      text NOT NULL,                   -- 'Planta principal' (valor semilla único; ver [SUPUESTO-01])
  status        text NOT NULL DEFAULT 'OK' CHECK (status IN ('OK','ALERT','CRITICAL')),
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE readings (
  id              bigserial PRIMARY KEY,
  meter_id        text NOT NULL REFERENCES meters(meter_id),
  timestamp       timestamptz NOT NULL,
  consumption_kwh numeric(10,3) NOT NULL,
  voltage_v       numeric(8,3)  NOT NULL,
  current_a       numeric(8,3)  NOT NULL,
  power_factor    numeric(5,3)  NOT NULL,
  status          text NOT NULL DEFAULT 'OK',
  UNIQUE (meter_id, timestamp)
);
CREATE INDEX readings_meter_ts_idx ON readings (meter_id, timestamp);

CREATE TABLE events (
  id          bigserial PRIMARY KEY,
  meter_id    text NOT NULL REFERENCES meters(meter_id),
  timestamp   timestamptz NOT NULL,
  type        text NOT NULL,                     -- 'OPERATIONAL_CHANGE' | 'SCHEDULED_OUTAGE' | 'UNKNOWN' | 'DATA_QUALITY' | otro valor del CSV
  description text NOT NULL DEFAULT ''
);

CREATE TABLE analyses (
  id             uuid PRIMARY KEY,
  status         text NOT NULL CHECK (status IN ('QUEUED','RUNNING','COMPLETED','FAILED')),
  steps          jsonb NOT NULL,                 -- ver 8.2.7
  summary        jsonb,                          -- {"anomalies_detected":4,"high_priority":2}
  error_message  text,
  started_at     timestamptz,
  finished_at    timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE anomalies (
  id                 uuid PRIMARY KEY,
  analysis_id        uuid NOT NULL REFERENCES analyses(id),
  meter_id           text NOT NULL REFERENCES meters(meter_id),
  detected_at        timestamptz NOT NULL,
  window_start       date NOT NULL,              -- primer día anómalo
  window_end         date NOT NULL,              -- último día anómalo
  type               text NOT NULL CHECK (type IN ('REAL_ANOMALY','EXPLAINABLE_ANOMALY','FALSE_POSITIVE','DATA_QUALITY')),
  severity           text NOT NULL CHECK (severity IN ('HIGH','MEDIUM','LOW')),
  confidence         numeric(4,2) NOT NULL CHECK (confidence BETWEEN 0.50 AND 0.98),
  priority_rank      integer NOT NULL,           -- 1 = primero
  reason             text NOT NULL,
  recommended_action text NOT NULL,
  explanation_points jsonb NOT NULL,             -- ["...", "..."]
  explanation_source text NOT NULL CHECK (explanation_source IN ('llm','template')),
  related_event_id   bigint REFERENCES events(id),
  status             text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','INVESTIGATING','RESOLVED','DISMISSED')),
  superseded         boolean NOT NULL DEFAULT false,
  UNIQUE (analysis_id, meter_id)
);

CREATE TABLE anomaly_evidence (
  id          bigserial PRIMARY KEY,
  anomaly_id  uuid NOT NULL REFERENCES anomalies(id) ON DELETE CASCADE,
  position    integer NOT NULL,                  -- orden de presentación, desde 1
  metric      text NOT NULL,                     -- ver enum en spec del motor
  observed    numeric(12,3),
  baseline    numeric(12,3),
  delta_pct   numeric(8,1),
  unit        text NOT NULL,                     -- 'kWh','V','A','','ratio','readings'
  window      text NOT NULL,                     -- '2026-09-12..2026-09-14'
  detail      text NOT NULL                      -- frase legible
);

CREATE TABLE users (
  id            bigserial PRIMARY KEY,
  email         text NOT NULL UNIQUE,
  password_hash text NOT NULL,                   -- bcrypt
  name          text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);
```

Semilla de `meters`: un registro por `meter_id` distinto en `readings.csv`, con `name = 'Medidor ' || meter_id` y `location = 'Planta principal'`.
Semilla de `users`: `DEMO_USER_EMAIL` / `DEMO_USER_PASSWORD` desde variables de entorno (valores por defecto `analista@energy.local` / `Demo1234!`).

### 8.2 Contratos de API

Base URL: `http://localhost:8080`. Todas las respuestas son `application/json; charset=utf-8`. Todos los endpoints salvo `GET /health` y `POST /auth/login` requieren `Authorization: Bearer <jwt>`.

Formato de error único:

```json
{ "error": { "code": "METER_NOT_FOUND", "message": "No existe el medidor M-999", "details": null } }
```

Códigos: `UNAUTHORIZED` (401), `INVALID_CREDENTIALS` (401), `METER_NOT_FOUND` (404), `ANOMALY_NOT_FOUND` (404), `ANALYSIS_NOT_FOUND` (404), `INVALID_QUERY` (400), `INVALID_BODY` (400), `ANALYSIS_IN_PROGRESS` (409), `INGEST_IN_PROGRESS` (503, mientras la ingesta inicial no ha terminado; solo en endpoints de datos), `TOO_MANY_ATTEMPTS` (429, rate limit de login), `NOT_FOUND` (404, ruta inexistente), `METHOD_NOT_ALLOWED` (405), `UNSUPPORTED_MEDIA_TYPE` (415), `PAYLOAD_TOO_LARGE` (413), `REQUEST_TIMEOUT` (503, timeout de 30 s por request), `INTERNAL_ERROR` (500). El detalle de cada uno está en `spec-energy-backend-api-v1.md`.

#### 8.2.1 `POST /auth/login`

Request:
```json
{ "email": "analista@energy.local", "password": "Demo1234!" }
```
Response 200:
```json
{ "token": "eyJhbGciOiJIUzI1NiJ9...", "expires_at": "2026-09-25T20:00:00Z", "user": { "email": "analista@energy.local", "name": "Analista Demo" } }
```
Errores: 401 `INVALID_CREDENTIALS`.

#### 8.2.2 `GET /meters`

Query: `status` ∈ `{OK, ALERT, CRITICAL}` (opcional, repetible), `q` (subcadena de `meter_id`, case-insensitive, opcional), `sort` ∈ `{consumption, variation, severity}` (default `severity`), `order` ∈ `{asc, desc}` (default `desc`).
Orden por `severity`: `CRITICAL` > `ALERT` > `OK`; dentro del mismo estado, severidad de anomalía `HIGH` > `MEDIUM` > `LOW` > sin anomalía; luego `meter_id` asc.

Response 200:
```json
{
  "items": [
    {
      "meter_id": "M-109",
      "name": "Medidor M-109",
      "location": "Planta principal",
      "status": "CRITICAL",
      "current_consumption_kwh": 2207.6,
      "baseline_daily_kwh": 1048.8,
      "variation_pct": 110.5,
      "period_consumption_kwh": 17526.0,
      "anomaly": { "id": "6f1c…", "type": "REAL_ANOMALY", "severity": "HIGH", "confidence": 0.98 }
    },
    {
      "meter_id": "M-101",
      "name": "Medidor M-101",
      "location": "Planta principal",
      "status": "OK",
      "current_consumption_kwh": 728.8,
      "baseline_daily_kwh": 729.4,
      "variation_pct": -0.1,
      "period_consumption_kwh": 10226.1,
      "anomaly": null
    }
  ],
  "total": 12
}
```

#### 8.2.3 `GET /meters/:meterId`

Response 200: el mismo objeto de un ítem de `GET /meters` más:
```json
{
  "…": "…",
  "period": { "start": "2026-09-01", "end": "2026-09-14", "readings_count": 336 },
  "electrical": {
    "voltage_v":    { "current_avg": 216.4, "baseline_avg": 219.9 },
    "current_a":    { "current_avg": 420.5, "baseline_avg": 200.5 },
    "power_factor": { "current_avg": 0.744, "baseline_avg": 0.939 }
  },
  "events": [ { "id": 3, "timestamp": "2026-09-12T14:00:00Z", "type": "UNKNOWN", "description": "No operational event reported" } ],
  "daily": [ { "date": "2026-09-01", "consumption_kwh": 1052.2, "deviation_pct": 0.3 }, { "date": "2026-09-14", "consumption_kwh": 2207.6, "deviation_pct": 110.5 } ]
}
```
`current_avg` = promedio del último día completo; `baseline_avg` = promedio de la ventana de baseline.

#### 8.2.4 `GET /meters/:meterId/readings`

Query: `from`, `to` (fecha-hora ISO 8601 UTC, opcionales, default = periodo completo), `granularity` ∈ `{hour, day}` (default `hour`).

Response 200 (`granularity=hour`):
```json
{
  "meter_id": "M-109",
  "granularity": "hour",
  "baseline_profile": [ { "hour": 0, "mean_kwh": 31.7, "std_kwh": 1.6 }, { "hour": 1, "mean_kwh": 32.7, "std_kwh": 1.6 } ],
  "items": [
    { "timestamp": "2026-09-14T13:00:00Z", "consumption_kwh": 117.51, "voltage_v": 213.43, "current_a": 504.29, "power_factor": 0.741, "status": "OK", "is_outlier": true }
  ]
}
```
`granularity=day` devuelve `items` con `{ "date", "consumption_kwh", "deviation_pct", "voltage_avg", "current_avg", "power_factor_avg" }` y omite `baseline_profile`.

#### 8.2.5 `GET /anomalies`

Query: `type`, `severity`, `status` (opcionales, repetibles), `include_superseded` (bool, default `false`).
Orden fijo: `priority_rank` asc.

Response 200:
```json
{
  "analysis_id": "b7a4…",
  "items": [
    { "id": "6f1c…", "analysis_id": "b7a4…", "superseded": false, "meter_id": "M-109", "type": "REAL_ANOMALY", "severity": "HIGH", "confidence": 0.98, "priority_rank": 1,
      "reason": "El consumo diario de M-109 subió 110,5 % sobre su baseline de 1.048,8 kWh durante 3 días consecutivos sin ningún evento operativo que lo explique; la corriente media diaria se multiplicó por 2,1 y el factor de potencia cayó de 0,94 a 0,74.",
      "recommended_action": "Investigar el medidor y la instalación.",
      "status": "OPEN", "detected_at": "2026-09-25T15:02:11Z", "window_start": "2026-09-12", "window_end": "2026-09-14" },
    { "id": "…", "meter_id": "M-112", "type": "DATA_QUALITY", "severity": "HIGH", "confidence": 0.98, "priority_rank": 2, "…": "…" },
    { "id": "…", "meter_id": "M-104", "type": "EXPLAINABLE_ANOMALY", "severity": "MEDIUM", "confidence": 0.95, "priority_rank": 3, "…": "…" },
    { "id": "…", "meter_id": "M-106", "type": "FALSE_POSITIVE", "severity": "LOW", "confidence": 0.80, "priority_rank": 4, "…": "…" }
  ],
  "total": 4
}
```
Todo ítem incluye `analysis_id` y `superseded` (omitidos en los ítems 2-4 del ejemplo por brevedad).

#### 8.2.6 `GET /anomalies/:id`

Response 200: el ítem de la lista (incluidos `analysis_id` y `superseded`; una anomalía supersedida se devuelve con 200 y `superseded: true`) más:
```json
{
  "…": "…",
  "explanation_source": "llm",
  "explanation_points": [
    "Consumo diario: 2.207,6 kWh el 2026-09-14 frente a un baseline de 1.048,8 kWh (+110,5 %).",
    "Cambio persistente: 3 días consecutivos por encima del 20 % (12, 13 y 14 de septiembre).",
    "Corriente media diaria: 420,5 A frente a 200,5 A de baseline (+109,7 %), coherente con el aumento de consumo.",
    "Factor de potencia medio: 0,744 frente a 0,939 (−0,195); el 13 y 14 cae por debajo de 0,75 en horas de carga.",
    "Sin evento operativo relacionado: el único evento del medidor es de tipo UNKNOWN."
  ],
  "related_event": null,
  "comparison": {
    "consumption_kwh": { "observed": 2207.6, "baseline": 1048.8, "delta_pct": 110.5 },
    "voltage_v":       { "observed": 216.4, "baseline": 219.9, "delta_pct": -1.6 },
    "current_a":       { "observed": 420.5, "baseline": 200.5, "delta_pct": 109.7 },
    "power_factor":    { "observed": 0.744, "baseline": 0.939, "delta_pct": -20.8 }
  },
  "evidence": [
    { "position": 1, "metric": "DAILY_CONSUMPTION", "observed": 2207.6, "baseline": 1048.8, "delta_pct": 110.5, "unit": "kWh", "window": "2026-09-14", "detail": "Total diario 110,5 % por encima del baseline" },
    { "position": 2, "metric": "PERSISTENCE_DAYS", "observed": 3, "baseline": null, "delta_pct": null, "unit": "days", "window": "2026-09-12..2026-09-14", "detail": "3 días consecutivos con desviación ≥ 20 %" },
    { "position": 3, "metric": "CURRENT_A", "observed": 420.5, "baseline": 200.5, "delta_pct": 109.7, "unit": "A", "window": "2026-09-14", "detail": "Corriente media diaria 2,1 veces el baseline" },
    { "position": 4, "metric": "POWER_FACTOR", "observed": 0.744, "baseline": 0.939, "delta_pct": -20.8, "unit": "", "window": "2026-09-14", "detail": "Factor de potencia cae 0,20 respecto al baseline" },
    { "position": 5, "metric": "RELATED_EVENT", "observed": null, "baseline": null, "delta_pct": null, "unit": "", "window": "2026-09-11..2026-09-13", "detail": "Sin evento operativo que explique el cambio (evento UNKNOWN ignorado)" }
  ]
}
```
`comparison.*.observed` = promedio (o total para consumo) del `window_end`; `baseline` = promedio de la ventana de baseline.

#### 8.2.7 `POST /ai/analyze` y `GET /ai/analysis/:id`

`POST /ai/analyze` — body vacío. Response 202:
```json
{ "analysis_id": "b7a4…", "status": "QUEUED", "created_at": "2026-09-25T15:02:00Z" }
```
Errores: 409 `ANALYSIS_IN_PROGRESS` con body:
```json
{ "error": { "code": "ANALYSIS_IN_PROGRESS", "message": "Ya hay un análisis en curso", "details": { "analysis_id": "b7a4…" } } }
```

`GET /ai/analysis/:id` — Response 200:
```json
{
  "id": "b7a4…",
  "status": "RUNNING",
  "steps": [
    { "key": "READINGS",       "label": "Lecturas",       "status": "COMPLETED", "started_at": "…", "finished_at": "…", "detail": "4.032 lecturas de 12 medidores" },
    { "key": "BASELINE",       "label": "Baseline",       "status": "COMPLETED", "started_at": "…", "finished_at": "…", "detail": "Baseline calculado para 12 medidores" },
    { "key": "DETECTION",      "label": "Detección",      "status": "RUNNING",   "started_at": "…", "finished_at": null, "detail": null },
    { "key": "CORRELATION",    "label": "Correlación",    "status": "PENDING",   "started_at": null, "finished_at": null, "detail": null },
    { "key": "EVENTS",         "label": "Eventos",        "status": "PENDING",   "started_at": null, "finished_at": null, "detail": null },
    { "key": "EXPLANATION",    "label": "Explicación",    "status": "PENDING",   "started_at": null, "finished_at": null, "detail": null },
    { "key": "RECOMMENDATION", "label": "Recomendación",  "status": "PENDING",   "started_at": null, "finished_at": null, "detail": null }
  ],
  "summary": null,
  "error_message": null,
  "started_at": "2026-09-25T15:02:00Z",
  "finished_at": null
}
```
Al terminar: `status="COMPLETED"`, `summary = { "anomalies_detected": 4, "high_priority": 2, "meters_analyzed": 12, "explanation_source": "llm" }`. `high_priority` = anomalías con `severity=HIGH`. Estados de step: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`.

#### 8.2.8 `GET /dashboard/summary`

Response 200:
```json
{
  "meters_total": 12,
  "total_consumption_kwh": 155250.8,
  "period": { "start": "2026-09-01", "end": "2026-09-14" },
  "anomalies_total": 4,
  "high_priority_total": 2,
  "ai_confidence_avg": 0.93,
  "meters_by_status": { "OK": 9, "ALERT": 2, "CRITICAL": 1 },
  "last_analysis": { "id": "b7a4…", "status": "COMPLETED", "finished_at": "2026-09-25T15:02:11Z" },
  "daily_consumption": [ { "date": "2026-09-01", "consumption_kwh": 10776.0 }, { "date": "2026-09-14", "consumption_kwh": 12502.4 } ],
  "top_anomalies": [ { "id": "6f1c…", "meter_id": "M-109", "type": "REAL_ANOMALY", "severity": "HIGH", "confidence": 0.98 } ]
}
```
`ai_confidence_avg` = promedio de `confidence` de las anomalías vigentes, redondeado a 2 decimales; `null` si no hay. `top_anomalies` = las primeras 3 por `priority_rank`. `daily_consumption` = suma de todos los medidores por día, los 14 días (el ejemplo muestra solo el primero y el último).

#### 8.2.9 `PATCH /anomalies/:id/status`

Request: `{ "status": "INVESTIGATING" }` con `status` ∈ `{OPEN, INVESTIGATING, RESOLVED, DISMISSED}`. Response 200: el objeto de `GET /anomalies/:id`. Errores: 400 `INVALID_BODY`, 404 `ANOMALY_NOT_FOUND`.

#### 8.2.10 `GET /health`

Response 200:
```json
{
  "status": "ok",
  "db": "ok",
  "ingest": { "status": "COMPLETED", "readings": 4032, "events": 4, "meters": 12, "rejected": 0, "duplicates": 0, "duration_ms": 1840, "started_at": "2026-09-25T15:00:01Z", "finished_at": "2026-09-25T15:00:03Z" }
}
```
`db` ∈ `{ok, error}` (`error` si la BD no responde a `SELECT 1` en 2 s; en ese caso `status="degraded"` y el HTTP es **503**, para que el healthcheck de Docker lo detecte). `ingest.status` ∈ `{PENDING, RUNNING, COMPLETED, FAILED}`. Sin auth. Detalle en `spec-energy-data-ingesta-v1.md` (ingesta) y `spec-energy-backend-api-v1.md` (códigos).

### 8.3 Derivación del estado de medidor

| Anomalía vigente del medidor | `meters.status` |
|---|---|
| Ninguna | `OK` |
| `FALSE_POSITIVE` (cualquier severidad) | `OK` (ver [DECISIÓN ABIERTA-01]) |
| `EXPLAINABLE_ANOMALY` | `ALERT` |
| `DATA_QUALITY` | `ALERT` |
| `REAL_ANOMALY` con `severity=MEDIUM` o `LOW` | `ALERT` |
| `REAL_ANOMALY` con `severity=HIGH` | `CRITICAL` |

El estado se recalcula y persiste en `meters.status` al completar cada análisis.

### 8.4 KPI del dashboard

| KPI | Fuente | Regla |
|---|---|---|
| Medidores | `meters_total` | `count(meters)` |
| Consumo | `total_consumption_kwh` | `sum(readings.consumption_kwh)` del periodo, 1 decimal |
| Anomalías IA | `anomalies_total` | `count(anomalies where superseded=false)` |
| Alta prioridad | `high_priority_total` | ídem con `severity='HIGH'` |
| Confianza IA | `ai_confidence_avg` | promedio de `confidence`, 2 decimales |
| Último análisis | `last_analysis` | análisis más reciente por `created_at`, cualquier estado |

### 8.5 Contrato con el LLM

Entrada (JSON en el mensaje de usuario), salida (JSON validado), prompt de sistema y plantillas de fallback están especificados en `spec-energy-motor-anomalias-ia-v1.md` sección 8. Regla transversal: el LLM recibe **solo** evidencia, evento relacionado y clasificación ya decidida; **nunca** recibe lecturas crudas completas ni `expected_results`.

## 9. Restricciones y decisiones tomadas

| Decisión/Restricción | Justificación |
|---|---|
| Backend en Go 1.22+ | Exigido por el PDF (sección 14). Go no está instalado en la máquina de desarrollo: el README DEBE incluir el paso de instalación y `docker compose` DEBE compilar el backend en contenedor para que el evaluador no dependa de Go local. |
| Router `github.com/go-chi/chi/v5` sobre `net/http` | Ligero, idiomático, middleware estándar; evita frameworks pesados para 10 endpoints. |
| Driver `github.com/jackc/pgx/v5` con `pgxpool`; migraciones con `github.com/golang-migrate/migrate/v4` embebidas en el binario | Estándar de facto; las migraciones corren al arrancar para que el arranque sea un solo comando. |
| PostgreSQL 16 en `docker-compose` | Elegido por el usuario. Volumen persistente `pgdata`. |
| Frontend React 18 + TypeScript + Vite + React Router 6 + TanStack Query 5 + Recharts 2 + Tailwind CSS 3 | Elegido por el usuario. Recharts para gráficas de series; TanStack Query para polling del análisis y caché. |
| Auth JWT HS256 (`github.com/golang-jwt/jwt/v5`), expiración 8 h, secreto en `JWT_SECRET` | Login exigido por el flujo de demo del PDF; un solo usuario demo hace innecesario un proveedor externo. |
| Contraseñas con bcrypt (`golang.org/x/crypto/bcrypt`, cost 10) | Nunca texto plano aunque sea demo. |
| Análisis asíncrono en goroutine dentro del proceso API; un análisis en curso a la vez (mutex + verificación en BD) | 12 medidores y 4.032 lecturas se procesan en segundos; una cola externa sería sobre-ingeniería. |
| Motor determinístico manda; LLM solo redacta | La rúbrica exige reproducibilidad (mismos 4 resultados siempre) y la demo no puede depender de red o cuota de API. |
| LLM: API de DeepSeek (`https://api.deepseek.com/chat/completions`, compatible con OpenAI) mediante `github.com/sashabaranov/go-openai` con `BaseURL` sobrescrita; modelo por defecto `deepseek-flash` (DeepSeek-V4.1-Flash) configurable por `DEEPSEEK_MODEL`; JSON mode (`response_format: {"type":"json_object"}`), `temperature` 0, `max_tokens` 2000, timeout 20 s, 1 reintento | Decisión del usuario (2026-09-25): el explicador usa DeepSeek-V4.1-Flash. No existe SDK oficial de DeepSeek en Go; su API es compatible con OpenAI y `go-openai` es la librería Go más usada para ese contrato. `deepseek-chat` ya no existe en la API (la v1.0 de este spec lo usaba); `GET /models` lista `deepseek-flash` y `deepseek-v4-pro`, ambos con JSON mode. Flash razona brevemente antes de responder y esos tokens cuentan en `max_tokens`; 2000 sobra. |
| Explicación por plantilla cuando no hay LLM | Garantiza que la demo funcione sin `DEEPSEEK_API_KEY`. |
| Ventana de baseline = primeros 7 días del periodo | Todos los cambios del dataset ocurren del día 8 en adelante; 7 días cubren un ciclo semanal completo. Documentado como [SUPUESTO-02]. |
| Voltaje nominal 220 V, banda aceptable ±5 % = [209, 231] V | Los 11 medidores sanos operan entre 216 y 225 V; ±5 % es la tolerancia habitual de red en baja tensión. [SUPUESTO-03]. |
| Timestamps de los CSV se interpretan como UTC | Los archivos no traen zona horaria. [SUPUESTO-04]. |
| Estructura de repositorio: `backend/` (Go), `frontend/` (React), `data/` (CSV), `docs/specs/`, `docker-compose.yml`, `Makefile`, `README.md` | Separación clara por lenguaje; los CSV se mueven de la raíz a `data/` en la implementación. |
| `expected_results.csv` prohibido | Exigencia explícita del PDF (sección 19). Se añade a `.gitignore` por si el evaluador lo coloca en el directorio. |
| Idioma de UI: español; enums y campos de API: inglés | El PDF y el evaluador están en español; los identificadores en inglés siguen el ejemplo JSON del PDF. |
| Puertos: API `8080`, frontend `5173`, PostgreSQL `5432` | Valores por defecto de las herramientas; configurables por variables de entorno. |

Variables de entorno del backend (todas con valor por defecto para `docker compose`):

| Variable | Default | Uso |
|---|---|---|
| `DATABASE_URL` | `postgres://energy:energy@db:5432/energy?sslmode=disable` | Conexión |
| `PORT` | `8080` | Puerto HTTP |
| `DATA_DIR` | `/app/data` | Carpeta con los CSV |
| `JWT_SECRET` | `change-me-in-prod` | Firma JWT |
| `DEMO_USER_EMAIL` | `analista@energy.local` | Usuario semilla |
| `DEMO_USER_PASSWORD` | `Demo1234!` | Contraseña semilla |
| `DEEPSEEK_API_KEY` | (vacío) | Si está vacío → fallback de explicación |
| `DEEPSEEK_MODEL` | `deepseek-flash` | Modelo LLM (DeepSeek-V4.1-Flash) |
| `DEEPSEEK_BASE_URL` | `https://api.deepseek.com` | Base URL de la API (permite apuntar a un mock en tests) |
| `LLM_TIMEOUT_SECONDS` | `20` | Timeout por llamada |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | CORS |
| `LOG_LEVEL` | `info` | Logs estructurados JSON |

## 10. Requisitos no funcionales

| # | Requisito | Métrica |
|---|---|---|
| RNF-01 | Duración del análisis completo sin LLM | ≤ 15 s desde `QUEUED` hasta `COMPLETED` sobre el dataset entregado, en un portátil con 4 vCPU |
| RNF-02 | Duración del análisis completo con LLM | ≤ 60 s (4 llamadas secuenciales de ≤ 20 s cada una en el peor caso) |
| RNF-03 | Latencia de endpoints GET | P95 ≤ 300 ms con 4.032 lecturas, medido con 20 peticiones consecutivas |
| RNF-04 | Carga inicial del frontend | ≤ 3 s hasta dashboard interactivo en red local |
| RNF-05 | Arranque desde cero | `docker compose up --build` ≤ 3 min; arranques posteriores ≤ 30 s |
| RNF-06 | Reproducibilidad del motor | Dos análisis consecutivos sobre los mismos datos producen `type`, `severity`, `confidence`, `priority_rank` y evidencia idénticos (el texto del LLM PUEDE variar) |
| RNF-07 | Cobertura de tests | ≥ 70 % de líneas en `backend/internal/engine`; ≥ 50 % en `backend/internal/http` |
| RNF-08 | Seguridad | Contraseñas con bcrypt; JWT con expiración; CORS restringido a `CORS_ALLOWED_ORIGINS`; ningún secreto en el repositorio (solo en `.env.example`) |
| RNF-09 | Privacidad del LLM | El payload enviado a la API de DeepSeek contiene ≤ 4 KB por anomalía, nunca lecturas crudas completas y ningún dato personal (solo `meter_id`, agregados y eventos) |
| RNF-10 | Observabilidad | Logs JSON con `level`, `msg`, `analysis_id` cuando aplique; cada llamada LLM registra `duration_ms`, `model`, `outcome` ∈ `{ok, timeout, error, invalid_json, fallback}` |
| RNF-11 | Compatibilidad de navegador | Chrome y Edge últimas 2 versiones; ancho mínimo 1280 px (no se exige diseño móvil) |
| RNF-12 | Calidad de código | `go vet` y `golangci-lint run` sin errores; `npm run lint` y `tsc --noEmit` sin errores |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| [SUPUESTO-01] | Los medidores no tienen ubicación real en los datos; se siembra `location = 'Planta principal'` para todos. Si resulta falso, solo cambia la semilla. | Jonnathan Sotelo | — |
| [SUPUESTO-02] | Ventana de baseline = primeros 7 días del periodo. Válido para el dataset entregado; para otro dataset el parámetro `BASELINE_DAYS` PUEDE cambiarse. | Jonnathan Sotelo | — |
| [SUPUESTO-03] | Voltaje nominal 220 V con tolerancia ±5 %. | Jonnathan Sotelo | — |
| [SUPUESTO-04] | Timestamps de los CSV en UTC. La UI muestra las horas tal cual, sin conversión de zona. | Jonnathan Sotelo | — |
| [SUPUESTO-05] | El evaluador ejecuta la app localmente con Docker; no se exige URL pública. | Jonnathan Sotelo | — |
| [DECISIÓN ABIERTA-01] | Un medidor cuya anomalía vigente es `FALSE_POSITIVE` se muestra con `status=OK` y badge de severidad `LOW` (opción elegida por defecto) **vs.** mostrarlo `ALERT`. Criterio: la tabla del PDF (sección 6) no incluye a `M-106`, y "no escalar" sugiere `OK`. Si el evaluador espera `ALERT`, cambiar solo la tabla 8.3. | Jonnathan Sotelo | Antes de la demo |
| [PENDIENTE-01] | Fecha límite de entrega de la prueba técnica. | Jonnathan Sotelo | — |
| [PENDIENTE-02] | ¿La demo será en vivo o grabada? Afecta si se necesita un video en el repositorio. | Jonnathan Sotelo | — |
| Resuelto el 2026-09-25 (antes PENDIENTE-03) | Análisis real con `deepseek-flash`: las 4 llamadas devolvieron JSON válido con `response_format: json_object` (`outcome=ok`, 2,6–4,9 s cada una, ≈ 1.000 tokens de entrada y 400–1.100 de salida), `summary.explanation_source = "llm"`. Texto original del pendiente: confirmar con una llamada real que la versión fijada de `go-openai` envía `response_format: {"type":"json_object"}` aceptado por DeepSeek. Si DeepSeek rechaza el parámetro, se omite y se mantiene la instrucción 6 del prompt de sistema más la validación en Go (comportamiento ya cubierto por CB-M05). | Implementador del motor | Durante la implementación |
| [SUPUESTO-06] | Los datos enviados a DeepSeek (agregados por medidor, sin personas) no están sujetos a restricciones de residencia de datos para esta prueba técnica. | Jonnathan Sotelo | — |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

La estrategia transversal (niveles, herramientas, comandos, cobertura, CI, smoke test, oráculo del dataset) está en `spec-energy-pruebas-v1.md`; esta tabla mapea los requisitos de este maestro a su verificación.

| Requisito | Verificación | Dónde |
|---|---|---|
| RF-M01 | Test de integración: arrancar con BD vacía y comprobar conteos; arrancar dos veces y comprobar idempotencia | `backend/internal/ingest/ingest_test.go` |
| RF-M02 | Test de aceptación del motor sobre `data/readings.csv` y `data/events.csv` que verifica los 4 casos y los 8 sin anomalía; test HTTP de `POST /ai/analyze` + polling hasta `COMPLETED` | `backend/internal/engine/engine_dataset_test.go`, `backend/internal/http/analysis_test.go` |
| RF-M03 | Test con `DEEPSEEK_API_KEY` vacía verificando `explanation_source=template` y campos no vacíos; test con cliente LLM simulado (fake) que devuelve JSON válido, JSON inválido y timeout | `backend/internal/llm/explainer_test.go` |
| RF-M04 | Test unitario de la tabla 8.3 con las 6 combinaciones | `backend/internal/domain/status_test.go` |
| RF-M05 | Test HTTP de `GET /dashboard/summary` tras análisis | `backend/internal/http/dashboard_test.go` |
| RF-M06 | Recorrido manual del guion de demo + `npm run lint` + `tsc --noEmit`; PUEDE añadirse un test E2E con Playwright | `frontend/` |
| RF-M07 | Test HTTP de `PATCH /anomalies/:id/status` con estado válido e inválido | `backend/internal/http/anomalies_test.go` |
| RF-M08 | Ejecutar `docker compose down -v && docker compose up --build` y cronometrar hasta `GET /health` = 200 | Ensayo de demo |
| RNF-01/02 | Medir `finished_at − started_at` del análisis en ambos modos | Log del análisis |
| RNF-03 | Script `scripts/bench.ps1` (o `.sh`) con 20 peticiones a cada GET | `scripts/` |
| RNF-06 | Ejecutar dos análisis y comparar `anomalies` (sin columnas de texto) | Test de integración |
| RNF-07 | `go test ./... -cover` | CI local |
| RNF-12 | `go vet ./...`, `golangci-lint run`, `npm run lint`, `tsc --noEmit` | CI local |

Datos de prueba: los CSV reales en `data/` más fixtures sintéticos en `backend/internal/engine/testdata/` (medidor con caída sin evento, medidor con < 2 días, baseline cero, evento fuera de rango).

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, autor de la prueba técnica y responsable de la entrega.
- **Qué criterios valida:** RF-M02, RF-M03, RF-M04, RF-M05, RF-M06, RF-M07 y RF-M08, ejecutando el guion de demo: `Login → Dashboard → Medidores (filtro "críticas") → M-109 (detalle) → Run AI Analysis (progreso de 7 pasos) → Anomalías IA (M-109 primero) → Investigación de M-109 (evidencia y explicación) → Acción (marcar "INVESTIGATING")`. Adicionalmente valida que `M-106` aparece como `FALSE_POSITIVE` y `M-112` como `DATA_QUALITY`.
- **Nivel de UAT:** Profundo → UAT formal completo: el firmante ejecuta el guion de demo paso a paso, registra cada criterio validado con resultado pasa/falla y firma el acta de aceptación antes de la entrega.
