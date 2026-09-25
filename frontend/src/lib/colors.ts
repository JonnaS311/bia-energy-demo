// Fuente de verdad de color (paleta bia.app sobre fondo oscuro). tailwind.config.ts replica estos valores.
import type { AnomalyStatus, AnomalyType, MeterStatus, Severity } from '@/types/api';

export const palette = {
  app: '#0a0a0a',
  canvas: '#0f0f0f',
  panel: '#141414',
  raised: '#1a1a1a',
  hover: '#1f1f1f',
  line: '#27272a',
  lineStrong: '#3f3f46',
  ink: '#ffffff',
  ink2: '#a1a1aa',
  ink3: '#71717a',
  mint: '#08ddbc',
  mintBright: '#17ffdb',
  violet: '#8b5cf6',
  amber: '#f59e0b',
  red: '#ef4444',
  blue: '#3b82f6',
} as const;

/** Un solo trazo para toda la iconografía lucide (contrato OWN-WORLD). */
export const STROKE = 1.5;

export const meterStatusColor: Record<MeterStatus, string> = {
  OK: palette.mint,
  ALERT: palette.amber,
  CRITICAL: palette.red,
};

export const severityColor: Record<Severity, string> = {
  HIGH: palette.red,
  MEDIUM: palette.amber,
  LOW: palette.ink2,
};

export const anomalyTypeColor: Record<AnomalyType, string> = {
  REAL_ANOMALY: palette.red,
  EXPLAINABLE_ANOMALY: palette.amber,
  FALSE_POSITIVE: palette.ink2,
  DATA_QUALITY: palette.violet,
};

export const anomalyStatusColor: Record<AnomalyStatus, string> = {
  OPEN: palette.mint,
  INVESTIGATING: palette.blue,
  RESOLVED: palette.ink2,
  DISMISSED: palette.ink3,
};

/** Color del texto de una variación según su valor absoluto (RF-F-05). */
export function variationColor(pct: number | null): string {
  if (pct === null) return palette.ink2;
  const v = Math.abs(pct);
  if (v >= 20) return palette.red;
  if (v >= 10) return palette.amber;
  return palette.ink;
}

/** Fondo translúcido para badges: color al 15 %. */
export function tint(hex: string, alpha = 0.15): string {
  const n = parseInt(hex.slice(1), 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
