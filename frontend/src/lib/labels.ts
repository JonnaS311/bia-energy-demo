// RF-F-11 — etiquetas en español para los enums de la API. Ningún enum se muestra en crudo.
import type { AnalysisStatusValue, AnomalyStatus, AnomalyType, MeterStatus, Severity } from '@/types/api';

const TYPE: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Anomalía real',
  EXPLAINABLE_ANOMALY: 'Anomalía explicable',
  FALSE_POSITIVE: 'Falso positivo',
  DATA_QUALITY: 'Calidad de datos',
};

const SEVERITY: Record<Severity, string> = { HIGH: 'Alta', MEDIUM: 'Media', LOW: 'Baja' };

const METER_STATUS: Record<MeterStatus, string> = { OK: 'OK', ALERT: 'ALERTA', CRITICAL: 'CRÍTICO' };

const ANOMALY_STATUS: Record<AnomalyStatus, string> = {
  OPEN: 'Abierta',
  INVESTIGATING: 'En investigación',
  RESOLVED: 'Resuelta',
  DISMISSED: 'Descartada',
};

const ANALYSIS_STATUS: Record<AnalysisStatusValue, string> = {
  QUEUED: 'En cola',
  RUNNING: 'En curso',
  COMPLETED: 'Completado',
  FAILED: 'Falló',
};

const METRIC: Record<string, string> = {
  DAILY_CONSUMPTION: 'Consumo diario',
  PERSISTENCE_DAYS: 'Días consecutivos',
  HOURLY_OUTLIERS: 'Lecturas fuera de patrón horario',
  CURRENT_A: 'Corriente',
  POWER_FACTOR: 'Factor de potencia',
  VOLTAGE_OUT_OF_RANGE: 'Voltaje fuera de rango',
  PHYSICAL_RATIO: 'Ratio físico kWh/(V·I·PF)',
  RELATED_EVENT: 'Evento relacionado',
  INSUFFICIENT_DATA: 'Datos insuficientes',
};

export const typeLabel = (t: AnomalyType): string => TYPE[t] ?? t;
export const severityLabel = (s: Severity): string => SEVERITY[s] ?? s;
export const statusLabel = (s: MeterStatus): string => METER_STATUS[s] ?? s;
export const anomalyStatusLabel = (s: AnomalyStatus): string => ANOMALY_STATUS[s] ?? s;
export const analysisStatusLabel = (s: AnalysisStatusValue): string => ANALYSIS_STATUS[s] ?? s;

export function confidenceLabel(c: number): string {
  if (c >= 0.9) return 'Alta';
  if (c >= 0.75) return 'Media/Alta';
  return 'Media';
}

export function metricLabel(metric: string): string {
  const label = METRIC[metric];
  if (!label) {
    console.warn('metric sin etiqueta', metric);
    return metric;
  }
  return label;
}

export const ANOMALY_TYPES: AnomalyType[] = ['REAL_ANOMALY', 'EXPLAINABLE_ANOMALY', 'FALSE_POSITIVE', 'DATA_QUALITY'];
export const SEVERITIES: Severity[] = ['HIGH', 'MEDIUM', 'LOW'];
export const ANOMALY_STATUSES: AnomalyStatus[] = ['OPEN', 'INVESTIGATING', 'RESOLVED', 'DISMISSED'];
