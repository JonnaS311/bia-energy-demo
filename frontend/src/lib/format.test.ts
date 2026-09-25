// RF-F-12 — ejecutado con TZ=America/Bogota (vite.config.ts) para verificar que no hay conversión de zona.
import { describe, expect, it } from 'vitest';
import { formatConfidence, formatDate, formatDateTime, formatDayShort, formatKwh, formatNumber, formatPct, formatPctInt, formatRelative, formatWithUnit } from './format';

describe('format', () => {
  it('formatKwh usa miles "." y decimal ","', () => {
    expect(formatKwh(155250.8)).toBe('155.250,8');
    expect(formatKwh(728.8)).toBe('728,8');
    expect(formatKwh(2207.6)).toBe('2.207,6');
  });

  it('formatPct lleva signo explícito y U+2212 para negativos', () => {
    expect(formatPct(110.5)).toBe('+110,5 %');
    expect(formatPct(-0.1)).toBe('−0,1 %');
    expect(formatPct(0)).toBe('+0,0 %');
    expect(formatPct(null)).toBe('—');
  });

  it('formatPctInt y formatConfidence', () => {
    expect(formatPctInt(0.93)).toBe('93 %');
    expect(formatPctInt(null)).toBe('—');
    expect(formatConfidence(0.98)).toBe('0,98');
  });

  it('formatNumber respeta decimales', () => {
    expect(formatNumber(420.5, 1)).toBe('420,5');
    expect(formatNumber(0.744, 3)).toBe('0,744');
    expect(formatNumber(-1234.5, 1)).toBe('−1.234,5');
  });

  it('fechas en UTC sin conversión local', () => {
    expect(formatDateTime('2026-09-25T15:02:11Z')).toBe('25 sep 2026 15:02');
    expect(formatDate('2026-09-14')).toBe('14 sep 2026');
    expect(formatDayShort('2026-09-14')).toBe('14 sep');
    expect(formatDateTime('2026-09-12T14:00:00Z')).toBe('12 sep 2026 14:00');
  });

  it('formatRelative', () => {
    const now = new Date('2026-09-25T15:04:11Z');
    expect(formatRelative('2026-09-25T15:04:00Z', now)).toBe('hace 11 s');
    expect(formatRelative('2026-09-25T15:02:11Z', now)).toBe('hace 2 min');
    expect(formatRelative('2026-09-25T13:02:11Z', now)).toBe('hace 2 h');
    expect(formatRelative('2026-09-22T15:02:11Z', now)).toBe('hace 3 d');
  });

  it('formatWithUnit por unidad de evidencia', () => {
    expect(formatWithUnit(2207.6, 'kWh')).toBe('2.207,6 kWh');
    expect(formatWithUnit(420.5, 'A')).toBe('420,5 A');
    expect(formatWithUnit(4.22, 'ratio')).toBe('4,22');
    expect(formatWithUnit(3, 'days')).toBe('3 d');
    expect(formatWithUnit(16, 'readings')).toBe('16');
    expect(formatWithUnit(0.744, '')).toBe('0,744');
    expect(formatWithUnit(null, 'kWh')).toBe('—');
  });
});
