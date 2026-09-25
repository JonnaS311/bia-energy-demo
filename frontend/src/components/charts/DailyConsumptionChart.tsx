import { useMemo } from 'react';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { formatDate, formatDayShort, formatKwh } from '@/lib/format';
import { axisProps, chart, ChartEmpty, ChartTooltip, compactTick, niceScale } from './chartTheme';

interface Props {
  data: { date: string; consumption_kwh: number }[];
  ariaLabel?: string;
  height?: number;
}

/** Barras del consumo diario total (dashboard, RF-F-04). Una serie: sin leyenda. */
export function DailyConsumptionChart({ data, ariaLabel = 'Consumo diario total en kWh', height = chart.MAIN_HEIGHT }: Props) {
  const scale = useMemo(() => niceScale(Math.max(0, ...data.map((d) => d.consumption_kwh))), [data]);
  if (!data.length) return <ChartEmpty height={height} />;
  return (
    <div role="img" aria-label={ariaLabel} style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 12, right: 12, left: 0, bottom: 0 }} barCategoryGap="30%">
          <CartesianGrid vertical={false} stroke={chart.grid} />
          <XAxis dataKey="date" tickFormatter={formatDayShort} {...axisProps} interval={0} minTickGap={8} />
          <YAxis domain={scale.domain} ticks={scale.ticks} tickFormatter={compactTick} width={44} {...axisProps} axisLine={false} />
          <Tooltip
            cursor={{ fill: 'rgba(255,255,255,0.04)' }}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              const p = payload[0].payload as Props['data'][number];
              return <ChartTooltip title={formatDate(p.date)} rows={[{ label: 'Consumo', value: `${formatKwh(p.consumption_kwh)} kWh`, color: chart.series }]} />;
            }}
          />
          <Bar dataKey="consumption_kwh" fill={chart.series} radius={[4, 4, 0, 0]} maxBarSize={24} isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
