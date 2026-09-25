# AI Energy Management Platform

MVP de gestión de 12 medidores eléctricos que detecta, clasifica, explica y prioriza anomalías sobre 4.032 lecturas horarias de 14 días. Backend en Go, frontend en React, PostgreSQL 16 y DeepSeek como LLM explicador (opcional).

## Arrancar todo

Requisito: Docker. Nada más.

```bash
docker compose up --build
```

| Servicio | URL |
|---|---|
| Aplicación | http://localhost:5173 |
| API | http://localhost:8080 (`GET /health` sin auth) |
| PostgreSQL | localhost:5432 (`energy` / `energy`) |

Credenciales de la demo: `analista@energy.local` / `Demo1234!`.

Para explicaciones generadas por LLM, exporta `DEEPSEEK_API_KEY` antes de `docker compose up`. Sin ella todo funciona igual y las explicaciones usan plantillas determinísticas.

Para empezar de cero (borra la base): `docker compose down -v`.

## Qué hace la IA

Un motor determinístico en `backend/internal/engine` compara cada día contra el baseline de los primeros 7 días y aplica cinco detectores: spike, cambio persistente, outlier horario, corroboración eléctrica y calidad de datos (incluido el ratio físico kWh/(V·I·PF)). Cruza con `events.csv` y clasifica con precedencia fija. El LLM solo redacta el texto a partir de la evidencia; nunca decide tipo, severidad ni confianza.

Resultado sobre el dataset entregado:

| Prioridad | Medidor | Tipo | Severidad | Confianza |
|---|---|---|---|---|
| 1 | M-109 | Anomalía real | Alta | 0,98 |
| 2 | M-112 | Calidad de datos | Alta | 0,98 |
| 3 | M-104 | Anomalía explicable | Media | 0,95 |
| 4 | M-106 | Falso positivo | Baja | 0,80 |

Los otros 8 medidores no generan anomalía.

## Estructura

```
backend/    API Go (chi, pgx, golang-migrate, JWT) + motor + ingesta + cliente DeepSeek
frontend/   React 18 + Vite + TypeScript + Tailwind + Recharts, servido con Caddy
data/       readings.csv y events.csv (montados en el contenedor del API)
docs/specs/ Especificaciones (maestro, data, motor, backend, frontend, pruebas)
scripts/    smoke.ps1 (smoke test del flujo principal)
```

## Desarrollo

Backend (Go 1.22+):

```bash
make test-db                      # PostgreSQL efímero en :55432
make test-integration             # go test ./... con TEST_DATABASE_URL
cd backend && go test ./...       # sin TEST_DATABASE_URL los tests de integración se saltan
```

`-race` requiere cgo. En Windows sin compilador C:

```bash
docker run --rm -v "$PWD/backend:/src:ro" -v "$PWD/data:/data:ro" -e CGO_ENABLED=1 \
  -e TEST_DATABASE_URL=postgres://energy:energy@host.docker.internal:55432/energy_test?sslmode=disable \
  golang:1.22 sh -c "cp -r /src /tmp/backend && cp -r /data /tmp/data && cd /tmp/backend && go test ./... -race"
```

Frontend: ver `frontend/README.md` (`npm run dev:mock` corre sin backend).

Smoke test con el stack levantado:

```bash
powershell -File scripts/smoke.ps1
```

## Verificación actual

| Chequeo | Resultado |
|---|---|
| `go vet ./...` y `go test ./... -race` (golang:1.22) | En verde |
| Cobertura backend | motor 95 %, http 81 %, llm 87 %, ingest 87 %, store+domain 82 % (tests de integración) |
| Frontend | 46 tests Vitest, cobertura 90 %, `tsc` y ESLint sin errores |
| `docker compose up --build` desde cero | ≈ 36 s; imagen del API 28,6 MB |
| `scripts/smoke.ps1` | 9/9 en 1,3 s |
| Guion de demo en navegador contra el backend real | 14 pantallas, 0 errores de consola |
