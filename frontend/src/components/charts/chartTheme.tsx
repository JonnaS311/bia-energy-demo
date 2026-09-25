// Reglas comunes de gráficas (RF-F-10 + dataviz): marcas finas, rejilla hairline recesiva, tooltip temado.
import type { ReactNode } from 'react';
import { palette } from '@/lib/colors';

export const chart = {
  series: palette.mint,
  seriesDim: 'rgba(8, 221, 188, 0.45)',
  grid: '#1f1f1f',
  axis: '#71717a',
  tick: { fill: '#a1a1aa', fontSize: 12 },
  reference: '#a1a1aa',
  event: palette.amber,
  outlier: palette.red,
  band: 'rgba(203, 213, 225, 0.16)',
  window: 'rgba(239, 68, 68, 0.14)',
  surface: palette.panel,
  MAIN_HEIGHT: 320,
  SMALL_HEIGHT: 180,
} as const;

export const axisProps = {
  tick: chart.tick,
  axisLine: { stroke: chart.grid },
  tickLine: false as const,
};

export function ChartTooltip({ title, rows }: { title: ReactNode; rows: { label: string; value: ReactNode; color?: string }[] }) {
  return (
    <div className="rounded-md border border-line bg-panel px-3 py-2 text-xs shadow-panel">
      <div className="mb-1 font-medium text-ink">{title}</div>
      {rows.map((r) => (
        <div key={r.label} className="flex items-center justify-between gap-4 text-ink-2">
          <span className="flex items-center gap-1.5">
            {r.color && <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: r.color }} aria-hidden="true" />}
            {r.label}
          </span>
          <span className="tnum text-ink">{r.value}</span>
        </div>
      ))}
    </div>
  );
}

/** Dominio [0, máximo redondeado] y ticks limpios (0 / 5.000 / 10.000…) para ejes de magnitud. */
export function niceScale(max: number, tickCount = 4): { domain: [number, number]; ticks: number[] } {
  if (!Number.isFinite(max) || max <= 0) return { domain: [0, 1], ticks: [0, 1] };
  const raw = max / tickCount;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const candidates = [1, 2, 2.5, 5, 10].map((m) => m * mag);
  const step = candidates.find((c) => c >= raw) ?? candidates[candidates.length - 1];
  const top = Math.ceil(max / step) * step;
  const ticks: number[] = [];
  for (let v = 0; v <= top + 1e-9; v += step) ticks.push(Math.round(v * 1000) / 1000);
  return { domain: [0, top], ticks };
}

export const compactTick = (v: number): string => (v >= 1000 ? `${Number((v / 1000).toFixed(1))}k` : String(v));

export function ChartEmpty({ height }: { height: number }) {
  return (
    <div className="flex items-center justify-center text-sm text-ink-2" style={{ height }}>
      Sin datos para mostrar
    </div>
  );
}
