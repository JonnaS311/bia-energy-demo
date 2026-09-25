# Specs — AI Energy Management Platform

Conjunto de especificaciones para implementar el MVP de la prueba técnica "AI Energy Management Platform" (Go + React + PostgreSQL + DeepSeek como LLM explicador). Escritas con un formato de spec de 12 secciones: lenguaje normativo (DEBE / DEBERÍA / PUEDE), requisitos numerados con criterios DADO / CUANDO / ENTONCES, y cero conocimiento tácito. Están pensadas para que un agente de IA o una persona sin contexto previo las ejecute sin preguntar al autor.

## Orden de lectura (obligatorio)

| # | Archivo | Qué define | Prefijos |
|---|---|---|---|
| 1 | [`spec-energy-plataforma-maestro-v1.md`](spec-energy-plataforma-maestro-v1.md) | Problema, métricas de éxito (rúbrica), glosario, alcance, **modelo de datos**, **contratos de API**, decisiones de stack, RNF, supuestos y UAT. Es la fuente de verdad: ningún hijo la contradice. | `RF-M`, `CB-M`, `RNF-` |
| 2 | [`spec-energy-data-ingesta-v1.md`](spec-energy-data-ingesta-v1.md) | Carga idempotente de `readings.csv` y `events.csv`, validaciones, semillas, baseline diario, perfil horario, agregados. | `RF-D`, `CB-D`, `RNF-D` |
| 3 | [`spec-energy-motor-anomalias-ia-v1.md`](spec-energy-motor-anomalias-ia-v1.md) | Pipeline de 7 etapas, detectores con umbrales, clasificación, severidad, confianza, priorización, evidencia, explicación con LLM y fallback. | `RF-E`, `CB-E`, `RNF-E` |
| 4 | [`spec-energy-backend-api-v1.md`](spec-energy-backend-api-v1.md) | Servicio Go: estructura, middleware, auth JWT, análisis asíncrono, validaciones por endpoint, errores, tests. | `RF-B`, `CB-B`, `RNF-B` |
| 5 | [`spec-energy-frontend-v1.md`](spec-energy-frontend-v1.md) | Aplicación React: rutas, layout, pantallas, estados, formato de datos, flujo "Run AI Analysis", guion de demo. | `RF-F`, `CB-F`, `RNF-F` |
| 6 | [`spec-energy-pruebas-v1.md`](spec-energy-pruebas-v1.md) | Estrategia transversal de pruebas: niveles y herramientas, oráculo del dataset, fixtures, fake de DeepSeek, golden files, cobertura, CI, smoke test, ensayo de demo y proceso de UAT. Los tests por requisito siguen en la sección 12a de cada spec de módulo. | `RF-T`, `CB-T`, `RNF-T` |

## Reglas para el implementador

1. Lee el maestro completo antes de abrir cualquier hijo.
2. Si un hijo necesita cambiar un contrato de datos o de API, se cambia primero en el maestro (sección 8) y luego en el hijo.
3. Los marcadores `[SUPUESTO]`, `[PENDIENTE]` y `[DECISIÓN ABIERTA]` están consolidados en la sección 11 de cada spec. No inventes la respuesta: aplica el valor por defecto que indica el marcador y deja el marcador visible en el código (comentario) si afecta a la implementación.
4. `expected_results.csv` está prohibido en código, tests, prompts y datos. Los casos esperados que citan los specs provienen del PDF de la prueba.
5. Orden de implementación recomendado: `docker-compose` + migraciones → ingesta → motor (con sus tests de aceptación sobre los CSV reales) → API → frontend → guion de demo. El `Makefile` con `make test`, `make lint` y `make smoke` del spec de pruebas se crea junto con el `docker-compose`, no al final.

## Resultado esperado sobre el dataset entregado

| Medidor | Tipo | Severidad | Confianza | Prioridad |
|---|---|---|---|---|
| M-109 | `REAL_ANOMALY` | `HIGH` | 0,98 | 1 |
| M-112 | `DATA_QUALITY` | `HIGH` | 0,98 | 2 |
| M-104 | `EXPLAINABLE_ANOMALY` | `MEDIUM` | 0,95 | 3 |
| M-106 | `FALSE_POSITIVE` | `LOW` | 0,80 | 4 |
| Otros 8 | sin anomalía | — | — | — |

Estos valores se derivan de las reglas del spec del motor aplicadas a `data/readings.csv` y `data/events.csv`; son el test de aceptación principal.
