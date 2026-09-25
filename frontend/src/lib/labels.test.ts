// RF-F-11 — etiquetas de enums.
import { describe, expect, it, vi } from 'vitest';
import { analysisStatusLabel, anomalyStatusLabel, confidenceLabel, metricLabel, severityLabel, statusLabel, typeLabel } from './labels';

describe('labels', () => {
  it('typeLabel', () => {
    expect(typeLabel('REAL_ANOMALY')).toBe('Anomalía real');
    expect(typeLabel('EXPLAINABLE_ANOMALY')).toBe('Anomalía explicable');
    expect(typeLabel('FALSE_POSITIVE')).toBe('Falso positivo');
    expect(typeLabel('DATA_QUALITY')).toBe('Calidad de datos');
  });

  it('severity / meter status / anomaly status / analysis status', () => {
    expect(severityLabel('HIGH')).toBe('Alta');
    expect(severityLabel('MEDIUM')).toBe('Media');
    expect(severityLabel('LOW')).toBe('Baja');
    expect(statusLabel('OK')).toBe('OK');
    expect(statusLabel('ALERT')).toBe('ALERTA');
    expect(statusLabel('CRITICAL')).toBe('CRÍTICO');
    expect(anomalyStatusLabel('OPEN')).toBe('Abierta');
    expect(anomalyStatusLabel('INVESTIGATING')).toBe('En investigación');
    expect(anomalyStatusLabel('RESOLVED')).toBe('Resuelta');
    expect(anomalyStatusLabel('DISMISSED')).toBe('Descartada');
    expect(analysisStatusLabel('QUEUED')).toBe('En cola');
    expect(analysisStatusLabel('RUNNING')).toBe('En curso');
    expect(analysisStatusLabel('COMPLETED')).toBe('Completado');
    expect(analysisStatusLabel('FAILED')).toBe('Falló');
  });

  it('confidenceLabel por rangos', () => {
    expect(confidenceLabel(0.9)).toBe('Alta');
    expect(confidenceLabel(0.98)).toBe('Alta');
    expect(confidenceLabel(0.8)).toBe('Media/Alta');
    expect(confidenceLabel(0.75)).toBe('Media/Alta');
    expect(confidenceLabel(0.74)).toBe('Media');
  });

  it('metricLabel devuelve el literal y avisa cuando no está mapeado', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    expect(metricLabel('DAILY_CONSUMPTION')).toBe('Consumo diario');
    expect(metricLabel('PHYSICAL_RATIO')).toBe('Ratio físico kWh/(V·I·PF)');
    expect(metricLabel('UNKNOWN_METRIC')).toBe('UNKNOWN_METRIC');
    expect(warn).toHaveBeenCalledWith('metric sin etiqueta', 'UNKNOWN_METRIC');
    warn.mockRestore();
  });
});
