import type { ReactNode } from 'react';

interface Props {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  valueColor?: string;
  /** Muestra el valor en 28 px (default) o 20 px para textos largos. */
  compact?: boolean;
}

export function KpiTile({ label, value, sub, valueColor, compact = false }: Props) {
  return (
    <div className="panel flex min-h-[104px] flex-col justify-between px-4 py-3">
      <div className="text-xs text-ink-2">{label}</div>
      <div>
        <div className={`${compact ? 'text-lg' : 'text-xl'} font-semibold leading-tight text-ink`} style={valueColor ? { color: valueColor } : undefined}>
          {value}
        </div>
        {sub !== undefined && <div className="mt-1 text-xs text-ink-2">{sub}</div>}
      </div>
    </div>
  );
}
