// Tipos escritos a mano desde los contratos del maestro §8.2 (spec-energy-plataforma-maestro-v1.md).
// Los nombres de campo son exactos; si un campo no aparece en el maestro, no existe aquí.

export type MeterStatus = 'OK' | 'ALERT' | 'CRITICAL';
export type AnomalyType = 'REAL_ANOMALY' | 'EXPLAINABLE_ANOMALY' | 'FALSE_POSITIVE' | 'DATA_QUALITY';
export type Severity = 'HIGH' | 'MEDIUM' | 'LOW';
export type AnomalyStatus = 'OPEN' | 'INVESTIGATING' | 'RESOLVED' | 'DISMISSED';
export type AnalysisStatusValue = 'QUEUED' | 'RUNNING' | 'COMPLETED' | 'FAILED';
export type StepStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
export type ExplanationSource = 'llm' | 'template';
export type Granularity = 'hour' | 'day';

export type EvidenceMetric =
  | 'DAILY_CONSUMPTION'
  | 'PERSISTENCE_DAYS'
  | 'HOURLY_OUTLIERS'
  | 'CURRENT_A'
  | 'POWER_FACTOR'
  | 'VOLTAGE_OUT_OF_RANGE'
  | 'PHYSICAL_RATIO'
  | 'RELATED_EVENT'
  | 'INSUFFICIENT_DATA'
  | (string & Record<never, never>);

// 8.2 formato de error
export interface ApiErrorBody {
  error: { code: string; message: string; details: unknown };
}

// 8.2.1
export interface LoginResponse {
  token: string;
  expires_at: string;
  user: { email: string; name: string };
}

// 8.2.2
export interface AnomalyRef {
  id: string;
  type: AnomalyType;
  severity: Severity;
  confidence: number;
}

export interface MeterListItem {
  meter_id: string;
  name: string;
  location: string;
  status: MeterStatus;
  current_consumption_kwh: number;
  baseline_daily_kwh: number;
  variation_pct: number | null;
  period_consumption_kwh: number;
  anomaly: AnomalyRef | null;
}

export interface MeterListResponse {
  items: MeterListItem[];
  total: number;
}

export type MeterSort = 'consumption' | 'variation' | 'severity';
export type SortOrder = 'asc' | 'desc';

export interface MeterListQuery {
  status?: MeterStatus;
  q?: string;
  sort?: MeterSort;
  order?: SortOrder;
}

// 8.2.3
export interface ElectricalStat {
  current_avg: number;
  baseline_avg: number;
}

export interface MeterEvent {
  id: number;
  timestamp: string;
  type: string;
  description: string;
}

export interface MeterDaily {
  date: string;
  consumption_kwh: number;
  deviation_pct: number | null;
}

export interface MeterDetail extends MeterListItem {
  period: { start: string; end: string; readings_count: number };
  electrical: {
    voltage_v: ElectricalStat;
    current_a: ElectricalStat;
    power_factor: ElectricalStat;
  };
  events: MeterEvent[];
  daily: MeterDaily[];
}

// 8.2.4
export interface BaselineProfilePoint {
  hour: number;
  mean_kwh: number;
  std_kwh: number;
}

export interface HourlyReading {
  timestamp: string;
  consumption_kwh: number;
  voltage_v: number;
  current_a: number;
  power_factor: number;
  status: string;
  is_outlier: boolean;
}

export interface DailyReading {
  date: string;
  consumption_kwh: number;
  deviation_pct: number | null;
  voltage_avg: number;
  current_avg: number;
  power_factor_avg: number;
}

export interface HourlyReadingsResponse {
  meter_id: string;
  granularity: 'hour';
  baseline_profile: BaselineProfilePoint[];
  items: HourlyReading[];
}

export interface DailyReadingsResponse {
  meter_id: string;
  granularity: 'day';
  items: DailyReading[];
}

export type ReadingsResponse = HourlyReadingsResponse | DailyReadingsResponse;

// 8.2.5
export interface AnomalyListItem {
  id: string;
  analysis_id: string;
  superseded: boolean;
  meter_id: string;
  type: AnomalyType;
  severity: Severity;
  confidence: number;
  priority_rank: number;
  reason: string;
  recommended_action: string;
  status: AnomalyStatus;
  detected_at: string;
  window_start: string;
  window_end: string;
}

export interface AnomalyListResponse {
  analysis_id: string | null;
  items: AnomalyListItem[];
  total: number;
}

export interface AnomalyListQuery {
  type?: AnomalyType;
  severity?: Severity;
  status?: AnomalyStatus;
  include_superseded?: boolean;
}

// 8.2.6
export interface ComparisonEntry {
  observed: number;
  baseline: number;
  delta_pct: number | null;
}

export interface EvidenceItem {
  position: number;
  metric: EvidenceMetric;
  observed: number | null;
  baseline: number | null;
  delta_pct: number | null;
  unit: string;
  window: string;
  detail: string;
}

export interface AnomalyDetail extends AnomalyListItem {
  explanation_source: ExplanationSource;
  explanation_points: string[];
  related_event: MeterEvent | null;
  comparison: {
    consumption_kwh: ComparisonEntry;
    voltage_v: ComparisonEntry;
    current_a: ComparisonEntry;
    power_factor: ComparisonEntry;
  };
  evidence: EvidenceItem[];
}

// 8.2.7
export type StepKey =
  | 'READINGS'
  | 'BASELINE'
  | 'DETECTION'
  | 'CORRELATION'
  | 'EVENTS'
  | 'EXPLANATION'
  | 'RECOMMENDATION';

export interface AnalysisStep {
  key: StepKey;
  label: string;
  status: StepStatus;
  started_at: string | null;
  finished_at: string | null;
  detail: string | null;
}

export interface AnalysisSummary {
  anomalies_detected: number;
  high_priority: number;
  meters_analyzed: number;
  explanation_source: ExplanationSource;
}

export interface AnalysisStatus {
  id: string;
  status: AnalysisStatusValue;
  steps: AnalysisStep[];
  summary: AnalysisSummary | null;
  error_message: string | null;
  started_at: string | null;
  finished_at: string | null;
}

export interface AnalyzeResponse {
  analysis_id: string;
  status: AnalysisStatusValue;
  created_at: string;
}

// 8.2.8
export interface DashboardSummary {
  meters_total: number;
  total_consumption_kwh: number;
  period: { start: string; end: string };
  anomalies_total: number;
  high_priority_total: number;
  ai_confidence_avg: number | null;
  meters_by_status: Record<MeterStatus, number>;
  last_analysis: { id: string; status: AnalysisStatusValue; finished_at: string | null } | null;
  daily_consumption: { date: string; consumption_kwh: number }[];
  top_anomalies: (AnomalyRef & { meter_id: string })[];
}

// 8.2.10
export interface HealthResponse {
  status: string;
  db: string;
  ingest: {
    status: string;
    readings: number;
    events: number;
    meters: number;
    rejected?: number;
    duplicates?: number;
    duration_ms?: number;
  } | null;
}
