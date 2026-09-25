# Spec: Estrategia de pruebas y aceptación

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8, heredado del maestro) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> Este spec hijo depende de `spec-energy-plataforma-maestro-v1.md` (en adelante "el maestro"). Los tests **por requisito** ya están listados en la sección 12a de cada spec de módulo (`RF-D`, `RF-E`, `RF-B`, `RF-F`); este documento no los repite. Aquí se define lo transversal: qué niveles de prueba existen, con qué herramientas, dónde viven, cómo se ejecutan, qué datos usan, qué umbral de cobertura exigen, cómo se integran en CI, cómo se prueba el arranque y la demo, y cómo se hace la aceptación de usuario. Prefijos: `RF-T`, `CB-T`, `RNF-T`.

## 1. Contexto y problema

Resumen del maestro, sección 1: la plataforma transforma 4.032 lecturas de 12 medidores en anomalías clasificadas, explicadas y priorizadas. La rúbrica del PDF asigna 10 puntos a "Testing / documentación / calidad" y 100 puntos de IA a que el motor produzca exactamente 4 resultados sobre el dataset. Sin una estrategia de pruebas única, cada módulo prueba a su manera, nadie sabe qué comando corre "todas las pruebas", el oráculo de los 4 casos queda disperso en varios archivos, y el riesgo principal (que un cambio de umbral rompa un caso de la rúbrica sin que nadie lo note) no está cubierto por una sola verificación automática. El segundo riesgo es la demo: la app puede pasar todos los tests unitarios y aun así fallar al arrancar con `docker compose` en la máquina del evaluador.

## 2. Objetivo y métricas de éxito

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Un solo comando ejecuta todas las pruebas | No existe | `make test` corre backend y frontend y termina con código 0 | En cada commit y antes de la entrega |
| Oráculo del dataset protegido | Disperso | Un único test de aceptación falla si cualquiera de los 12 medidores cambia de tipo, severidad, confianza o rank | En cada `go test` |
| Cobertura backend | 0 % | `internal/engine ≥ 70 %`, `internal/llm ≥ 60 %`, `internal/http ≥ 50 %`, `internal/ingest ≥ 60 %` | `make test` (falla si baja del umbral) |
| Cobertura frontend | 0 % | `src/lib ≥ 80 %`, `src/pages ≥ 60 %` (líneas) | `npm test` (falla si baja del umbral) |
| Duración de la suite completa | — | ≤ 3 minutos en portátil con 4 vCPU; ≤ 5 minutos en CI | `make test` cronometrado |
| Arranque verificado automáticamente | — | `scripts/smoke` pasa contra `docker compose up` desde cero en ≤ 3 minutos | Antes de cada entrega y en el ensayo de demo |
| CI | — | Workflow de GitHub Actions en verde en `main` antes de la entrega | Última semana |
| Aceptación de usuario | — | Acta firmada con 0 defectos S1 y 0 S2 sin aceptación registrada | Antes de la entrega |

## 3. Glosario

Los términos de dominio están en el maestro, sección 3. Términos propios de este documento:

| Término | Definición |
|---|---|
| Test unitario | Prueba de una función o paquete sin red, sin base de datos y sin sistema de archivos externo al `testdata/` del paquete. |
| Test de integración | Prueba que usa PostgreSQL real (contenedor efímero) y/o el servidor HTTP completo en memoria. |
| Test de aceptación del dataset | Test que carga `data/readings.csv` y `data/events.csv`, ejecuta el motor y compara los 12 medidores contra el oráculo. |
| Oráculo | Tabla de resultados esperados escrita en código (`oracle_test.go`), transcrita de la sección 9 del PDF de la prueba y de los cálculos de este conjunto de specs. Nunca proviene de `expected_results.csv`. |
| Fixture | Archivo CSV o JSON sintético bajo `testdata/` que reproduce un caso borde que el dataset real no cubre. |
| Fake de DeepSeek | Servidor HTTP local (`httptest.Server`) que imita `POST /chat/completions` con respuestas predefinidas. Reemplaza a la API real en todos los tests. |
| Golden file | Archivo JSON con la respuesta esperada de un endpoint; el test compara la respuesta real contra él tras normalizar campos volátiles (`id`, fechas). |
| Test de contrato | Test que verifica que la respuesta de un endpoint tiene exactamente la forma (campos, tipos, enums) del maestro, sección 8.2. |
| E2E | Prueba que abre el frontend en un navegador real contra el backend real y recorre el guion de demo. |
| Smoke test | Script que, contra una instancia ya levantada, verifica en ≤ 60 s que el flujo principal responde: salud, login, medidores, análisis, anomalías. |
| MSW | Mock Service Worker: librería que intercepta `fetch` en tests de frontend y devuelve respuestas definidas por el test. |
| UAT | Aceptación de usuario: validación de que se construyó lo que el dueño de la entrega necesitaba, ejecutada contra los criterios numerados de los specs, distinta de la verificación técnica. |
| S1 / S2 / S3 / S4 | Severidades de defecto: crítico / alto / medio / cosmético (tabla en RF-T-14). |
| Flaky | Test que pasa o falla sin que cambie el código, por depender de tiempo, red u orden. |

## 4. Actores y casos de uso

| Actor | Rol en las pruebas |
|---|---|
| Implementador (persona o agente) | Escribe y ejecuta tests unitarios, de integración, de contrato y frontend; mantiene la cobertura sobre el umbral; corre `make test` antes de cada commit. |
| Firmante (Jonnathan Sotelo, dueño de la entrega) | Ejecuta el UAT contra el guion de demo, registra defectos, firma el acta. |
| Evaluador de la prueba técnica | Levanta la app con `docker compose`, sigue el guion y califica. Es el "usuario final" real; no ejecuta tests. |
| CI (GitHub Actions) | Ejecuta lint, tests y cobertura en cada push; publica el reporte. |

Historias:
- Como implementador quiero un solo comando que corra todas las pruebas, para saber en 3 minutos si rompí algo.
- Como implementador quiero que el oráculo de los 12 medidores sea un test, para que un cambio de umbral que rompa la rúbrica falle en local antes de llegar a la demo.
- Como firmante quiero un guion de UAT derivado de los criterios numerados, para validar la entrega sin releer los specs.
- Como evaluador quiero que la app arranque a la primera, para dedicar los 10 minutos a evaluar y no a depurar.

## 5. Alcance y no-alcance

**Incluye:**
- Definición de niveles de prueba, herramientas, ubicación y comandos para backend (Go) y frontend (React).
- Oráculo del dataset como test de aceptación único.
- Catálogo de fixtures sintéticos compartidos entre módulos.
- Fake de DeepSeek y mock de API para frontend.
- Tests de contrato contra los ejemplos JSON del maestro.
- Umbrales de cobertura con fallo automático.
- Smoke test post-arranque y ensayo de demo.
- Pipeline de CI.
- Proceso de UAT: roles, severidades, criterios de entrada y salida, acta.

**NO incluye (explícito):**
- NO pruebas de carga o estrés (el sistema es monousuario; RNF-03 se mide con 20 peticiones secuenciales, no con herramientas de carga).
- NO llamadas a la API real de DeepSeek en ninguna suite automatizada. La única prueba contra la API real es manual, en el ensayo de demo, y queda registrada en el acta de UAT.
- NO pruebas de seguridad automatizadas (SAST, DAST, escaneo de dependencias) más allá de `go vet`, `eslint` y el grep de secretos de RF-T-10.
- NO pruebas de regresión visual (comparación de screenshots).
- NO pruebas de compatibilidad multi-navegador; solo Chromium (RNF-11 del maestro).
- NO tests de mutación.
- NO generación del plan de UAT con casos derivados en este documento: se genera cuando los marcadores `[PENDIENTE]` y `[DECISIÓN ABIERTA]` de los cinco specs estén cerrados (ver RF-T-14 y sección 11).

## 6. Requisitos funcionales

### RF-T-01 — Niveles de prueba, herramientas y comandos
**Prioridad:** DEBE
**Descripción:** El repositorio DEBE implementar los niveles de la tabla siguiente, con la herramienta, la ubicación y el comando exactos. `make test` DEBE ejecutar, en este orden, los niveles marcados con ✓ en la columna "make test" y terminar con código distinto de 0 si cualquiera falla.

| Nivel | Qué cubre | Herramienta | Ubicación | Comando | make test | CI |
|---|---|---|---|---|---|---|
| Unitario backend | `engine`, `llm` (con fake), `ingest` (parser), `domain`, `config` | `go test` + `testing` estándar; sin librerías de aserción obligatorias (PUEDE usarse `github.com/stretchr/testify`) | `backend/internal/<pkg>/*_test.go` | `cd backend && go test ./... -race -short` | ✓ | ✓ |
| Aceptación del dataset | Motor completo sobre los CSV reales vs. oráculo | `go test` | `backend/internal/engine/engine_dataset_test.go` + `oracle_test.go` | Incluido en el anterior (no lleva `-short` skip) | ✓ | ✓ |
| Integración backend | `store`, ingesta contra Postgres real, handlers HTTP con BD real, job de análisis | `go test` + `github.com/testcontainers/testcontainers-go` (módulo `postgres`) | `backend/internal/store/*_test.go`, `backend/internal/http/*_test.go`, `backend/internal/ingest/ingest_integration_test.go` | `cd backend && go test ./... -race -run Integration` (los tests de integración llevan sufijo `Integration` en el nombre y `t.Skip` si `SKIP_INTEGRATION=1`) | ✓ | ✓ |
| Contrato de API | Forma exacta de cada respuesta del maestro 8.2 | `go test` + golden files | `backend/internal/http/testdata/golden/*.json`, `contract_test.go` | Incluido en integración | ✓ | ✓ |
| Unitario frontend | `src/lib` (formato, etiquetas), hooks, componentes de estado | Vitest 2 + Testing Library + MSW 2 | `frontend/src/**/*.test.ts(x)` | `cd frontend && npm test -- --run --coverage` | ✓ | ✓ |
| Páginas frontend | Render de cada página con datos del maestro vía MSW | Vitest + Testing Library + MSW | `frontend/src/pages/*.test.tsx` | Incluido en el anterior | ✓ | ✓ |
| E2E | Guion de demo en navegador | Playwright 1.4x (Chromium) | `frontend/e2e/demo.spec.ts` | `cd frontend && npx playwright test` (requiere backend y BD levantados) | — | DEBERÍA (job separado, ver RF-T-08) |
| Smoke | Flujo principal contra instancia levantada | Script PowerShell y Bash | `scripts/smoke.ps1`, `scripts/smoke.sh` | `make smoke` (asume `make up` previo) | — | ✓ tras `docker compose up` |
| Estático | Formato, vet, lint, tipos, secretos | `gofmt`, `go vet`, `golangci-lint`, `eslint`, `tsc`, grep | — | `make lint` | — (target aparte) | ✓ |

**Criterios de aceptación:**
- DADO el repositorio en un portátil con Go 1.22, Node 20 y Docker, CUANDO se ejecuta `make test`, ENTONCES corren los niveles marcados ✓, se imprime el porcentaje de cobertura por paquete, y el comando termina en ≤ 3 minutos con código 0.
- DADO `SKIP_INTEGRATION=1`, CUANDO se ejecuta `make test`, ENTONCES los tests de integración se saltan con mensaje `skipped: SKIP_INTEGRATION=1` y el resto corre sin Docker.
- DADO un test unitario que falla, CUANDO se ejecuta `make test`, ENTONCES el código de salida es distinto de 0 y la salida contiene el nombre del test.

### RF-T-02 — Oráculo del dataset como test único
**Prioridad:** DEBE
**Descripción:** El sistema DEBE tener un único archivo `backend/internal/engine/oracle_test.go` con la tabla de resultados esperados para los 12 medidores del dataset entregado, y un test `TestEngineDataset` en `engine_dataset_test.go` que carga `data/readings.csv` y `data/events.csv` (ruta relativa al módulo: `../../../data/`), ejecuta `Pipeline.Run` en modo plantilla (sin API key) y compara cada medidor contra la tabla. El oráculo DEBE llevar un comentario que indique que sus valores provienen de la sección 9 del PDF de la prueba y de `spec-energy-motor-anomalias-ia-v1.md` RF-E-10 a RF-E-13, y NUNCA de `expected_results.csv`.

Tabla del oráculo (contenido exacto):

| `meter_id` | `type` | `severity` | `confidence` | `priority_rank` | `window_start` | `window_end` | `min_evidence` |
|---|---|---|---|---|---|---|---|
| M-109 | `REAL_ANOMALY` | `HIGH` | 0.98 | 1 | 2026-09-12 | 2026-09-14 | 5 |
| M-112 | `DATA_QUALITY` | `HIGH` | 0.98 | 2 | 2026-09-13 | 2026-09-14 | 5 |
| M-104 | `EXPLAINABLE_ANOMALY` | `MEDIUM` | 0.95 | 3 | 2026-09-11 | 2026-09-14 | 5 |
| M-106 | `FALSE_POSITIVE` | `LOW` | 0.80 | 4 | 2026-09-08 | 2026-09-08 | 5 |
| M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111 | sin anomalía | — | — | — | — | — | — |

Además el test DEBE verificar: `detail` final del pipeline = `"4 anomalías · 2 requieren atención prioritaria"`; `recommended_action` por tipo igual a RF-E-15; baseline diario de M-109 = 1048.8 (±0.1); variación de M-109 = +110.5 % (±0.1).

**Criterios de aceptación:**
- DADO el código conforme a los specs, CUANDO se ejecuta `go test ./internal/engine -run TestEngineDataset`, ENTONCES pasa en ≤ 5 s.
- DADO que un implementador cambia `Thresholds.RatioMax` de 2.0 a 1.4, CUANDO se ejecuta el mismo test, ENTONCES falla con un mensaje que contiene `M-109: type = DATA_QUALITY, want REAL_ANOMALY`.
- DADO que `data/readings.csv` no existe en la ruta esperada, CUANDO se ejecuta el test, ENTONCES falla (no se salta) con mensaje `dataset not found: <ruta absoluta>`.

### RF-T-03 — Catálogo de fixtures sintéticos
**Prioridad:** DEBE
**Descripción:** Los fixtures de casos borde DEBEN vivir en `backend/internal/engine/testdata/` (motor) y `backend/internal/ingest/testdata/` (ingesta), ser generados por `backend/internal/engine/testdata/gen/main.go` (programa Go determinístico, sin aleatoriedad, con `go:generate`) y estar versionados en el repositorio. Cada fixture DEBE tener un comentario de cabecera (línea que empieza por `#`, que el parser DEBE ignorar) con el caso que reproduce y el resultado esperado.

| Fixture | Contenido | Resultado esperado | RF que cubre |
|---|---|---|---|
| `short_meter.csv` | 1 medidor, 30 lecturas | `insufficient_data`, sin anomalía | RF-E-03, CB-E-01 |
| `six_days.csv` | 1 medidor, 6 días completos, subida 40 % el día 6 | Baseline = días 1-5, `REAL_ANOMALY MEDIUM` | CB-E-01 |
| `hourly_pattern.csv` | 14 días planos, un día con 8 lecturas al 150 % del perfil sin superar 20 % diario | `REAL_ANOMALY MEDIUM` por patrón horario | RF-E-06 |
| `gap_day.csv` | 14 días, un día con 20 lecturas | Día DQ por (d); `DATA_QUALITY HIGH` | RF-E-08 |
| `drop_no_event.csv` + `events_empty.csv` | Caída 40 % un día, sin eventos | `REAL_ANOMALY MEDIUM` | RF-E-10 |
| `rise_with_outage.csv` + `events_outage.csv` | Subida 40 % con `SCHEDULED_OUTAGE` ese día | `REAL_ANOMALY MEDIUM` | CB-E-10 |
| `long_drop_with_outage.csv` + `events_outage.csv` | Caída 30 % durante 4 días con `SCHEDULED_OUTAGE` el día 1 | `REAL_ANOMALY MEDIUM` | CB-E-11 |
| `dq_with_change.csv` + `events_change.csv` | Días DQ y `OPERATIONAL_CHANGE` | `DATA_QUALITY HIGH` | CB-E-12 |
| `two_events.csv` | Dos eventos del mismo medidor a 0 h y 12 h del inicio | Gana el de 0 h | CB-E-06 |
| `zero_baseline.csv` | 7 días con consumo 0, luego consumo normal | `variation_pct = null`, solo evaluación DQ | CB-E-02 |
| `no_readings_meter.csv` + `events_orphan.csv` | Evento de un medidor sin lecturas | `EVENT_ORPHAN` en log, sin anomalía | CB-E-05, CB-E-21 |
| `ingest_bom_crlf.csv` | Encabezado con BOM y saltos CRLF | Todas las filas insertadas, 0 rechazos, encabezado reconocido sin el BOM | RF-D-02 |
| `ingest_bad_rows.csv` | 10 filas válidas + 5 inválidas (PF > 1, timestamp mal, consumo negativo, meter_id `X-1`, columna vacía) | 10 insertadas, 5 rechazadas con motivo | RF-D-04 |
| `ingest_duplicates.csv` | 3 pares `(meter_id, timestamp)` repetidos | Última gana; `duplicates = 3` | RF-D-05, CB-M10 |
| `ingest_extra_columns.csv` | Columna adicional `note` | Ignorada con warning | RF-D-02 |
| `ingest_missing_column.csv` | Sin `power_factor` | `INGEST_SCHEMA_MISMATCH`, archivo abortado | RF-D-02 |

**Criterios de aceptación:**
- DADO el repositorio, CUANDO se ejecuta `go generate ./internal/engine/testdata/...` dos veces, ENTONCES los archivos generados son byte a byte idénticos entre ejecuciones (`git status` limpio).
- DADO cada fixture de la tabla, CUANDO se ejecuta su test asociado, ENTONCES produce el resultado esperado de la tabla.

### RF-T-04 — Fake de DeepSeek
**Prioridad:** DEBE
**Descripción:** Los tests de `internal/llm` DEBEN usar un `httptest.Server` (`backend/internal/llm/fake_test.go`) que responde a `POST /chat/completions` según un escenario configurado por test, apuntando el cliente con `DEEPSEEK_BASE_URL` a la URL del servidor y `DEEPSEEK_API_KEY=test`. Ningún test DEBE hacer una petición a `api.deepseek.com`; el fake DEBE registrar el número de peticiones recibidas y el cuerpo de cada una para que los tests puedan verificar el payload (RNF-E-03: ≤ 4.096 bytes; sin la cadena `"consumption_kwh":[`, que indicaría lecturas crudas).

Escenarios obligatorios del fake:

| Escenario | Respuesta | Resultado esperado en el `Explainer` |
|---|---|---|
| `ok` | 200 con `choices[0].message.content` = JSON válido del schema 8.3 | `explanation_source = "llm"`, 1 petición |
| `ok_extra_fields` | 200 con JSON válido más `"severity":"LOW","type":"FALSE_POSITIVE"` | Campos ignorados; clasificación intacta |
| `invalid_json` | 200 con `content = "Claro, aquí está la explicación..."` | 2 peticiones, `cause=invalid_json`, plantilla |
| `schema_invalid` | 200 con JSON con `explanation_points` de 1 elemento | 2 peticiones, `cause=schema_invalid`, plantilla |
| `finish_length` | 200 con `finish_reason = "length"` | Tratado como `schema_invalid` |
| `slow` | Duerme `LLM_TIMEOUT_SECONDS + 1` s | 2 peticiones, `cause=timeout`, plantilla; duración total ≤ 2 × timeout + 1 s |
| `unauthorized` | 401 | 1 petición, `cause=http_error`, plantilla para esa y todas las anomalías siguientes del mismo análisis |
| `rate_limited_then_ok` | 429 en la primera, 200 en la segunda | 2 peticiones, `outcome=ok` |
| `server_error` | 500 dos veces | 2 peticiones, `cause=http_error`, plantilla |
| `no_key` | (no se levanta servidor) `DEEPSEEK_API_KEY=""` | 0 peticiones, `explanation_source = "template"` |

**Criterios de aceptación:**
- DADO cada escenario, CUANDO se ejecuta `go test ./internal/llm -run TestExplainer`, ENTONCES el número de peticiones y el resultado coinciden con la tabla.
- DADO el escenario `ok` sobre las 4 anomalías del dataset, ENTONCES cada cuerpo enviado mide ≤ 4.096 bytes y no contiene lecturas horarias.

### RF-T-05 — Integración con PostgreSQL real
**Prioridad:** DEBE
**Descripción:** Los tests de integración DEBEN levantar PostgreSQL 16 con `testcontainers-go` (imagen `postgres:16-alpine`), aplicar las migraciones del repositorio, y ejecutarse sobre una base limpia por test (`TRUNCATE` de todas las tablas en `t.Cleanup`). DONDE `TEST_DATABASE_URL` esté definida, el sistema DEBE usarla en lugar de levantar un contenedor (útil en CI con servicio Postgres). Los tests DEBEN cubrir: ingesta completa de los CSV reales con conteos exactos; idempotencia (dos ingestas); `POST /ai/analyze` + polling hasta `COMPLETED` con los 4 resultados del oráculo persistidos; supersede de anomalías al re-ejecutar; recálculo de `meters.status`; `PATCH /anomalies/:id/status`; y el análisis huérfano marcado `FAILED` al arrancar (RF-B-12).
**Criterios de aceptación:**
- DADO Docker disponible, CUANDO se ejecuta `go test ./... -run Integration`, ENTONCES el contenedor se crea una vez por paquete, todos los tests pasan y el contenedor se elimina al terminar (no quedan contenedores `testcontainers` tras `docker ps -a`).
- DADO `TEST_DATABASE_URL` apuntando a un Postgres levantado, CUANDO se ejecutan los mismos tests, ENTONCES no se crea ningún contenedor y pasan igual.

### RF-T-06 — Tests de contrato de API con golden files
**Prioridad:** DEBE
**Descripción:** Para cada endpoint del maestro 8.2 DEBE existir un golden file en `backend/internal/http/testdata/golden/<nombre>.json` con la respuesta esperada sobre el dataset tras un análisis en modo plantilla, y un test `TestContract_<Endpoint>` que: ejecuta la petición contra el servidor con BD real, normaliza campos volátiles (`id`, `analysis_id`, `detected_at`, `started_at`, `finished_at`, `expires_at`, `token`, `created_at`, `duration_ms` → valor fijo `"<volatile>"`), y compara byte a byte con el golden. El flag `-update` DEBE regenerar los golden files. Golden obligatorios: `login_ok`, `meters_default`, `meters_filter_critical`, `meters_sort_consumption_desc`, `meter_M-109`, `readings_M-109_hour`, `readings_M-109_day`, `anomalies_default`, `anomaly_M-109`, `anomaly_M-112`, `analysis_completed`, `dashboard_summary`, `health_ok`, `error_meter_not_found`, `error_invalid_query`, `error_unauthorized`, `error_analysis_in_progress`.
**Criterios de aceptación:**
- DADO los golden generados, CUANDO un implementador renombra `variation_pct` a `variationPct`, ENTONCES `TestContract_Meters` falla mostrando el diff.
- DADO `go test ./internal/http -run TestContract -update`, ENTONCES los 17 archivos se regeneran y `git diff` muestra solo los cambios reales.

### RF-T-07 — Tests de frontend
**Prioridad:** DEBE
**Descripción:** El frontend DEBE usar Vitest con entorno `jsdom`, Testing Library y MSW 2 para interceptar la API. Los handlers de MSW (`frontend/src/test/handlers.ts`) DEBEN devolver exactamente los JSON de ejemplo del maestro 8.2 (copiados a `frontend/src/test/fixtures/*.json`, un archivo por endpoint, mantenidos en sincronía con los golden del backend mediante el test RF-T-06: el mismo archivo se usa en ambos lados vía `frontend/src/test/fixtures -> ../../../backend/internal/http/testdata/golden` como copia verificada por un test que compara hashes). Cobertura mínima de tests por página según la sección 12a de `spec-energy-frontend-v1.md`, más los siguientes transversales:

| Test | Qué verifica |
|---|---|
| `labels.test.ts` | Mapeo completo de enums (RF-F-11): cada valor del enum tiene etiqueta; un valor desconocido devuelve el literal |
| `format.test.ts` | Formato es-CO: `2207.6 → "2.207,6"`, `110.5 → "+110,5 %"`, `0.98 → "98 %"`, fecha UTC |
| `auth.test.tsx` | Sin token → redirige a `/login`; 401 en cualquier query → limpia sesión y redirige una sola vez |
| `analysis.test.tsx` | Con `vi.useFakeTimers()`: polling cada 1.000 ms, stepper avanza, al `COMPLETED` invalida queries y muestra `"4 anomalías detectadas · 2 requieren atención prioritaria"`; 409 se adjunta al análisis existente; recarga recupera `analysis_id` de `sessionStorage` |
| `pages/*.test.tsx` | Render de cada página con fixtures: loading, datos, vacío, error |

**Criterios de aceptación:**
- DADO `cd frontend && npm test -- --run --coverage`, ENTONCES termina en ≤ 60 s, código 0, y la cobertura de `src/lib` ≥ 80 % y `src/pages` ≥ 60 %; si baja, el comando falla (`coverage.thresholds` en `vitest.config.ts`).
- DADO el fixture `anomalies_default.json` con el orden `M-109, M-112, M-104, M-106`, CUANDO se renderiza la página de anomalías, ENTONCES las filas aparecen en ese orden y la primera muestra "Anomalía real", "Alta", "98 %".

### RF-T-08 — E2E del guion de demo
**Prioridad:** DEBERÍA
**Descripción:** DONDE se implemente (ver [DECISIÓN ABIERTA-F-01] del spec de frontend), `frontend/e2e/demo.spec.ts` DEBE recorrer con Playwright los 8 pasos del guion de demo de `spec-energy-frontend-v1.md` 12b contra `docker compose up` con `DEEPSEEK_API_KEY` vacía, verificando en cada paso el texto visible que ese guion especifica, y DEBE terminar en ≤ 120 s. El test DEBE esperar al `COMPLETED` del análisis con timeout de 60 s.
**Criterios de aceptación:**
- DADO la app levantada, CUANDO se ejecuta `npx playwright test`, ENTONCES el test pasa y genera `playwright-report/` con captura por paso.

### RF-T-09 — Reproducibilidad y rendimiento como tests
**Prioridad:** DEBE
**Descripción:** Los requisitos no funcionales medibles del maestro DEBEN tener test o script:

| RNF | Verificación | Umbral | Dónde |
|---|---|---|---|
| RNF-01 (análisis sin LLM ≤ 15 s) | `time.Since` alrededor del job en el test de integración | Falla si > 15 s | `http/analysis_test.go` |
| RNF-06 (reproducibilidad) | Dos ejecuciones del motor, serialización JSON canónica sin campos volátiles, comparación byte a byte | Igualdad exacta | `engine/pipeline_test.go` |
| RNF-03 (P95 GET ≤ 300 ms) | Script con 20 peticiones a cada GET, calcula P95 | Informe; falla si P95 > 300 ms | `scripts/bench.ps1`, `scripts/bench.sh` |
| RNF-E-01 (motor ≤ 2 s) | `testing.B` `BenchmarkPipeline` + aserción en el test de aceptación | Falla si > 2 s | `engine/engine_dataset_test.go` |
| RNF-E-02 (heap ≤ 100 MB) | `runtime.ReadMemStats` antes y después | Falla si `HeapAlloc` delta > 100 MB | Ídem |
| RNF-E-03 (payload ≤ 4 KB) | Longitud del cuerpo capturado por el fake | Falla si > 4.096 bytes | `llm/explainer_test.go` |

**Criterios de aceptación:**
- DADO `make test`, ENTONCES los tests de la tabla corren y sus umbrales se imprimen con el valor medido (ej. `analysis duration: 3.2s (limit 15s)`).

### RF-T-10 — Calidad estática y control de secretos
**Prioridad:** DEBE
**Descripción:** `make lint` DEBE ejecutar, y fallar si cualquiera falla: `gofmt -l ./backend` (sin salida), `go vet ./...`, `golangci-lint run` con la configuración `backend/.golangci.yml` (linters: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `misspell`), `npm run lint` (ESLint con `@typescript-eslint/recommended` y `react-hooks`), `tsc --noEmit`, y dos verificaciones por grep sobre todo el repositorio excepto `docs/specs`, `.gitignore` y `README.md`: (a) ninguna coincidencia de `expected_results`; (b) ninguna coincidencia de patrones de secretos `sk-[A-Za-z0-9]{20,}`, `JWT_SECRET=.{8,}` fuera de `.env.example`.
**Criterios de aceptación:**
- DADO un archivo Go con una variable sin usar, CUANDO se ejecuta `make lint`, ENTONCES falla en `golangci-lint` con el nombre del archivo.
- DADO un test que contiene la cadena `expected_results.csv`, CUANDO se ejecuta `make lint`, ENTONCES falla con `forbidden reference: expected_results`.

### RF-T-11 — Pipeline de integración continua
**Prioridad:** DEBERÍA
**Descripción:** El repositorio DEBERÍA incluir `.github/workflows/ci.yml` con estos jobs, ejecutados en `push` y `pull_request` sobre `main`:

| Job | Runner | Pasos | Duración objetivo |
|---|---|---|---|
| `lint` | `ubuntu-latest` | checkout, setup-go 1.22, setup-node 20, `make lint` | ≤ 3 min |
| `backend` | `ubuntu-latest` con servicio `postgres:16-alpine` y `TEST_DATABASE_URL` | `go test ./... -race -coverprofile`, verificación de umbrales (script `scripts/coverage-check.sh`), subir `coverage.out` como artefacto | ≤ 5 min |
| `frontend` | `ubuntu-latest` | `npm ci`, `npm test -- --run --coverage`, `npm run build` | ≤ 3 min |
| `smoke` | `ubuntu-latest` | `docker compose up -d --build`, esperar `GET /health` 200 (máx. 180 s), `scripts/smoke.sh`, `docker compose logs` como artefacto en fallo | ≤ 6 min |
| `e2e` (si RF-T-08) | `ubuntu-latest` | depende de `smoke`; `npx playwright test`; subir `playwright-report/` | ≤ 5 min |

**Criterios de aceptación:**
- DADO un push a `main` con el código conforme, ENTONCES los jobs `lint`, `backend`, `frontend` y `smoke` terminan en verde en ≤ 15 minutos en total.
- DADO un push que rompe el oráculo, ENTONCES el job `backend` falla y el resumen del workflow muestra el nombre `TestEngineDataset`.

### RF-T-12 — Smoke test y ensayo de demo
**Prioridad:** DEBE
**Descripción:** `scripts/smoke.ps1` y `scripts/smoke.sh` DEBEN ejecutar contra `API_URL` (default `http://localhost:8080`) esta secuencia y fallar en el primer paso que no cumpla:

| Paso | Petición | Verificación |
|---|---|---|
| 1 | `GET /health` | 200, `db="ok"`, `ingest.status="COMPLETED"`, `ingest.readings=4032` |
| 2 | `POST /auth/login` con credenciales demo | 200, `token` no vacío |
| 3 | `GET /meters` | 200, `total=12` |
| 4 | `POST /ai/analyze` | 202 (o 409 con `details.analysis_id`, en cuyo caso usa ese id) |
| 5 | `GET /ai/analysis/:id` cada 1 s hasta 90 s | `status="COMPLETED"`, `summary.anomalies_detected=4`, `summary.high_priority=2` |
| 6 | `GET /anomalies` | `items[0..3].meter_id` = `M-109, M-112, M-104, M-106` |
| 7 | `GET /meters/M-109` | `status="CRITICAL"`, `variation_pct` entre 110.4 y 110.6 |
| 8 | `GET /dashboard/summary` | `anomalies_total=4`, `high_priority_total=2`, `ai_confidence_avg=0.93` |
| 9 | `GET http://localhost:5173/login` | 200 y el HTML contiene `<div id="root">` |

El **ensayo de demo** es un procedimiento manual documentado en `README.md` sección "Ensayo de demo": `docker compose down -v`, `docker compose up --build` cronometrado (≤ 3 min hasta `/health` 200), `make smoke`, y luego el guion de 8 pasos del spec de frontend con `DEEPSEEK_API_KEY` real, anotando la duración de cada paso y si alguna explicación vino por plantilla. El resultado del ensayo se registra en el acta de UAT (RF-T-14).
**Criterios de aceptación:**
- DADO `docker compose up --build` recién terminado, CUANDO se ejecuta `make smoke`, ENTONCES los 9 pasos pasan en ≤ 60 s y el script imprime `SMOKE OK (9/9)`.
- DADO el backend detenido, CUANDO se ejecuta `make smoke`, ENTONCES falla en el paso 1 con `FAIL step 1: connection refused` y código de salida 1.

### RF-T-13 — Trazabilidad requisito → test
**Prioridad:** DEBE
**Descripción:** Todo requisito con prioridad DEBE de los specs de módulo DEBE tener al menos un test automatizado que lo referencie por número en el nombre o en un comentario de la forma `// covers: RF-E-10, CB-E-12`. El script `scripts/traceability.ps1` (y `.sh`) DEBE listar los RF DEBE de `docs/specs/*.md` (regex `^### (RF-[DEBF]-\d{2})`, excluyendo los que tengan `**Prioridad:** DEBERÍA|PUEDE`), buscar cada uno en `backend/**/*_test.go` y `frontend/src/**/*.test.ts*`, e imprimir la tabla `RF | archivo(s) | estado`. `make lint` DEBE fallar si algún RF DEBE queda sin test.

Resumen de la trazabilidad esperada (los detalles por RF están en la sección 12a de cada spec):

| Spec | RF DEBE | Archivos de test principales |
|---|---|---|
| Data / ingesta | RF-D-01 … RF-D-10 | `ingest/csv_test.go`, `ingest/ingest_integration_test.go`, `store/aggregates_test.go` |
| Motor / IA | RF-E-01 … RF-E-18 | `engine/{pipeline,baseline,detectors,classify,confidence,evidence,engine_dataset}_test.go`, `llm/{explainer,templates}_test.go` |
| Backend API | RF-B-01 … RF-B-18 | `http/{auth,meters,readings,anomalies,analysis,dashboard,health,errors,contract}_test.go`, `cmd/api/main_test.go` |
| Frontend | RF-F-01 … RF-F-14 | `src/lib/*.test.ts`, `src/pages/*.test.tsx`, `src/app/*.test.tsx`, `e2e/demo.spec.ts` |

**Criterios de aceptación:**
- DADO el repositorio completo, CUANDO se ejecuta `scripts/traceability.ps1`, ENTONCES imprime 0 filas con estado `MISSING` y código 0.
- DADO que se elimina el test de RF-B-15, ENTONCES el script imprime `RF-B-15 | — | MISSING` y `make lint` falla.

### RF-T-14 — Aceptación de usuario (UAT)
**Prioridad:** DEBE
**Descripción:** La aceptación de usuario DEBE seguir estas reglas. El plan con casos derivados (`UAT-NN ← RF-NN`) se genera en un documento aparte, `docs/uat/plan-uat-v1.md`, **solo cuando** la sección 11 de los cinco specs de módulo no tenga marcadores `[PENDIENTE]` ni `[DECISIÓN ABIERTA]` abiertos (los `[SUPUESTO]` son compatibles y se confirman durante la ejecución).

**Roles.** Ejecutor: Jonnathan Sotelo. Facilitador (ambiente): Jonnathan Sotelo. Firmante: Jonnathan Sotelo como dueño de la entrega, con la salvedad de [SUPUESTO-T-01]. Aceptación externa real: el evaluador de la prueba técnica, mediante la rúbrica; su resultado no forma parte del acta porque ocurre después de la entrega.

**Derivación de casos.** Un caso `UAT-NN` por cada criterio DADO/CUANDO/ENTONCES de los RF DEBE observables en pantalla (los de `RF-F` completos, y los de `RF-M02` a `RF-M08`, `RF-E-10` a `RF-E-17`, `RF-B-04`, `RF-B-11` a `RF-B-15`). Cada caso: precondiciones, pasos numerados ejecutables por un lector sin contexto, resultado esperado copiado literalmente del criterio, campo de evidencia (captura de pantalla con nombre `uat/evidencia/UAT-NN.png`). Se añaden casos exploratorios `EXP-NN` sin RF: recorrido completo cronometrado, uso con `DEEPSEEK_API_KEY` real, cierre de sesión y reingreso, ventana de 1280 px de ancho.

**Severidad y bloqueo.**

| Severidad | Definición para este producto | Bloquea la entrega |
|---|---|---|
| S1 | Cualquiera de los 4 resultados del oráculo incorrecto en pantalla; la app no arranca con `docker compose`; login imposible; análisis termina `FAILED` con el dataset entregado | Siempre; se corrige y se re-prueba |
| S2 | Un paso del guion de demo no se puede completar o muestra datos distintos a los specs (orden, KPI, estado de medidor); explicación por plantilla con API key válida | Bloquea salvo aceptación escrita del firmante con fecha compromiso, registrada en el acta |
| S3 | Filtro, orden, gráfica o texto secundario incorrecto con alternativa disponible | No bloquea; va al acta con fecha objetivo |
| S4 | Texto, alineación, color sin impacto funcional | No bloquea; backlog |

La severidad la propone quien registra el defecto con justificación contra la tabla; solo puede subirse, nunca bajarse sin dejarlo escrito en el acta.

**Criterios de entrada al UAT** (todos): specs en estado Aprobado con sección 11 cerrada; `make test`, `make lint` y `make smoke` en verde en el commit candidato; plan de UAT con casos derivados; ambiente `docker compose` levantado desde cero en la máquina del ejecutor con el dataset entregado (sin datos personales: los CSV no contienen personas).

**Criterios de salida** (todos): 100 % de los casos derivados ejecutados; 0 S1 abiertos; 0 S2 sin aceptación registrada; S3/S4 listados con fecha; evidencia por caso; acta de sign-off en `docs/uat/acta-uat-v1.md` con: versión de los specs y commit validado, resultados por caso, defectos aceptados con condiciones, resultado del ensayo de demo (RF-T-12), y registro de la aceptación (nombre, rol, fecha, medio).

**Regresión.** Los casos UAT que pasen y no tengan ya test automatizado se listan al final del acta como candidatos a E2E (RF-T-08).

**Criterios de aceptación:**
- DADO un marcador `[DECISIÓN ABIERTA]` abierto en cualquier spec de módulo, CUANDO se solicita generar el plan de UAT, ENTONCES no se genera y se lista el marcador que lo impide.
- DADO el plan generado, ENTONCES cada `UAT-NN` referencia exactamente un criterio `RF-XX-NN` y ningún caso derivado carece de referencia.
- DADO un defecto S1 abierto, ENTONCES no existe acta de sign-off.

## 7. Casos borde y manejo de errores

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-T-01 | Docker no disponible en la máquina (ni `TEST_DATABASE_URL`) | Los tests de integración se saltan con mensaje `skipped: docker not available` y `make test` termina 0; `make smoke` falla con mensaje claro. En CI nunca se salta (falla). |
| CB-T-02 | `DEEPSEEK_API_KEY` definida en el entorno del desarrollador o de CI | Los tests DEBEN sobrescribirla (`t.Setenv("DEEPSEEK_API_KEY", …)`) para que ningún test toque la API real; un test verifica que `DEEPSEEK_BASE_URL` en tests nunca apunta a `api.deepseek.com`. |
| CB-T-03 | Test flaky por tiempo (polling, timeouts) | Prohibido `time.Sleep` como sincronización en tests unitarios; usar canales, `require.Eventually` o timers falsos. En frontend, `vi.useFakeTimers()`. Un test que falle 1 de 20 ejecuciones (`go test -count=20`) se considera defecto S2 del propio test. |
| CB-T-04 | `data/readings.csv` movido o renombrado | `TestEngineDataset` falla con la ruta absoluta buscada; no se salta (RF-T-02). |
| CB-T-05 | Puerto 5432, 8080 o 5173 ocupado durante `make smoke` o E2E | `docker compose` falla al publicar el puerto; el script reporta `port in use: <n>` y sugiere `docker compose down`. |
| CB-T-06 | Runner de CI en zona horaria distinta de UTC | Todos los tests fijan `TZ=UTC` (`t.Setenv("TZ","UTC")` y `process.env.TZ='UTC'` en `vitest.config.ts`). |
| CB-T-07 | Desarrollo en Windows (CRLF, rutas con `\`) | `.gitattributes` fuerza LF en `*.csv`, `*.go`, `*.ts*`, `*.json`, `*.sh`; los tests usan `filepath.Join`; los fixtures generados usan LF. |
| CB-T-08 | Cobertura por debajo del umbral | `scripts/coverage-check.sh` imprime `coverage internal/engine 64.2% < 70%` y falla; el umbral no se baja sin registrar la decisión en la sección 11. |
| CB-T-09 | Golden file desactualizado tras un cambio intencional de contrato | El test falla con diff; el implementador actualiza primero el maestro 8.2, luego regenera con `-update` y revisa el diff en el commit. |
| CB-T-10 | Contenedor de `testcontainers` huérfano por test abortado | `testcontainers` usa Ryuk para limpiar; el `Makefile` incluye `make clean-test` que ejecuta `docker rm -f $(docker ps -aq --filter label=org.testcontainers=true)`. |
| CB-T-11 | La API real de DeepSeek falla durante el ensayo de demo | El ensayo registra que las explicaciones vinieron por plantilla; no es S1 (el fallback es comportamiento especificado) pero sí S2 si la API key es válida, porque la demo pierde valor. |
| CB-T-12 | `go generate` de fixtures produce diff en CI | El job `lint` ejecuta `go generate` y falla si `git status --porcelain` no está vacío. |

## 8. Contratos de datos e interfaces

### 8.1 Estructura de archivos de prueba

```
backend/
  internal/engine/
    oracle_test.go              # tabla RF-T-02
    engine_dataset_test.go      # aceptación sobre CSV reales
    *_test.go                   # unitarios por archivo
    testdata/                   # fixtures RF-T-03
      gen/main.go               # generador determinístico
  internal/llm/
    fake_test.go                # httptest.Server DeepSeek
    explainer_test.go
  internal/http/
    contract_test.go
    testdata/golden/*.json
    testutil_test.go            # levanta servidor + BD para tests
  internal/store/
    postgres_test.go            # helper testcontainers / TEST_DATABASE_URL
frontend/
  vitest.config.ts              # jsdom, coverage.thresholds, TZ=UTC
  src/test/
    setup.ts                    # MSW server, cleanup
    handlers.ts                 # handlers por endpoint
    fixtures/*.json             # copia verificada de los golden
  e2e/demo.spec.ts
scripts/
  smoke.ps1  smoke.sh
  bench.ps1  bench.sh
  traceability.ps1  traceability.sh
  coverage-check.sh
.github/workflows/ci.yml
```

### 8.2 Oráculo en Go (forma exacta)

```go
// oracle_test.go
// Fuente: sección 9 del PDF de la prueba técnica y spec-energy-motor-anomalias-ia-v1.md RF-E-10..13.
// Estos valores NO provienen de expected_results.csv (archivo reservado al evaluador, no presente en el repo).
type oracleCase struct {
    MeterID     string
    Type        string  // "" = sin anomalía
    Severity    string
    Confidence  float64
    Rank        int
    WindowStart string
    WindowEnd   string
    MinEvidence int
}

var datasetOracle = []oracleCase{
    {"M-109", "REAL_ANOMALY", "HIGH", 0.98, 1, "2026-09-12", "2026-09-14", 5},
    {"M-112", "DATA_QUALITY", "HIGH", 0.98, 2, "2026-09-13", "2026-09-14", 5},
    {"M-104", "EXPLAINABLE_ANOMALY", "MEDIUM", 0.95, 3, "2026-09-11", "2026-09-14", 5},
    {"M-106", "FALSE_POSITIVE", "LOW", 0.80, 4, "2026-09-08", "2026-09-08", 5},
    {"M-101", "", "", 0, 0, "", "", 0}, {"M-102", "", "", 0, 0, "", "", 0},
    {"M-103", "", "", 0, 0, "", "", 0}, {"M-105", "", "", 0, 0, "", "", 0},
    {"M-107", "", "", 0, 0, "", "", 0}, {"M-108", "", "", 0, 0, "", "", 0},
    {"M-110", "", "", 0, 0, "", "", 0}, {"M-111", "", "", 0, 0, "", "", 0},
}
```

### 8.3 Respuesta del fake de DeepSeek (escenario `ok`)

```json
{
  "id": "fake-1",
  "object": "chat.completion",
  "model": "deepseek-flash",
  "choices": [
    { "index": 0, "finish_reason": "stop",
      "message": { "role": "assistant",
        "content": "{\"reason\":\"El consumo diario de M-109 subió 110,5 % sobre su baseline de 1.048,8 kWh durante 3 días consecutivos sin evento operativo que lo explique.\",\"recommended_action\":\"Investigar el medidor y la instalación.\",\"explanation_points\":[\"Consumo diario: 2.207,6 kWh frente a 1.048,8 kWh de baseline (+110,5 %).\",\"Corriente media diaria: 420,5 A frente a 200,5 A (+109,7 %).\"]}" } }
  ],
  "usage": { "prompt_tokens": 812, "completion_tokens": 96, "total_tokens": 908 }
}
```

### 8.4 Targets del `Makefile` relacionados con pruebas

| Target | Comando |
|---|---|
| `test` | `cd backend && go test ./... -race -coverprofile=coverage.out && ../scripts/coverage-check.sh coverage.out && cd ../frontend && npm test -- --run --coverage` |
| `test-unit` | `cd backend && go test ./... -race -short` |
| `test-integration` | `cd backend && go test ./... -race -run Integration` |
| `lint` | `gofmt`, `go vet`, `golangci-lint`, `eslint`, `tsc`, greps, `scripts/traceability.*` |
| `smoke` | `scripts/smoke.sh` o `.ps1` según SO |
| `bench` | `scripts/bench.*` |
| `e2e` | `cd frontend && npx playwright test` |
| `clean-test` | Limpieza de contenedores de prueba |

### 8.5 Formato de la tabla de defectos del UAT

| Campo | Valor |
|---|---|
| `id` | `DEF-NN` |
| `caso` | `UAT-NN` o `EXP-NN` de origen |
| `descripción` | Pasos para reproducir + resultado observado + resultado esperado (literal del criterio) |
| `severidad` | S1-S4 con justificación de una línea |
| `estado` | `abierto` / `corregido` / `aceptado` / `descartado` |
| `decisión` | Quién, cuándo y por qué medio (solo para `aceptado`) |
| `evidencia` | Ruta a captura o log |

## 9. Restricciones y decisiones tomadas

Las restricciones de stack están en el maestro, sección 9. Decisiones propias de las pruebas:

| Decisión/Restricción | Justificación |
|---|---|
| El oráculo vive en un solo archivo Go, no en JSON ni CSV | Evita cualquier confusión con `expected_results.csv`; los valores se leen en el code review y no pueden "cargarse" desde fuera. |
| `testcontainers-go` en lugar de mocks de base de datos | Los tests de `store` y de handlers deben probar SQL real; 4.032 filas caben en segundos. `TEST_DATABASE_URL` como alternativa evita Docker-en-Docker en CI. |
| Los handlers HTTP se prueban con el motor real, no simulado | El motor es determinístico y corre en < 2 s; simularlo ocultaría errores de integración entre etapas. Solo el LLM se simula. |
| Fake de DeepSeek por HTTP (`httptest`) y no por interfaz Go | Prueba también la serialización, cabeceras, `BaseURL`, timeouts y errores HTTP reales de `go-openai`. Una interfaz falsa no probaría nada de eso. |
| Golden files compartidos con el frontend | Un solo origen de verdad para la forma de la API; si el backend cambia un campo, los tests de frontend fallan en el mismo commit. |
| Sin librería de aserciones obligatoria en Go | Menos dependencias; `testify` se permite por comodidad, no se exige. |
| Playwright como DEBERÍA y no DEBE | Depende del tiempo disponible ([DECISIÓN ABIERTA-F-01]); el smoke test cubre el flujo principal por API con costo mucho menor. |
| Umbrales de cobertura por paquete y no globales | La cobertura global ocultaría un `engine` mal cubierto compensado por handlers triviales. |
| Prohibición de red real en tests | Reproducibilidad y costo cero; la única prueba contra DeepSeek real es el ensayo de demo, manual y registrado. |
| El UAT usa el dataset real, no datos enmascarados | Los CSV no contienen personas ni datos sensibles; el dataset es el objeto mismo de la evaluación. |

## 10. Requisitos no funcionales

| # | Requisito | Métrica |
|---|---|---|
| RNF-T-01 | Duración de `make test` | ≤ 3 min local (4 vCPU), ≤ 5 min en CI |
| RNF-T-02 | Duración de la suite unitaria backend (`-short`) | ≤ 30 s |
| RNF-T-03 | Duración de la suite frontend | ≤ 60 s |
| RNF-T-04 | Determinismo | `go test ./... -count=20` sin ningún fallo; `vitest --run` 5 veces seguidas sin fallo |
| RNF-T-05 | Aislamiento de red | 0 conexiones salientes durante `make test` (verificable con `DEEPSEEK_BASE_URL` apuntando a `http://127.0.0.1:1` en el entorno de CI: todo test que intente red falla de inmediato) |
| RNF-T-06 | Tamaño de fixtures | ≤ 200 KB en total bajo `testdata/` |
| RNF-T-07 | Evidencia de UAT | Una captura por caso derivado, ≤ 500 KB cada una, en `docs/uat/evidencia/` |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| [SUPUESTO-T-01] | En una prueba técnica individual el autor de los specs, el implementador y el firmante del UAT son la misma persona. Se acepta esta excepción a la separación de roles porque no existe otro dueño del proceso antes de la entrega; la aceptación independiente real la hace el evaluador con la rúbrica. Se registra en el acta. | Jonnathan Sotelo | — |
| [SUPUESTO-T-02] | Los runners de GitHub Actions tienen Docker disponible para el job `smoke` (cierto para `ubuntu-latest` a la fecha). | Jonnathan Sotelo | — |
| [PENDIENTE-T-01] | El plan de UAT con casos derivados no se puede generar hasta cerrar estos marcadores en los specs de módulo: maestro [PENDIENTE-01] (fecha de entrega), [PENDIENTE-02] (demo en vivo o grabada), [PENDIENTE-03] (`response_format` en DeepSeek), [DECISIÓN ABIERTA-01] (estado de medidor con falso positivo); data [PENDIENTE-D-01], [DECISIÓN ABIERTA-D-01]; motor [DECISIÓN ABIERTA-E-01]; backend [PENDIENTE-B01], [PENDIENTE-B02], [DECISIÓN ABIERTA-B01]; frontend [PENDIENTE-F-02], [DECISIÓN ABIERTA-F-01]. | Jonnathan Sotelo | Antes del UAT |
| [DECISIÓN ABIERTA-F-01] (frontend) | Playwright E2E sí o no. Afecta RF-T-08 y el job `e2e`. | Jonnathan Sotelo | Antes de la entrega |
| [DECISIÓN ABIERTA-T-01] | Versión de `golangci-lint`: fijarla en el `Makefile` (instalación reproducible; añade ≈ 1 min a la primera ejecución de `make lint`) vs. usar la del sistema. Criterio: si el evaluador va a correr `make lint`, fijarla. | Implementador | Al crear el `Makefile` |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

Este spec se verifica por la existencia y el resultado de lo que exige:

| Requisito | Verificación |
|---|---|
| RF-T-01 | `make test` existe y termina 0; `SKIP_INTEGRATION=1 make test` termina 0 sin Docker |
| RF-T-02 | `go test ./internal/engine -run TestEngineDataset -v` pasa; mutar `RatioMax` a 1.4 lo hace fallar con el mensaje esperado |
| RF-T-03 | `go generate` idempotente; cada fixture tiene su test |
| RF-T-04 | `go test ./internal/llm -v` muestra los 10 escenarios; `grep -r "api.deepseek.com" backend/internal/llm/*_test.go` vacío |
| RF-T-05 | `go test ./... -run Integration` pasa con Docker y con `TEST_DATABASE_URL` |
| RF-T-06 | 17 golden files existen; renombrar un campo rompe el test |
| RF-T-07 | `npm test -- --run --coverage` con umbrales; hash de fixtures = hash de golden |
| RF-T-09 | Los umbrales aparecen impresos en la salida de `make test` |
| RF-T-10 | `make lint` falla ante variable sin usar y ante la cadena `expected_results` fuera de las rutas permitidas |
| RF-T-11 | Workflow en verde en `main` |
| RF-T-12 | `make smoke` imprime `SMOKE OK (9/9)` tras `docker compose up --build` |
| RF-T-13 | `scripts/traceability.*` sin `MISSING` |
| RF-T-14 | Existe `docs/uat/plan-uat-v1.md` derivado y `docs/uat/acta-uat-v1.md` con criterios de salida cumplidos |

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, dueño de la entrega (ver [SUPUESTO-T-01]).
- **Qué criterios valida:** RF-T-01 (existe un solo comando), RF-T-12 (el ensayo de demo pasa) y RF-T-14 (el acta cumple los criterios de salida). El resto de RF-T son verificación técnica y no requieren aceptación de usuario.
- **Nivel de UAT:** Profundo → el firmante ejecuta `make test`, `make lint` y `make smoke` sobre el commit candidato, registra el resultado con captura, y firma el acta junto con la del producto.
