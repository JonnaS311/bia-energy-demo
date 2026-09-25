// Anomalías simuladas para los mocks: los 4 casos del dataset con los valores de los specs
// (maestro §8.2.6 para M-109; spec del motor RF-E-14 para M-112 y M-106; M-104 derivado del dataset).
// Los números eléctricos se completan en tiempo de ejecución desde dataset.json (ver handlers.ts).
import type { AnomalyStatus, AnomalyType, EvidenceItem, ExplanationSource, Severity } from '@/types/api';

export interface MockAnomalySeed {
  id: string;
  meter_id: string;
  type: AnomalyType;
  severity: Severity;
  confidence: number;
  priority_rank: number;
  window_start: string;
  window_end: string;
  related_event_id: number | null;
  reason: string;
  recommended_action: string;
  explanation_points: string[];
  explanation_source: ExplanationSource;
  status: AnomalyStatus;
  /** Evidencia fija; los ítems con `observed: null` y métrica eléctrica se rellenan desde el dataset. */
  evidence: EvidenceItem[];
}

export const ANOMALY_SEEDS: MockAnomalySeed[] = [
  {
    id: '6f1c2a3e-9b4d-4c7a-8e1f-0a1b2c3d4e5f',
    meter_id: 'M-109',
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.98,
    priority_rank: 1,
    window_start: '2026-09-12',
    window_end: '2026-09-14',
    related_event_id: null,
    reason:
      'El consumo diario de M-109 subió 110,5 % sobre su baseline de 1.048,8 kWh durante 3 días consecutivos sin ningún evento operativo que lo explique; la corriente media diaria se multiplicó por 2,1 y el factor de potencia cayó de 0,94 a 0,74.',
    recommended_action: 'Investigar el medidor y la instalación.',
    explanation_points: [
      'Consumo diario: 2.207,6 kWh el 2026-09-14 frente a un baseline de 1.048,8 kWh (+110,5 %).',
      'Cambio persistente: 3 días consecutivos por encima del 20 % (12, 13 y 14 de septiembre).',
      'Corriente media diaria: 420,5 A frente a 200,5 A de baseline (+109,7 %), coherente con el aumento de consumo.',
      'Factor de potencia medio: 0,744 frente a 0,939 (−0,195); el 13 y 14 cae por debajo de 0,75 en horas de carga.',
      'Sin evento operativo relacionado: el único evento del medidor es de tipo UNKNOWN.',
    ],
    explanation_source: 'llm',
    status: 'OPEN',
    evidence: [
      { position: 1, metric: 'DAILY_CONSUMPTION', observed: 2207.6, baseline: 1048.8, delta_pct: 110.5, unit: 'kWh', window: '2026-09-14', detail: 'Total diario 110,5 % por encima del baseline' },
      { position: 2, metric: 'PERSISTENCE_DAYS', observed: 3, baseline: null, delta_pct: null, unit: 'days', window: '2026-09-12..2026-09-14', detail: '3 días consecutivos con desviación ≥ 20 %' },
      { position: 3, metric: 'CURRENT_A', observed: 420.5, baseline: 200.5, delta_pct: 109.7, unit: 'A', window: '2026-09-14', detail: 'Corriente media diaria 2,1 veces el baseline' },
      { position: 4, metric: 'POWER_FACTOR', observed: 0.744, baseline: 0.939, delta_pct: -20.8, unit: '', window: '2026-09-14', detail: 'Factor de potencia cae 0,20 respecto al baseline' },
      { position: 5, metric: 'HOURLY_OUTLIERS', observed: 48, baseline: null, delta_pct: null, unit: 'readings', window: '2026-09-13..2026-09-14', detail: '48 lecturas fuera del perfil horario (z ≥ 3) en los dos últimos días' },
      { position: 6, metric: 'RELATED_EVENT', observed: null, baseline: null, delta_pct: null, unit: '', window: '2026-09-11..2026-09-13', detail: 'Sin evento operativo que explique el cambio (evento UNKNOWN ignorado)' },
    ],
  },
  {
    id: '2b7d9e10-5c3a-4f61-9d2e-7a8b9c0d1e2f',
    meter_id: 'M-112',
    type: 'DATA_QUALITY',
    severity: 'HIGH',
    confidence: 0.98,
    priority_rank: 2,
    window_start: '2026-09-13',
    window_end: '2026-09-14',
    related_event_id: 4,
    reason:
      'Las lecturas eléctricas de M-112 entre el 13 y el 14 de septiembre son inconsistentes entre sí: 16 lecturas con voltaje fuera de 209-231 V, un ratio kWh/(V·I·PF) de hasta 4,22 y un factor de potencia mínimo de 0,58, mientras el consumo diario se mantiene estable (+0,1 %). Esto apunta a un problema del sensor o de la transmisión de datos, no del consumo.',
    recommended_action: 'Validar el sensor y la calidad de las lecturas del medidor.',
    explanation_points: [
      'Voltaje: 16 lecturas fuera del rango 209-231 V (mínimo 201,6 V, máximo 241,2 V).',
      'Relación física: ratio kWh/(V·I·PF) máximo de 4,22, cuando el rango coherente es 0,5-2,0.',
      'Factor de potencia mínimo de 0,58 frente a 0,951 de baseline; 12 lecturas por debajo de 0,85.',
      'Consumo diario estable: 662,7 kWh frente a 662,2 kWh de baseline (+0,1 %).',
      'Evento relacionado el 2026-09-13 00:00 (DATA_QUALITY): Intermittent readings and abnormal electrical jumps.',
    ],
    explanation_source: 'llm',
    status: 'OPEN',
    evidence: [
      { position: 1, metric: 'VOLTAGE_OUT_OF_RANGE', observed: 16, baseline: null, delta_pct: null, unit: 'readings', window: '2026-09-13..2026-09-14', detail: '16 lecturas con voltaje fuera de 209-231 V (mín. 201,6 V, máx. 241,2 V)' },
      { position: 2, metric: 'PHYSICAL_RATIO', observed: 4.22, baseline: 1.0, delta_pct: 322.0, unit: 'ratio', window: '2026-09-13..2026-09-14', detail: '7 lecturas con ratio kWh/(V·I·PF) fuera de 0,5-2,0; máximo 4,22' },
      { position: 3, metric: 'POWER_FACTOR', observed: 0.58, baseline: 0.951, delta_pct: -39.0, unit: '', window: '2026-09-13..2026-09-14', detail: 'Factor de potencia mínimo 0,58 frente a 0,951 de baseline; 12 lecturas por debajo de 0,85' },
      { position: 4, metric: 'DAILY_CONSUMPTION', observed: 662.7, baseline: 662.2, delta_pct: 0.1, unit: 'kWh', window: '2026-09-14', detail: 'Consumo diario estable (+0,1 %) pese a las lecturas eléctricas inconsistentes' },
      { position: 5, metric: 'RELATED_EVENT', observed: null, baseline: null, delta_pct: null, unit: '', window: '2026-09-13', detail: 'Evento DATA_QUALITY 2026-09-13 00:00: Intermittent readings and abnormal electrical jumps' },
    ],
  },
  {
    id: '9c4e1f22-7a6b-4d83-b1c5-3e4f5a6b7c8d',
    meter_id: 'M-104',
    type: 'EXPLAINABLE_ANOMALY',
    severity: 'MEDIUM',
    confidence: 0.95,
    priority_rank: 3,
    window_start: '2026-09-11',
    window_end: '2026-09-14',
    related_event_id: 1,
    reason:
      'El consumo diario de M-104 está +47,5 % respecto a su baseline de 1.169,7 kWh durante 4 días consecutivos, y el cambio coincide con el evento operativo del 2026-09-11 00:00: New production line activated. La corriente media subió 48,2 % de forma coherente con el consumo y el factor de potencia se mantiene estable.',
    recommended_action: 'Validar con operaciones que el cambio corresponde al evento registrado.',
    explanation_points: [
      'Consumo diario: 1.725,6 kWh el 2026-09-14 frente a 1.169,7 kWh de baseline (+47,5 %).',
      'Duración: 4 días consecutivos con desviación ≥ 20 % desde el 2026-09-11.',
      'Corriente media diaria: +48,2 % respecto al baseline, coherente con una nueva carga productiva.',
      'Evento relacionado el 2026-09-11 00:00 (OPERATIONAL_CHANGE): New production line activated.',
    ],
    explanation_source: 'llm',
    status: 'OPEN',
    evidence: [
      { position: 1, metric: 'DAILY_CONSUMPTION', observed: 1725.6, baseline: 1169.7, delta_pct: 47.5, unit: 'kWh', window: '2026-09-14', detail: 'Total diario 47,5 % por encima del baseline' },
      { position: 2, metric: 'PERSISTENCE_DAYS', observed: 4, baseline: null, delta_pct: null, unit: 'days', window: '2026-09-11..2026-09-14', detail: '4 días consecutivos con desviación ≥ 20 %' },
      { position: 3, metric: 'CURRENT_A', observed: 324.7, baseline: 219.1, delta_pct: 48.2, unit: 'A', window: '2026-09-14', detail: 'Corriente media diaria 48,2 % por encima del baseline, misma dirección que el consumo' },
      { position: 4, metric: 'POWER_FACTOR', observed: 0.893, baseline: 0.91, delta_pct: -1.9, unit: '', window: '2026-09-14', detail: 'Factor de potencia sin cambio relevante (−0,017)' },
      { position: 5, metric: 'RELATED_EVENT', observed: null, baseline: null, delta_pct: null, unit: '', window: '2026-09-11', detail: 'Evento OPERATIONAL_CHANGE 2026-09-11 00:00: New production line activated' },
    ],
  },
  {
    id: 'd5a8b3c4-1e2f-4a9b-8c7d-6e5f4a3b2c1d',
    meter_id: 'M-106',
    type: 'FALSE_POSITIVE',
    severity: 'LOW',
    confidence: 0.8,
    priority_rank: 4,
    window_start: '2026-09-08',
    window_end: '2026-09-08',
    related_event_id: 2,
    reason:
      'El consumo diario de M-106 bajó 36,8 % respecto a su baseline de 1.349,1 kWh el 2026-09-08, y la caída está explicada por la parada programada del 2026-09-08 00:00: Scheduled maintenance outage for 12 hours. Al día siguiente el consumo vuelve al baseline. No se requiere escalamiento.',
    recommended_action: 'No escalar; el cambio está explicado por la parada programada.',
    explanation_points: [
      'Consumo diario: 852,4 kWh el 2026-09-08 frente a 1.349,1 kWh de baseline (−36,8 %).',
      'Duración: 1 día; el 2026-09-09 el consumo vuelve al baseline (−0,1 %).',
      'Corriente media diaria: −37,2 %, consistente con equipos apagados durante 12 horas.',
      'Evento relacionado el 2026-09-08 00:00 (SCHEDULED_OUTAGE): Scheduled maintenance outage for 12 hours.',
    ],
    explanation_source: 'llm',
    status: 'OPEN',
    evidence: [
      { position: 1, metric: 'DAILY_CONSUMPTION', observed: 852.4, baseline: 1349.1, delta_pct: -36.8, unit: 'kWh', window: '2026-09-08', detail: 'Total diario 36,8 % por debajo del baseline' },
      { position: 2, metric: 'PERSISTENCE_DAYS', observed: 1, baseline: null, delta_pct: null, unit: 'days', window: '2026-09-08', detail: 'Un solo día con desviación ≥ 20 %; el 2026-09-09 vuelve al baseline (−0,1 %)' },
      { position: 3, metric: 'CURRENT_A', observed: 161.6, baseline: 257.4, delta_pct: -37.2, unit: 'A', window: '2026-09-08', detail: 'Corriente media diaria 37,2 % por debajo del baseline; 12 horas entre 38,7 y 61,1 A' },
      { position: 4, metric: 'POWER_FACTOR', observed: 0.916, baseline: 0.921, delta_pct: -0.5, unit: '', window: '2026-09-08', detail: 'Factor de potencia sin cambio relevante (−0,005)' },
      { position: 5, metric: 'HOURLY_OUTLIERS', observed: 12, baseline: null, delta_pct: null, unit: 'readings', window: '2026-09-08', detail: '12 lecturas fuera del perfil horario (z ≥ 3), todas entre 00:00 y 11:00' },
      { position: 6, metric: 'RELATED_EVENT', observed: null, baseline: null, delta_pct: null, unit: '', window: '2026-09-08', detail: 'Evento SCHEDULED_OUTAGE 2026-09-08 00:00: Scheduled maintenance outage for 12 hours' },
    ],
  },
];
