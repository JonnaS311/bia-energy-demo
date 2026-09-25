# Spec: Motor de anomalías e IA explicativa

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8, heredado del maestro) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> Este spec hijo depende de `spec-energy-plataforma-maestro-v1.md` (en adelante "el maestro"). Los enums, tablas y contratos de API viven en el maestro; aquí se especifica **cómo el motor decide** y **cómo se produce la explicación**. Prefijos de este documento: `RF-E`, `CB-E`, `RNF-E`.

## 1. Contexto y problema

Resumen del maestro, sección 1: 12 medidores industriales generan lecturas horarias de consumo, voltaje, corriente y factor de potencia durante 14 días (4.032 lecturas) y existen 4 eventos operativos conocidos. Nadie puede decir en minutos qué medidor se desvió, si la desviación es real, explicable o un sensor defectuoso, ni cuál investigar primero. Este módulo es el corazón de la respuesta: transforma lecturas y eventos en anomalías con tipo, severidad, confianza, evidencia, explicación y acción. La rúbrica de IA del PDF (100 puntos) evalúa exclusivamente la salida de este módulo: detectar `M-109` (30), priorizarlo (25), no tratar `M-106` como real (15), detectar `M-112` como calidad de datos (10), explicar con evidencia (10) y recomendar acción coherente (10).

## 2. Objetivo y métricas de éxito

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Clasificación de los 4 casos del dataset | 0 de 4 | `M-109 REAL_ANOMALY/HIGH`, `M-112 DATA_QUALITY/HIGH`, `M-104 EXPLAINABLE_ANOMALY/MEDIUM`, `M-106 FALSE_POSITIVE/LOW` | Test de aceptación `engine_dataset_test.go` en cada ejecución de `go test` |
| Falsos positivos en medidores sanos | — | 0 anomalías en `M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111` | Ídem |
| Orden de prioridad | — | `priority_rank` 1..4 = `M-109, M-112, M-104, M-106` | Ídem |
| Confianza calibrada | — | `M-109 = 0.98`, `M-112 = 0.98`, `M-104 = 0.95`, `M-106 = 0.80` (valores exactos con 2 decimales) | Ídem |
| Evidencia por anomalía | — | ≥ 3 ítems con `observed`, `baseline` y `delta_pct` cuando la métrica lo permite | Ídem |
| Explicación disponible sin red | — | 100 % de las anomalías con `reason`, `recommended_action` y ≥ 2 `explanation_points` cuando `DEEPSEEK_API_KEY` está vacía | Test `explainer_test.go` |
| Reproducibilidad | — | Dos ejecuciones sobre los mismos datos producen `type`, `severity`, `confidence`, `priority_rank` y evidencia idénticos | Test de reproducibilidad (RNF-06 del maestro) |

## 3. Glosario

Los términos de dominio (baseline diario, perfil horario, desviación diaria, spike, cambio persistente, outlier horario, ratio físico, corroboración eléctrica, evento relacionado, tipos de anomalía, severidad, confianza, evidencia, fallback de explicación) están definidos en el maestro, sección 3, y se usan aquí con el mismo significado. Términos adicionales de este documento:

| Término | Definición |
|---|---|
| Día | Bloque de 24 lecturas horarias de un medidor con la misma fecha calendario UTC (`YYYY-MM-DD`). |
| Día spike | Día cuya desviación diaria cumple `|dev| ≥ 0,20`. |
| Día DQ | Día en el que el detector de calidad de datos (RF-E-08) dispara. |
| Inicio de anomalía | Fecha del primer día spike o, para `DATA_QUALITY`, del primer día DQ. |
| Ventana de anomalía | `window_start` = inicio de anomalía; `window_end` = último día spike (o último día DQ). |
| `max_dev` | Máximo de `|dev|` entre los días spike del medidor. |
| `signed_dev` | Valor de `dev` (con signo) del día cuyo `|dev|` es máximo. |
| Lectura inconsistente | Lectura con `voltage_v` fuera de [209, 231] o con ratio físico fuera de [0,5; 2,0]. |
| Candidato | Medidor que disparó al menos un detector creador de anomalía (RF-E-04, RF-E-05, RF-E-06 con ≥ 6 outliers, RF-E-08). |

## 4. Actores y casos de uso

Ver maestro, sección 4. Para este módulo el único actor humano es el analista de operaciones, que invoca el análisis y consume sus resultados; el actor técnico es el job de análisis (goroutine del backend) que ejecuta el pipeline. Historia central: "Como analista quiero que el análisis me diga para cada medidor anómalo qué pasó, de qué tipo es, qué tan grave, con cuánta certeza, con qué evidencia y qué hacer, para decidir sin abrir los CSV".

## 5. Alcance y no-alcance

**Incluye:**
- Cálculo de baseline diario, perfil horario, agregados diarios y ratio físico por medidor (a partir de las lecturas ya ingeridas; la ingesta es del spec de data).
- Cinco detectores determinísticos (spike, persistente, outlier horario, eléctrico, calidad de datos).
- Correlación con eventos y clasificación en los 4 tipos con precedencia fija.
- Severidad, confianza, priorización y evidencia estructurada.
- Generación de `reason`, `recommended_action` y `explanation_points` con LLM y fallback por plantilla.
- Emisión del progreso de las 7 etapas del pipeline.
- Tests unitarios, de aceptación sobre el dataset real y fixtures sintéticos.

**NO incluye (explícito):**
- NO lectura de CSV ni acceso a PostgreSQL directo: el motor recibe estructuras en memoria y devuelve estructuras en memoria; la persistencia la hace el backend (spec backend).
- NO endpoints HTTP ni polling: viven en el spec backend.
- NO modelos entrenados (Isolation Forest, ARIMA, redes), NO aprendizaje a partir de resultados previos, NO ajuste automático de umbrales.
- NO uso de `expected_results.csv` en código, tests, fixtures ni prompts. Los valores esperados de los tests provienen de la sección 9 del PDF de la prueba.
- NO envío de lecturas crudas al LLM; solo el payload de la sección 8.1.
- NO detección multi-medidor (correlaciones entre medidores distintos).
- NO detección intradía de eventos de duración < 1 hora.

## 6. Requisitos funcionales

### RF-E-01 — Pipeline de siete etapas
**Prioridad:** DEBE
**Descripción:** CUANDO el backend invoca `Pipeline.Run(ctx, input)`, el motor DEBE ejecutar en orden las etapas de la tabla siguiente, invocar el callback `onStep(key, status, detail)` al iniciar (`RUNNING`) y al terminar (`COMPLETED` o `FAILED`) cada una, y devolver la lista de anomalías con evidencia. SI una etapa falla, ENTONCES el motor DEBE marcar esa etapa `FAILED`, no ejecutar las siguientes y devolver el error.

| # | `key` | `label` | Entrada | Salida | `detail` al completar |
|---|---|---|---|---|---|
| 1 | `READINGS` | Lecturas | `input.Readings` (todas), `input.Events` | Lecturas agrupadas por medidor y día; conteo de días con < 24 lecturas | `"{n_readings} lecturas de {n_meters} medidores"` (ej. `"4.032 lecturas de 12 medidores"`) |
| 2 | `BASELINE` | Baseline | Grupos por medidor | Por medidor: baseline diario, perfil horario, medias de baseline de V/I/PF; marca `insufficient_data` | `"Baseline calculado para {n} medidores"` más `"; {k} con datos insuficientes"` si `k > 0` |
| 3 | `DETECTION` | Detección | Agregados diarios | Por medidor y día: flags de spike, outliers, calidad de datos; por medidor: persistencia, `max_dev` | `"{n} medidores con desviaciones o inconsistencias"` |
| 4 | `CORRELATION` | Correlación | Flags + agregados eléctricos | Por día spike: corroboración eléctrica; por medidor: ratio físico y contadores | `"Corroboración eléctrica evaluada en {n} medidores"` |
| 5 | `EVENTS` | Eventos | Candidatos + `input.Events` | Por candidato: evento relacionado (o `nil`); tipo, severidad, confianza, evidencia, `priority_rank` | `"{n} anomalías clasificadas"` |
| 6 | `EXPLANATION` | Explicación | Anomalías clasificadas | Por anomalía: `reason`, `explanation_points`, `explanation_source` | `"{n_llm} explicaciones por LLM, {n_tpl} por plantilla"` |
| 7 | `RECOMMENDATION` | Recomendación | Anomalías explicadas | Por anomalía: `recommended_action` (del LLM validado o de la tabla 6.9) | `"{n} anomalías · {h} requieren atención prioritaria"` (ej. `"4 anomalías · 2 requieren atención prioritaria"`) |

**Criterios de aceptación:**
- DADO el dataset entregado, CUANDO se ejecuta el pipeline, ENTONCES `onStep` se invoca exactamente 14 veces (7 `RUNNING` + 7 `COMPLETED`) en el orden de la tabla y el `detail` final es `"4 anomalías · 2 requieren atención prioritaria"`.
- DADO un `input.Readings` vacío, CUANDO se ejecuta el pipeline, ENTONCES las 7 etapas terminan `COMPLETED`, la lista de anomalías es vacía y el `detail` de `READINGS` es `"0 lecturas de 0 medidores"`.

### RF-E-02 — Cálculos base por medidor
**Prioridad:** DEBE
**Descripción:** El motor DEBE calcular por medidor, con `BASELINE_DAYS = 7` (constante configurable por variable de entorno `BASELINE_DAYS`, entero ≥ 2):

| Cálculo | Definición exacta |
|---|---|
| Total diario `total[d]` | Suma de `consumption_kwh` de las lecturas del día `d`. |
| Ventana de baseline | Los primeros `BASELINE_DAYS` días calendario distintos del medidor, ordenados ascendentemente. Si el medidor tiene ≤ `BASELINE_DAYS` días completos, aplica CB-E-01. |
| Baseline diario `B` | Promedio aritmético de `total[d]` en la ventana de baseline. |
| Desviación diaria `dev[d]` | `(total[d] − B) / B`. `null` si `B = 0` (ver CB-E-02). |
| Consumo actual | `total[último día calendario del medidor]`. |
| Variación | `dev[último día]`, en porcentaje con 1 decimal. |
| Perfil horario `P[h]` | Para `h ∈ 0..23`: media `μ[h]` y desviación estándar muestral `σ[h]` (denominador `n − 1`) de `consumption_kwh` de las lecturas de la ventana de baseline cuya hora es `h`. Si `n < 2`, `σ[h] = 0`. |
| Media diaria de V, I, PF | Promedio aritmético de `voltage_v`, `current_a`, `power_factor` de las lecturas del día. |
| Media de baseline de V, I, PF | Promedio aritmético sobre todas las lecturas de la ventana de baseline. |
| Ratio físico `r` | Por lectura: `consumption_kwh / (voltage_v × current_a × power_factor / 1000)`. Si el denominador es 0, `r = +∞` (cuenta como fuera de rango). |

**Criterios de aceptación:**
- DADO `M-109` del dataset, CUANDO se calcula el baseline, ENTONCES `B = 1048.8` (±0,1), media de baseline `I = 200.5` (±0,1) y `PF = 0.939` (±0,001).
- DADO `M-109`, CUANDO se calcula `dev` del 2026-09-14, ENTONCES el resultado es `1.105` (±0,001), es decir `+110,5 %`.
- DADO `M-101`, CUANDO se calcula el perfil horario, ENTONCES `σ[h] > 0` para las 24 horas y exactamente 1 lectura tiene `|z| ≥ 3` (`2026-09-11 22:00`, z = 3,13).

### RF-E-03 — Exclusión por datos insuficientes
**Prioridad:** DEBE
**Descripción:** SI un medidor tiene menos de 2 días calendario con ≥ 24 lecturas, ENTONCES el motor DEBE marcarlo `insufficient_data = true`, excluirlo de todos los detectores y NO generar anomalía para él; el backend lo expone con `status = OK` y `variation_pct = null`.
**Criterios de aceptación:**
- DADO un fixture con un medidor de 30 lecturas, CUANDO se ejecuta el pipeline, ENTONCES no existe anomalía para ese medidor y el `detail` de `BASELINE` termina en `"; 1 con datos insuficientes"`.

### RF-E-04 — Detector de spike
**Prioridad:** DEBE
**Descripción:** El motor DEBE marcar como día spike todo día `d` con `|dev[d]| ≥ 0.20`. Un medidor con ≥ 1 día spike es candidato.
**Criterios de aceptación:**
- DADO `M-104`, CUANDO se ejecuta la detección, ENTONCES los días spike son exactamente `2026-09-11, 2026-09-12, 2026-09-13, 2026-09-14` con `dev` entre `+0.459` y `+0.475`.
- DADO `M-106`, ENTONCES el único día spike es `2026-09-08` con `dev = −0.368` (±0,002).
- DADO `M-109`, ENTONCES los días spike son `2026-09-12 (+0.496), 2026-09-13 (+1.092), 2026-09-14 (+1.105)` (±0,002).
- DADO cualquiera de los 8 medidores sanos, ENTONCES no hay días spike (su `max |dev|` está entre 1,2 % y 3,2 %).

### RF-E-05 — Detector de cambio persistente
**Prioridad:** DEBE
**Descripción:** El motor DEBE marcar `persistent = true` cuando existen ≥ 2 días spike consecutivos (fechas calendario adyacentes).
**Criterios de aceptación:**
- DADO `M-104` y `M-109`, ENTONCES `persistent = true`. DADO `M-106`, ENTONCES `persistent = false`.

### RF-E-06 — Detector de outliers horarios
**Prioridad:** DEBE
**Descripción:** El motor DEBE calcular por lectura `z = (consumption_kwh − μ[h]) / σ[h]` (con `σ[h] > 0`; si `σ[h] = 0`, `z = 0`) y marcar `is_outlier = true` cuando `|z| ≥ 3`. Los outliers son **evidencia** (`HOURLY_OUTLIERS`) y NO crean anomalía por sí solos. DONDE un día tenga ≥ 6 outliers y NO sea día spike, el motor DEBE tratar ese día como "patrón horario anormal" y el medidor como candidato de tipo consumo con `max_dev` = `|dev|` de ese día.
**Justificación del umbral 6:** en el dataset, los medidores sanos acumulan como máximo 4 outliers en un día (`M-107`) y 14 en total (`M-107`: 14, `M-105`: 6, `M-110`: 6, `M-102`: 4, `M-108`: 3, `M-101`/`M-103`/`M-111`: 1); 6 por día no dispara en ninguno. Valores medidos en implementación (la v1.0 citaba conteos menores de un script que subcontaba).
**Criterios de aceptación:**
- DADO `M-107`, CUANDO se ejecuta la detección, ENTONCES el medidor tiene 14 lecturas `is_outlier = true`, ningún día con ≥ 6 (máximo 4) y no es candidato.
- DADO `M-109`, ENTONCES el 2026-09-13 y el 2026-09-14 tienen 24 outliers cada uno.
- DADO un fixture sintético con un medidor sin días spike y un día con 8 lecturas de `consumption_kwh = μ[h] × 1.5` (z ≥ 3 en las 8), ENTONCES el medidor es candidato y produce `REAL_ANOMALY` con `severity = MEDIUM`.

### RF-E-07 — Corroboración eléctrica
**Prioridad:** DEBE
**Descripción:** Para cada día spike `d`, el motor DEBE marcar `electrical[d] = true` cuando se cumple al menos una:
- `|I_dia[d] − I_base| / I_base ≥ 0.20` **y** `sign(I_dia[d] − I_base) = sign(dev[d])`; o
- `PF_base − PF_dia[d] ≥ 0.10`.
Donde `I_dia`, `PF_dia` son medias diarias y `I_base`, `PF_base` medias de baseline (RF-E-02).
**Criterios de aceptación:**
- DADO `M-109` el 2026-09-14, ENTONCES `I_dia = 420.5`, `I_base = 200.5` (+109,7 %, mismo signo que `dev`) y `PF_base − PF_dia = 0.195` → `electrical = true`.
- DADO `M-104` el 2026-09-14, ENTONCES `I_dia = 324.7` vs `219.1` (+48,2 %) → `electrical = true`; `PF` cae solo 0,017 (no cumple la segunda condición sola).
- DADO `M-106` el 2026-09-08, ENTONCES `I_dia = 161.6` vs `257.4` (−37,2 %, mismo signo que `dev = −0.368`) → `electrical = true`.

### RF-E-08 — Detector de calidad de datos
**Prioridad:** DEBE
**Descripción:** El motor DEBE marcar un día `d` como día DQ cuando se cumple al menos una:
- (a) ≥ 3 lecturas del día con `voltage_v < 209` o `voltage_v > 231`;
- (b) ≥ 3 lecturas del día con ratio físico `r < 0.5` o `r > 2.0`;
- (c) ≥ 3 lecturas del día con `power_factor < 0.85` **y** `|dev[d]| < 0.10`;
- (d) el día tiene < 24 lecturas, o la ingesta reportó duplicados `(meter_id, timestamp)` para ese día (`input.IngestStats`).
Un medidor con ≥ 1 día DQ es candidato de tipo `DATA_QUALITY`.
**Por qué (b) separa `M-109` de `M-112`:** en `M-109` el consumo, la corriente y el PF se mueven juntos y el ratio físico llega como máximo a 1,47, dentro de [0,5; 2,0]; en `M-112` el consumo es estable pero V, I y PF saltan sin relación entre sí y el ratio llega a 4,22.
**Criterios de aceptación:**
- DADO `M-112`, CUANDO se ejecuta la detección, ENTONCES los días DQ son `2026-09-13` (`vOut = 8`, `rOut = 5`, `pfLow = 5`, `|dev| = 0.0003`) y `2026-09-14` (`vOut = 8`, `rOut = 2`, `pfLow = 7`), y ningún otro.
- DADO `M-109`, ENTONCES `vOut = 0` y `rOut = 0` en los 14 días (ratio máximo 1,47; voltaje mínimo 213,4 V) y no es día DQ aunque tenga `pfLow = 24` el 2026-09-13, porque `|dev| = 1.092 ≥ 0.10`.
- DADO los 8 medidores sanos, ENTONCES ningún día es DQ.
- DADO un fixture con un día de 20 lecturas, ENTONCES ese día es DQ por (d).

### RF-E-09 — Evento relacionado
**Prioridad:** DEBE
**Descripción:** Para cada candidato, el motor DEBE buscar en `input.Events` los eventos del mismo `meter_id` con `type ≠ "UNKNOWN"` y `|event.timestamp − inicio_de_anomalía a las 00:00 UTC| ≤ 24 h`. SI hay varios, ENTONCES DEBE elegir el de menor distancia temporal; en empate, el de mayor precedencia `SCHEDULED_OUTAGE > OPERATIONAL_CHANGE > DATA_QUALITY > otros`. SI no hay ninguno, `related_event = nil`.
**Criterios de aceptación:**
- DADO `M-109` (inicio `2026-09-12`, evento `UNKNOWN` el `2026-09-12 14:00`), ENTONCES `related_event = nil`.
- DADO `M-104` (inicio `2026-09-11`, evento `OPERATIONAL_CHANGE` el `2026-09-11 00:00`), ENTONCES `related_event` es ese evento (distancia 0 h).
- DADO `M-106` (inicio `2026-09-08`, evento `SCHEDULED_OUTAGE` el `2026-09-08 00:00`), ENTONCES `related_event` es ese evento.
- DADO `M-112` (inicio `2026-09-13`, evento `DATA_QUALITY` el `2026-09-13 00:00`), ENTONCES `related_event` es ese evento.

### RF-E-10 — Clasificación con precedencia fija
**Prioridad:** DEBE
**Descripción:** Para cada candidato el motor DEBE asignar `type` evaluando las reglas en este orden y deteniéndose en la primera que cumple:
1. `DATA_QUALITY` — si existe ≥ 1 día DQ (RF-E-08), independientemente de spikes o eventos.
2. `FALSE_POSITIVE` — si `related_event.type = "SCHEDULED_OUTAGE"` **y** `signed_dev < 0` **y** número de días spike ≤ 2.
3. `EXPLAINABLE_ANOMALY` — si `related_event.type = "OPERATIONAL_CHANGE"`.
4. `REAL_ANOMALY` — en cualquier otro caso (incluye: sin evento, evento `UNKNOWN`, `SCHEDULED_OUTAGE` con `signed_dev > 0`, `SCHEDULED_OUTAGE` con > 2 días spike, cualquier otro `type` de evento).
Un medidor produce como máximo una anomalía por análisis; un medidor sin candidatura no produce ninguna.
**Criterios de aceptación:**
- DADO el dataset, ENTONCES `M-109 → REAL_ANOMALY`, `M-112 → DATA_QUALITY`, `M-104 → EXPLAINABLE_ANOMALY`, `M-106 → FALSE_POSITIVE` y los otros 8 medidores no producen anomalía.
- DADO un fixture con un medidor cuyo consumo cae 40 % un día y **no** tiene eventos, ENTONCES `type = REAL_ANOMALY` (no `FALSE_POSITIVE`).
- DADO un fixture con un medidor cuyo consumo **sube** 40 % un día y tiene `SCHEDULED_OUTAGE` ese día, ENTONCES `type = REAL_ANOMALY` porque `signed_dev > 0`.
- DADO un fixture con un medidor con días DQ y además `OPERATIONAL_CHANGE` relacionado, ENTONCES `type = DATA_QUALITY` (regla 1 gana).

### RF-E-11 — Severidad
**Prioridad:** DEBE
**Descripción:** El motor DEBE asignar `severity` así:

| `type` | `severity` |
|---|---|
| `DATA_QUALITY` | `HIGH` |
| `FALSE_POSITIVE` | `LOW` |
| `EXPLAINABLE_ANOMALY` | `MEDIUM` (nunca `HIGH`, aunque `max_dev ≥ 0.50`) |
| `REAL_ANOMALY` | `HIGH` si `max_dev ≥ 0.50`; `MEDIUM` si `0.20 ≤ max_dev < 0.50`; `MEDIUM` si la candidatura proviene solo de patrón horario (RF-E-06) |

**Criterios de aceptación:**
- DADO el dataset, ENTONCES `M-109 HIGH` (`max_dev = 1.105`), `M-112 HIGH`, `M-104 MEDIUM` (`max_dev = 0.475`), `M-106 LOW`.
- DADO un fixture `EXPLAINABLE_ANOMALY` con `max_dev = 0.80`, ENTONCES `severity = MEDIUM`.

### RF-E-12 — Confianza
**Prioridad:** DEBE
**Descripción:** El motor DEBE calcular `confidence` con la fórmula aditiva siguiente, acotar el resultado a `[0.50, 0.98]` y redondear a 2 decimales (redondeo half-up).

Para `type ∈ {REAL_ANOMALY, EXPLAINABLE_ANOMALY, FALSE_POSITIVE}`:

| Término | Condición | Valor |
|---|---|---|
| Base | siempre | `+0.50` |
| Magnitud | `max_dev ≥ 0.50` | `+0.20` |
| Magnitud | `0.20 ≤ max_dev < 0.50` | `+0.05` |
| Persistencia | `persistent = true` | `+0.15` |
| Corroboración eléctrica | `electrical[d] = true` en ≥ 1 día spike | `+0.15` |
| Coherencia | (`EXPLAINABLE_ANOMALY` y `signed_dev > 0`) o (`FALSE_POSITIVE` y `signed_dev < 0`) o (`REAL_ANOMALY` y `related_event = nil` y corroboración eléctrica) | `+0.10` |
| Penalización | `persistent = false` **y** sin corroboración eléctrica | `−0.10` |

Para `type = DATA_QUALITY`:

| Término | Condición | Valor |
|---|---|---|
| Base | siempre | `+0.50` |
| Volumen | suma de lecturas inconsistentes (`vOut + rOut`) en todos los días ≥ 6 | `+0.20` |
| Persistencia | ≥ 2 días DQ | `+0.15` |
| Violación física | ≥ 1 día con `rOut ≥ 3` | `+0.15` |
| Coherencia | `related_event.type = "DATA_QUALITY"` | `+0.10` |

Cálculo desglosado obligatorio sobre el dataset (los tests DEBEN reproducirlo):

| Medidor | Base | Magnitud | Persist. | Eléctrico / físico | Coherencia | Penal. | Suma | `confidence` |
|---|---|---|---|---|---|---|---|---|
| `M-109` | 0,50 | +0,20 (1,105) | +0,15 | +0,15 | +0,10 (sin evento + eléctrico) | 0 | 1,10 | **0,98** (tope) |
| `M-112` | 0,50 | +0,20 (23 lecturas inconsistentes) | +0,15 (2 días) | +0,15 (`rOut = 5` el 09-13) | +0,10 (evento `DATA_QUALITY`) | — | 1,10 | **0,98** (tope) |
| `M-104` | 0,50 | +0,05 (0,475) | +0,15 | +0,15 | +0,10 (`OPERATIONAL_CHANGE`, `dev > 0`) | 0 | 0,95 | **0,95** |
| `M-106` | 0,50 | +0,05 (0,368) | 0 | +0,15 | +0,10 (`SCHEDULED_OUTAGE`, `dev < 0`) | 0 (tiene eléctrico) | 0,80 | **0,80** |

**Criterios de aceptación:**
- DADO el dataset, ENTONCES las confianzas son exactamente `0.98, 0.98, 0.95, 0.80` para `M-109, M-112, M-104, M-106`.
- DADO un fixture `REAL_ANOMALY` de un solo día, sin corroboración eléctrica y `max_dev = 0.30`, ENTONCES `confidence = 0.50` (0,50 + 0,05 − 0,10 = 0,45 → acotado a 0,50).

### RF-E-13 — Priorización
**Prioridad:** DEBE
**Descripción:** El motor DEBE asignar `priority_rank` (1 = primero) ordenando las anomalías por: (1) `severity` `HIGH > MEDIUM > LOW`; (2) `type` `REAL_ANOMALY > DATA_QUALITY > EXPLAINABLE_ANOMALY > FALSE_POSITIVE`; (3) `confidence` descendente; (4) `meter_id` ascendente.
**Criterios de aceptación:**
- DADO el dataset, ENTONCES `priority_rank`: `M-109 = 1`, `M-112 = 2`, `M-104 = 3`, `M-106 = 4`.
- DADO dos `REAL_ANOMALY HIGH` con confianza 0,98 en `M-105` y `M-102`, ENTONCES `M-102` va antes.

### RF-E-14 — Evidencia estructurada
**Prioridad:** DEBE
**Descripción:** El motor DEBE generar por anomalía una lista ordenada de ítems `{position, metric, observed, baseline, delta_pct, unit, window, detail}` con `metric` del enum exacto: `DAILY_CONSUMPTION`, `PERSISTENCE_DAYS`, `HOURLY_OUTLIERS`, `CURRENT_A`, `POWER_FACTOR`, `VOLTAGE_OUT_OF_RANGE`, `PHYSICAL_RATIO`, `RELATED_EVENT`, `INSUFFICIENT_DATA`. `delta_pct = (observed − baseline) / baseline × 100` con 1 decimal cuando `baseline` no es `null` ni 0. `observed` y `baseline` para `CURRENT_A`/`POWER_FACTOR` son la media diaria de `window_end` y la media de baseline. Reglas de generación:

| `type` | Ítems obligatorios (en este orden) | Ítems condicionales |
|---|---|---|
| `REAL_ANOMALY`, `EXPLAINABLE_ANOMALY`, `FALSE_POSITIVE` | 1 `DAILY_CONSUMPTION` (día de `max_dev`); 2 `PERSISTENCE_DAYS` (número de días spike consecutivos, ≥ 1); 3 `CURRENT_A`; 4 `POWER_FACTOR`; último `RELATED_EVENT` (con `detail` del evento o `"Sin evento operativo que explique el cambio"`) | `HOURLY_OUTLIERS` si la ventana tiene ≥ 1 outlier (`observed` = conteo, `unit = "readings"`) |
| `DATA_QUALITY` | 1 `VOLTAGE_OUT_OF_RANGE` (`observed` = conteo total de `vOut`, `unit = "readings"`); 2 `PHYSICAL_RATIO` (`observed` = ratio máximo, `baseline = 1.0`, `unit = "ratio"`); 3 `POWER_FACTOR` (`observed` = PF mínimo de la ventana); 4 `DAILY_CONSUMPTION` (para mostrar que el consumo es estable); último `RELATED_EVENT` | `CURRENT_A` si `|Δ I| ≥ 0.20` |
| Medidor `insufficient_data` | No genera anomalía; el backend PUEDE exponer un único ítem `INSUFFICIENT_DATA` en el detalle del medidor | — |

Toda anomalía DEBE tener ≥ 3 ítems. `detail` es una frase en español de ≤ 140 caracteres con los números formateados con coma decimal y punto de miles.

Ejemplo obligatorio `M-112` (los tests comparan `metric`, `observed`, `baseline`, `delta_pct`, `unit`, `window`; `detail` se compara solo por no vacío):

```json
[
  { "position": 1, "metric": "VOLTAGE_OUT_OF_RANGE", "observed": 16, "baseline": null, "delta_pct": null, "unit": "readings", "window": "2026-09-13..2026-09-14", "detail": "16 lecturas con voltaje fuera de 209-231 V (mín. 201,6 V, máx. 241,2 V)" },
  { "position": 2, "metric": "PHYSICAL_RATIO", "observed": 4.22, "baseline": 1.0, "delta_pct": 322.0, "unit": "ratio", "window": "2026-09-13..2026-09-14", "detail": "7 lecturas con ratio kWh/(V·I·PF) fuera de 0,5-2,0; máximo 4,22" },
  { "position": 3, "metric": "POWER_FACTOR", "observed": 0.58, "baseline": 0.951, "delta_pct": -39.0, "unit": "", "window": "2026-09-13..2026-09-14", "detail": "Factor de potencia mínimo 0,58 frente a 0,951 de baseline; 12 lecturas por debajo de 0,85" },
  { "position": 4, "metric": "DAILY_CONSUMPTION", "observed": 662.7, "baseline": 662.2, "delta_pct": 0.1, "unit": "kWh", "window": "2026-09-14", "detail": "Consumo diario estable (+0,1 %) pese a las lecturas eléctricas inconsistentes" },
  { "position": 5, "metric": "RELATED_EVENT", "observed": null, "baseline": null, "delta_pct": null, "unit": "", "window": "2026-09-13", "detail": "Evento DATA_QUALITY 2026-09-13 00:00: Intermittent readings and abnormal electrical jumps" }
]
```

Ejemplo obligatorio `M-106`:

```json
[
  { "position": 1, "metric": "DAILY_CONSUMPTION", "observed": 852.4, "baseline": 1349.1, "delta_pct": -36.8, "unit": "kWh", "window": "2026-09-08", "detail": "Total diario 36,8 % por debajo del baseline" },
  { "position": 2, "metric": "PERSISTENCE_DAYS", "observed": 1, "baseline": null, "delta_pct": null, "unit": "days", "window": "2026-09-08", "detail": "Un solo día con desviación ≥ 20 %; el 2026-09-09 vuelve al baseline (−0,1 %)" },
  { "position": 3, "metric": "CURRENT_A", "observed": 161.6, "baseline": 257.4, "delta_pct": -37.2, "unit": "A", "window": "2026-09-08", "detail": "Corriente media diaria 37,2 % por debajo del baseline; 12 horas entre 38,7 y 61,1 A" },
  { "position": 4, "metric": "POWER_FACTOR", "observed": 0.916, "baseline": 0.921, "delta_pct": -0.5, "unit": "", "window": "2026-09-08", "detail": "Factor de potencia sin cambio relevante (−0,005)" },
  { "position": 5, "metric": "HOURLY_OUTLIERS", "observed": 12, "baseline": null, "delta_pct": null, "unit": "readings", "window": "2026-09-08", "detail": "12 lecturas fuera del perfil horario (z ≥ 3), todas entre 00:00 y 11:00" },
  { "position": 6, "metric": "RELATED_EVENT", "observed": null, "baseline": null, "delta_pct": null, "unit": "", "window": "2026-09-08", "detail": "Evento SCHEDULED_OUTAGE 2026-09-08 00:00: Scheduled maintenance outage for 12 hours" }
]
```

**Criterios de aceptación:**
- DADO el dataset, ENTONCES `M-109`, `M-104` y `M-106` tienen ≥ 5 ítems y `M-112` tiene 5 ítems, con los `metric` y valores numéricos de los ejemplos (tolerancia ±0,1 en `observed`/`baseline`, ±0,2 en `delta_pct`).
- DADO cualquier anomalía, ENTONCES `position` es 1..n sin huecos y `detail` no está vacío.

### RF-E-15 — Acción recomendada por defecto
**Prioridad:** DEBE
**Descripción:** El motor DEBE asignar `action_default` por tipo con el texto exacto de la tabla; este texto es la `recommended_action` final cuando se usa el fallback y el valor `action_default` que se envía al LLM.

| `type` | `action_default` |
|---|---|
| `REAL_ANOMALY` | `Investigar el medidor y la instalación.` |
| `DATA_QUALITY` | `Validar el sensor y la calidad de las lecturas del medidor.` |
| `EXPLAINABLE_ANOMALY` | `Validar con operaciones que el cambio corresponde al evento registrado.` |
| `FALSE_POSITIVE` | `No escalar; el cambio está explicado por la parada programada.` |

**Criterios de aceptación:**
- DADO `DEEPSEEK_API_KEY` vacía, CUANDO se ejecuta el análisis, ENTONCES `recommended_action` de `M-109` es exactamente `Investigar el medidor y la instalación.`

### RF-E-16 — Explicación por LLM
**Prioridad:** DEBE
**Descripción:** DONDE `DEEPSEEK_API_KEY` no esté vacía, el motor DEBE, por cada anomalía y en secuencia, enviar al LLM el payload de la sección 8.1 con el prompt de sistema 8.2, validar la respuesta contra el schema 8.3 y, si valida, asignar `reason`, `recommended_action`, `explanation_points` y `explanation_source = "llm"`. El LLM NO DEBE poder alterar `type`, `severity`, `confidence`, `priority_rank` ni la evidencia: esos campos se fijan antes de la etapa `EXPLANATION` y no se leen de la respuesta.
**Criterios de aceptación:**
- DADO un cliente LLM simulado que devuelve un JSON válido, CUANDO se explica `M-109`, ENTONCES `explanation_source = "llm"`, `reason` es el del JSON y `type/severity/confidence` son iguales a los calculados por RF-E-10/11/12.
- DADO el mismo cliente devolviendo `{"reason":"…","recommended_action":"…","explanation_points":["a","b"],"severity":"LOW"}`, ENTONCES `severity` de la anomalía sigue siendo la calculada y el campo extra se ignora.

### RF-E-17 — Fallback de explicación
**Prioridad:** DEBE
**Descripción:** SI `DEEPSEEK_API_KEY` está vacía, o la llamada al LLM falla (timeout, error HTTP, error de red), o la respuesta no valida el schema 8.3 tras 1 reintento, ENTONCES el motor DEBE generar `reason` y `explanation_points` con las plantillas de la sección 8.4, `recommended_action = action_default`, `explanation_source = "template"`, registrar un log `LLM_FALLBACK` con `meter_id` y `cause ∈ {no_api_key, timeout, http_error, invalid_json, schema_invalid}`, y continuar con la siguiente anomalía. El análisis termina `COMPLETED`.
**Criterios de aceptación:**
- DADO `DEEPSEEK_API_KEY` vacía, ENTONCES las 4 anomalías tienen `explanation_source = "template"`, `reason` no vacío y ≥ 2 `explanation_points`, y no se realiza ninguna llamada de red.
- DADO un cliente simulado que devuelve texto no JSON dos veces seguidas, ENTONCES la anomalía usa plantilla y el log contiene `cause=invalid_json`.
- DADO un cliente simulado que tarda 25 s, ENTONCES la llamada se cancela a los 20 s, se reintenta una vez, y si vuelve a tardar se usa plantilla con `cause=timeout`; el análisis completa en ≤ 60 s.

### RF-E-18 — Reproducibilidad
**Prioridad:** DEBE
**Descripción:** El motor DEBE ser determinístico: MIENTRAS las lecturas, eventos y parámetros no cambien, dos ejecuciones DEBEN producir `type`, `severity`, `confidence`, `priority_rank`, `window_start`, `window_end`, `related_event_id` y la lista de evidencia (sin `detail`) idénticos. El motor NO DEBE usar aleatoriedad ni depender del reloj para decidir; `detected_at` es el único campo que cambia.
**Criterios de aceptación:**
- DADO dos ejecuciones consecutivas sobre el dataset con `DEEPSEEK_API_KEY` vacía, ENTONCES la serialización JSON de las anomalías (excluyendo `id`, `analysis_id`, `detected_at`) es byte a byte idéntica.

### RF-E-19 — Estructura de código
**Prioridad:** DEBERÍA
**Descripción:** El motor DEBERÍA organizarse en `backend/internal/engine/{baseline.go, detectors.go, classify.go, confidence.go, evidence.go, pipeline.go}` y la explicación en `backend/internal/llm/{client.go, explainer.go, templates.go}`, con una interfaz `Explainer` que permita inyectar un cliente simulado en tests. Los umbrales DEBERÍAN vivir en una struct `Thresholds` con valores por defecto (`DevSpike = 0.20`, `DevHigh = 0.50`, `ZOutlier = 3.0`, `OutliersPerDay = 6`, `CurrentDelta = 0.20`, `PFDrop = 0.10`, `VoltageMin = 209`, `VoltageMax = 231`, `RatioMin = 0.5`, `RatioMax = 2.0`, `MinInconsistent = 3`, `EventWindowHours = 24`, `BaselineDays = 7`).
**Criterios de aceptación:**
- DADO el paquete `engine`, CUANDO se construye con `Thresholds{}` vacío, ENTONCES se aplican los valores por defecto y el test de aceptación pasa.

## 7. Casos borde y manejo de errores

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-E-01 | Medidor con < 2 días completos (≥ 24 lecturas) | `insufficient_data = true`, sin anomalía (RF-E-03). Si tiene entre 2 y 6 días completos, la ventana de baseline son todos los días disponibles menos el último; se registra log `BASELINE_SHORT`. |
| CB-E-02 | Baseline diario `B = 0` (todas las lecturas de baseline en 0) | `dev[d] = null` para todos los días; el medidor no puede ser candidato por consumo (RF-E-04/05/06); sí se evalúa RF-E-08. `variation_pct = null`. |
| CB-E-03 | `σ[h] = 0` en el perfil horario | `z = 0` para esa hora; nunca outlier. |
| CB-E-04 | Denominador del ratio físico = 0 (`voltage_v`, `current_a` o `power_factor` = 0) | La lectura cuenta como inconsistente (`rOut`). |
| CB-E-05 | Evento cuyo `meter_id` no existe en las lecturas | Se ignora en la correlación; log `EVENT_ORPHAN` con el `meter_id`. |
| CB-E-06 | Dos o más eventos del mismo medidor dentro de ±24 h del inicio | Gana el de menor distancia temporal; en empate, precedencia `SCHEDULED_OUTAGE > OPERATIONAL_CHANGE > DATA_QUALITY > otros`. Solo uno es `related_event`. |
| CB-E-07 | Evento a más de 24 h del inicio de anomalía | Se ignora; `related_event = nil`; la anomalía se clasifica como si no hubiera evento. |
| CB-E-08 | Evento `UNKNOWN` | Nunca es `related_event`; se muestra en el detalle del medidor pero no influye en la clasificación. `M-109` es el caso real. |
| CB-E-09 | Evento con `type` no contemplado (ej. `MAINTENANCE`) dentro de ±24 h | Es `related_event` (para mostrarlo) pero no activa las reglas 2 ni 3 de RF-E-10; el tipo resultante es `REAL_ANOMALY` o `DATA_QUALITY`. |
| CB-E-10 | `SCHEDULED_OUTAGE` relacionado pero `signed_dev > 0` | `REAL_ANOMALY` (una parada no explica una subida). |
| CB-E-11 | `SCHEDULED_OUTAGE` relacionado, `signed_dev < 0`, pero > 2 días spike | `REAL_ANOMALY` con evidencia `RELATED_EVENT` indicando que la parada declarada no cubre la duración observada. |
| CB-E-12 | Medidor con días DQ **y** días spike | `DATA_QUALITY` (regla 1). La evidencia incluye `DAILY_CONSUMPTION` con el `max_dev` para que el analista vea ambas señales. |
| CB-E-13 | LLM responde JSON válido pero `explanation_points` con 1 o con 7 elementos, o `reason` vacío | `schema_invalid` → reintento una vez → fallback. |
| CB-E-14 | LLM responde JSON con campos adicionales (`severity`, `type`, `confidence`) | Se ignoran; la clasificación no cambia (RF-E-16). |
| CB-E-15 | LLM responde con `finish_reason = "length"` (cortado por `max_tokens`) o `"content_filter"`, o con `choices` vacío | Se trata como `schema_invalid`. |
| CB-E-16 | HTTP 401/403 del LLM (clave inválida) | `http_error`; NO se reintenta (el reintento no cambiaría el resultado); fallback para esa anomalía y para todas las siguientes del mismo análisis (se marca el cliente como no disponible durante el análisis). |
| CB-E-17 | HTTP 429 o 5xx del LLM | `go-openai` no reintenta por sí solo: el motor espera 2 s y reintenta una vez (el mismo reintento único de RF-E-17); si persiste, `http_error` y fallback. |
| CB-E-18 | Timeout de 20 s | `timeout` → 1 reintento → fallback. |
| CB-E-19 | Análisis re-ejecutado | El motor no sabe de análisis previos; el backend marca `superseded = true` en las anomalías del análisis anterior al persistir las nuevas (spec backend). |
| CB-E-20 | Lecturas con `timestamp` fuera de orden en `input.Readings` | El motor ordena por `(meter_id, timestamp)` antes de agrupar; el resultado no depende del orden de entrada. |
| CB-E-21 | Medidor presente en `meters` pero sin lecturas | No es candidato; `insufficient_data = true`. |
| CB-E-22 | `BASELINE_DAYS` configurado con valor < 2 o no numérico | Se usa 7 y se registra log `CONFIG_INVALID`. |
| CB-E-23 | Contexto cancelado (`ctx.Done()`) durante el pipeline | La etapa en curso termina `FAILED` con `error_message = "context canceled"`; no se persisten anomalías parciales. |

## 8. Contratos de datos e interfaces

### 8.0 Interfaz Go del motor

```go
package engine

type Input struct {
    Readings    []Reading      // todas las lecturas del periodo
    Events      []Event
    IngestStats IngestStats    // duplicados y huecos reportados por la ingesta, por meter_id y día
    Thresholds  Thresholds
}

type StepCallback func(key, status, detail string)

type Result struct {
    Anomalies []Anomaly         // ordenadas por PriorityRank
    Meters    []MeterSummary    // baseline, variación, insufficient_data por medidor
}

func (p *Pipeline) Run(ctx context.Context, in Input, onStep StepCallback) (Result, error)
```

`Anomaly` contiene exactamente los campos de la tabla `anomalies` del maestro (sección 8.1) más `Evidence []EvidenceItem` y `ActionDefault string`; `EvidenceItem` contiene los campos de `anomaly_evidence`.

### 8.1 Payload de entrada al LLM

Un mensaje de usuario con el JSON siguiente (≤ 4 KB; nunca lecturas crudas). Ejemplo real para `M-109`:

```json
{
  "meter_id": "M-109",
  "classification": { "type": "REAL_ANOMALY", "severity": "HIGH", "confidence": 0.98 },
  "window": { "start": "2026-09-12", "end": "2026-09-14", "baseline_window": "2026-09-01..2026-09-07" },
  "comparison": {
    "consumption_kwh": { "observed": 2207.6, "baseline": 1048.8, "delta_pct": 110.5 },
    "voltage_v":       { "observed": 216.4, "baseline": 219.9, "delta_pct": -1.6 },
    "current_a":       { "observed": 420.5, "baseline": 200.5, "delta_pct": 109.7 },
    "power_factor":    { "observed": 0.744, "baseline": 0.939, "delta_pct": -20.8 }
  },
  "evidence": [
    { "metric": "DAILY_CONSUMPTION", "observed": 2207.6, "baseline": 1048.8, "delta_pct": 110.5, "unit": "kWh", "window": "2026-09-14", "detail": "Total diario 110,5 % por encima del baseline" },
    { "metric": "PERSISTENCE_DAYS", "observed": 3, "baseline": null, "delta_pct": null, "unit": "days", "window": "2026-09-12..2026-09-14", "detail": "3 días consecutivos con desviación ≥ 20 %" },
    { "metric": "CURRENT_A", "observed": 420.5, "baseline": 200.5, "delta_pct": 109.7, "unit": "A", "window": "2026-09-14", "detail": "Corriente media diaria 2,1 veces el baseline" },
    { "metric": "POWER_FACTOR", "observed": 0.744, "baseline": 0.939, "delta_pct": -20.8, "unit": "", "window": "2026-09-14", "detail": "Factor de potencia cae 0,20 respecto al baseline" },
    { "metric": "RELATED_EVENT", "observed": null, "baseline": null, "delta_pct": null, "unit": "", "window": "2026-09-11..2026-09-13", "detail": "Sin evento operativo que explique el cambio (evento UNKNOWN ignorado)" }
  ],
  "related_event": null,
  "action_default": "Investigar el medidor y la instalación."
}
```

Cuando hay evento relacionado: `"related_event": { "type": "OPERATIONAL_CHANGE", "timestamp": "2026-09-11T00:00:00Z", "description": "New production line activated" }`.

### 8.2 Prompt de sistema (texto exacto)

```
Eres un analista senior de gestión energética. Recibes la clasificación ya decidida de una anomalía en un medidor eléctrico industrial, junto con la evidencia numérica que la sustenta.

Tu tarea es redactar, en español y para un operador de planta, una explicación breve y una acción recomendada.

Reglas obligatorias:
1. No cambies ni cuestiones la clasificación (type, severity, confidence): ya está decidida por un motor determinístico.
2. Usa únicamente los números que aparecen en "comparison" y "evidence". No inventes cifras, fechas ni causas que no estén en los datos.
3. Si "related_event" es null, di explícitamente que no hay evento operativo que explique el cambio.
4. Si "related_event" existe, menciónalo con su tipo y descripción y explica cómo se relaciona con el cambio.
5. "recommended_action" debe ser una sola frase imperativa coherente con "action_default"; puedes precisarla con datos de la evidencia, pero no cambiar su sentido (por ejemplo, no recomiendes escalar un FALSE_POSITIVE).
6. Responde ÚNICAMENTE con un objeto JSON válido, sin texto antes ni después, sin bloques de código, con esta forma exacta:
{"reason": "<1 a 3 frases>", "recommended_action": "<1 frase>", "explanation_points": ["<punto 1>", "<punto 2>", "... entre 2 y 6 puntos"]}
7. Cada punto de "explanation_points" cita al menos un número de la evidencia con su unidad.
8. Formato numérico: coma decimal y punto de miles (ej. 1.048,8 kWh; 110,5 %).
```

### 8.3 Schema de salida del LLM y validación

```json
{
  "type": "object",
  "additionalProperties": true,
  "required": ["reason", "recommended_action", "explanation_points"],
  "properties": {
    "reason": { "type": "string", "minLength": 20, "maxLength": 600 },
    "recommended_action": { "type": "string", "minLength": 10, "maxLength": 200 },
    "explanation_points": { "type": "array", "minItems": 2, "maxItems": 6, "items": { "type": "string", "minLength": 10, "maxLength": 300 } }
  }
}
```

Validación en Go (`explainer.go`), en este orden; cualquier fallo → `schema_invalid`:
1. Tomar `choices[0].message.content`; recortar espacios; si empieza por "```", quitar el cerco de código.
2. `json.Unmarshal` a `struct { Reason string; RecommendedAction string; ExplanationPoints []string }` — fallo → `invalid_json`.
3. Longitudes según el schema. `reason` DEBE contener entre 1 y 3 frases (contar terminadores `.`, `!`, `?` seguidos de espacio o fin; ≥ 1 y ≤ 3).
4. `recommended_action` DEBE terminar en `.` y no contener saltos de línea.
5. DEBERÍA: cada número decimal citado en `explanation_points` (regex `\d{1,3}(\.\d{3})*(,\d+)?`) corresponde, tras normalizar a punto decimal, a algún `observed`, `baseline`, `delta_pct` o conteo de la evidencia con tolerancia ±0,1; si no, se registra log `LLM_NUMBER_MISMATCH` pero NO se rechaza la respuesta.
6. Campos adicionales (`type`, `severity`, `confidence`, otros) se ignoran.

### 8.4 Plantillas de fallback (texto exacto)

Placeholders: `{meter_id}`, `{delta_pct}` (con signo y 1 decimal, ej. `+110,5`), `{baseline}` (kWh, 1 decimal), `{observed}` (kWh, 1 decimal), `{days}` (entero), `{event_description}`, `{event_date}` (`YYYY-MM-DD HH:MM`), `{current_delta_pct}`, `{pf_observed}`, `{pf_baseline}`, `{v_out_count}`, `{ratio_max}`, `{pf_min}`, `{window_start}`, `{window_end}`.

| `type` | `reason` | `explanation_points` |
|---|---|---|
| `REAL_ANOMALY` | `El consumo diario de {meter_id} está {delta_pct} % respecto a su baseline de {baseline} kWh durante {days} día(s) consecutivo(s), sin ningún evento operativo que lo explique. La corriente media varió {current_delta_pct} % y el factor de potencia pasó de {pf_baseline} a {pf_observed}.` | `["Consumo diario: {observed} kWh el {window_end} frente a {baseline} kWh de baseline ({delta_pct} %).", "Duración: {days} día(s) consecutivo(s) con desviación ≥ 20 % ({window_start} a {window_end}).", "Corriente media diaria: {current_delta_pct} % respecto al baseline.", "Factor de potencia: {pf_observed} frente a {pf_baseline} de baseline.", "Sin evento operativo registrado que explique el cambio."]` |
| `EXPLAINABLE_ANOMALY` | `El consumo diario de {meter_id} está {delta_pct} % respecto a su baseline de {baseline} kWh durante {days} día(s), y el cambio coincide con el evento operativo del {event_date}: {event_description}.` | `["Consumo diario: {observed} kWh el {window_end} frente a {baseline} kWh de baseline ({delta_pct} %).", "Duración: {days} día(s) consecutivo(s) desde {window_start}.", "Corriente media diaria: {current_delta_pct} %, coherente con el cambio de consumo.", "Evento relacionado el {event_date}: {event_description}."]` |
| `FALSE_POSITIVE` | `El consumo diario de {meter_id} bajó {delta_pct} % respecto a su baseline de {baseline} kWh el {window_start}, y la caída está explicada por la parada programada del {event_date}: {event_description}. No se requiere escalamiento.` | `["Consumo diario: {observed} kWh el {window_start} frente a {baseline} kWh de baseline ({delta_pct} %).", "Duración: {days} día(s); el consumo vuelve al baseline después.", "Corriente media diaria: {current_delta_pct} %, consistente con equipos apagados.", "Evento relacionado el {event_date}: {event_description}."]` |
| `DATA_QUALITY` | `Las lecturas eléctricas de {meter_id} entre {window_start} y {window_end} son inconsistentes entre sí: {v_out_count} lecturas con voltaje fuera de 209-231 V, ratio kWh/(V·I·PF) de hasta {ratio_max} y factor de potencia mínimo de {pf_min}, mientras el consumo diario se mantiene estable ({delta_pct} %). Esto apunta a un problema del sensor o de la transmisión de datos, no del consumo.` | `["Voltaje: {v_out_count} lecturas fuera del rango 209-231 V.", "Relación física: ratio kWh/(V·I·PF) máximo de {ratio_max} (rango coherente 0,5-2,0).", "Factor de potencia mínimo: {pf_min} frente a {pf_baseline} de baseline.", "Consumo diario estable: {observed} kWh frente a {baseline} kWh ({delta_pct} %).", "Evento relacionado el {event_date}: {event_description}."]` (si no hay evento, el último punto es `"Sin evento operativo registrado."`) |

`recommended_action` en fallback = `action_default` (RF-E-15).

### 8.5 Cliente LLM

- Proveedor: **DeepSeek**, API compatible con OpenAI en `https://api.deepseek.com` (endpoint `POST /chat/completions`). Modelo por defecto `deepseek-flash` (DeepSeek-V4.1-Flash), configurable con `DEEPSEEK_MODEL`; `deepseek-v4-pro` PUEDE usarse cambiando la variable. Ambos soportan `response_format` y `temperature`. Si se configura un modelo cuyo id contiene `reasoner`, el cliente DEBE omitir ambos parámetros y depender solo de la instrucción 6 del prompt más la validación 8.3.
- Librería: `github.com/sashabaranov/go-openai`. Construcción:
  ```go
  cfg := openai.DefaultConfig(os.Getenv("DEEPSEEK_API_KEY"))
  cfg.BaseURL = getenvDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
  client := openai.NewClientWithConfig(cfg)
  ```
  SI `DEEPSEEK_API_KEY` está vacía, ENTONCES el `Explainer` se construye en modo `template` y no se instancia el cliente.
- Llamada:
  ```go
  resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
      Model:       model, // "deepseek-flash"
      Temperature: 0,
      MaxTokens:   2000,
      ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
      Messages: []openai.ChatCompletionMessage{
          {Role: openai.ChatMessageRoleSystem, Content: <prompt 8.2>},
          {Role: openai.ChatMessageRoleUser,   Content: <payload 8.1>},
      },
  })
  ```
  El JSON mode de DeepSeek exige que la palabra "json" aparezca en el prompt; la instrucción 6 del prompt 8.2 ya la contiene. `Temperature: 0` reduce la variación del texto entre ejecuciones (no la elimina; RNF-06 del maestro excluye el texto del LLM de la reproducibilidad).
- `ctx` = `context.WithTimeout(parent, LLM_TIMEOUT_SECONDS s)` (default 20). Un reintento propio del motor tras `timeout`, `schema_invalid`/`invalid_json` o HTTP 429/5xx (con espera de 2 s); ninguno tras 401/403. Los errores HTTP se detectan con `errors.As(err, &openai.APIError{})` y `apiErr.HTTPStatusCode`.
- Cada llamada registra `{"msg":"llm_call","meter_id":…,"model":…,"duration_ms":…,"outcome":"ok|timeout|http_error|invalid_json|schema_invalid|fallback","prompt_tokens":resp.Usage.PromptTokens,"completion_tokens":resp.Usage.CompletionTokens}`.
- [PENDIENTE-03] (del maestro): confirmar con una llamada real que DeepSeek acepta `response_format: {"type":"json_object"}` desde la versión fijada de `go-openai`; si lo rechaza, se omite el parámetro y la validación 8.3 se mantiene igual.
- Tests: el cliente fake se implementa apuntando `DEEPSEEK_BASE_URL` a un `httptest.Server` que devuelve respuestas `chat/completions` predefinidas (JSON válido, texto no JSON, 401, 429, respuesta lenta); no se usa red real en `go test`.

## 9. Restricciones y decisiones tomadas

Las restricciones de stack (Go 1.22+, API de DeepSeek vía `go-openai`, modelo por defecto `deepseek-flash`, JSON mode, timeout 20 s, motor determinístico manda) y los supuestos SUPUESTO-02 (baseline = 7 días), SUPUESTO-03 (220 V ±5 %) y SUPUESTO-04 (UTC) están en el maestro, sección 9. Decisiones propias de este módulo:

| Decisión/Restricción | Justificación |
|---|---|
| Umbral de spike 20 % | Los 8 medidores sanos no superan 3,2 % de desviación diaria; 20 % deja margen 6× y captura los 3 casos de consumo (36,8 %, 47,5 %, 110,5 %). |
| Umbral HIGH 50 % | Separa el "duplicó el consumo" de `M-109` (110 %) del cambio operativo de `M-104` (47,5 %); el PDF describe `M-109` como "aumento > 100 %". |
| Outliers horarios solo como evidencia (salvo ≥ 6/día) | Con z ≥ 3 los medidores sanos producen hasta 12 outliers; crear anomalías por outlier aislado rompería la métrica de 0 falsos positivos. |
| Corroboración eléctrica con signo | Evita contar como corroboración una variación de corriente en sentido contrario al consumo, que sería a su vez señal de calidad de datos. |
| Ratio físico [0,5; 2,0] como criterio de imposibilidad | Los sanos están en [0,82; 1,30]; `M-109` sube a 1,47 por el cambio de PF pero sigue siendo físicamente plausible; 4,22 en `M-112` no lo es. Un rango más estrecho (p. ej. [0,75; 1,30]) marcaría `M-109` como calidad de datos y perdería 30 + 25 puntos de rúbrica. |
| Precedencia `DATA_QUALITY` primero | Si las lecturas no son fiables, ninguna conclusión sobre consumo lo es; el analista debe validar el sensor antes de investigar consumo. |
| `FALSE_POSITIVE` exige `dev < 0` y ≤ 2 días | Una parada programada explica caídas cortas, no subidas ni caídas prolongadas. |
| `EXPLAINABLE_ANOMALY` nunca `HIGH` | La rúbrica y el PDF la esperan `Medium`; una subida explicada por operación no compite en prioridad con una real. |
| Fórmula de confianza aditiva con tope 0,98 | Transparente y explicable al evaluador; el tope evita prometer certeza absoluta. |
| LLM secuencial (no paralelo) | 4 llamadas de ≤ 20 s cumplen RNF-02 (≤ 60 s) y evitan límites de tasa en la demo. |
| Reintento propio solo para timeout, JSON inválido y 429/5xx | Un 401 no se arregla reintentando; un timeout, una respuesta mal formada o una saturación transitoria sí pueden. `go-openai` no reintenta por sí mismo, así que el motor lo hace una vez. |
| `deepseek-flash` (DeepSeek-V4.1-Flash) como modelo por defecto y no `deepseek-v4-pro` | Elegido por el usuario. Soporta JSON mode y `temperature`, cuesta ≈ 4× menos y responde en ≈ 1 s; la capacidad extra de Pro no aporta a una tarea de redacción sobre evidencia ya calculada. |

## 10. Requisitos no funcionales

Heredados del maestro, sección 10: RNF-01 (≤ 15 s sin LLM), RNF-02 (≤ 60 s con LLM), RNF-06 (reproducibilidad), RNF-07 (cobertura ≥ 70 % en `internal/engine`), RNF-09 (payload LLM ≤ 4 KB sin lecturas crudas), RNF-10 (logs de cada llamada LLM). Propios de este módulo:

| # | Requisito | Métrica |
|---|---|---|
| RNF-E-01 | Tiempo de cómputo del motor sin LLM sobre el dataset | ≤ 2 s en un portátil con 4 vCPU (medido en el test de aceptación con `testing.B` o `time.Since`) |
| RNF-E-02 | Memoria | ≤ 100 MB de heap durante el pipeline con 4.032 lecturas |
| RNF-E-03 | Tamaño del payload al LLM | ≤ 4.096 bytes por anomalía; el test lo verifica sobre las 4 anomalías |
| RNF-E-04 | Aislamiento del LLM | 0 dependencias de red en `internal/engine`; solo `internal/llm` importa `go-openai` |
| RNF-E-05 | Cobertura | ≥ 70 % en `internal/engine`, ≥ 60 % en `internal/llm` |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| [SUPUESTO-02] (maestro) | Ventana de baseline = primeros 7 días. Aplica a todos los cálculos de este documento. | Jonnathan Sotelo | — |
| [SUPUESTO-03] (maestro) | Voltaje nominal 220 V ±5 % → [209, 231] V. | Jonnathan Sotelo | — |
| [SUPUESTO-04] (maestro) | Timestamps en UTC; "día" = fecha calendario UTC. | Jonnathan Sotelo | — |
| [SUPUESTO-E-01] | Los umbrales de la sección 9 se calibraron sobre el dataset entregado; para otro dataset PUEDEN requerir ajuste vía `Thresholds`. | Jonnathan Sotelo | — |
| [SUPUESTO-E-02] | El valor `expected_results.csv` no se usa en ningún test; los valores esperados de los tests se transcriben de la sección 9 del PDF de la prueba. | Jonnathan Sotelo | — |
| [PENDIENTE-03] (maestro) | Confirmar con una llamada real que DeepSeek acepta `response_format: {"type":"json_object"}` desde `go-openai`; si no, se omite y se mantiene la instrucción 6 del prompt + validación 8.3. | Implementador del motor | Durante la implementación |
| [SUPUESTO-06] (maestro) | Los agregados enviados a DeepSeek no tienen restricciones de residencia de datos para esta prueba. | Jonnathan Sotelo | — |
| [DECISIÓN ABIERTA-E-01] | Si el LLM devuelve una `recommended_action` que contradice `action_default` (ej. "Escalar" en un `FALSE_POSITIVE`), hoy se acepta si valida el schema. Alternativa: rechazarla y usar `action_default`. Criterio: revisar 5 respuestas reales del LLM en la demo; si ≥ 1 contradice, implementar el rechazo por lista de verbos prohibidos por tipo. | Jonnathan Sotelo | Antes de la demo |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

| Requisito | Test | Archivo |
|---|---|---|
| RF-E-01 | Test del pipeline con `onStep` espía: 14 invocaciones, orden, `detail` final; test con entrada vacía | `pipeline_test.go` |
| RF-E-02 | Table-driven: baseline, `dev`, perfil horario de `M-109` y `M-101` con tolerancias del RF | `baseline_test.go` |
| RF-E-03 | Fixture `testdata/short_meter.csv` (30 lecturas) → sin anomalía, `insufficient_data` | `baseline_test.go` |
| RF-E-04/05 | Table-driven sobre `M-104`, `M-106`, `M-109` y los 8 sanos: días spike exactos y `persistent` | `detectors_test.go` |
| RF-E-06 | `M-107` (12 outliers, no candidato), `M-109` (24/día), fixture `testdata/hourly_pattern.csv` (8 outliers en un día sin spike → `REAL_ANOMALY MEDIUM`) | `detectors_test.go` |
| RF-E-07 | Table-driven con los tres ejemplos del RF | `detectors_test.go` |
| RF-E-08 | `M-112` (días y contadores exactos), `M-109` (no DQ), sanos (no DQ), fixture `testdata/gap_day.csv` (20 lecturas) | `detectors_test.go` |
| RF-E-09 | Los 4 casos del RF + CB-E-05/06/07/08/09 con fixtures de eventos | `classify_test.go` |
| RF-E-10 | Los 4 casos del dataset + fixtures: caída sin evento → `REAL_ANOMALY`; subida con `SCHEDULED_OUTAGE` → `REAL_ANOMALY`; DQ + `OPERATIONAL_CHANGE` → `DATA_QUALITY` | `classify_test.go` |
| RF-E-11 | Table-driven de la tabla de severidad incluyendo `EXPLAINABLE` con `max_dev = 0.80` | `classify_test.go` |
| RF-E-12 | Table-driven reproduciendo la tabla de desglose (4 filas) + caso de tope inferior 0,50 | `confidence_test.go` |
| RF-E-13 | Orden sobre el dataset + caso de empate por `meter_id` | `classify_test.go` |
| RF-E-14 | Comparación de evidencia de `M-109`, `M-112`, `M-106` contra los ejemplos (tolerancias del RF); invariantes `position` y `detail` | `evidence_test.go` |
| RF-E-15 | `recommended_action` exacta por tipo en modo template | `templates_test.go` |
| RF-E-16 | `Explainer` con cliente fake: JSON válido; JSON con `severity` extra | `explainer_test.go` |
| RF-E-17 | `httptest.Server` como DeepSeek falso vía `DEEPSEEK_BASE_URL`: sin API key (0 llamadas); texto no JSON ×2; respuesta lenta (con `LLM_TIMEOUT_SECONDS=1` en test); 401 (sin reintento, resto del análisis en template); 429 seguido de 200 (un reintento, `outcome=ok`); `finish_reason="length"` | `explainer_test.go` |
| RF-E-18 | Ejecutar dos veces y comparar JSON canónico | `pipeline_test.go` |
| RF-E-19 | `Thresholds{}` → defaults; compilación de `internal/engine` sin importar `internal/llm` ni el SDK (`go list -deps`) | `pipeline_test.go`, CI |
| Aceptación global | `engine_dataset_test.go`: carga `data/readings.csv` y `data/events.csv`, ejecuta el pipeline en modo template y verifica los 12 medidores (tipo, severidad, confianza, rank, ≥ 3 evidencias, acción) y el `detail` final | `engine_dataset_test.go` |
| RNF-E-01/02/03 | Medición de tiempo y `runtime.MemStats` en el test de aceptación; `len(payload) ≤ 4096` | `engine_dataset_test.go`, `explainer_test.go` |

Fixtures en `backend/internal/engine/testdata/`: `short_meter.csv`, `hourly_pattern.csv`, `gap_day.csv`, `drop_no_event.csv`, `rise_with_outage.csv`, `dq_with_change.csv`, `events_*.csv`, `zero_baseline.csv`, `no_readings_meter.csv`. Ninguno contiene ni deriva de `expected_results.csv`.

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, autor de la prueba técnica.
- **Qué criterios valida:** RF-E-10, RF-E-11, RF-E-12, RF-E-13, RF-E-14, RF-E-15, RF-E-16 y RF-E-17, observando en la pantalla de Anomalías IA y en la Investigación de cada uno de los 4 medidores que tipo, severidad, confianza, orden, evidencia, explicación y acción coinciden con este documento, tanto con `DEEPSEEK_API_KEY` definida como sin ella.
- **Nivel de UAT:** Profundo → UAT formal completo: el firmante valida cada criterio listado con resultado pasa/falla y firma el acta de aceptación antes de la entrega.
