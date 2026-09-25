import { useMemo } from 'react';
import { Bar, BarChart, CartesianGrid, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { formatDate, formatDayShort, formatKwh, formatPct } from '@/lib/format';
import { axisProps, chart, ChartEmpty, ChartTooltip, compactTick, niceScale } from './chartTheme';

interface Props {
  data: { date: string; consumption_kwh: number; deviation_pct: number | null }[];
  baselineDailyKwh: number | null;
  /** Ventana sombreada [start, end] inclusive (investigación, RF-F-08). */
  window?: { start: string; end: string } | null;
  events?: { id: number; timestamp: string; type: string; description: string }[];
  ariaLabel: string;
  height?: number;
}

/** Barras diarias con línea de baseline, ventana sombreada y marcadores de eventos (RF-F-06 día, RF-F-08). */
export function DailyBarsChart({ data, baselineDailyKwh, window, events = [], ariaLabel, height = chart.MAIN_HEIGHT }: Props) {
  const scale = useMemo(() => niceScale(Math.max(0, baselineDailyKwh ?? 0, ...data.map((d) => d.consumption_kwh))), [data, baselineDailyKwh]);
  if (!data.length) return <ChartEmpty height={height} />;
  return (
    <div>
      <div className="mb-2 flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-ink-2" aria-hidden="true">
        {baselineDailyKwh !== null && (
          <span className="inline-flex items-center gap-2">
            <span className="inline-block h-0 w-5 border-t border-dashed" style={{ borderColor: chart.reference }} />
            Baseline diario {formatKwh(baselineDailyKwh)} kWh
          </span>
        )}
        {window && (
          <span className="inline-flex items-center gap-2">
            <span className="inline-block h-3 w-5 rounded-sm" style={{ backgroundColor: 'rgba(239, 68, 68, 0.28)' }} />
            Ventana de la anomalía {formatDayShort(window.start)} – {formatDayShort(window.end)}
          </span>
        )}
        {events.length > 0 && (
          <span className="inline-flex items-center gap-2">
            <span className="inline-block h-3 w-0 border-l border-dashed" style={{ borderColor: chart.event }} />
            Evento operativo
          </span>
        )}
      </div>
      <div role="img" aria-label={ariaLabel} style={{ height }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 12, right: 12, left: 0, bottom: 0 }} barCategoryGap="30%">
            <CartesianGrid vertical={false} stroke={chart.grid} />
            <XAxis dataKey="date" tickFormatter={formatDayShort} {...axisProps} interval={0} />
            <YAxis domain={scale.domain} ticks={scale.ticks} tickFormatter={compactTick} width={44} {...axisProps} axisLine={false} />
            <Tooltip
              cursor={{ fill: 'rgba(255,255,255,0.04)' }}
              content={({ active, payload }) => {
                if (!active || !payload?.length) return null;
                const p = payload[0].payload as Props['data'][number];
                return (
                  <ChartTooltip
                    title={formatDate(p.date)}
                    rows={[
                      { label: 'Consumo', value: `${formatKwh(p.consumption_kwh)} kWh`, color: chart.series },
                      { label: 'Desviación', value: formatPct(p.deviation_pct) },
                    ]}
                  />
                );
              }}
            />
            {window && <ReferenceArea x1={window.start} x2={window.end} fill={chart.window} stroke="none" ifOverflow="extendDomain" />}
            {baselineDailyKwh !== null && <ReferenceLine y={baselineDailyKwh} stroke={chart.reference} strokeDasharray="4 3" />}
            {events.map((e) => (
              <ReferenceLine key={e.id} x={e.timestamp.slice(0, 10)} stroke={chart.event} strokeDasharray="4 3" />
            ))}
            <Bar dataKey="consumption_kwh" fill={chart.series} radius={[4, 4, 0, 0]} maxBarSize={24} isAnimationActive={false} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
