# Frontend — Energy Management

SPA en React 18 + TypeScript + Vite + Tailwind + Recharts que implementa `docs/specs/spec-energy-frontend-v1.md` con el tema oscuro y la paleta de bia.app (acento menta `#08ddbc`).

## Requisitos

- Node 20+ y npm 10+.
- Backend en `http://localhost:8080` (`VITE_API_URL`), **o** el modo mock (no necesita backend).

## Comandos

| Comando | Qué hace |
|---|---|
| `npm install` | Instala dependencias |
| `npm run dev:mock` | Dev server en `http://localhost:5173` con **MSW en el navegador**: responde con el dataset real (`data/readings.csv`, `data/events.csv`) y simula el análisis de IA con progreso por etapas. Credenciales: `analista@energy.local` / `Demo1234!` |
| `npm run dev` | Dev server contra el backend real (`VITE_API_URL`) |
| `npm run mock:dataset` | Regenera `src/mocks/data/dataset.json` desde los CSV (baseline, perfil horario, outliers) |
| `npm run build` | `tsc --noEmit` + build de producción en `dist/` |
| `npm run lint` / `npm run typecheck` | ESLint sin warnings / TypeScript estricto |
| `npm test` / `npm run test:run` | Vitest (+ cobertura con `test:run`); las páginas se prueban con los mismos handlers MSW |
| `node scripts/screenshots.mjs [dir]` | Recorre el guion de demo con Playwright (Edge/Chrome instalado) y captura cada pantalla a 1440 px. Requiere `npm i --no-save playwright-core` |

## Docker

```bash
docker build -t energy-frontend --build-arg VITE_API_URL=http://localhost:8080 .
docker run -p 5173:5173 energy-frontend
```

La imagen sirve `dist/` con **Caddy 2** (`Caddyfile`): fallback SPA a `index.html`, compresión y caché inmutable de `/assets/*`.

## Estructura

```
src/
  app/          App, router (guardas de sesión), providers, layout (Sidebar, Header, AppLayout)
  pages/        Login, Dashboard, Meters, MeterDetail, Anomalies, AnomalyDetail, NotFound
  components/   Badge, KpiTile, NetworkStrip, AnalysisModal, Panel, Segmented, States, Toast, ConfidenceBar, charts/
  api/          client (Authorization, 401, ApiError) y un módulo por recurso
  hooks/        useAuth, useAnalysis (arranque + modal + 409 + reanudación), useAnalysisPolling, useDebounce
  lib/          format (es-CO, UTC), labels (enum → español), colors (tokens), storage
  mocks/        handlers MSW, dataset.json generado, anomalías simuladas, worker/server
  types/        api.ts (contratos del maestro §8.2)
```

## Variables de entorno

Ver `.env.example`: `VITE_API_URL` (default `http://localhost:8080`) y `VITE_USE_MOCKS` (`true` solo en desarrollo).
