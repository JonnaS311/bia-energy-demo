import { useMemo } from 'react';
import { Area, CartesianGrid, ComposedChart, Line, ReferenceLine, ResponsiveContainer, Scatter, Tooltip, XAxis, YAxis } from 'recharts';
import { formatDayShort, formatHour, formatKwh } from '@/lib/format';
import type { BaselineProfilePoint, HourlyReading, MeterEvent } from '@/types/api';
import { axisProps, chart, ChartEmpty, ChartTooltip } from './chartTheme';

interface Props {
  items: HourlyReading[];
  baselineProfile: BaselineProfilePoint[];
  events: MeterEvent[];
  meterId: string;
  height?: number;
}

interface Point {
  ts: number;
  timestamp: string;
  kwh: number;
  bandLow: number | null;
  bandHigh: number | null;
  outlier: number | null;
}

/** Línea horaria con banda baseline (media ± 2σ), outliers en rojo y marcadores de eventos (RF-F-06). */
export function HourlyConsumptionChart({ items, baselineProfile, events, meterId, height = chart.MAIN_HEIGHT }: Props) {
  const data = useMemo<Point[]>(() => {
    const byHour = new Map(baselineProfile.map((p) => [p.hour, p]));
    return items.map((r) => {
      const d = new Date(r.timestamp);
      const p = byHour.get(d.getUTCHours());
      return {
        ts: d.getTime(),
        timestamp: r.timestamp,
        kwh: r.consumption_kwh,
        bandLow: p ? Math.max(0, p.mean_kwh - 2 * p.std_kwh) : null,
        bandHigh: p ? p.mean_kwh + 2 * p.std_kwh : null,
        outlier: r.is_outlier ? r.consumption_kwh : null,
      };
    });
  }, [items, baselineProfile]);

  if (!items.length) return <ChartEmpty height={height} />;

  const hasBand = baselineProfile.length > 0;
  const dayTicks = data.filter((p) => new Date(p.ts).getUTCHours() === 0).map((p) => p.ts);

  return (
    <div role="img" aria-label={`Consumo horario de ${meterId} con banda baseline`} style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={data} margin={{ top: 12, right: 12, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} stroke={chart.grid} />
          <XAxis dataKey="ts" type="number" domain={['dataMin', 'dataMax']} ticks={dayTicks} tickFormatter={(v: number) => formatDayShort(new Date(v).toISOString())} {...axisProps} />
          <YAxis width={40} {...axisProps} axisLine={false} />
          <Tooltip
            cursor={{ stroke: chart.axis, strokeWidth: 1 }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              const p = payload[0].payload as Point;
              const rows: { label: string; value: string; color?: string }[] = [{ label: 'Consumo', value: `${formatKwh(p.kwh)} kWh`, color: chart.series }];
              if (p.bandLow !== null && p.bandHigh !== null) rows.push({ label: 'Banda baseline', value: `${formatKwh(p.bandLow)} – ${formatKwh(p.bandHigh)}`, color: '#cbd5e1' });
              if (p.outlier !== null) rows.push({ label: 'Fuera de patrón', value: 'z ≥ 3', color: chart.outlier });
              return <ChartTooltip title={formatHour(p.timestamp)} rows={rows} />;
            }}
          />
          {hasBand && <Area dataKey="bandHigh" stroke="none" fill={chart.band} isAnimationActive={false} dot={false} activeDot={false} />}
          {hasBand && <Area dataKey="bandLow" stroke="none" fill={chart.surface} isAnimationActive={false} dot={false} activeDot={false} />}
          {events.map((e) => (
            <ReferenceLine
              key={e.id}
              x={new Date(e.timestamp).getTime()}
              stroke={chart.event}
              strokeDasharray="4 3"
              label={{ value: e.type, position: 'insideTopRight', fill: chart.event, fontSize: 11 }}
            />
          ))}
          <Line dataKey="kwh" stroke={chart.series} strokeWidth={2} dot={false} activeDot={{ r: 4, stroke: chart.surface, strokeWidth: 2 }} isAnimationActive={false} />
          <Scatter dataKey="outlier" fill={chart.outlier} shape={(props: { cx?: number; cy?: number }) => (props.cy == null ? <g /> : <circle cx={props.cx} cy={props.cy} r={4} fill={chart.outlier} stroke={chart.surface} strokeWidth={2} />)} isAnimationActive={false} />
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
