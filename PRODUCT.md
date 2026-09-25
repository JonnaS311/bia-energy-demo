# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

React 18 + TypeScript 5 + Vite 5 + React Router 6 + TanStack Query 5 + Recharts 2 + Tailwind CSS 3 (fijado por el usuario en `docs/specs/spec-energy-plataforma-maestro-v1.md` §9 y `spec-energy-frontend-v1.md` RF-F-01). Backend en Go (aún no construido); en desarrollo el frontend corre contra mocks MSW en el navegador (`VITE_USE_MOCKS=true`) que devuelven los JSON de ejemplo del maestro §8.2.

## Users

Analista de operaciones de una empresa con 12 medidores eléctricos industriales. Trabaja en escritorio (≥ 1280 px), en horario laboral, revisando si algún medidor se salió de su comportamiento y decidiendo qué investigar primero. Único rol de la aplicación; inicia sesión con credenciales demo.

Segundo público confirmado: el evaluador de la prueba técnica, que recorre la app en 5-10 minutos siguiendo el guion `Login → Dashboard → M-109 → Run AI Analysis → Anomalía → Explicación → Acción` y califica con una rúbrica (Frontend/UX 20 pts; IA 100 pts sobre 4 casos).

## Product Purpose

Convertir 4.032 lecturas horarias (consumo, voltaje, corriente, factor de potencia) de 14 días en una decisión operativa: qué medidor requiere atención, si la anomalía es real, explicable por un evento operativo o un problema de calidad de datos, con qué confianza, por qué, y qué acción tomar. Éxito: la persona identifica el medidor prioritario y entiende la razón sin abrir los CSV.

## Positioning

La IA no devuelve "sí/no": un motor determinístico clasifica en 4 tipos (`REAL_ANOMALY`, `EXPLAINABLE_ANOMALY`, `FALSE_POSITIVE`, `DATA_QUALITY`), calcula severidad y confianza a partir de evidencia numérica verificable (baseline, persistencia, corroboración eléctrica, ratio físico kWh/(V·I·PF), eventos), y un LLM solo redacta la explicación. Cada anomalía muestra su evidencia estructurada; nada es una caja negra.

## Operating Context

- Datos: `data/readings.csv` (12 medidores M-101…M-112, 2026-09-01 a 2026-09-14) y `data/events.csv` (4 eventos). Sin datos personales.
- Flujo: Dashboard → Medidores → Detalle → Anomalías IA → Investigación → Acción. El análisis se ejecuta bajo demanda ("Run AI Analysis") y tarda segundos; el progreso se muestra en 7 etapas.
- Resultado esperado sobre el dataset: M-109 anomalía real HIGH 0,98 (prioridad 1), M-112 calidad de datos HIGH 0,98, M-104 explicable MEDIUM 0,95, M-106 falso positivo LOW 0,80; otros 8 sin anomalía.
- Demo local con `docker compose up`; posible despliegue gratuito (Render + Neon + Vercel) sin cambios de arquitectura.

## Capabilities and Constraints

- Pantallas: Login, Dashboard (6 KPI + consumo diario + top anomalías), Medidores (filtros, búsqueda, orden), Detalle de medidor (consumo horario con banda baseline y eventos, V/I/PF), Anomalías IA (tabla priorizada), Investigación (7 secciones literales), modal de análisis con stepper y polling.
- Contratos de API fijos en el maestro §8.2; textos de UI literales fijos en `spec-energy-frontend-v1.md`; enums en inglés, UI en español; fechas en UTC; formato numérico es-CO.
- Sin tema claro (decisión del usuario 2026-09-25: tema oscuro), sin responsive < 1280 px, sin i18n, sin paginación, sin WebSockets.
- Accesibilidad: WCAG AA en contraste, foco visible, tablas semánticas, gráficas con `role="img"`.

## Brand Commitments

- Nombre visible: "Energy Management". Tipografía Inter.
- Paleta pinned por el usuario: la de https://www.bia.app/ (comercializadora de energía colombiana, tagline "Energía Inteligente"): fondo casi negro `#0f0f0f` / `#0a0a0a`, superficies `#1a1a1a`, `#1f1f1f`, `#27272a`, `#2a2a2a`; texto principal blanco, secundario `#a1a1aa`, terciario `#9ca3af`/`#d1d5db`; acento principal menta `#08ddbc` (con brillo `#17ffdb` en gradientes y halos `rgba(8,221,188,.3)`); secundarios violeta `#8b5cf6`, azul `#3b82f6`, cian `#06b6d4`; semánticos ámbar `#f59e0b`, rojo `#ef4444`, verde `#10b981`.
- Estilo de la referencia: oscuro, sobrio, superficies planas con bordes sutiles, acento menta reservado a acciones y datos clave, brillos suaves en elementos activos.

## Evidence on Hand

- Specs completos en `docs/specs/` (maestro, data, motor, backend, frontend, pruebas) con JSON de ejemplo reales para cada endpoint.
- Dataset real en `readings.csv` y `events.csv` (raíz; se moverán a `data/`).
- No existen logos, ilustraciones, fotografías ni testimonios. No inventar clientes, cifras comerciales ni certificaciones.

## Product Principles

1. La prioridad se ve antes que el detalle: quien abre la app sabe en segundos qué medidor atender.
2. Toda conclusión de la IA va acompañada de su evidencia numérica; nunca una afirmación sin dato.
3. Los textos, umbrales y resultados del spec son contrato: la UI no reinterpreta ni reordena la decisión del motor.
4. Herramienta de operación, no landing: densidad útil, consistencia entre pantallas, el color acentúa estado y acción.
5. Funciona sin red externa: fallback por plantillas y mocks locales garantizan la demo.

## Accessibility & Inclusion

Contraste WCAG AA (≥ 4,5:1 texto normal, ≥ 3:1 texto grande y badges) sobre fondos oscuros; anillo de foco visible de 2 px; `aria-label` en botones de solo icono; tablas con `<th scope="col">`.
