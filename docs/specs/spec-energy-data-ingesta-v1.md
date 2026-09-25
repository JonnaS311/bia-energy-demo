# Spec: Ingesta de datos y cálculos derivados

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8, heredado del maestro) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> Spec hijo de `spec-energy-plataforma-maestro-v1.md`. El maestro define el problema, el glosario completo, el modelo de datos (sección 8.1) y los contratos de API (sección 8.2). Este documento detalla **cómo entran los datos** a PostgreSQL y **cómo se calculan** los valores derivados (baseline, variación, perfil horario, agregados) que consumen el motor de anomalías y la API. Cualquier contradicción se resuelve a favor del maestro.

## 1. Contexto y problema

Resumen del maestro (sección 1): la plataforma parte de dos archivos CSV, `readings.csv` (4.032 lecturas horarias de 12 medidores durante 14 días) y `events.csv` (4 eventos operativos), y un tercer archivo `expected_results.csv` reservado al evaluador que NO se usa. Sin una ingesta confiable e idempotente y sin una definición única de "baseline", "consumo actual" y "variación", el motor de anomalías y las pantallas darían números distintos para el mismo medidor. Este spec fija esa definición una sola vez: todo cálculo derivado que aparezca en la API o en el motor se implementa como está escrito en la sección 8.

## 2. Objetivo y métricas de éxito

Resultado esperado: que al arrancar el backend los datos queden cargados y verificables con conteos exactos, y que cada número derivado (baseline, variación, totales) coincida con los valores de referencia calculados sobre el dataset entregado.

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Lecturas cargadas | 0 | 4.032 filas en `readings`, 336 por medidor | Al terminar la ingesta en un arranque con BD vacía |
| Medidores y eventos cargados | 0 | 12 filas en `meters`, 4 en `events` | Ídem |
| Filas rechazadas y duplicadas sobre el dataset entregado | — | `rejected = 0`, `duplicates = 0` | Ídem |
| Idempotencia | — | Un segundo arranque deja los mismos conteos y `rejected = 0` | Segundo arranque consecutivo |
| Exactitud de derivados | — | Los 12 valores de referencia de la sección 12a coinciden con tolerancia ±0,05 | Test `aggregates_dataset_test.go` |
| Duración de la ingesta | — | ≤ 5 s para los 2 archivos (RNF-D-01) | Log `ingest_completed.duration_ms` |

## 3. Glosario

El glosario completo vive en el maestro, sección 3. Términos que este spec introduce o precisa:

| Término | Definición |
|---|---|
| Ingesta | Proceso que lee los CSV desde `DATA_DIR`, valida cada fila y la escribe en PostgreSQL. Ocurre una vez por arranque del backend. |
| Fila rechazada | Fila del CSV que no pasa la validación de la sección 8.2 y por tanto no se escribe. Se cuenta en `ingest_stats.rejected`. |
| Duplicado | Dos o más filas del **mismo archivo** con igual clave natural (`(meter_id, timestamp)` en lecturas; `(meter_id, timestamp, type)` en eventos). |
| `ingest_stats` | Estructura en memoria con contadores de la última ingesta; se expone en `GET /health` bajo `ingest`. |
| Día completo | Fecha calendario (UTC) para la que un medidor tiene exactamente 24 lecturas, una por hora 00..23. |
| Último día completo | El día completo con la fecha más reciente de un medidor. Con el dataset entregado es `2026-09-14` para los 12 medidores. |
| Periodo del dataset | Rango `[min(timestamp), max(timestamp)]` sobre toda la tabla `readings`, truncado a fechas. Con el dataset entregado: `2026-09-01` a `2026-09-14`. |
| `BASELINE_DAYS` | Parámetro entero, valor por defecto 7. Número de días calendario, contados desde el primer día del periodo, que forman la ventana de baseline. |
| `insufficient_data` | Marca booleana por medidor: `true` cuando el medidor tiene menos de 2 días completos. Excluye al medidor de la detección por consumo (ver CB-M11 del maestro). |
| BOM | Byte Order Mark: los 3 bytes `EF BB BF` que algunos editores anteponen a un archivo UTF-8. |

## 4. Actores y casos de uso

Resumen del maestro (sección 4): el único rol humano es el analista de operaciones; la ingesta no tiene interfaz de usuario. Actores de este spec:

- **Sistema (proceso de arranque del backend):** ejecuta la ingesta antes de abrir el puerto HTTP a endpoints de datos.
- **Motor de anomalías** (`spec-energy-motor-anomalias-ia-v1.md`): consume los cálculos derivados de la sección 8.4 tal como están definidos aquí.
- **API HTTP** (`spec-energy-backend-api-v1.md`): consume los mismos cálculos para `GET /meters`, `GET /meters/:meterId`, `GET /meters/:meterId/readings` y `GET /dashboard/summary`.

Historia: como implementador del motor o de la API quiero una única función por cálculo derivado, con nombre y firma fijos, para que dashboard, detalle y motor muestren el mismo baseline y la misma variación para un mismo medidor.

## 5. Alcance y no-alcance

**Incluye:**
- Lectura, validación y carga idempotente de `data/readings.csv` y `data/events.csv` al arrancar.
- Semilla de `meters` (derivada de los datos) y de `users` (desde variables de entorno).
- Contadores de ingesta expuestos en `GET /health` y en log.
- Definición y consultas exactas de los cálculos derivados: totales diarios, baseline diario, perfil horario, consumo actual, variación, promedios eléctricos, `is_outlier`, consumo total del periodo y consumo diario global.
- Manejo de datasets distintos al entregado (medidores con pocos días, días incompletos) sin fallar.

**NO incluye (explícito):**
- NO carga de archivos desde la UI ni por API; NO endpoint `POST /ingest`. La única vía es el arranque del proceso.
- NO lectura, copia ni referencia a `expected_results.csv`. El archivo va en `.gitignore` y, si existe en `DATA_DIR`, se ignora sin registrar su contenido.
- NO tablas materializadas ni vistas materializadas para agregados (decisión en sección 9).
- NO limpieza ni imputación de datos: una fila inválida se rechaza, nunca se corrige ni se interpola.
- NO conversión de zona horaria: los timestamps se interpretan como UTC y se almacenan tal cual.
- NO detección de anomalías: este spec entrega insumos; las reglas viven en el spec del motor.
- NO soporte de otros formatos (Excel, JSON, Parquet) ni de otros separadores distintos de la coma.

## 6. Requisitos funcionales

### RF-D-01 — Ubicación y descubrimiento de archivos
**Prioridad:** DEBE
**Descripción:** CUANDO el backend arranca, el sistema DEBE buscar `readings.csv` y `events.csv` en el directorio `DATA_DIR` (valor por defecto `/app/data`; en el repositorio los archivos viven en `data/`). El sistema DEBE ejecutar la ingesta antes de aceptar peticiones a endpoints de datos; MIENTRAS la ingesta está en curso, `GET /health` DEBE responder 200 con `ingest.status = "RUNNING"` y los endpoints protegidos de datos DEBEN responder 503 `INGEST_IN_PROGRESS`.
**Criterios de aceptación:**
- DADO `DATA_DIR=/app/data` con ambos archivos, CUANDO el backend arranca, ENTONCES el log contiene `ingest_started` con las dos rutas absolutas y, al terminar, `ingest_completed` con los contadores de RF-D-06.
- DADO que la ingesta está en curso, CUANDO se pide `GET /meters`, ENTONCES la respuesta es 503 con código `INGEST_IN_PROGRESS`.

### RF-D-02 — Formato de archivo y encabezados
**Prioridad:** DEBE
**Descripción:** El sistema DEBE leer los archivos como texto UTF-8, separados por coma, con salto de línea `LF` o `CRLF`, y DEBE tolerar un BOM inicial descartándolo. El sistema DEBE exigir que la primera línea sea exactamente, en ese orden y sin espacios:
- `readings.csv`: `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status`
- `events.csv`: `meter_id,event_timestamp,event_type,description`

SI el archivo tiene columnas adicionales a la derecha de las obligatorias, ENTONCES el sistema DEBE ignorarlas y registrar un warning `INGEST_EXTRA_COLUMNS` con sus nombres. SI falta cualquiera de las columnas obligatorias o el orden difiere, ENTONCES el sistema DEBE registrar el error `INGEST_SCHEMA_MISMATCH` con las columnas esperadas y las encontradas, y DEBE abortar la ingesta de **ese archivo** (el otro archivo se procesa igual).
**Criterios de aceptación:**
- DADO un `readings.csv` con BOM y `CRLF`, CUANDO se ingesta, ENTONCES se cargan las mismas 4.032 filas que sin BOM y con `LF`.
- DADO un `readings.csv` con la columna extra `site` al final, CUANDO se ingesta, ENTONCES se cargan 4.032 filas y el log contiene `INGEST_EXTRA_COLUMNS` con `["site"]`.
- DADO un `readings.csv` cuyo encabezado es `meter_id,timestamp,kwh,...`, CUANDO se ingesta, ENTONCES no se escribe ninguna lectura, el log contiene `INGEST_SCHEMA_MISMATCH` y los eventos sí se cargan.

### RF-D-03 — Parseo de tipos y timestamps
**Prioridad:** DEBE
**Descripción:** El sistema DEBE parsear cada campo así:
- `timestamp` (readings) con el layout Go `2006-01-02 15:04:05`.
- `event_timestamp` (events) con el layout Go `2006-01-02 15:04`; SI el valor tiene segundos, ENTONCES el sistema DEBE aceptarlo también con `2006-01-02 15:04:05`.
- Ambos timestamps se interpretan como UTC y se almacenan como `timestamptz` en UTC.
- `consumption_kwh`, `voltage_v`, `current_a`, `power_factor`: número decimal con punto como separador (`strconv.ParseFloat` en base 10, 64 bits).
- `meter_id`, `status`, `event_type`, `description`: texto; el sistema DEBE recortar espacios en blanco al inicio y al final de cada campo antes de validar.
**Criterios de aceptación:**
- DADO el valor `2026-09-01 00:00:00` en readings, CUANDO se parsea, ENTONCES se almacena `2026-09-01T00:00:00Z`.
- DADO el valor `2026-09-11 00:00` en events, CUANDO se parsea, ENTONCES se almacena `2026-09-11T00:00:00Z`.
- DADO el valor ` M-101 ` con espacios, CUANDO se parsea, ENTONCES `meter_id` almacenado es `M-101`.

### RF-D-04 — Validación por fila y rechazos
**Prioridad:** DEBE
**Descripción:** El sistema DEBE validar cada fila de datos contra la tabla de la sección 8.2. SI una fila no cumple una regla, ENTONCES el sistema DEBE omitirla, incrementar `ingest_stats.rejected` y registrar un log `INGEST_ROW_REJECTED` con `file`, `line` (número de línea en el archivo, contando el encabezado como línea 1), `reason` (código de la tabla) y `meter_id` si se pudo leer. Una fila rechazada NO aborta la ingesta.
**Criterios de aceptación:**
- DADO un archivo con una fila cuyo `power_factor` es `1.2`, CUANDO se ingesta, ENTONCES esa fila no existe en `readings`, `rejected = 1` y el log contiene `reason = "PF_OUT_OF_RANGE"` con su número de línea.
- DADO el dataset entregado, CUANDO se ingesta, ENTONCES `rejected = 0`.

### RF-D-05 — Escritura idempotente y duplicados
**Prioridad:** DEBE
**Descripción:** El sistema DEBE escribir las lecturas con `INSERT ... ON CONFLICT (meter_id, timestamp) DO UPDATE SET consumption_kwh = EXCLUDED.consumption_kwh, voltage_v = EXCLUDED.voltage_v, current_a = EXCLUDED.current_a, power_factor = EXCLUDED.power_factor, status = EXCLUDED.status`, y los eventos con `INSERT ... ON CONFLICT (meter_id, timestamp, type) DO UPDATE SET description = EXCLUDED.description`. CUANDO dos filas del mismo archivo comparten clave natural, el sistema DEBE conservar la que aparece **más abajo** en el archivo e incrementar `ingest_stats.duplicates` por cada fila sobrescrita. Cada archivo DEBE escribirse dentro de **una** transacción; SI la transacción falla, ENTONCES el sistema DEBE hacer rollback de ese archivo completo y registrar `INGEST_TX_FAILED`.
**Criterios de aceptación:**
- DADO una BD ya cargada, CUANDO el backend arranca de nuevo, ENTONCES `count(readings) = 4032`, `count(events) = 4`, `rejected = 0`, `duplicates = 0`.
- DADO un archivo donde la línea 10 y la línea 500 tienen `(M-101, 2026-09-01 08:00:00)` con consumos `35.07` y `99.9`, CUANDO se ingesta, ENTONCES `readings` tiene `99.9` para esa clave y `duplicates = 1`.
- DADO que la BD se cae en la fila 2.000 de readings, CUANDO la transacción falla, ENTONCES `count(readings) = 0` (o el valor previo si ya había datos) y el log contiene `INGEST_TX_FAILED`.

### RF-D-06 — Semillas de medidores y usuario
**Prioridad:** DEBE
**Descripción:** Antes de escribir lecturas, el sistema DEBE crear en `meters` una fila por cada `meter_id` distinto válido encontrado en `readings.csv`, con `name = 'Medidor ' || meter_id`, `location = 'Planta principal'`, `status = 'OK'`, usando `ON CONFLICT (meter_id) DO NOTHING` (no sobrescribe `status` de un análisis previo). SI `events.csv` contiene un `meter_id` válido que no existe en `readings.csv`, ENTONCES el sistema DEBE crearlo igualmente en `meters` y registrar `INGEST_METER_WITHOUT_READINGS` con ese `meter_id`. El sistema DEBE crear el usuario demo en `users` con `email = DEMO_USER_EMAIL`, `name = 'Analista Demo'` y `password_hash = bcrypt(DEMO_USER_PASSWORD, cost 10)`, usando `ON CONFLICT (email) DO NOTHING`.
**Criterios de aceptación:**
- DADO el dataset entregado, CUANDO termina la ingesta, ENTONCES `meters` tiene 12 filas con `meter_id` de `M-101` a `M-112` y `name` `Medidor M-101` … `Medidor M-112`.
- DADO un `events.csv` con una fila para `M-150` y sin lecturas de `M-150`, CUANDO termina la ingesta, ENTONCES `meters` contiene `M-150` y el log contiene `INGEST_METER_WITHOUT_READINGS`.
- DADO `meters.status = 'CRITICAL'` para `M-109` por un análisis previo, CUANDO el backend arranca de nuevo, ENTONCES `M-109` sigue en `CRITICAL`.

### RF-D-07 — Estadísticas de ingesta
**Prioridad:** DEBE
**Descripción:** Al terminar la ingesta el sistema DEBE guardar en memoria y registrar en log `ingest_completed` la estructura `ingest_stats` con los campos enteros `readings`, `events`, `meters`, `rejected`, `duplicates` y `duration_ms`, más `status` ∈ `{RUNNING, COMPLETED, FAILED}`. `GET /health` DEBE devolverla bajo la clave `ingest` (el maestro, sección 8.2.10, muestra los tres primeros campos; este spec añade los restantes).
**Criterios de aceptación:**
- DADO el dataset entregado, CUANDO se pide `GET /health` tras la ingesta, ENTONCES `ingest` es `{ "status": "COMPLETED", "readings": 4032, "events": 4, "meters": 12, "rejected": 0, "duplicates": 0, "duration_ms": <entero ≤ 5000> }`.

### RF-D-08 — Cálculos derivados como fuente única
**Prioridad:** DEBE
**Descripción:** El sistema DEBE exponer, en el paquete `backend/internal/store`, una función por cada cálculo de la sección 8.4 con la firma y la semántica allí definidas. El motor y la API DEBEN consumir esas funciones y NO DEBEN reimplementar el cálculo. Los cálculos se ejecutan sobre `readings` con el índice `readings_meter_ts_idx`, sin tablas materializadas; el sistema PUEDE mantener una caché en memoria por proceso que se invalida al terminar cada ingesta.
**Criterios de aceptación:**
- DADO el dataset entregado, CUANDO se llama `DailyBaseline("M-109")`, ENTONCES devuelve `1048.8` (±0,05).
- DADO el dataset entregado, CUANDO se llama `CurrentConsumption("M-109")`, ENTONCES devuelve `2207.6` (±0,05) con `date = 2026-09-14`.
- DADO el dataset entregado, CUANDO se llama `Variation("M-109")`, ENTONCES devuelve `110.5` (±0,1).
- DADO el dataset entregado, CUANDO se llama `PeriodTotal()`, ENTONCES devuelve `155250.8` (±0,5).

### RF-D-09 — Comportamiento con datos escasos
**Prioridad:** DEBE
**Descripción:** SI un medidor tiene menos de `BASELINE_DAYS` días completos dentro de la ventana de baseline, ENTONCES el sistema DEBE calcular el baseline con los días completos disponibles. SI un medidor tiene menos de 2 días completos en todo el periodo, ENTONCES el sistema DEBE marcar `insufficient_data = true`, devolver `baseline_daily_kwh = null` y `variation_pct = null` para ese medidor. SI el baseline diario calculado es `0`, ENTONCES `variation_pct` DEBE ser `null`.
**Criterios de aceptación:**
- DADO un fixture con `M-201` y 3 días completos, CUANDO se calcula el baseline, ENTONCES es el promedio de esos 3 días e `insufficient_data = false`.
- DADO un fixture con `M-202` y 1 día completo más 5 lecturas de otro día, CUANDO se calcula, ENTONCES `insufficient_data = true` y `variation_pct = null`.
- DADO un fixture con `M-203` cuyas lecturas de baseline son todas `0`, CUANDO se calcula, ENTONCES `baseline_daily_kwh = 0` y `variation_pct = null`.

### RF-D-10 — Exclusión de `expected_results.csv`
**Prioridad:** DEBE
**Descripción:** El sistema NO DEBE abrir ni leer ningún archivo de `DATA_DIR` distinto de `readings.csv` y `events.csv`. El repositorio DEBE incluir la línea `data/expected_results.csv` en `.gitignore`. Ningún módulo, test, fixture o prompt DEBE contener la cadena `expected_results`.
**Criterios de aceptación:**
- DADO un tercer archivo cualquiera presente en `DATA_DIR` (el test usa `otro_archivo.csv`; el comportamiento no depende del nombre), CUANDO el backend arranca, ENTONCES el log no contiene su nombre ni su contenido y los contadores son idénticos a un arranque sin ese archivo.
- DADO el repositorio, CUANDO se ejecuta `git grep -i expected_results -- ':!docs/specs' ':!.gitignore' ':!README.md'`, ENTONCES no hay coincidencias.

## 7. Casos borde y manejo de errores

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-D-01 | `readings.csv` no existe en `DATA_DIR` | Log `INGEST_FILE_MISSING` con ruta absoluta; `ingest.status = "COMPLETED"` con `readings = 0`; el backend sigue arrancando (coincide con CB-M01 del maestro). |
| CB-D-02 | `events.csv` no existe | Ídem con `events = 0`; las lecturas se cargan normalmente. |
| CB-D-03 | Archivo con solo encabezado (0 filas de datos) | Encabezado se valida (RF-D-02); contadores en 0; sin error. |
| CB-D-04 | Archivo con 1 medidor y 3 días completos | Se carga; baseline = promedio de 3 días; `insufficient_data = false` (RF-D-09). |
| CB-D-05 | Medidor con 1 solo día completo | `insufficient_data = true`; `baseline_daily_kwh = null`, `variation_pct = null`; sigue apareciendo en `GET /meters` con `status = OK`. |
| CB-D-06 | Día incompleto (< 24 lecturas) | El día NO cuenta como "día completo": no entra al baseline ni puede ser "último día completo". Sí se incluye en totales del periodo y en el histórico horario. |
| CB-D-07 | Timestamps no consecutivos u ordenados aleatoriamente en el archivo | El orden del archivo es irrelevante; los cálculos ordenan por `timestamp` en SQL. |
| CB-D-08 | Lectura con hora repetida en el mismo día (duplicado exacto de clave) | Regla de RF-D-05: última gana, `duplicates += 1`. |
| CB-D-09 | Dos ingestas concurrentes en el mismo proceso | Imposible por diseño: la ingesta corre una vez en el arranque, protegida por `sync.Mutex`; una segunda llamada retorna sin hacer nada y registra `INGEST_ALREADY_RUNNING`. |
| CB-D-10 | BD caída a mitad de la escritura de un archivo | Rollback del archivo completo; `INGEST_TX_FAILED`; `ingest.status = "FAILED"`; el backend sigue vivo y `GET /health` devuelve `db = "error"` hasta que reconecte. |
| CB-D-11 | Fila con campos vacíos (`M-101,2026-09-01 00:00:00,,221.9,101.28,0.954,OK`) | Rechazo con `reason = "NOT_A_NUMBER"` en `consumption_kwh`. |
| CB-D-12 | Fila con espacios alrededor de valores (`M-101 , 2026-09-01 00:00:00 , 23.5 ,…`) | Se recortan espacios (RF-D-03) y la fila se acepta. |
| CB-D-13 | Fila con número de columnas distinto al encabezado (columnas de más o de menos en esa línea) | Rechazo con `reason = "COLUMN_COUNT_MISMATCH"`. |
| CB-D-14 | `status` de lectura con valor distinto de `OK` | Se acepta y almacena tal cual (el dataset entregado solo trae `OK`; el motor no usa este campo). |
| CB-D-15 | `event_type` con un valor fuera de los 4 conocidos | Se acepta y almacena tal cual; el motor lo trata como "sin evento" (regla en el spec del motor). |
| CB-D-16 | Timestamp de evento fuera del periodo del dataset | Se acepta y almacena; no se correlacionará con ninguna anomalía porque no hay lecturas a ≤ 24 h. |
| CB-D-17 | Archivo con codificación distinta de UTF-8 (por ejemplo Latin-1 en `description`) | Los bytes inválidos en `description` se reemplazan por `U+FFFD`; las columnas numéricas y `meter_id` no admiten bytes no ASCII y la fila se rechaza con `INVALID_METER_ID` o `NOT_A_NUMBER` según el campo afectado. |
| CB-D-18 | `DEMO_USER_PASSWORD` vacío | El backend termina con exit code 1 y log `CONFIG_INVALID` (`DEMO_USER_PASSWORD` no puede ser vacío). |

## 8. Contratos de datos e interfaces

### 8.1 Archivos de entrada

| Archivo | Ruta en repo | Ruta en contenedor | Encabezado exacto | Ejemplo de fila real |
|---|---|---|---|---|
| Lecturas | `data/readings.csv` | `/app/data/readings.csv` | `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status` | `M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK` |
| Eventos | `data/events.csv` | `/app/data/events.csv` | `meter_id,event_timestamp,event_type,description` | `M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated` |

Contenido completo de `events.csv` entregado (4 filas):

```
meter_id,event_timestamp,event_type,description
M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated
M-106,2026-09-08 00:00,SCHEDULED_OUTAGE,Scheduled maintenance outage for 12 hours
M-109,2026-09-12 14:00,UNKNOWN,No operational event reported
M-112,2026-09-13 00:00,DATA_QUALITY,Intermittent readings and abnormal electrical jumps
```

### 8.2 Reglas de validación por fila

| Código `reason` | Aplica a | Regla | Ejemplo que se rechaza |
|---|---|---|---|
| `COLUMN_COUNT_MISMATCH` | ambos | Número de campos ≠ número de columnas del encabezado | `M-101,2026-09-01 00:00:00,23.5` |
| `INVALID_METER_ID` | ambos | `meter_id` no cumple la expresión regular `^M-\d{3}$` | `m101`, `M-1`, `M-1000` |
| `INVALID_TIMESTAMP` | ambos | No parsea con los layouts de RF-D-03 | `2026/09/01 00:00`, `01-09-2026 00:00:00` |
| `NOT_A_NUMBER` | readings | Cualquiera de los 4 campos numéricos vacío o no parseable | `abc`, `` (vacío), `23,5` |
| `NEGATIVE_CONSUMPTION` | readings | `consumption_kwh < 0` | `-1.5` |
| `VOLTAGE_NOT_POSITIVE` | readings | `voltage_v <= 0` | `0`, `-220` |
| `NEGATIVE_CURRENT` | readings | `current_a < 0` | `-3` |
| `PF_OUT_OF_RANGE` | readings | `power_factor < 0` o `power_factor > 1` | `1.2`, `-0.1` |
| `EMPTY_EVENT_TYPE` | events | `event_type` vacío tras recortar espacios | `M-104,2026-09-11 00:00,,texto` |

Orden de evaluación: `COLUMN_COUNT_MISMATCH` → `INVALID_METER_ID` → `INVALID_TIMESTAMP` → numéricos en el orden del encabezado. Se reporta solo el primer motivo encontrado.

### 8.3 Estructura `ingest_stats`

```json
{
  "status": "COMPLETED",
  "readings": 4032,
  "events": 4,
  "meters": 12,
  "rejected": 0,
  "duplicates": 0,
  "duration_ms": 1840,
  "started_at": "2026-09-25T15:00:00Z",
  "finished_at": "2026-09-25T15:00:01Z"
}
```

Log de ingesta (JSON por línea, ver RNF-10 del maestro):

```json
{"level":"info","msg":"ingest_started","readings_path":"/app/data/readings.csv","events_path":"/app/data/events.csv"}
{"level":"warn","msg":"INGEST_ROW_REJECTED","file":"readings.csv","line":117,"reason":"PF_OUT_OF_RANGE","meter_id":"M-101"}
{"level":"info","msg":"ingest_completed","readings":4032,"events":4,"meters":12,"rejected":0,"duplicates":0,"duration_ms":1840}
```

### 8.4 Cálculos derivados (fuente de verdad)

Todas las fechas son UTC. `date` = `(timestamp AT TIME ZONE 'UTC')::date`. Las funciones viven en `backend/internal/store/aggregates.go`. Redondeo: kWh y V/A a 1 decimal, PF a 3 decimales, porcentajes a 1 decimal, solo al serializar en la API; los cálculos internos usan `float64` sin redondear.

#### 8.4.1 Periodo del dataset — `Period() (start, end date)`

```sql
SELECT min(timestamp)::date AS start, max(timestamp)::date AS end FROM readings;
```
Dataset entregado: `2026-09-01`, `2026-09-14`.

#### 8.4.2 Días completos por medidor — `CompleteDays(meterID) []date`

```sql
SELECT (timestamp AT TIME ZONE 'UTC')::date AS date
FROM readings WHERE meter_id = $1
GROUP BY 1 HAVING count(*) = 24 AND count(DISTINCT extract(hour FROM timestamp)) = 24
ORDER BY 1;
```

#### 8.4.3 Totales diarios por medidor — `DailyTotals(meterID) []DailyTotal`

```sql
SELECT (timestamp AT TIME ZONE 'UTC')::date AS date,
       sum(consumption_kwh) AS consumption_kwh,
       avg(voltage_v) AS voltage_avg, avg(current_a) AS current_avg, avg(power_factor) AS power_factor_avg,
       count(*) AS readings_count
FROM readings WHERE meter_id = $1
GROUP BY 1 ORDER BY 1;
```
Cada fila incluye `is_complete = (readings_count = 24)`. Ejemplo `M-109`: `2026-09-14 → 2207.6 kWh, V 216.4, I 420.5, PF 0.744, 24 lecturas`.

#### 8.4.4 Ventana de baseline — `BaselineWindow() (start, end date)`

`start = Period().start`; `end = start + (BASELINE_DAYS − 1)` días. Con `BASELINE_DAYS = 7`: `2026-09-01` a `2026-09-07` inclusive. La ventana es **global** (misma para todos los medidores).

#### 8.4.5 Baseline diario — `DailyBaseline(meterID) (value float64, days int, insufficient bool)`

`value` = promedio aritmético de `consumption_kwh` de `DailyTotals` restringido a `is_complete = true` y `date` dentro de la ventana de baseline. `days` = número de días usados. SI el medidor tiene < 2 días completos en **todo** el periodo, ENTONCES `insufficient = true` y `value` no se usa.

```sql
SELECT avg(d.consumption_kwh), count(*)
FROM ( /* consulta 8.4.3 */ ) d
WHERE d.readings_count = 24 AND d.date BETWEEN $2 AND $3;
```
Valores de referencia (dataset entregado): `M-101 = 729.4`, `M-104 = 1169.7`, `M-106 = 1349.1`, `M-109 = 1048.8`, `M-112 = 662.2`, `days = 7` en todos.

#### 8.4.6 Perfil horario baseline — `HourlyProfile(meterID) [24]HourStat`

Para cada `hour ∈ 0..23`: `mean` y `std` (desviación estándar **muestral**, divisor `n − 1`) de `consumption_kwh` de las lecturas del medidor con `date` en la ventana de baseline y `extract(hour) = hour`.

```sql
SELECT extract(hour FROM timestamp)::int AS hour,
       avg(consumption_kwh) AS mean, stddev_samp(consumption_kwh) AS std, count(*) AS n
FROM readings
WHERE meter_id = $1 AND (timestamp AT TIME ZONE 'UTC')::date BETWEEN $2 AND $3
GROUP BY 1 ORDER BY 1;
```
SI `n < 2` para una hora, ENTONCES `std = 0`. Referencia `M-109`: `hour 0 → mean 31.7, std 1.6`; `hour 1 → mean 32.7, std 1.6`; `hour 13 → mean 51.3, std 1.4`.

#### 8.4.7 Consumo actual — `CurrentConsumption(meterID) (value float64, date date)`

`date` = último día completo del medidor (8.4.2, el más reciente); `value` = `consumption_kwh` de `DailyTotals` para esa fecha. SI no hay ningún día completo, ENTONCES `value = null`. Referencia: `M-109 → 2207.6, 2026-09-14`; `M-101 → 728.8, 2026-09-14`.

#### 8.4.8 Variación — `Variation(meterID) *float64`

`(CurrentConsumption − DailyBaseline) / DailyBaseline × 100`. `null` SI `insufficient = true`, SI `DailyBaseline = 0` o SI `CurrentConsumption = null`. Referencia: `M-109 → 110.5`; `M-101 → -0.1`.

#### 8.4.9 Desviación diaria — `DailyDeviations(meterID) []DailyDeviation`

Para cada fila de `DailyTotals` con `is_complete = true`: `deviation_pct = (consumption_kwh − DailyBaseline) / DailyBaseline × 100`; `null` con las mismas condiciones de 8.4.8. Referencia `M-109`: `2026-09-12 → 49.6`, `2026-09-13 → 109.2`, `2026-09-14 → 110.5`.

#### 8.4.10 Promedios eléctricos — `ElectricalSummary(meterID) ElectricalSummary`

Para `voltage_v`, `current_a`, `power_factor`: `current_avg` = promedio del último día completo (8.4.7); `baseline_avg` = promedio de todas las lecturas del medidor con `date` en la ventana de baseline. Referencia `M-109`: `V 216.4 / 219.9`, `I 420.5 / 200.5`, `PF 0.744 / 0.939`.

#### 8.4.11 Outlier horario — `IsOutlier(reading, profile) bool`

`z = |consumption_kwh − profile[hour].mean| / profile[hour].std`; `is_outlier = (profile[hour].std > 0) AND (z ≥ 3)`. Se calcula al servir `GET /meters/:meterId/readings` con `granularity=hour` y lo consume el motor. Referencia: `M-109 2026-09-14 13:00:00` (117,51 kWh, perfil hora 13 media 51,3 y desviación 1,4) → `is_outlier = true`; `M-101` tiene exactamente 1 outlier en el periodo (`2026-09-11 22:00:00`, z = 3,13). Corregido en implementación: la v1.0 decía "ningún outlier", valor que venía de un script de validación que subcontaba.

#### 8.4.12 Total del periodo y consumo diario global

```sql
SELECT sum(consumption_kwh) FROM readings;                                       -- PeriodTotal(): 155250.8
SELECT (timestamp AT TIME ZONE 'UTC')::date AS date, sum(consumption_kwh)
FROM readings GROUP BY 1 ORDER BY 1;                                             -- GlobalDailyTotals(): 14 filas
```
Referencia: `2026-09-01 → 10776.0`, `2026-09-14 → 12502.4`.

#### 8.4.13 Consumo del periodo por medidor — `PeriodConsumption(meterID) float64`

`sum(consumption_kwh)` del medidor en todo el periodo. Referencia: `M-109 → 17526.0`, `M-101 → 10226.1`. Alimenta `period_consumption_kwh` en `GET /meters`.

### 8.5 Tipos Go (contrato entre paquetes)

```go
package store

type DailyTotal struct {
    Date            time.Time // 00:00 UTC
    ConsumptionKWh  float64
    VoltageAvg      float64
    CurrentAvg      float64
    PowerFactorAvg  float64
    ReadingsCount   int
    IsComplete      bool
}

type HourStat struct { Hour int; Mean, Std float64; N int }

type Baseline struct {
    MeterID      string
    DailyKWh     float64
    Days         int
    Insufficient bool
    WindowStart  time.Time
    WindowEnd    time.Time
    Profile      [24]HourStat
}

type IngestStats struct {
    Status     string // RUNNING | COMPLETED | FAILED
    Readings   int
    Events     int
    Meters     int
    Rejected   int
    Duplicates int
    DurationMs int64
    StartedAt  time.Time
    FinishedAt time.Time
}
```

## 9. Restricciones y decisiones tomadas

Las restricciones de stack (Go 1.22+, `pgx/v5`, `golang-migrate`, PostgreSQL 16, variables de entorno `DATA_DIR`, `DEMO_USER_EMAIL`, `DEMO_USER_PASSWORD`) están en el maestro, sección 9. Decisiones propias de este spec:

| Decisión/Restricción | Justificación |
|---|---|
| Ingesta solo al arrancar, sin endpoint | El PDF entrega archivos estáticos; un endpoint de carga añadiría superficie de ataque y trabajo sin puntos en la rúbrica. |
| Una transacción por archivo | Permite cargar eventos aunque falle lecturas (y viceversa) y garantiza que nunca queda un archivo a medias. |
| "Última fila gana" en duplicados | Regla determinística y simple de explicar; la alternativa (rechazar ambas) perdería datos válidos. |
| Cálculos on-the-fly, sin materializar | 4.032 filas con índice `(meter_id, timestamp)` se agregan en < 50 ms; materializar añade invalidación y migraciones. |
| Baseline con `stddev_samp` (n − 1) | Convención estadística para muestras; con 7 valores por hora la diferencia con la poblacional es relevante y debe ser una sola. |
| Ventana de baseline global y no por medidor | Todos los medidores comparten periodo; una ventana única evita comparar días distintos entre medidores en el dashboard. |
| `ON CONFLICT (meter_id) DO NOTHING` en `meters` | Preserva `meters.status` escrito por el motor entre reinicios. |
| Estructura de paquetes: `backend/internal/ingest/{csv.go, readings.go, events.go, seed.go}` y `backend/internal/store/{readings.go, aggregates.go}` | DEBERÍA. Separa parseo (ingest) de persistencia y agregados (store); los tests del motor pueden usar `store` con una BD de prueba. |
| Migración `0001_init.up.sql` con el DDL del maestro 8.1 y el índice `readings_meter_ts_idx` | Un solo archivo inicial simplifica el arranque desde cero. |

## 10. Requisitos no funcionales

Los RNF transversales (RNF-01…RNF-12) están en el maestro, sección 10. Propios de este spec:

| # | Requisito | Métrica |
|---|---|---|
| RNF-D-01 | Duración de la ingesta completa (2 archivos, semillas incluidas) | ≤ 5 s en un portátil con 4 vCPU y PostgreSQL local en Docker |
| RNF-D-02 | Escritura por lotes | Inserciones en lotes de ≥ 500 filas (`pgx.Batch` o `COPY`); nunca 1 `INSERT` por fila |
| RNF-D-03 | Latencia de cada función de la sección 8.4 | ≤ 50 ms por llamada con 4.032 lecturas, medido en test de integración |
| RNF-D-04 | Memoria de la ingesta | Streaming por línea con `encoding/csv`; el archivo completo no se carga en memoria (pico ≤ 50 MB para un archivo de 1 millón de filas) |
| RNF-D-05 | Cobertura | ≥ 80 % de líneas en `backend/internal/ingest` |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| [SUPUESTO-D-01] | Los CSV entregados son los definitivos; no llegará una versión con más medidores o más días antes de la demo. Si cambia, solo cambian los valores de referencia de la sección 12a. | Jonnathan Sotelo | — |
| [SUPUESTO-D-02] | Un "día completo" exige exactamente 24 lecturas horarias distintas. Con lecturas cada 15 minutos (96 por día) la regla debería parametrizarse; no aplica al dataset entregado. | Jonnathan Sotelo | — |
| [SUPUESTO-D-03] | `readings.status` no aporta información (siempre `OK` en el dataset); se almacena pero no se usa en ningún cálculo. | Jonnathan Sotelo | — |
| [SUPUESTO-D-04] | Hereda [SUPUESTO-02] (ventana de 7 días), [SUPUESTO-04] (UTC) y [SUPUESTO-01] (`location` fijo) del maestro. | Jonnathan Sotelo | — |
| [DECISIÓN ABIERTA-D-01] | Bloquear endpoints de datos con 503 durante la ingesta (opción elegida) **vs.** servir datos parciales. Criterio: la ingesta dura < 5 s; un 503 breve es preferible a mostrar 3 medidores en el dashboard. Si la ingesta creciera a minutos, revisar. | Jonnathan Sotelo | Antes de la demo |
| [PENDIENTE-D-01] | Confirmar si `pgx.CopyFrom` (COPY) se usa en vez de `pgx.Batch` para lecturas; ambos cumplen RNF-D-02. Se decide en implementación según simplicidad del UPSERT (COPY no admite `ON CONFLICT` directo; requeriría tabla temporal). | Implementador de ingesta | Durante la implementación |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

| Requisito | Verificación | Dónde |
|---|---|---|
| RF-D-01 | Test de arranque con `DATA_DIR` apuntando a fixtures; verificar logs `ingest_started`/`ingest_completed` y 503 durante ingesta (con un fixture grande o un hook que retarde la escritura) | `backend/internal/ingest/ingest_test.go` |
| RF-D-02 | Tests unitarios del parser con fixtures `with_bom_crlf.csv`, `extra_column.csv`, `bad_header.csv` | `backend/internal/ingest/csv_test.go`, `backend/internal/ingest/testdata/` |
| RF-D-03 | Tests unitarios de parseo de timestamps (con y sin segundos), decimales y recorte de espacios | `backend/internal/ingest/csv_test.go` |
| RF-D-04 | Test table-driven con una fila por código de la tabla 8.2; verificar `reason` y `line` | `backend/internal/ingest/readings_test.go`, `events_test.go` |
| RF-D-05 | Test de integración: ingestar dos veces y comparar conteos; fixture `duplicates.csv`; test de rollback cerrando la conexión a mitad | `backend/internal/ingest/ingest_test.go` |
| RF-D-06 | Test de integración: verificar 12 `meters` con nombres; fixture `events_unknown_meter.csv`; verificar que `status` previo se conserva | `backend/internal/ingest/seed_test.go` |
| RF-D-07 | Test HTTP de `GET /health.ingest` tras ingesta | `backend/internal/http/health_test.go` |
| RF-D-08 | Test de integración contra los CSV reales verificando los 12 valores de referencia listados abajo | `backend/internal/store/aggregates_dataset_test.go` |
| RF-D-09 | Fixtures `few_days.csv` (3 días), `one_day.csv`, `zero_baseline.csv` | `backend/internal/store/aggregates_edge_test.go` |
| RF-D-10 | Test que coloca un tercer archivo `otro_archivo.csv` en un `DATA_DIR` temporal y verifica logs y conteos (el test no puede nombrar el archivo prohibido sin violar la propia regla); comando `git grep` de la sección 6 en el `Makefile` (`make check-forbidden`) | `backend/internal/ingest/ingest_test.go`, `Makefile` |
| RNF-D-01/03 | Medir `duration_ms` y tiempos de cada función en el test de integración; fallar si superan el umbral | `aggregates_dataset_test.go` |
| RNF-D-05 | `go test ./internal/ingest/... -cover` | CI local |

Valores de referencia para `aggregates_dataset_test.go` (tolerancia ±0,05 salvo indicación):

| Cálculo | Medidor / fecha | Valor esperado |
|---|---|---|
| `count(readings)` | — | 4032 (exacto) |
| `count(readings)` por medidor | cada uno de los 12 | 336 (exacto) |
| `count(meters)` / `count(events)` | — | 12 / 4 (exacto) |
| `DailyBaseline` | M-101 | 729.4 |
| `DailyBaseline` | M-104 | 1169.7 |
| `DailyBaseline` | M-106 | 1349.1 |
| `DailyBaseline` | M-109 | 1048.8 |
| `DailyBaseline` | M-112 | 662.2 |
| `CurrentConsumption` | M-109 | 2207.6 en `2026-09-14` |
| `Variation` | M-109 | 110.5 (±0,1) |
| `PeriodTotal` | — | 155250.8 (±0,5) |
| `GlobalDailyTotals` | 2026-09-01 / 2026-09-14 | 10776.0 / 12502.4 |
| `HourlyProfile` | M-109 hora 0 | mean 31.7, std 1.6 |

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, autor de la prueba técnica y responsable de la entrega.
- **Qué criterios valida:** RF-D-05 (arrancar dos veces y ver los mismos conteos en `GET /health`), RF-D-07 (contadores visibles en `GET /health`), RF-D-08 (los valores de `M-109` en la pantalla de detalle coinciden con la tabla de referencia: baseline 1.048,8 kWh, consumo actual 2.207,6 kWh, variación +110,5 %) y RF-D-10 (`make check-forbidden` sin coincidencias).
- **Nivel de UAT:** Profundo → UAT formal completo: el firmante valida cada criterio listado con resultado pasa/falla y firma el acta de aceptación antes de la entrega.
