// RF-F-02 — textos del chip de análisis.
import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/client';
import type { AnalysisStatus, DashboardSummary } from '@/types/api';
import { chipState } from './Header';

const now = new Date('2026-09-25T15:04:11Z');
const summary = (last: DashboardSummary['last_analysis']): DashboardSummary => ({
  meters_total: 12, total_consumption_kwh: 0, period: { start: '', end: '' }, anomalies_total: 0, high_priority_total: 0, ai_confidence_avg: null,
  meters_by_status: { OK: 12, ALERT: 0, CRITICAL: 0 }, last_analysis: last, daily_consumption: [], top_anomalies: [],
});
const steps = (runningIndex: number): AnalysisStatus['steps'] =>
  ['Lecturas', 'Baseline', 'Detección', 'Correlación', 'Eventos', 'Explicación', 'Recomendación'].map((label, i) => ({
    key: 'READINGS', label, status: i < runningIndex ? 'COMPLETED' : i === runningIndex ? 'RUNNING' : 'PENDING', started_at: null, finished_at: null, detail: null,
  }));

describe('chipState', () => {
  it('Sin análisis', () => expect(chipState(summary(null), null, null, false, now).text).toBe('Sin análisis'));
  it('En cola', () => expect(chipState(undefined, null, null, true, now).text).toBe('En curso · En cola'));
  it('En curso · Detección 3/7', () => {
    const a: AnalysisStatus = { id: 'x', status: 'RUNNING', steps: steps(2), summary: null, error_message: null, started_at: null, finished_at: null };
    expect(chipState(undefined, null, a, true, now).text).toBe('En curso · Detección 3/7');
  });
  it('Completado hace 2 min', () => expect(chipState(summary({ id: 'x', status: 'COMPLETED', finished_at: '2026-09-25T15:02:11Z' }), null, null, false, now).text).toBe('Completado hace 2 min'));
  it('Falló', () => expect(chipState(summary({ id: 'x', status: 'FAILED', finished_at: null }), null, null, false, now).text).toBe('Falló'));
  it('Sin conexión', () => expect(chipState(undefined, new ApiError(0, 'NETWORK_ERROR', 'x'), null, false, now).text).toBe('Sin conexión'));
});
