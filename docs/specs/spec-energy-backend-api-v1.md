# Spec: AI Energy Management Platform — Backend API (Go)

| Campo | Valor |
|---|---|
| Autor | Jonnathan Sotelo |
| Fecha | 2026-09-25 |
| Versión | v1.0 |
| Estado | Aprobado |
| Nivel | Profundo (D8, heredado del maestro) |
| Revisores | Jonnathan Sotelo (autor y único revisor; prueba técnica individual) |
| Área | Energy Management — prueba técnica |

> Spec hijo de `spec-energy-plataforma-maestro-v1.md` (en adelante, "el maestro"). Los contratos JSON de request/response de cada endpoint viven en la sección 8.2 del maestro y **no se repiten aquí**; este documento especifica lo que el maestro no dice: estructura del servicio, middleware, validaciones, reglas de orden, comportamiento interno del análisis asíncrono, errores, observabilidad y tests. Ante cualquier diferencia entre este documento y el maestro, manda el maestro.
>
> Dependencias: este backend invoca el motor especificado en `spec-energy-motor-anomalias-ia-v1.md` (paquete `internal/engine`) y la ingesta especificada en `spec-energy-data-ingesta-v1.md` (paquete `internal/ingest`). Este spec define cómo se orquestan, no qué calculan.

## 1. Contexto y problema

Resumen del maestro, sección 1: la empresa tiene 4.032 lecturas horarias de 12 medidores y 4 eventos operativos en CSV, y nadie puede responder en menos de una hora qué medidor se salió de su comportamiento ni si el cambio es real, explicable o un problema de datos. El backend es la pieza que persiste esos datos, ejecuta el análisis de IA bajo demanda y expone todo por HTTP al frontend. Sin él no existe la aplicación: es el único proceso que toca la base de datos y el único que llama al LLM.

## 2. Objetivo y métricas de éxito

Resultado: una API HTTP en Go que un frontend y un evaluador pueden consumir sin leer código, que produce los 4 resultados esperados del dataset (maestro, sección 2) y que arranca con `docker compose up --build`.

| Métrica | Hoy | Objetivo | Cuándo se mide |
|---|---|---|---|
| Endpoints del maestro implementados (8.2.1 a 8.2.10) | 0 de 10 | 10 de 10 responden según su contrato | Tests HTTP de 12a |
| `POST /ai/analyze` sobre el dataset entregado | — | Termina `COMPLETED` con `summary.anomalies_detected = 4` y `summary.high_priority = 2` | Test de integración `analysis_test.go` |
| Latencia de GET con 4.032 lecturas | — | P95 ≤ 300 ms en 20 peticiones consecutivas por endpoint | `scripts/bench.ps1` |
| Cobertura de tests | 0 % | ≥ 50 % en `internal/http`, ≥ 70 % en `internal/engine` (este último se mide en el spec del motor) | `go test ./... -race -cover` |
| Lint | — | `go vet ./...` y `golangci-lint run` sin errores | Antes de cada commit |

## 3. Glosario

Aplica íntegro el glosario del maestro, sección 3 (medidor, lectura, baseline, anomalía, tipos, severidad, confianza, análisis, pipeline, estado de medidor, anomalía vigente, JWT). Términos adicionales propios de este spec:

| Término | Definición |
|---|---|
| Handler | Función Go que atiende una ruta HTTP y devuelve una respuesta JSON. |
| Middleware | Función que envuelve a todos los handlers y se ejecuta antes/después de ellos (logging, auth, etc.). |
| Request ID | Identificador único por petición HTTP (UUID v4), generado por el primer middleware y devuelto en el header `X-Request-Id`. |
| Job de análisis | Goroutine que ejecuta el pipeline de 7 etapas fuera del ciclo de vida de la petición HTTP que lo disparó. |
| Análisis huérfano | Fila de `analyses` con `status` `QUEUED` o `RUNNING` cuyo proceso ya no existe (el servidor se reinició mientras corría). |
| Graceful shutdown | Al recibir `SIGTERM`/`SIGINT`, el servidor deja de aceptar conexiones nuevas, espera hasta 10 s a que terminen las peticiones en curso y luego cierra la base de datos y el proceso. |
| Store | Paquete `internal/store`: única capa que ejecuta SQL. Ningún handler ni el motor ejecutan SQL directamente. |

## 4. Actores y casos de uso

Aplican los actores del maestro, sección 4 (analista de operaciones, evaluador, sistema/job de análisis). Desde la perspectiva del backend hay dos consumidores: el frontend (`spec-energy-frontend-v1.md`), que envía el JWT en cada petición, y las herramientas de prueba (`curl`, tests `httptest`), que lo obtienen de `POST /auth/login`.

## 5. Alcance y no-alcance

**Incluye:**
- Servicio HTTP en Go con los 10 endpoints del maestro (8.2.1 a 8.2.10).
- Arranque completo con un solo binario: configuración, conexión a PostgreSQL con reintentos, migraciones embebidas, ingesta de CSV, semilla de usuario demo, servidor HTTP con graceful shutdown.
- Middleware: request-id, logging JSON, recover, CORS, auth JWT, timeout por petición, límite de tamaño de body.
- Ejecución asíncrona del análisis con persistencia de progreso por etapa, exclusión mutua y recuperación de análisis huérfanos.
- Mapa de errores único, validación de parámetros de query y body.
- `Dockerfile` multi-stage, `Makefile`, `.env.example`, tests unitarios e integración.

**NO incluye (explícito):**
- NO la lógica de detección, clasificación, severidad, confianza ni explicación: vive en `internal/engine` e `internal/llm` y se especifica en el spec del motor. Este spec solo define cómo se invoca y cómo se persiste su resultado.
- NO el parseo y validación de los CSV: vive en `internal/ingest` y se especifica en el spec de data. Este spec solo define cuándo se invoca y qué pasa si falla.
- NO gestión de usuarios (registro, roles, recuperación de contraseña), NO refresh tokens, NO logout en servidor (el frontend borra el token).
- NO paginación en `GET /meters` ni `GET /anomalies` (maestro, sección 5).
- NO cola externa (Redis, RabbitMQ), NO workers separados, NO cron de análisis automático.
- NO versionado de API (`/v1`), NO OpenAPI generado automáticamente (PUEDE incluirse un `openapi.yaml` escrito a mano; no es requisito).
- NO HTTPS/TLS en el propio proceso (se asume `localhost` o un proxy externo).
- NO endpoints de escritura sobre `meters`, `readings` ni `events`.

## 6. Requisitos funcionales

Numeración `RF-B-NN`. Los contratos de request/response referenciados como "maestro 8.2.N" son normativos y no se repiten.

### RF-B-01 — Estructura del proyecto y módulo Go
**Prioridad:** DEBE
**Descripción:** El backend DEBE vivir en `backend/` con la siguiente estructura exacta y el módulo Go `github.com/jsotelo/energy-platform/backend` ([SUPUESTO-B01]):

```
backend/
  cmd/api/main.go                 # punto de entrada; solo cablea dependencias
  internal/config/                # lectura y validación de variables de entorno
  internal/http/                  # router, middleware, handlers, DTOs de respuesta
  internal/domain/                # tipos de dominio, enums, derivación de estado (tabla 8.3 del maestro)
  internal/store/                 # acceso a PostgreSQL (pgx); única capa con SQL
  internal/engine/                # motor de anomalías (spec del motor)
  internal/llm/                   # cliente DeepSeek (go-openai) + fallback de explicación (spec del motor)
  internal/ingest/                # carga de CSV (spec de data)
  migrations/0001_init.up.sql     # DDL exacto de la sección 8.1 del maestro
  migrations/0001_init.down.sql   # DROP de las 7 tablas en orden inverso
  Dockerfile                      # multi-stage: golang:1.22-alpine (build) → alpine:3.20 (runtime)
  go.mod, go.sum
  Makefile
  .env.example
```
Las migraciones DEBEN embeberse en el binario con `embed.FS` y ejecutarse con `golang-migrate` al arrancar.
**Criterios de aceptación:**
- DADO el repositorio clonado, CUANDO se ejecuta `cd backend && go build ./...`, ENTONCES compila sin errores y existe cada ruta de la lista anterior.
- DADO el binario compilado, CUANDO se inspecciona con `go list -m`, ENTONCES el módulo es `github.com/jsotelo/energy-platform/backend`.

### RF-B-02 — Secuencia de arranque
**Prioridad:** DEBE
**Descripción:** CUANDO el proceso arranca, el sistema DEBE ejecutar, en este orden y abortando en el paso que falle salvo donde se indica: (1) leer configuración (variables de la sección 9 del maestro) y validar que `JWT_SECRET` no esté vacío; (2) conectar a PostgreSQL reintentando cada 2 s hasta 30 intentos (CB-M02 del maestro), terminando con exit code 1 y log `DB_UNAVAILABLE` si agota; (3) aplicar migraciones pendientes; (4) marcar como `FAILED` con `error_message = "process restarted"` todo análisis con `status IN ('QUEUED','RUNNING')` (análisis huérfanos); (5) ejecutar la ingesta de `DATA_DIR/readings.csv` y `DATA_DIR/events.csv` — SI falta un archivo, ENTONCES registrar `INGEST_FILE_MISSING` con la ruta absoluta y continuar (CB-M01 del maestro); (6) sembrar el usuario demo con `DEMO_USER_EMAIL` / `DEMO_USER_PASSWORD` (upsert por `email`, hash bcrypt cost 10); (7) escuchar en `:PORT`. CUANDO recibe `SIGTERM` o `SIGINT`, el sistema DEBE dejar de aceptar conexiones, esperar hasta 10 s a las peticiones en curso, cerrar el pool de BD y salir con exit code 0.
**Criterios de aceptación:**
- DADO PostgreSQL detenido, CUANDO arranca el backend, ENTONCES los logs muestran ≥ 1 línea `{"level":"warn","msg":"db retry","attempt":N}` y, al levantar PostgreSQL antes del intento 30, el arranque continúa y `GET /health` devuelve 200.
- DADO una fila en `analyses` con `status='RUNNING'`, CUANDO el backend arranca, ENTONCES esa fila queda `status='FAILED'`, `error_message='process restarted'`, `finished_at` no nulo.
- DADO `JWT_SECRET=""`, CUANDO el backend arranca, ENTONCES termina con exit code 1 y log `CONFIG_INVALID` con `field="JWT_SECRET"`.
- DADO el backend sirviendo una petición de 3 s, CUANDO recibe `SIGTERM`, ENTONCES la petición completa con 200 y el proceso sale con código 0 en ≤ 10 s.

### RF-B-03 — Cadena de middleware
**Prioridad:** DEBE
**Descripción:** El router DEBE aplicar a toda petición, en este orden: (1) **request-id**: genera UUID v4 y lo escribe en `X-Request-Id` de la respuesta; si la petición ya trae `X-Request-Id`, lo reutiliza; (2) **logger**: al terminar la petición emite una línea JSON con los campos de RF-B-14; (3) **recover**: SI un handler entra en pánico, ENTONCES responde 500 `INTERNAL_ERROR` con `message="Error interno"` sin incluir stack ni mensaje del pánico en el body, y registra el stack completo en log con `level="error"`; (4) **CORS**: permite los orígenes exactos de `CORS_ALLOWED_ORIGINS` (lista separada por comas), métodos `GET, POST, PATCH, OPTIONS`, headers `Authorization, Content-Type, X-Request-Id`, y responde `204` a `OPTIONS`; (5) **body limit**: rechaza bodies > 1 MB con 413 `PAYLOAD_TOO_LARGE`; (6) **timeout**: cancela el contexto de la petición a los 30 s y responde 503 `REQUEST_TIMEOUT` si el handler no terminó; (7) **auth JWT**: en toda ruta salvo `GET /health` y `POST /auth/login`, exige `Authorization: Bearer <token>` válido (RF-B-04); si falta o es inválido responde 401 `UNAUTHORIZED`.
**Criterios de aceptación:**
- DADO cualquier petición, CUANDO se responde, ENTONCES el header `X-Request-Id` está presente y coincide con el campo `request_id` de la línea de log.
- DADO un handler de prueba que hace `panic("boom")`, CUANDO se invoca, ENTONCES la respuesta es 500 con body exactamente `{"error":{"code":"INTERNAL_ERROR","message":"Error interno","details":null}}` y el log contiene `"boom"`.
- DADO `CORS_ALLOWED_ORIGINS=http://localhost:5173`, CUANDO llega `OPTIONS /meters` con `Origin: http://evil.local`, ENTONCES la respuesta no incluye `Access-Control-Allow-Origin`.
- DADO `GET /meters` sin header `Authorization`, CUANDO se invoca, ENTONCES responde 401 `UNAUTHORIZED`.
- DADO `GET /health` sin header `Authorization`, CUANDO se invoca, ENTONCES responde 200.

### RF-B-04 — Autenticación (`POST /auth/login`, maestro 8.2.1)
**Prioridad:** DEBE
**Descripción:** CUANDO llega `POST /auth/login` con body `{ "email", "password" }`, el sistema DEBE buscar el usuario por `email` (comparación exacta, sin normalizar mayúsculas) y comparar `password` contra `password_hash` con `bcrypt.CompareHashAndPassword`. SI el usuario no existe O la contraseña no coincide, ENTONCES DEBE responder 401 `INVALID_CREDENTIALS` con body idéntico en ambos casos y con tiempo de respuesta equivalente (ejecutar la comparación bcrypt contra un hash fijo cuando el usuario no existe). Si coincide, DEBE emitir un JWT HS256 firmado con `JWT_SECRET` y claims exactos: `sub` = email, `name` = nombre del usuario, `iat` = ahora (epoch s), `exp` = `iat + 28800` (8 h). La respuesta sigue el maestro 8.2.1 con `expires_at` = `exp` en RFC 3339 UTC. El sistema DEBERÍA limitar a 10 intentos por minuto por IP de origen (`X-Forwarded-For` si existe, si no `RemoteAddr`), respondiendo 429 `TOO_MANY_ATTEMPTS` al exceder.
**Criterios de aceptación:**
- DADO el usuario demo sembrado, CUANDO se envía `{"email":"analista@energy.local","password":"Demo1234!"}`, ENTONCES responde 200, el token decodificado tiene `sub="analista@energy.local"` y `exp - iat = 28800`.
- DADO el usuario demo sembrado, CUANDO se envía la contraseña `"incorrecta"`, ENTONCES responde 401 con body `{"error":{"code":"INVALID_CREDENTIALS","message":"Credenciales inválidas","details":null}}`.
- DADO un email inexistente `"nadie@energy.local"`, CUANDO se envía cualquier contraseña, ENTONCES responde 401 con el mismo body exacto del criterio anterior.
- DADO 11 peticiones de login fallidas desde la misma IP en 60 s, CUANDO llega la número 11, ENTONCES responde 429 `TOO_MANY_ATTEMPTS` (solo si se implementa el rate limit; es DEBERÍA).

### RF-B-05 — Validación del token JWT
**Prioridad:** DEBE
**Descripción:** CUANDO el middleware de auth recibe un token, el sistema DEBE verificar: algoritmo exactamente `HS256` (rechazar `none` y cualquier otro), firma con `JWT_SECRET`, `exp` en el futuro, y presencia de `sub`. SI cualquier verificación falla, ENTONCES 401 `UNAUTHORIZED` con `message` fijo `"Token inválido o expirado"` (sin distinguir la causa en el body; la causa va al log con `reason ∈ {missing, malformed, bad_signature, expired, bad_alg}`). El email del `sub` DEBE quedar en el contexto de la petición para logging.
**Criterios de aceptación:**
- DADO un token firmado con otro secreto, CUANDO se envía a `GET /meters`, ENTONCES 401 y log con `reason="bad_signature"`.
- DADO un token con `exp` en el pasado, CUANDO se envía, ENTONCES 401 y log con `reason="expired"`.
- DADO un token con header `alg=none`, CUANDO se envía, ENTONCES 401 y log con `reason="bad_alg"`.
- DADO `Authorization: Basic xxx`, CUANDO se envía, ENTONCES 401 y log con `reason="malformed"`.

### RF-B-06 — Listado de medidores (`GET /meters`, maestro 8.2.2)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE validar los parámetros de query: `status` (repetible) con valores exactos del conjunto `{OK, ALERT, CRITICAL}`, `q` (texto, máximo 20 caracteres), `sort` ∈ `{consumption, variation, severity}` con default `severity`, `order` ∈ `{asc, desc}` con default `desc`. SI un valor no pertenece a su conjunto, ENTONCES 400 `INVALID_QUERY` con `details = { "param": "<nombre>", "allowed": [<valores>] }`; SI `q` supera 20 caracteres, ENTONCES 400 `INVALID_QUERY` con `details = { "param": "q", "reason": "max length 20" }`. Varios `status` se combinan con OR. `q` filtra por `meter_id ILIKE '%' || q || '%'`. Los campos calculados (`current_consumption_kwh`, `baseline_daily_kwh`, `variation_pct`, `period_consumption_kwh`) siguen las definiciones del glosario del maestro y se redondean a 1 decimal. El comparador de orden DEBE ser:
- `sort=consumption`: por `current_consumption_kwh`; desempate por `meter_id` ascendente.
- `sort=variation`: por `variation_pct` (valor con signo, no absoluto); medidores con `variation_pct = null` van al final en ambos `order`; desempate por `meter_id` ascendente.
- `sort=severity`: clave compuesta (a) rango de estado `CRITICAL=3, ALERT=2, OK=1`; (b) rango de severidad de la anomalía vigente `HIGH=3, MEDIUM=2, LOW=1, sin anomalía=0`; (c) `confidence` de la anomalía vigente (0 si no hay); (d) `meter_id` ascendente. `order=desc` invierte (a), (b) y (c); (d) siempre ascendente.
**Criterios de aceptación:**
- DADO el análisis completado sobre el dataset, CUANDO se consulta `GET /meters` sin parámetros, ENTONCES `items[0].meter_id="M-109"`, `items[1].meter_id="M-112"`, `items[2].meter_id="M-104"`, `items[3].meter_id="M-106"`, y `items[4..11]` son `M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111` en ese orden; `total=12`.
- DADO el análisis completado, CUANDO se consulta `?status=CRITICAL`, ENTONCES `items` contiene solo `M-109`.
- DADO el análisis completado, CUANDO se consulta `?status=CRITICAL&status=ALERT`, ENTONCES `items` contiene `M-109, M-112, M-104` y `total=3`.
- DADO cualquier estado, CUANDO se consulta `?q=10`, ENTONCES `items` contiene exactamente `M-101` a `M-110` (10 medidores: `M-110` también contiene la subcadena "10"; corregido en implementación, la v1.0 decía 9).
- DADO cualquier estado, CUANDO se consulta `?sort=consumption&order=desc`, ENTONCES `items[0].meter_id="M-109"` (2.207,6 kWh) y `items[11].meter_id="M-107"`.
- DADO cualquier estado, CUANDO se consulta `?sort=foo`, ENTONCES 400 con `details={"param":"sort","allowed":["consumption","variation","severity"]}`.

### RF-B-07 — Detalle de medidor (`GET /meters/:meterId`, maestro 8.2.3)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE buscar el medidor por igualdad exacta de `meter_id` (sin normalizar mayúsculas ni espacios). SI no existe, ENTONCES 404 `METER_NOT_FOUND` con `message="No existe el medidor <meterId>"`. La respuesta incluye `daily` con los 14 días ordenados por fecha ascendente, `events` del medidor ordenados por `timestamp` ascendente (incluyendo los de tipo `UNKNOWN`), y `electrical` con promedios del último día completo y de la ventana de baseline redondeados a 1 decimal (voltaje, corriente) y 3 decimales (PF).
**Criterios de aceptación:**
- DADO el dataset cargado, CUANDO se consulta `GET /meters/M-109`, ENTONCES `daily` tiene 14 elementos, `daily[13].date="2026-09-14"`, `daily[13].consumption_kwh=2207.6`, `daily[13].deviation_pct=110.5`, `electrical.current_a.current_avg=420.5`, `electrical.current_a.baseline_avg=200.5`, `events` tiene 1 elemento con `type="UNKNOWN"`.
- DADO el dataset cargado, CUANDO se consulta `GET /meters/m-109`, ENTONCES 404 `METER_NOT_FOUND`.
- DADO el dataset cargado, CUANDO se consulta `GET /meters/M-999`, ENTONCES 404 con `message="No existe el medidor M-999"`.

### RF-B-08 — Lecturas de medidor (`GET /meters/:meterId/readings`, maestro 8.2.4)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE validar: `from` y `to` en RFC 3339 (`2026-09-12T00:00:00Z`) o fecha `YYYY-MM-DD` (interpretada como 00:00:00 UTC para `from` y 23:59:59 UTC para `to`); ambos opcionales e independientes (si falta `from`, se usa el inicio del periodo; si falta `to`, el fin); SI `from > to`, ENTONCES 400 `INVALID_QUERY` con `details={"param":"from","reason":"from must be <= to"}`; `granularity` ∈ `{hour, day}`, default `hour`. Con `granularity=hour`, `is_outlier` se calcula con el perfil horario baseline (z-score ≥ 3, definición del maestro) y `baseline_profile` trae exactamente 24 elementos ordenados por `hour`. Con `granularity=day` se omite `baseline_profile` y `items` agrupa por fecha UTC. Un rango sin lecturas devuelve `items=[]` con 200.
**Criterios de aceptación:**
- DADO el dataset, CUANDO se consulta `GET /meters/M-109/readings`, ENTONCES `items` tiene 336 elementos ordenados por `timestamp` ascendente y `baseline_profile` tiene 24 elementos con `baseline_profile[0].mean_kwh=31.7`.
- DADO el dataset, CUANDO se consulta `?from=2026-09-14&to=2026-09-14`, ENTONCES `items` tiene 24 elementos y el elemento con `timestamp="2026-09-14T13:00:00Z"` tiene `consumption_kwh=117.51` e `is_outlier=true`.
- DADO el dataset, CUANDO se consulta `?granularity=day`, ENTONCES `items` tiene 14 elementos y la respuesta no contiene la clave `baseline_profile`.
- DADO el dataset, CUANDO se consulta `?from=2026-09-20&to=2026-09-21`, ENTONCES 200 con `items=[]`.
- DADO el dataset, CUANDO se consulta `?from=2026-09-14&to=2026-09-01`, ENTONCES 400 `INVALID_QUERY`.
- DADO el dataset, CUANDO se consulta `?from=14/09/2026`, ENTONCES 400 `INVALID_QUERY` con `details.param="from"`.

### RF-B-09 — Listado de anomalías (`GET /anomalies`, maestro 8.2.5)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE validar `type` ∈ `{REAL_ANOMALY, EXPLAINABLE_ANOMALY, FALSE_POSITIVE, DATA_QUALITY}`, `severity` ∈ `{HIGH, MEDIUM, LOW}`, `status` ∈ `{OPEN, INVESTIGATING, RESOLVED, DISMISSED}` (los tres repetibles, OR dentro del mismo parámetro, AND entre parámetros distintos) e `include_superseded` ∈ `{true, false}` con default `false`. `analysis_id` en la respuesta es el del último análisis `COMPLETED`, o `null` si no existe. Orden fijo `priority_rank` ascendente; con `include_superseded=true` se ordena por `detected_at` descendente y luego `priority_rank` ascendente. MIENTRAS no exista análisis `COMPLETED`, responde 200 con `items=[]`, `total=0`, `analysis_id=null`.
**Criterios de aceptación:**
- DADO el análisis completado, CUANDO se consulta `GET /anomalies`, ENTONCES `items` tiene 4 elementos con `meter_id` en orden `M-109, M-112, M-104, M-106` y `priority_rank` `1, 2, 3, 4`.
- DADO el análisis completado, CUANDO se consulta `?severity=HIGH`, ENTONCES `items` contiene `M-109` y `M-112` en ese orden.
- DADO dos análisis completados, CUANDO se consulta sin `include_superseded`, ENTONCES `total=4`; CUANDO se consulta con `include_superseded=true`, ENTONCES `total=8`.
- DADO ningún análisis, CUANDO se consulta, ENTONCES `{"analysis_id":null,"items":[],"total":0}`.

### RF-B-10 — Detalle de anomalía (`GET /anomalies/:id`, maestro 8.2.6)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE validar que `:id` sea un UUID; SI no lo es o no existe, ENTONCES 404 `ANOMALY_NOT_FOUND`. Las anomalías supersedidas DEBEN poder consultarse (histórico) y la respuesta incluye el campo `superseded` (booleano). `evidence` se devuelve ordenada por `position` ascendente. `related_event` es el objeto de evento completo (`id, timestamp, type, description`) o `null`. `comparison` sigue la definición del maestro 8.2.6.
**Criterios de aceptación:**
- DADO la anomalía de `M-109`, CUANDO se consulta por su `id`, ENTONCES `evidence` tiene ≥ 3 elementos con `position` consecutivos desde 1, `comparison.consumption_kwh.observed=2207.6`, `related_event=null`, `superseded=false`.
- DADO la anomalía de `M-104`, CUANDO se consulta, ENTONCES `related_event.type="OPERATIONAL_CHANGE"`.
- DADO `GET /anomalies/no-es-uuid`, CUANDO se invoca, ENTONCES 404 `ANOMALY_NOT_FOUND`.

### RF-B-11 — Disparo del análisis (`POST /ai/analyze`, maestro 8.2.7)
**Prioridad:** DEBE
**Descripción:** CUANDO llega `POST /ai/analyze`, el sistema DEBE, dentro de una transacción: (1) intentar tomar el mutex de proceso `analysisMu` con `TryLock`; SI está tomado, ENTONCES leer (sin bloqueo) el `id` del análisis `QUEUED`/`RUNNING` y responder 409 `ANALYSIS_IN_PROGRESS` con ese `id` en `details`; (2) ejecutar `SELECT id FROM analyses WHERE status IN ('QUEUED','RUNNING') FOR UPDATE`; SI devuelve una fila, ENTONCES hacer rollback, liberar el mutex y responder 409 `ANALYSIS_IN_PROGRESS` con `details={"analysis_id":"<id>"}`; (3) insertar una fila en `analyses` con `id` UUID v4, `status='QUEUED'`, `steps` = las 7 etapas en estado `PENDING` con las claves y labels exactos del maestro 8.2.7, `created_at=now()`; (4) commit y liberar el mutex; (5) lanzar la goroutine del job (RF-B-12) con `context.Background()` y `context.WithTimeout` de 120 s, nunca con el contexto de la petición; (6) responder 202 según maestro 8.2.7 en ≤ 100 ms. El body de la petición se ignora (vacío o cualquier JSON).
**Criterios de aceptación:**
- DADO ningún análisis en curso, CUANDO se invoca, ENTONCES 202 con `status="QUEUED"` y en `analyses` existe la fila con 7 `steps` en `PENDING`.
- DADO un análisis `RUNNING`, CUANDO se invoca, ENTONCES 409 con `details.analysis_id` igual al id en curso.
- DADO ningún análisis en curso, CUANDO se envían 10 peticiones concurrentes, ENTONCES exactamente una responde 202 y nueve responden 409.
- DADO una petición cuyo cliente cierra la conexión inmediatamente después del 202, CUANDO pasan 15 s, ENTONCES el análisis está `COMPLETED` (el job no se canceló con la petición).

### RF-B-12 — Ejecución del job de análisis
**Prioridad:** DEBE
**Descripción:** El job DEBE: (1) actualizar `analyses.status='RUNNING'` y `started_at=now()`; (2) por cada etapa en orden (`READINGS, BASELINE, DETECTION, CORRELATION, EVENTS, EXPLANATION, RECOMMENDATION`), escribir en BD `steps[i].status='RUNNING'` y `started_at` antes de ejecutarla, y `status='COMPLETED'`, `finished_at` y `detail` al terminar (una escritura por transición, no en memoria hasta el final); las etapas ejecutan las funciones del paquete `internal/engine` e `internal/llm` según el spec del motor; (3) al completar la última etapa, en **una sola transacción**: `UPDATE anomalies SET superseded=true WHERE superseded=false`; insertar las nuevas filas en `anomalies` y `anomaly_evidence`; `UPDATE meters SET status=<derivado>` para los 12 medidores según la tabla 8.3 del maestro; `UPDATE analyses SET status='COMPLETED', summary=<json>, finished_at=now()`; (4) SI cualquier etapa devuelve error o entra en pánico (el job DEBE tener su propio `recover`), ENTONCES escribir `steps[i].status='FAILED'`, `analyses.status='FAILED'`, `error_message=<mensaje>`, `finished_at=now()`, y NO ejecutar ninguna escritura sobre `anomalies` ni `meters` (las anomalías previas siguen vigentes); (5) SI el contexto de 120 s expira, ENTONCES tratarlo como error con `error_message="analysis timeout after 120s"`. El fallo del LLM NO es un error de etapa: el spec del motor define el fallback y la etapa `EXPLANATION` termina `COMPLETED` con `detail` indicando `source=template`.
**Criterios de aceptación:**
- DADO el dataset entregado y `DEEPSEEK_API_KEY` vacía, CUANDO se ejecuta el job, ENTONCES en ≤ 15 s `status="COMPLETED"`, `summary={"anomalies_detected":4,"high_priority":2,"meters_analyzed":12,"explanation_source":"template"}`, los 7 `steps` están `COMPLETED` con `started_at ≤ finished_at`, y `meters.status` es `CRITICAL` para `M-109`, `ALERT` para `M-104` y `M-112`, `OK` para los 9 restantes.
- DADO un motor de prueba (fake) que devuelve error en la etapa `DETECTION`, CUANDO se ejecuta el job, ENTONCES `status="FAILED"`, `steps[2].status="FAILED"`, `steps[3..6].status="PENDING"`, `error_message` no vacío, y `SELECT count(*) FROM anomalies WHERE superseded=false` no cambia respecto a antes del job.
- DADO un análisis previo `COMPLETED` con 4 anomalías, CUANDO se ejecuta un segundo job exitoso, ENTONCES las 4 anteriores quedan `superseded=true`, hay 4 nuevas con `superseded=false`, y `GET /anomalies` devuelve el `analysis_id` del segundo.
- DADO un motor de prueba que hace `panic`, CUANDO se ejecuta el job, ENTONCES el proceso sigue vivo, `GET /health` responde 200 y el análisis queda `FAILED`.

### RF-B-13 — Consulta de análisis (`GET /ai/analysis/:id`, maestro 8.2.7)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE responder con el estado actual leído de BD en cada petición (sin caché en memoria) y con header `Cache-Control: no-store`. SI `:id` no es UUID o no existe, ENTONCES 404 `ANALYSIS_NOT_FOUND`.
**Criterios de aceptación:**
- DADO un análisis en curso, CUANDO se consulta cada 1 s, ENTONCES en alguna respuesta al menos un `step` tiene `status="RUNNING"` y la respuesta final tiene `status="COMPLETED"` con `summary` no nulo.
- DADO cualquier consulta, CUANDO se responde, ENTONCES el header `Cache-Control` es exactamente `no-store`.
- DADO `GET /ai/analysis/00000000-0000-0000-0000-000000000000`, CUANDO se invoca, ENTONCES 404 `ANALYSIS_NOT_FOUND`.

### RF-B-14 — Resumen de dashboard (`GET /dashboard/summary`, maestro 8.2.8)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE calcular los campos con las reglas de la tabla 8.4 del maestro. `daily_consumption` DEBE contener los 14 días ordenados por fecha ascendente con la suma de los 12 medidores redondeada a 1 decimal. `top_anomalies` = primeras 3 por `priority_rank` (menos si hay menos). `last_analysis` = el más reciente por `created_at` sin importar su estado, o `null`. `meters_by_status` DEBE incluir siempre las tres claves aunque valgan 0.
**Criterios de aceptación:**
- DADO el análisis completado, CUANDO se consulta, ENTONCES `meters_total=12`, `total_consumption_kwh=155250.8`, `anomalies_total=4`, `high_priority_total=2`, `ai_confidence_avg=0.93`, `meters_by_status={"OK":9,"ALERT":2,"CRITICAL":1}`, `daily_consumption` tiene 14 elementos con `daily_consumption[0].consumption_kwh=10776.0` y `daily_consumption[13].consumption_kwh=12502.4`, `top_anomalies` tiene 3 elementos empezando por `M-109`.
- DADO ningún análisis, CUANDO se consulta, ENTONCES `anomalies_total=0`, `high_priority_total=0`, `ai_confidence_avg=null`, `last_analysis=null`, `top_anomalies=[]`, `meters_by_status={"OK":12,"ALERT":0,"CRITICAL":0}`.
- DADO un análisis `FAILED` posterior a uno `COMPLETED`, CUANDO se consulta, ENTONCES `last_analysis.status="FAILED"` y `anomalies_total=4` (las del `COMPLETED` siguen vigentes).

### RF-B-15 — Cambio de estado de anomalía (`PATCH /anomalies/:id/status`, maestro 8.2.9)
**Prioridad:** DEBERÍA
**Descripción:** CUANDO llega `PATCH /anomalies/:id/status`, el sistema DEBE validar `Content-Type: application/json`, body con la única clave `status` y valor ∈ `{OPEN, INVESTIGATING, RESOLVED, DISMISSED}`; SI falla, ENTONCES 400 `INVALID_BODY` con `details={"field":"status","allowed":[...]}`. Toda transición está permitida, incluida volver a `OPEN` y repetir el mismo estado (idempotente) ([DECISIÓN ABIERTA-B01]). Las anomalías supersedidas también pueden editarse (decisión: el histórico sigue siendo gestionable; ver sección 9). Responde 200 con el objeto de `GET /anomalies/:id`.
**Criterios de aceptación:**
- DADO la anomalía de `M-109` en `OPEN`, CUANDO se envía `{"status":"INVESTIGATING"}`, ENTONCES 200 con `status="INVESTIGATING"` y `GET /anomalies/:id` devuelve `status="INVESTIGATING"`.
- DADO una anomalía en `RESOLVED`, CUANDO se envía `{"status":"OPEN"}`, ENTONCES 200 con `status="OPEN"`.
- DADO cualquier anomalía, CUANDO se envía `{"status":"CLOSED"}`, ENTONCES 400 con `details.allowed=["OPEN","INVESTIGATING","RESOLVED","DISMISSED"]`.
- DADO cualquier anomalía, CUANDO se envía `{"status":"OPEN","extra":1}`, ENTONCES 400 `INVALID_BODY` con `details.field="extra"` (claves desconocidas se rechazan con `DisallowUnknownFields`).
- DADO una anomalía con `superseded=true`, CUANDO se envía `{"status":"DISMISSED"}`, ENTONCES 200.

### RF-B-16 — Salud (`GET /health`, maestro 8.2.10)
**Prioridad:** DEBE
**Descripción:** El sistema DEBE ejecutar `SELECT 1` con timeout 2 s; SI falla, ENTONCES responde 503 con `{"status":"degraded","db":"error","ingest":<último IngestStats conocido o null>}`. Si responde, devuelve 200 con el objeto `ingest` completo del maestro 8.2.10 (`status`, `readings`, `events`, `meters`, `rejected`, `duplicates`, `duration_ms`, `started_at`, `finished_at`), tomado del `IngestStats` que expone el paquete de ingesta (`spec-energy-data-ingesta-v1.md` RF-D-07). MIENTRAS `ingest.status ∈ {PENDING, RUNNING}`, los endpoints de datos (todos salvo `/health` y `/auth/login`) DEBEN responder 503 `INGEST_IN_PROGRESS`. Sin auth.
**Criterios de aceptación:**
- DADO el backend con datos cargados, CUANDO se consulta, ENTONCES 200 con `db="ok"`, `ingest.status="COMPLETED"`, `ingest.readings=4032`, `ingest.events=4`, `ingest.meters=12`, `ingest.rejected=0`, `ingest.duplicates=0`.
- DADO la ingesta en curso, CUANDO se pide `GET /meters`, ENTONCES 503 `INGEST_IN_PROGRESS` y `GET /health` responde 200 con `ingest.status="RUNNING"`.
- DADO PostgreSQL detenido, CUANDO se consulta, ENTONCES 503 con `db="error"` en ≤ 2,5 s.

### RF-B-17 — Formato y mapa de errores
**Prioridad:** DEBE
**Descripción:** Toda respuesta de error DEBE usar el formato del maestro 8.2 (`error.code`, `error.message`, `error.details`). El mapa código → HTTP es exacto:

| `code` | HTTP | `details` |
|---|---|---|
| `INVALID_CREDENTIALS` | 401 | `null` |
| `UNAUTHORIZED` | 401 | `null` |
| `TOO_MANY_ATTEMPTS` | 429 | `{"retry_after_seconds": 60}` |
| `INVALID_QUERY` | 400 | `{"param": "<nombre>", "allowed": [...]}` o `{"param": "<nombre>", "reason": "<texto>"}` |
| `INVALID_BODY` | 400 | `{"field": "<nombre>", "allowed": [...]}` o `{"field": "<nombre>", "reason": "<texto>"}` |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | `{"expected": "application/json"}` |
| `PAYLOAD_TOO_LARGE` | 413 | `{"max_bytes": 1048576}` |
| `METER_NOT_FOUND` | 404 | `null` |
| `ANOMALY_NOT_FOUND` | 404 | `null` |
| `ANALYSIS_NOT_FOUND` | 404 | `null` |
| `ANALYSIS_IN_PROGRESS` | 409 | `{"analysis_id": "<uuid>"}` |
| `INGEST_IN_PROGRESS` | 503 | `{"ingest_status": "RUNNING"}` |
| `REQUEST_TIMEOUT` | 503 | `null` |
| `INTERNAL_ERROR` | 500 | `null` |

Un fallo de base de datos durante una petición DEBE responder 500 `INTERNAL_ERROR` y registrar `DB_ERROR` con el error original (decisión: no se distingue 503 porque el cliente no puede hacer nada diferente). Rutas no definidas responden 404 con `code="NOT_FOUND"`; método no permitido responde 405 con `code="METHOD_NOT_ALLOWED"`.
**Criterios de aceptación:**
- DADO cada fila de la tabla, CUANDO se provoca el error, ENTONCES el HTTP y la forma de `details` coinciden con la fila.
- DADO `GET /no-existe`, CUANDO se invoca con token válido, ENTONCES 404 `{"error":{"code":"NOT_FOUND",...}}`.
- DADO `DELETE /meters`, CUANDO se invoca con token válido, ENTONCES 405 `METHOD_NOT_ALLOWED`.

### RF-B-18 — Logging estructurado
**Prioridad:** DEBE
**Descripción:** El sistema DEBE emitir logs JSON (una línea por evento) con `log/slog` a stdout. Toda línea DEBE tener `time` (RFC 3339), `level` ∈ `{debug, info, warn, error}`, `msg`. La línea de acceso HTTP DEBE tener además `request_id`, `method`, `path`, `status` (HTTP), `duration_ms`, `user` (email del `sub` o `""`), `remote_ip`. Las líneas del job DEBEN tener `analysis_id` y `step`. Cada llamada al LLM DEBE registrar `analysis_id`, `meter_id`, `model`, `duration_ms`, `outcome` ∈ `{ok, timeout, error, invalid_json, fallback}` (RNF-10 del maestro). `LOG_LEVEL` controla el nivel mínimo. Los logs NUNCA incluyen `password`, `JWT_SECRET`, `DEEPSEEK_API_KEY` ni el token completo. El sistema PUEDE exponer `GET /metrics` en formato Prometheus; no es requisito.
**Criterios de aceptación:**
- DADO `GET /meters` con token válido, CUANDO se responde, ENTONCES stdout contiene una línea JSON con `"method":"GET"`, `"path":"/meters"`, `"status":200`, `"user":"analista@energy.local"` y `duration_ms` numérico.
- DADO `POST /auth/login`, CUANDO se registra, ENTONCES ninguna línea de log contiene la cadena `Demo1234!`.
- DADO un análisis con `DEEPSEEK_API_KEY` vacía, CUANDO termina, ENTONCES hay 4 líneas con `"outcome":"fallback"`.

## 7. Casos borde y manejo de errores

Aplican además CB-M01 a CB-M12 del maestro.

| # | Situación | Comportamiento esperado |
|---|---|---|
| CB-B-01 | Body JSON malformado en `POST /auth/login` o `PATCH /anomalies/:id/status` | 400 `INVALID_BODY` con `details={"field":"body","reason":"invalid JSON"}`. |
| CB-B-02 | `Content-Type` distinto de `application/json` (con o sin `; charset=utf-8`) en endpoints con body | 415 `UNSUPPORTED_MEDIA_TYPE`. `POST /ai/analyze` es la excepción: acepta cualquier `Content-Type` o ninguno porque ignora el body. |
| CB-B-03 | Token con firma inválida | 401 `UNAUTHORIZED`, log `reason="bad_signature"` (RF-B-05). |
| CB-B-04 | Token expirado | 401 `UNAUTHORIZED`, log `reason="expired"`. El frontend borra la sesión y redirige (spec del frontend). |
| CB-B-05 | `meterId` en minúsculas (`/meters/m-109`) o con espacios | 404 `METER_NOT_FOUND`. No se normaliza: los identificadores son exactos. |
| CB-B-06 | Parámetro repetido `status=OK&status=ALERT` | Se combinan con OR. Parámetros distintos (`type` y `severity`) se combinan con AND. |
| CB-B-07 | `from` sin `to` (o viceversa) en readings | El faltante toma el límite del periodo (inicio `2026-09-01T00:00:00Z`, fin `2026-09-14T23:59:59Z`). |
| CB-B-08 | Rango de readings válido pero sin lecturas | 200 con `items=[]`; con `granularity=hour` se sigue devolviendo `baseline_profile` completo. |
| CB-B-09 | Análisis huérfano (`QUEUED`/`RUNNING`) al arrancar | Se marca `FAILED`, `error_message="process restarted"`, `finished_at=now()` antes de aceptar tráfico (RF-B-02). |
| CB-B-10 | Dos `POST /ai/analyze` en el mismo milisegundo | Uno recibe 202 y el otro 409; garantizado por mutex de proceso + `SELECT … FOR UPDATE` (RF-B-11). |
| CB-B-11 | PostgreSQL cae durante una petición | 500 `INTERNAL_ERROR`, log `DB_ERROR` con el error de pgx. El pool reconecta automáticamente en la siguiente petición. |
| CB-B-12 | PostgreSQL cae durante el job de análisis | El job falla al escribir la siguiente transición y queda en memoria como fallido; al recuperarse la BD, si la fila sigue `RUNNING`, se marcará `FAILED` en el próximo arranque (CB-B-09). El job DEBE reintentar la escritura de `FAILED` 3 veces con 1 s de espera antes de rendirse. |
| CB-B-13 | Body > 1 MB | 413 `PAYLOAD_TOO_LARGE` sin leer el resto del body. |
| CB-B-14 | Handler tarda > 30 s | 503 `REQUEST_TIMEOUT`; el contexto cancelado aborta la query en pgx. No aplica al job (usa su propio contexto). |
| CB-B-15 | Migración falla al arrancar (SQL inválido o BD en versión incompatible) | Exit code 1, log `MIGRATION_FAILED` con el error. |
| CB-B-16 | Ingesta parcial (CSV con filas inválidas) | El backend arranca; `GET /health.ingest` refleja los conteos reales; el detalle de rechazos lo define el spec de data. |
| CB-B-17 | `DEMO_USER_PASSWORD` vacía | Exit code 1, log `CONFIG_INVALID` con `field="DEMO_USER_PASSWORD"`. |
| CB-B-18 | `CORS_ALLOWED_ORIGINS` vacío | Se usa el default `http://localhost:5173`; se registra `warn` `CORS_DEFAULT_ORIGIN`. |
| CB-B-19 | `X-Request-Id` entrante > 128 caracteres o con caracteres fuera de `[A-Za-z0-9-]` | Se ignora y se genera uno nuevo. |
| CB-B-20 | `id` con formato UUID válido pero de otra tabla (un `analysis_id` enviado a `/anomalies/:id`) | 404 `ANOMALY_NOT_FOUND`. |
| CB-B-21 | `Authorization` con token válido en `GET /health` o `POST /auth/login` | Se ignora; ambos endpoints no evalúan el token. |
| CB-B-22 | Reloj del servidor adelantado/atrasado respecto al cliente | `exp` se evalúa con el reloj del servidor; no hay tolerancia (`leeway=0`). Documentado para que el frontend confíe en `expires_at` de la respuesta de login y no en su reloj local. |

## 8. Contratos de datos e interfaces

### 8.1 Contratos HTTP
Normativos en el maestro, sección 8.2 (8.2.1 a 8.2.10) y formato de error en 8.2. Este spec añade los códigos `TOO_MANY_ATTEMPTS`, `UNSUPPORTED_MEDIA_TYPE`, `PAYLOAD_TOO_LARGE`, `REQUEST_TIMEOUT`, `NOT_FOUND` y `METHOD_NOT_ALLOWED` (tabla en RF-B-17).

### 8.2 Modelo de datos
Normativo en el maestro, sección 8.1. `migrations/0001_init.up.sql` DEBE contener ese DDL literal. `0001_init.down.sql`:

```sql
DROP TABLE IF EXISTS anomaly_evidence;
DROP TABLE IF EXISTS anomalies;
DROP TABLE IF EXISTS analyses;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS readings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS meters;
```

### 8.3 Interfaces internas (Go)

Firmas que los paquetes DEBEN exponer para que los specs hijos encajen. Los tipos de dominio viven en `internal/domain`.

```go
// internal/ingest — definido en detalle en spec-energy-data-ingesta-v1.md
type Stats struct { Meters, Readings, Events, Duplicates, Rejected int }
func Run(ctx context.Context, st *store.Store, dataDir string) (Stats, error)

// internal/engine — definido en detalle en spec-energy-motor-anomalias-ia-v1.md
type StepFn func(ctx context.Context, report func(detail string)) error
type Pipeline interface {
    // Steps devuelve las 7 etapas en orden; el job las ejecuta una a una.
    Steps() []Step          // Step{Key, Label string; Run StepFn}
    // Result devuelve las anomalías finales (con evidencia y texto) tras la etapa 7.
    Result() []domain.AnomalyResult
}
func NewPipeline(st *store.Store, explainer llm.Explainer, cfg Config) Pipeline

// internal/llm — definido en detalle en spec-energy-motor-anomalias-ia-v1.md
type Explainer interface {
    Explain(ctx context.Context, in ExplainInput) (ExplainOutput, Source, error) // Source ∈ {"llm","template"}
}

// internal/store
type Store struct{ pool *pgxpool.Pool }
func (s *Store) CreateAnalysis(ctx, steps []domain.Step) (uuid.UUID, error)      // dentro de tx con FOR UPDATE
func (s *Store) UpdateAnalysisStep(ctx, id uuid.UUID, idx int, step domain.Step) error
func (s *Store) CompleteAnalysis(ctx, id uuid.UUID, results []domain.AnomalyResult, meterStatus map[string]string, summary domain.Summary) error // una sola tx
func (s *Store) FailAnalysis(ctx, id uuid.UUID, stepIdx int, msg string) error
func (s *Store) FailOrphanAnalyses(ctx) (int, error)
```

Ejemplo de `domain.Step` serializado en `analyses.steps` (idéntico al maestro 8.2.7):
```json
{ "key": "DETECTION", "label": "Detección", "status": "COMPLETED", "started_at": "2026-09-25T15:02:03Z", "finished_at": "2026-09-25T15:02:04Z", "detail": "4 candidatos en 12 medidores" }
```

### 8.4 Claims del JWT

```json
{ "sub": "analista@energy.local", "name": "Analista Demo", "iat": 1790348400, "exp": 1790377200 }
```

### 8.5 Línea de log de acceso

```json
{ "time": "2026-09-25T15:02:11.482Z", "level": "info", "msg": "http", "request_id": "3f0c9c2e-7b1a-4b9e-9c0d-1e2f3a4b5c6d", "method": "GET", "path": "/meters", "status": 200, "duration_ms": 12, "user": "analista@energy.local", "remote_ip": "172.18.0.1" }
```

### 8.6 `.env.example`

```
DATABASE_URL=postgres://energy:energy@db:5432/energy?sslmode=disable
PORT=8080
DATA_DIR=/app/data
JWT_SECRET=change-me-in-prod
DEMO_USER_EMAIL=analista@energy.local
DEMO_USER_PASSWORD=Demo1234!
DEEPSEEK_API_KEY=
DEEPSEEK_MODEL=deepseek-flash
DEEPSEEK_BASE_URL=https://api.deepseek.com
LLM_TIMEOUT_SECONDS=20
CORS_ALLOWED_ORIGINS=http://localhost:5173
LOG_LEVEL=info
```

## 9. Restricciones y decisiones tomadas

Aplican todas las decisiones del maestro, sección 9 (Go 1.22+, chi, pgx, golang-migrate, PostgreSQL 16, JWT HS256 8 h, bcrypt, análisis en goroutine, puertos, variables de entorno). Decisiones propias de este spec:

| Decisión/Restricción | Justificación |
|---|---|
| Librerías con versión mínima: `github.com/go-chi/chi/v5 ≥ 5.1`, `github.com/jackc/pgx/v5 ≥ 5.6`, `github.com/golang-migrate/migrate/v4 ≥ 4.17` (driver `postgres` + source `iofs`), `github.com/golang-jwt/jwt/v5 ≥ 5.2`, `golang.org/x/crypto` (bcrypt), `github.com/google/uuid ≥ 1.6`, `github.com/sashabaranov/go-openai ≥ 1.30` (cliente de la API de DeepSeek, compatible con OpenAI), `log/slog` de la librería estándar | Todas mantenidas, sin CGO, compilan en `alpine`. `slog` evita una dependencia de logging. |
| Sin ORM | 7 tablas y ~15 consultas; SQL explícito en `internal/store` deja cada consulta visible en el código sin una dependencia adicional. |
| `Dockerfile` multi-stage con `CGO_ENABLED=0 go build -ldflags="-s -w"` y runtime `alpine:3.20` con `ca-certificates` | Binario estático ≤ 50 MB; `ca-certificates` es necesario para llamar a la API de DeepSeek por HTTPS. |
| `Makefile` con targets `up` (`docker compose up --build`), `down` (`docker compose down -v`), `test` (`cd backend && go test ./... -race -cover`), `lint` (`go vet ./... && golangci-lint run`), `seed` (reejecuta la ingesta contra la BD levantada vía `go run ./cmd/api --seed-only`) | Un comando por acción habitual; `seed` permite recargar datos sin reiniciar contenedores. |
| Análisis huérfanos → `FAILED` al arrancar | Sin esto, un reinicio durante un análisis bloquearía `POST /ai/analyze` para siempre con 409. |
| `SELECT … FOR UPDATE` además del mutex de proceso | El mutex cubre concurrencia dentro del proceso; el bloqueo de fila cubre un segundo proceso apuntando a la misma BD (por ejemplo, `make seed` mientras la API corre). |
| Una sola transacción para publicar resultados | Evita un estado intermedio donde las anomalías viejas ya están supersedidas y las nuevas aún no existen; el lector siempre ve un conjunto completo. |
| `PATCH` permite cualquier transición y también sobre supersedidas | Es un MVP con un solo usuario; restringir transiciones añadiría código sin valor para la demo. Las supersedidas son histórico consultable y su estado forma parte de ese histórico. |
| Fallo de BD en petición → 500, no 503 | El cliente no distingue acciones diferentes; simplifica el manejo en el frontend a un único "error del servidor". |
| `DisallowUnknownFields` en todos los decoders JSON | Un cliente que envía claves inesperadas suele tener un bug; rechazar la petición con 400 lo hace visible en la primera llamada. |
| Rate limit de login en memoria (`golang.org/x/time/rate` por IP), DEBERÍA | Suficiente para un solo proceso; no se persiste porque no hay requisito de multi-instancia. |
| Timeouts del `http.Server`: `ReadHeaderTimeout=5s`, `ReadTimeout=10s`, `WriteTimeout=35s`, `IdleTimeout=60s` | `WriteTimeout` > timeout de middleware (30 s) para que el middleware responda antes de que el servidor corte. |

## 10. Requisitos no funcionales

Aplican RNF-01 a RNF-12 del maestro. Específicos del backend:

| # | Requisito | Métrica |
|---|---|---|
| RNF-B-01 | Latencia GET | P95 ≤ 300 ms por endpoint en 20 peticiones consecutivas con 4.032 lecturas (mismo umbral del maestro RNF-03, medido por endpoint). |
| RNF-B-02 | `POST /ai/analyze` | Responde 202 o 409 en ≤ 100 ms; el trabajo ocurre en el job. |
| RNF-B-03 | Pool de conexiones | `pgxpool` con `MaxConns=10`, `MinConns=2`, `MaxConnLifetime=30m`. |
| RNF-B-04 | Tamaño de imagen | Imagen Docker del backend ≤ 50 MB; binario ≤ 30 MB. |
| RNF-B-05 | Tests | `go test ./... -race -cover` pasa sin carreras detectadas; cobertura ≥ 50 % en `internal/http` y ≥ 60 % en `internal/store`. |
| RNF-B-06 | Arranque | Con BD disponible y datos ya cargados, `GET /health` responde 200 en ≤ 5 s desde el inicio del proceso. |
| RNF-B-07 | Memoria | RSS del proceso ≤ 256 MB durante un análisis completo con el dataset entregado. |
| RNF-B-08 | Seguridad | Ningún secreto en logs ni en respuestas; `JWT_SECRET` obligatorio; bcrypt cost 10; CORS por lista exacta. |
| RNF-B-09 | Determinismo | Dos jobs consecutivos sin cambios en datos producen filas de `anomalies` idénticas en `type, severity, confidence, priority_rank, window_start, window_end` (RNF-06 del maestro). |

## 11. Supuestos, pendientes y decisiones abiertas

| Marcador | Detalle | Responsable | Fecha límite |
|---|---|---|---|
| [SUPUESTO-B01] | Nombre del módulo Go `github.com/jsotelo/energy-platform/backend`. Si el repositorio se publica bajo otro usuario u organización, se cambia en `go.mod` y en los imports; nada más depende del nombre. | Jonnathan Sotelo | Antes del primer commit |
| [SUPUESTO-B02] | Un solo proceso de API contra la BD. El mutex de proceso más `FOR UPDATE` cubren también un segundo proceso accidental, pero no se prueba un despliegue multi-instancia. | Jonnathan Sotelo | — |
| [SUPUESTO-B03] | `X-Forwarded-For` es confiable para el rate limit de login porque el único proxy previsto es el dev server de Vite o ninguno. | Jonnathan Sotelo | — |
| [DECISIÓN ABIERTA-B01] | Transiciones de `anomalies.status`: hoy cualquiera → cualquiera. Alternativa: máquina de estados (`OPEN → INVESTIGATING → RESOLVED|DISMISSED`, sin retroceso). Criterio: si la demo necesita mostrar "reabrir", se mantiene libre; si el evaluador pide trazabilidad, se restringe y se añade `status_changed_at`. | Jonnathan Sotelo | Antes de la demo |
| Resuelto (antes PENDIENTE-B01) | `make lint` ejecuta `golangci-lint` solo si está instalado; si no, degrada a `go vet` con aviso. | — | — |
| Resuelto (antes PENDIENTE-B02) | `go-openai` fijado en `v1.36.1`. Modelo por defecto `deepseek-flash` (DeepSeek-V4.1-Flash); `deepseek-chat` ya no existe en la API. Verificado con llamada real el 2026-09-25: `GET /models` lista `deepseek-flash` y `deepseek-v4-pro`, y `response_format: json_object` devuelve JSON válido. | — | — |

## 11b. Notas de implementación (2026-09-25)

Decisiones tomadas al implementar, con su porqué. No cambian contratos HTTP.

| Tema | Decisión | Porqué |
|---|---|---|
| DDL | `anomaly_evidence."window"` va entre comillas y `events` lleva `UNIQUE (meter_id, timestamp, type)`. | `WINDOW` es palabra reservada en PostgreSQL (el DDL literal falla); el UPSERT de eventos del spec de data necesita esa restricción. |
| Cálculos derivados | Implementación única y pura en `internal/domain/calc.go`; `store/aggregates.go` expone las funciones con nombre de §8.4 del spec de data delegando en ella (salvo `PeriodTotal` y `GlobalDailyTotals`, en SQL). El motor usa la misma función sobre datos en memoria. | Cumple a la vez "una sola implementación" (spec de data RF-D-08) y "el motor no accede a la BD" (spec del motor §5). |
| Interfaz del motor | `engine.Pipeline.Run(ctx, Input, StepCallback) (Result, error)` con callback por etapa, en lugar de `Steps()/Result()` de §8.3. | Es la firma del spec del motor §8.0; el job persiste cada transición desde el callback. El callback devuelve `error` para abortar si falla la escritura en BD. |
| Explainer | Interfaz definida en `internal/engine` e implementada en `internal/llm`; nunca devuelve error (el fallback es interno). | `internal/engine` no importa red (RNF-E-04). |
| `POST /ai/analyze` | Además de mutex y `FOR UPDATE`, `LOCK TABLE analyses IN SHARE ROW EXCLUSIVE MODE` dentro de la transacción. | `FOR UPDATE` no bloquea nada cuando aún no existe fila activa; el bloqueo de tabla serializa dos procesos que crean a la vez. |
| Tests de integración | `TEST_DATABASE_URL` (opción PUEDE de §12a) en lugar de `testcontainers-go`; `make test-db` levanta el contenedor. | Sin dependencia adicional; mismo PostgreSQL 16. |
| Fixtures del motor | Generados en memoria en `internal/engine/fixtures_test.go` (determinísticos) en lugar de CSV bajo `testdata/`. | Mismos escenarios del catálogo del spec de pruebas RF-T-03 sin mantener archivos. |
| Criterio `?q=10` | Corregido a 10 medidores (RF-B-06). | `M-110` contiene "10". |
| Consumo total | `total_consumption_kwh = 155250.9` sobre el dataset (suma exacta 155.250,85 redondeada). Los specs citan 155.250,8 con tolerancia ±0,5, que se cumple. | Redondeo half-away-from-zero. |
| `-race` | Requiere cgo; en Windows sin compilador C se ejecuta dentro de `golang:1.22` (ver README). | RNF-B-05. |

## 12. Verificación y definición de terminado

### 12a. Verificación técnica

Comandos exactos que DEBEN pasar antes de dar por terminado el backend:

```
cd backend
go vet ./...
golangci-lint run
go test ./... -race -cover
```

Estrategia: tests unitarios con `net/http/httptest` sobre el router completo (middleware incluido) usando un `store` real contra PostgreSQL de prueba. La BD de prueba DEBERÍA levantarse con `testcontainers-go` (imagen `postgres:16-alpine`); PUEDE usarse en su lugar un servicio `db-test` en `docker-compose.test.yml` con `TEST_DATABASE_URL`. El motor y el LLM se sustituyen por fakes en los tests de `internal/http` (interfaces `Pipeline` y `Explainer` de 8.3); los tests de integración de RF-B-12 usan el motor real con los CSV de `data/`.

| Requisito | Test | Archivo |
|---|---|---|
| RF-B-01 | `go build ./...` + test que verifica `embed.FS` contiene `0001_init.up.sql` | `internal/store/migrate_test.go` |
| RF-B-02 | Test de arranque con BD caída (reintentos), con análisis huérfano, con `JWT_SECRET` vacío, y de graceful shutdown con una petición de 5 s en curso (debe completarse antes de los 10 s de gracia) | `cmd/api/main_test.go` |
| RF-B-03 | Tests de request-id, recover (handler que hace panic), CORS con origen permitido y no permitido, 413, 503 por timeout, 401 sin token | `internal/http/middleware_test.go` |
| RF-B-04 | Login con credenciales válidas, contraseña errónea, email inexistente (bodies idénticos), claims del token, rate limit | `internal/http/auth_test.go` |
| RF-B-05 | Token con otro secreto, expirado, `alg=none`, esquema `Basic` | `internal/http/auth_test.go` |
| RF-B-06 | Orden por defecto sobre el dataset (12 ids exactos), filtros `status`, `q`, `sort`/`order`, `sort` inválido | `internal/http/meters_test.go` |
| RF-B-07 | Detalle `M-109` (valores exactos), `m-109` → 404, `M-999` → 404 | `internal/http/meters_test.go` |
| RF-B-08 | 336 lecturas, rango de un día con outlier, `granularity=day`, rango vacío, `from > to`, formato inválido | `internal/http/readings_test.go` |
| RF-B-09 | Orden `M-109, M-112, M-104, M-106`; filtros; `include_superseded` con dos análisis; sin análisis | `internal/http/anomalies_test.go` |
| RF-B-10 | Detalle `M-109` (evidencia, comparison, `related_event=null`), `M-104` (`related_event.type`), id inválido | `internal/http/anomalies_test.go` |
| RF-B-11 | 202 con steps `PENDING`, 409 con `RUNNING`, 10 peticiones concurrentes (1×202, 9×409), cliente que cierra conexión | `internal/http/analysis_test.go` |
| RF-B-12 | Job completo sin LLM (≤ 15 s, summary, steps, `meters.status`), fake que falla en `DETECTION`, segundo job supersede, fake con panic | `internal/http/analysis_job_test.go` |
| RF-B-13 | Polling hasta `COMPLETED`, header `Cache-Control`, id inexistente | `internal/http/analysis_test.go` |
| RF-B-14 | Summary tras análisis (valores exactos), sin análisis, tras `FAILED` posterior | `internal/http/dashboard_test.go` |
| RF-B-15 | Transición válida, retroceso a `OPEN`, valor inválido, clave desconocida, supersedida | `internal/http/anomalies_test.go` |
| RF-B-16 | `/health` con datos, con BD caída (503 ≤ 2,5 s) | `internal/http/health_test.go` |
| RF-B-17 | Un test por fila de la tabla de errores; 404 ruta desconocida; 405 | `internal/http/errors_test.go` |
| RF-B-18 | Captura de stdout: campos de la línea de acceso; ausencia de la contraseña; 4 líneas `fallback` | `internal/http/logging_test.go` |
| CB-B-01, CB-B-02, CB-B-13 | JSON malformado, `Content-Type` distinto de `application/json`, body > 1 MB | `internal/http/errors_test.go` |
| CB-B-12 | Fake de store que falla en `UpdateAnalysisStep`; verificar 3 reintentos de `FailAnalysis` | `internal/http/analysis_job_test.go` |
| RNF-B-01, RNF-B-02 | `scripts/bench.ps1`: 20 peticiones por GET y 1 `POST /ai/analyze`, imprime P95 | `scripts/bench.ps1` |
| RNF-B-04 | `docker image ls` tras `make up` | Manual |
| RNF-B-09 | Dos jobs consecutivos; comparar `anomalies` sin columnas de texto | `internal/http/analysis_job_test.go` |

Datos de prueba: `data/readings.csv` y `data/events.csv` reales; fixtures de `internal/engine/testdata/` definidos en el spec del motor.

### 12b. Aceptación de usuario (UAT)

- **Quién firma:** Jonnathan Sotelo, autor de la prueba técnica y responsable de la entrega.
- **Qué criterios valida:** RF-B-04 (login con usuario demo), RF-B-06 (orden y filtros de medidores), RF-B-09 (orden `M-109, M-112, M-104, M-106`), RF-B-11 y RF-B-12 (disparo y finalización del análisis con `summary.anomalies_detected=4`), RF-B-14 (KPI del dashboard), RF-B-15 (cambio de estado de la anomalía de `M-109`), RF-B-16 (`/health` tras `docker compose up --build`). Se validan ejecutando el guion de demo del maestro 12b desde el frontend y contrastando con `curl` los valores de `GET /dashboard/summary` y `GET /anomalies`.
- **Nivel de UAT:** Profundo → UAT formal completo: el firmante valida cada criterio listado con resultado pasa/falla y firma el acta de aceptación antes de la entrega.
