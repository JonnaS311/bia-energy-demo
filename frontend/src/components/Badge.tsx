// Badges de estado, severidad, tipo y estado de anomalía: 12 px, radio 4 px, fondo del color al 15 %,
// glifo además de color (el estado es una marca, no solo un tono).
import { AlertTriangle, Ban, Check, CircleDot, Database, Octagon, Search, ShieldCheck, type LucideIcon } from 'lucide-react';
import { STROKE, anomalyStatusColor, anomalyTypeColor, meterStatusColor, severityColor, tint } from '@/lib/colors';
import { anomalyStatusLabel, severityLabel, statusLabel, typeLabel } from '@/lib/labels';
import type { AnomalyStatus, AnomalyType, MeterStatus, Severity } from '@/types/api';

interface BaseProps {
  color: string;
  label: string;
  icon?: LucideIcon;
  title?: string;
  className?: string;
}

export function Badge({ color, label, icon: Icon, title, className = '' }: BaseProps) {
  return (
    <span
      title={title}
      className={`inline-flex h-5 items-center gap-1 whitespace-nowrap rounded-badge px-1.5 text-xs font-semibold leading-none ${className}`}
      style={{ color, backgroundColor: tint(color) }}
    >
      {Icon && <Icon size={12} strokeWidth={STROKE} aria-hidden="true" />}
      {label}
    </span>
  );
}

const METER_ICON: Record<MeterStatus, LucideIcon> = { OK: Check, ALERT: AlertTriangle, CRITICAL: Octagon };
export function StatusBadge({ status }: { status: MeterStatus }) {
  return <Badge color={meterStatusColor[status]} label={statusLabel(status)} icon={METER_ICON[status]} />;
}

export function SeverityBadge({ severity }: { severity: Severity }) {
  return <Badge color={severityColor[severity]} label={severityLabel(severity)} />;
}

const TYPE_ICON: Record<AnomalyType, LucideIcon> = {
  REAL_ANOMALY: Octagon,
  EXPLAINABLE_ANOMALY: AlertTriangle,
  FALSE_POSITIVE: ShieldCheck,
  DATA_QUALITY: Database,
};
export function TypeBadge({ type }: { type: AnomalyType }) {
  return <Badge color={anomalyTypeColor[type]} label={typeLabel(type)} icon={TYPE_ICON[type]} />;
}

const ANOMALY_STATUS_ICON: Record<AnomalyStatus, LucideIcon> = { OPEN: CircleDot, INVESTIGATING: Search, RESOLVED: Check, DISMISSED: Ban };
export function AnomalyStatusBadge({ status }: { status: AnomalyStatus }) {
  return <Badge color={anomalyStatusColor[status]} label={anomalyStatusLabel(status)} icon={ANOMALY_STATUS_ICON[status]} />;
}
