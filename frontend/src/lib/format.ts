// RF-F-12 — formato es-CO: miles ".", decimal ",", fechas en UTC sin conversión de zona.

const MONTHS = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];
const MINUS = '−';

function groupThousands(intPart: string): string {
  return intPart.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
}

export function formatNumber(n: number, decimals: number): string {
  const negative = n < 0;
  const fixed = Math.abs(n).toFixed(decimals);
  const [intPart, decPart] = fixed.split('.');
  const grouped = groupThousands(intPart);
  const body = decPart !== undefined ? `${grouped},${decPart}` : grouped;
  return negative ? `${MINUS}${body}` : body;
}

export function formatKwh(n: number): string {
  return formatNumber(n, 1);
}

export function formatPct(n: number | null): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—';
  const rounded = Math.round(n * 10) / 10;
  const sign = rounded < 0 ? MINUS : '+';
  return `${sign}${formatNumber(Math.abs(rounded), 1)} %`;
}

export function formatPctInt(fraction: number | null): string {
  if (fraction === null || fraction === undefined) return '—';
  return `${Math.round(fraction * 100)} %`;
}

export function formatConfidence(n: number): string {
  return formatNumber(n, 2);
}

export function formatInt(n: number): string {
  return formatNumber(n, 0);
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

function toDate(iso: string): Date {
  // Fechas 'YYYY-MM-DD' se interpretan como UTC medianoche.
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) return new Date(`${iso}T00:00:00Z`);
  return new Date(iso);
}

export function formatDateTime(iso: string): string {
  const d = toDate(iso);
  return `${pad(d.getUTCDate())} ${MONTHS[d.getUTCMonth()]} ${d.getUTCFullYear()} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

export function formatDate(iso: string): string {
  const d = toDate(iso);
  return `${pad(d.getUTCDate())} ${MONTHS[d.getUTCMonth()]} ${d.getUTCFullYear()}`;
}

export function formatDayShort(iso: string): string {
  const d = toDate(iso);
  return `${pad(d.getUTCDate())} ${MONTHS[d.getUTCMonth()]}`;
}

export function formatHour(iso: string): string {
  const d = toDate(iso);
  return `${pad(d.getUTCDate())} ${MONTHS[d.getUTCMonth()]} ${pad(d.getUTCHours())}:00`;
}

export function formatRelative(iso: string, now: Date = new Date()): string {
  const diff = Math.max(0, Math.floor((now.getTime() - toDate(iso).getTime()) / 1000));
  if (diff < 60) return `hace ${diff} s`;
  const min = Math.floor(diff / 60);
  if (min < 60) return `hace ${min} min`;
  const h = Math.floor(min / 60);
  if (h < 24) return `hace ${h} h`;
  return `hace ${Math.floor(h / 24)} d`;
}

/** Formato de un valor de evidencia según su unidad (RF-F-08). */
export function formatWithUnit(value: number | null, unit: string): string {
  if (value === null || value === undefined) return '—';
  switch (unit) {
    case 'kWh':
      return `${formatNumber(value, 1)} kWh`;
    case 'V':
      return `${formatNumber(value, 1)} V`;
    case 'A':
      return `${formatNumber(value, 1)} A`;
    case 'ratio':
      return formatNumber(value, 2);
    case 'days':
      return `${formatInt(value)} d`;
    case 'readings':
      return formatInt(value);
    case '':
      return Number.isInteger(value) ? formatInt(value) : formatNumber(value, 3);
    default:
      return `${formatNumber(value, 1)} ${unit}`;
  }
}
