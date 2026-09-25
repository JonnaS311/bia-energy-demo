import { useMemo } from 'react';
import { CartesianGrid, Line, LineChart, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { formatDayShort, formatHour, formatNumber } from '@/lib/format';
import type { HourlyReading } from '@/types/api';
import { axisProps, chart, ChartEmpty, ChartTooltip } from './chartTheme';

interface Props {
  items: HourlyReading[];
  metric: 'voltage_v' | 'current_a' | 'power_factor';
  baselineAvg: number;
  meterId: string;
  /** Líneas de referencia adicionales (banda de voltaje 209–231). */
  limits?: number[];
  height?: number;
}

const META = {
  voltage_v: { label: 'Voltaje', unit: 'V', decimals: 1 },
  current_a: { label: 'Corriente', unit: 'A', decimals: 1 },
  power_factor: { label: 'Factor de potencia', unit: '', decimals: 3 },
} as const;

/** Gráfica pequeña de V / I / PF con línea de baseline (RF-F-06). */
export function ElectricalChart({ items, metric, baselineAvg, meterId, limits = [], height = chart.SMALL_HEIGHT }: Props) {
  const meta = META[metric];
  const data = useMemo(() => items.map((r) => ({ ts: new Date(r.timestamp).getTime(), timestamp: r.timestamp, v: r[metric] })), [items, metric]);
  if (!items.length) return <ChartEmpty height={height} />;
  // Un tick cada 4 días desde el primero (01 / 05 / 09 / 13 sep): sin colisiones a 230 px de ancho.
  const firstDay = data.length ? Date.UTC(new Date(data[0].ts).getUTCFullYear(), new Date(data[0].ts).getUTCMonth(), new Date(data[0].ts).getUTCDate()) : 0;
  const dayTicks = data.filter((p) => new Date(p.ts).getUTCHours() === 0 && Math.round((p.ts - firstDay) / 86_400_000) % 4 === 0).map((p) => p.ts);
  const fmt = (v: number) => `${formatNumber(v, meta.decimals)}${meta.unit ? ` ${meta.unit}` : ''}`;
  return (
    <div role="img" aria-label={`${meta.label} horario de ${meterId} con referencia de baseline`} style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} stroke={chart.grid} />
          <XAxis dataKey="ts" type="number" domain={['dataMin', 'dataMax']} ticks={dayTicks} tickFormatter={(v: number) => formatDayShort(new Date(v).toISOString())} {...axisProps} />
          <YAxis width={44} domain={['auto', 'auto']} tickFormatter={(v: number) => formatNumber(v, metric === 'power_factor' ? 2 : 0)} {...axisProps} axisLine={false} />
          <Tooltip
            cursor={{ stroke: chart.axis, strokeWidth: 1 }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              const p = payload[0].payload as (typeof data)[number];
              return <ChartTooltip title={formatHour(p.timestamp)} rows={[{ label: meta.label, value: fmt(p.v), color: chart.series }, { label: 'Baseline', value: fmt(baselineAvg) }]} />;
            }}
          />
          <ReferenceLine y={baselineAvg} stroke={chart.reference} strokeDasharray="4 3" />
          {limits.map((l) => (
            <ReferenceLine key={l} y={l} stroke="rgba(239, 68, 68, 0.5)" strokeDasharray="2 3" />
          ))}
          <Line dataKey="v" stroke={chart.series} strokeWidth={1.5} dot={false} activeDot={{ r: 4, stroke: chart.surface, strokeWidth: 2 }} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
