// Handlers MSW que imitan el backend del maestro §8.2 sobre el dataset real (dataset.json) y
// simulan el análisis de IA con progreso por etapas. Se usan en el navegador (VITE_USE_MOCKS=true)
// y en los tests.
import { HttpResponse, delay, http } from 'msw';
import dataset from './data/dataset.json';
import { ANOMALY_SEEDS, type MockAnomalySeed } from './anomalies';
import type {
  AnalysisStatus,
  AnalysisStep,
  AnomalyDetail,
  AnomalyListItem,
  AnomalyStatus,
  DashboardSummary,
  MeterDetail,
  MeterEvent,
  MeterListItem,
  MeterStatus,
  Severity,
  StepKey,
} from '@/types/api';

type Dataset = typeof dataset;
type MockMeter = Dataset['meters'][keyof Dataset['meters']];

const BASE = (import.meta.env.VITE_API_URL as string | undefined) || 'http://localhost:8080';
const url = (path: string) => `${BASE}${path}`;
const LATENCY = () => delay(typeof process !== 'undefined' && process.env.VITEST ? 0 : 180);

export const DEMO_EMAIL = 'analista@energy.local';
export const DEMO_PASSWORD = 'Demo1234!';
export const DEMO_TOKEN = 'mock.eyJzdWIiOiJhbmFsaXN0YUBlbmVyZ3kubG9jYWwifQ.signature';

const STEP_DEFS: { key: StepKey; label: string; detail: string }[] = [
  { key: 'READINGS', label: 'Lecturas', detail: '4.032 lecturas de 12 medidores' },
  { key: 'BASELINE', label: 'Baseline', detail: 'Baseline calculado para 12 medidores' },
  { key: 'DETECTION', label: 'Detección', detail: '4 medidores con desviaciones o inconsistencias' },
  { key: 'CORRELATION', label: 'Correlación', detail: 'Corroboración eléctrica evaluada en 4 medidores' },
  { key: 'EVENTS', label: 'Eventos', detail: '4 anomalías clasificadas' },
  { key: 'EXPLANATION', label: 'Explicación', detail: '4 explicaciones por LLM, 0 por plantilla' },
  { key: 'RECOMMENDATION', label: 'Recomendación', detail: '4 anomalías · 2 requieren atención prioritaria' },
];

export const MOCK_STEP_MS = 650;

interface MockState {
  analyses: Map<string, AnalysisStatus>;
  anomalies: Map<string, AnomalyDetail>;
  currentAnalysisId: string | null;
  timers: ReturnType<typeof setTimeout>[];
}

const state: MockState = { analyses: new Map(), anomalies: new Map(), currentAnalysisId: null, timers: [] };

export function resetMockState(): void {
  state.timers.forEach(clearTimeout);
  state.timers = [];
  state.analyses.clear();
  state.anomalies.clear();
  state.currentAnalysisId = null;
}

const meterById = (id: string): MockMeter | undefined => (dataset.meters as Record<string, MockMeter>)[id];
const allMeters = (): MockMeter[] => Object.values(dataset.meters as Record<string, MockMeter>);

const isAuthed = (req: Request) => req.headers.get('authorization') === `Bearer ${DEMO_TOKEN}`;
const unauthorized = () =>
  HttpResponse.json({ error: { code: 'UNAUTHORIZED', message: 'Token ausente o inválido', details: null } }, { status: 401 });
const notFound = (code: string, message: string) =>
  HttpResponse.json({ error: { code, message, details: null } }, { status: 404 });

function currentAnomalyFor(meterId: string): AnomalyDetail | null {
  for (const a of state.anomalies.values()) if (!a.superseded && a.meter_id === meterId) return a;
  return null;
}

function meterStatusFor(meterId: string): MeterStatus {
  const a = currentAnomalyFor(meterId);
  if (!a) return 'OK';
  if (a.type === 'FALSE_POSITIVE') return 'OK';
  if (a.type === 'REAL_ANOMALY' && a.severity === 'HIGH') return 'CRITICAL';
  return 'ALERT';
}

function toListItem(m: MockMeter): MeterListItem {
  const a = currentAnomalyFor(m.meter_id);
  return {
    meter_id: m.meter_id,
    name: m.name,
    location: m.location,
    status: meterStatusFor(m.meter_id),
    current_consumption_kwh: m.current_consumption_kwh,
    baseline_daily_kwh: m.baseline_daily_kwh,
    variation_pct: m.variation_pct,
    period_consumption_kwh: m.period_consumption_kwh,
    anomaly: a ? { id: a.id, type: a.type, severity: a.severity, confidence: a.confidence } : null,
  };
}

const SEV_RANK: Record<Severity, number> = { HIGH: 3, MEDIUM: 2, LOW: 1 };
const STATUS_RANK: Record<MeterStatus, number> = { CRITICAL: 3, ALERT: 2, OK: 1 };

function sortMeters(items: MeterListItem[], sort: string, order: string): MeterListItem[] {
  const dir = order === 'asc' ? 1 : -1;
  const bySeverity = (a: MeterListItem, b: MeterListItem) => {
    const s = STATUS_RANK[a.status] - STATUS_RANK[b.status];
    if (s !== 0) return s;
    const r = (a.anomaly ? SEV_RANK[a.anomaly.severity] : 0) - (b.anomaly ? SEV_RANK[b.anomaly.severity] : 0);
    if (r !== 0) return r;
    return b.meter_id.localeCompare(a.meter_id);
  };
  const cmp: (a: MeterListItem, b: MeterListItem) => number =
    sort === 'consumption'
      ? (a, b) => a.current_consumption_kwh - b.current_consumption_kwh
      : sort === 'variation'
        ? (a, b) => (a.variation_pct ?? -Infinity) - (b.variation_pct ?? -Infinity)
        : bySeverity;
  return [...items].sort((a, b) => dir * cmp(a, b) || a.meter_id.localeCompare(b.meter_id));
}

function buildAnomaly(seed: MockAnomalySeed, analysisId: string, detectedAt: string): AnomalyDetail {
  const m = meterById(seed.meter_id)!;
  const day = m.daily.find((d) => d.date === seed.window_end) ?? m.daily[m.daily.length - 1];
  const ev = (dataset.events as MeterEvent[]).find((e) => e.id === seed.related_event_id) ?? null;
  const pct = (obs: number, base: number) => (base ? Math.round(((obs - base) / base) * 1000) / 10 : null);
  return {
    id: seed.id,
    analysis_id: analysisId,
    superseded: false,
    meter_id: seed.meter_id,
    type: seed.type,
    severity: seed.severity,
    confidence: seed.confidence,
    priority_rank: seed.priority_rank,
    reason: seed.reason,
    recommended_action: seed.recommended_action,
    status: seed.status,
    detected_at: detectedAt,
    window_start: seed.window_start,
    window_end: seed.window_end,
    explanation_source: seed.explanation_source,
    explanation_points: seed.explanation_points,
    related_event: ev,
    comparison: {
      consumption_kwh: { observed: day.consumption_kwh, baseline: m.baseline_daily_kwh, delta_pct: pct(day.consumption_kwh, m.baseline_daily_kwh) },
      voltage_v: { observed: day.voltage_avg, baseline: m.electrical.voltage_v.baseline_avg, delta_pct: pct(day.voltage_avg, m.electrical.voltage_v.baseline_avg) },
      current_a: { observed: day.current_avg, baseline: m.electrical.current_a.baseline_avg, delta_pct: pct(day.current_avg, m.electrical.current_a.baseline_avg) },
      power_factor: { observed: day.power_factor_avg, baseline: m.electrical.power_factor.baseline_avg, delta_pct: pct(day.power_factor_avg, m.electrical.power_factor.baseline_avg) },
    },
    evidence: seed.evidence,
  };
}

function toListAnomaly(a: AnomalyDetail): AnomalyListItem {
  const { explanation_source: _s, explanation_points: _p, related_event: _e, comparison: _c, evidence: _v, ...rest } = a;
  return rest;
}

function currentAnomalies(): AnomalyDetail[] {
  return [...state.anomalies.values()].filter((a) => !a.superseded).sort((a, b) => a.priority_rank - b.priority_rank);
}

function completeAnalysis(analysis: AnalysisStatus): void {
  const finishedAt = new Date().toISOString();
  for (const a of state.anomalies.values()) a.superseded = true;
  for (const seed of ANOMALY_SEEDS) {
    const previous = state.anomalies.get(seed.id);
    const built = buildAnomaly(seed, analysis.id, finishedAt);
    if (previous) built.status = previous.status;
    state.anomalies.set(seed.id, built);
  }
  analysis.status = 'COMPLETED';
  analysis.finished_at = finishedAt;
  analysis.summary = { anomalies_detected: 4, high_priority: 2, meters_analyzed: 12, explanation_source: 'llm' };
  state.currentAnalysisId = null;
}

function scheduleAnalysis(analysis: AnalysisStatus): void {
  const stepMs = MOCK_STEP_MS;
  STEP_DEFS.forEach((def, i) => {
    state.timers.push(
      setTimeout(() => {
        analysis.status = 'RUNNING';
        if (!analysis.started_at) analysis.started_at = new Date().toISOString();
        const step = analysis.steps[i];
        step.status = 'RUNNING';
        step.started_at = new Date().toISOString();
      }, stepMs * i + 120),
    );
    state.timers.push(
      setTimeout(() => {
        const step = analysis.steps[i];
        step.status = 'COMPLETED';
        step.finished_at = new Date().toISOString();
        step.detail = def.detail;
        if (i === STEP_DEFS.length - 1) completeAnalysis(analysis);
      }, stepMs * (i + 1)),
    );
  });
}

function newAnalysis(): AnalysisStatus {
  const id = crypto.randomUUID();
  const steps: AnalysisStep[] = STEP_DEFS.map((d) => ({ key: d.key, label: d.label, status: 'PENDING', started_at: null, finished_at: null, detail: null }));
  const analysis: AnalysisStatus = { id, status: 'QUEUED', steps, summary: null, error_message: null, started_at: null, finished_at: null };
  state.analyses.set(id, analysis);
  state.currentAnalysisId = id;
  scheduleAnalysis(analysis);
  return analysis;
}

function lastAnalysis(): AnalysisStatus | null {
  const all = [...state.analyses.values()];
  return all.length ? all[all.length - 1] : null;
}

function dashboard(): DashboardSummary {
  const anomalies = currentAnomalies();
  const byStatus: Record<MeterStatus, number> = { OK: 0, ALERT: 0, CRITICAL: 0 };
  for (const m of allMeters()) byStatus[meterStatusFor(m.meter_id)] += 1;
  const last = lastAnalysis();
  return {
    meters_total: allMeters().length,
    total_consumption_kwh: dataset.total_consumption_kwh,
    period: dataset.period,
    anomalies_total: anomalies.length,
    high_priority_total: anomalies.filter((a) => a.severity === 'HIGH').length,
    ai_confidence_avg: anomalies.length ? Math.round((anomalies.reduce((s, a) => s + a.confidence, 0) / anomalies.length) * 100) / 100 : null,
    meters_by_status: byStatus,
    last_analysis: last ? { id: last.id, status: last.status, finished_at: last.finished_at } : null,
    daily_consumption: dataset.daily_consumption,
    top_anomalies: anomalies.slice(0, 3).map((a) => ({ id: a.id, meter_id: a.meter_id, type: a.type, severity: a.severity, confidence: a.confidence })),
  };
}

export const handlers = [
  http.get(url('/health'), () =>
    HttpResponse.json({ status: 'ok', db: 'ok', ingest: { status: 'COMPLETED', readings: 4032, events: 4, meters: 12, rejected: 0, duplicates: 0, duration_ms: 1840 } }),
  ),

  http.post(url('/auth/login'), async ({ request }) => {
    await LATENCY();
    const body = (await request.json().catch(() => ({}))) as { email?: string; password?: string };
    if (body.email !== DEMO_EMAIL || body.password !== DEMO_PASSWORD) {
      return HttpResponse.json({ error: { code: 'INVALID_CREDENTIALS', message: 'Credenciales inválidas', details: null } }, { status: 401 });
    }
    return HttpResponse.json({ token: DEMO_TOKEN, expires_at: new Date(Date.now() + 8 * 3600 * 1000).toISOString(), user: { email: DEMO_EMAIL, name: 'Analista Demo' } });
  }),

  http.get(url('/meters'), async ({ request }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const u = new URL(request.url);
    const statuses = u.searchParams.getAll('status');
    const q = (u.searchParams.get('q') ?? '').toLowerCase();
    const sort = u.searchParams.get('sort') ?? 'severity';
    const order = u.searchParams.get('order') ?? 'desc';
    let items = allMeters().map(toListItem);
    if (statuses.length) items = items.filter((m) => statuses.includes(m.status));
    if (q) items = items.filter((m) => m.meter_id.toLowerCase().includes(q));
    items = sortMeters(items, sort, order);
    return HttpResponse.json({ items, total: items.length });
  }),

  http.get(url('/meters/:meterId'), async ({ request, params }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const m = meterById(String(params.meterId));
    if (!m) return notFound('METER_NOT_FOUND', `No existe el medidor ${String(params.meterId)}`);
    const detail: MeterDetail = { ...toListItem(m), period: m.period, electrical: m.electrical, events: m.events, daily: m.daily.map((d) => ({ date: d.date, consumption_kwh: d.consumption_kwh, deviation_pct: d.deviation_pct })) };
    return HttpResponse.json(detail);
  }),

  http.get(url('/meters/:meterId/readings'), async ({ request, params }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const m = meterById(String(params.meterId));
    if (!m) return notFound('METER_NOT_FOUND', `No existe el medidor ${String(params.meterId)}`);
    const u = new URL(request.url);
    const granularity = u.searchParams.get('granularity') ?? 'hour';
    if (granularity === 'day') return HttpResponse.json({ meter_id: m.meter_id, granularity: 'day', items: m.daily });
    const items = m.readings.map(({ meter_id: _id, ...r }) => r);
    return HttpResponse.json({ meter_id: m.meter_id, granularity: 'hour', baseline_profile: m.baseline_profile, items });
  }),

  http.get(url('/anomalies'), async ({ request }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const u = new URL(request.url);
    const types = u.searchParams.getAll('type');
    const sevs = u.searchParams.getAll('severity');
    const sts = u.searchParams.getAll('status');
    let items = currentAnomalies();
    if (types.length) items = items.filter((a) => types.includes(a.type));
    if (sevs.length) items = items.filter((a) => sevs.includes(a.severity));
    if (sts.length) items = items.filter((a) => sts.includes(a.status));
    const last = [...state.analyses.values()].filter((a) => a.status === 'COMPLETED').pop();
    return HttpResponse.json({ analysis_id: last?.id ?? null, items: items.map(toListAnomaly), total: items.length });
  }),

  http.get(url('/anomalies/:id'), async ({ request, params }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const a = state.anomalies.get(String(params.id));
    if (!a) return notFound('ANOMALY_NOT_FOUND', 'La anomalía no existe');
    return HttpResponse.json(a);
  }),

  http.patch(url('/anomalies/:id/status'), async ({ request, params }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const a = state.anomalies.get(String(params.id));
    if (!a) return notFound('ANOMALY_NOT_FOUND', 'La anomalía no existe');
    const body = (await request.json().catch(() => ({}))) as { status?: AnomalyStatus };
    const allowed: AnomalyStatus[] = ['OPEN', 'INVESTIGATING', 'RESOLVED', 'DISMISSED'];
    if (!body.status || !allowed.includes(body.status)) {
      return HttpResponse.json({ error: { code: 'INVALID_BODY', message: 'status inválido', details: { field: 'status', allowed } } }, { status: 400 });
    }
    a.status = body.status;
    return HttpResponse.json(a);
  }),

  http.post(url('/ai/analyze'), async ({ request }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    const running = state.currentAnalysisId ? state.analyses.get(state.currentAnalysisId) : null;
    if (running && (running.status === 'QUEUED' || running.status === 'RUNNING')) {
      return HttpResponse.json({ error: { code: 'ANALYSIS_IN_PROGRESS', message: 'Ya hay un análisis en curso', details: { analysis_id: running.id } } }, { status: 409 });
    }
    const analysis = newAnalysis();
    return HttpResponse.json({ analysis_id: analysis.id, status: analysis.status, created_at: new Date().toISOString() }, { status: 202 });
  }),

  http.get(url('/ai/analysis/:id'), async ({ request, params }) => {
    if (!isAuthed(request)) return unauthorized();
    const a = state.analyses.get(String(params.id));
    if (!a) return notFound('ANALYSIS_NOT_FOUND', 'El análisis no existe');
    return HttpResponse.json(a, { headers: { 'Cache-Control': 'no-store' } });
  }),

  http.get(url('/dashboard/summary'), async ({ request }) => {
    if (!isAuthed(request)) return unauthorized();
    await LATENCY();
    return HttpResponse.json(dashboard());
  }),
];
