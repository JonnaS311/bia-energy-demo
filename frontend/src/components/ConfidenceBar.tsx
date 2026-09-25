import { palette } from '@/lib/colors';
import { formatPctInt } from '@/lib/format';
import { confidenceLabel } from '@/lib/labels';

interface Props {
  value: number;
  width?: number;
  showLabel?: boolean;
}

/** Porcentaje entero + barra cuyo relleno es confidence × 100 % (RF-F-07). */
export function ConfidenceBar({ value, width = 80, showLabel = false }: Props) {
  const pct = Math.round(value * 100);
  return (
    <span className="inline-flex items-center gap-2" title={confidenceLabel(value)}>
      <span className="tnum text-sm text-ink">{formatPctInt(value)}</span>
      <span
        className="relative inline-block h-1.5 overflow-hidden rounded-full"
        style={{ width, backgroundColor: 'rgba(8, 221, 188, 0.18)' }}
        role="progressbar"
        aria-valuenow={pct}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`Confianza ${pct} %`}
      >
        <span className="absolute inset-y-0 left-0 rounded-full" style={{ width: `${pct}%`, backgroundColor: palette.mint }} />
      </span>
      {showLabel && <span className="text-xs text-ink-2">({confidenceLabel(value)})</span>}
    </span>
  );
}
