// RF-F-06 — Detalle de medidor.
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight } from 'lucide-react';
import { getMeter, getReadings } from '@/api/meters';
import { isApiError } from '@/api/client';
import { AppLayout } from '@/app/layout/AppLayout';
import { SeverityBadge, StatusBadge } from '@/components/Badge';
import { KpiTile } from '@/components/KpiTile';
import { Panel } from '@/components/Panel';
import { Segmented } from '@/components/Segmented';
import { EmptyState, ErrorState, Skeleton } from '@/components/States';
import { DailyBarsChart } from '@/components/charts/DailyBarsChart';
import { ElectricalChart } from '@/components/charts/ElectricalChart';
import { HourlyConsumptionChart } from '@/components/charts/HourlyConsumptionChart';
import { variationColor } from '@/lib/colors';
import { formatConfidence, formatDateTime, formatDayShort, formatKwh, formatPct } from '@/lib/format';
import { typeLabel } from '@/lib/labels';
import type { Granularity } from '@/types/api';

const GRANULARITY = [
  { value: 'hour' as Granularity, label: 'Hora' },
  { value: 'day' as Granularity, label: 'Día' },
];

export function MeterDetailPage() {
  const { meterId = '' } = useParams();
  const [granularity, setGranularity] = useState<Granularity>('hour');
  const meter = useQuery({ queryKey: ['meter', meterId], queryFn: () => getMeter(meterId), enabled: Boolean(meterId) });
  const hourly = useQuery({ queryKey: ['readings', meterId, 'hour'], queryFn: () => getReadings(meterId, 'hour'), enabled: Boolean(meterId), staleTime: 5 * 60_000 });
  const daily = useQuery({ queryKey: ['readings', meterId, 'day'], queryFn: () => getReadings(meterId, 'day'), enabled: Boolean(meterId) && granularity === 'day', staleTime: 5 * 60_000 });

  const title = `Medidor ${meterId}`;
  const back = (
    <Link to="/meters" className="mb-4 inline-flex items-center gap-1.5 text-sm text-ink-2 hover:text-ink">
      <ArrowLeft size={14} strokeWidth={1.5} aria-hidden="true" /> Medidores
    </Link>
  );

  if (meter.isError) {
    const notFound = isApiError(meter.error) && meter.error.status === 404;
    return (
      <AppLayout title={title}>
        {back}
        {notFound ? (
          <EmptyState text={`El medidor ${meterId} no existe`} actionLabel="Volver a medidores" onAction={() => window.history.back()} />
        ) : (
          <ErrorState error={meter.error} text="No se pudo cargar el medidor" onRetry={() => void meter.refetch()} />
        )}
      </AppLayout>
    );
  }

  if (meter.isPending) {
    return (
      <AppLayout title={title}>
        {back}
        <div className="flex flex-col gap-4" aria-busy="true">
          <div className="grid grid-cols-4 gap-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-[104px]" />
            ))}
          </div>
          <Skeleton className="h-[380px]" />
          <div className="grid grid-cols-3 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-[240px]" />
            ))}
          </div>
        </div>
      </AppLayout>
    );
  }

  const m = meter.data;
  const readings = hourly.data?.items ?? [];
  const noBaseline = hourly.data ? hourly.data.baseline_profile.length === 0 : false;

  return (
    <AppLayout title={title}>
      {back}
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-4 gap-4">
          <KpiTile label="Consumo actual" value={`${formatKwh(m.current_consumption_kwh)} kWh`} sub={`último día (${formatDayShort(m.period.end)})`} />
          <KpiTile label="Baseline" value={`${formatKwh(m.baseline_daily_kwh)} kWh/día`} sub="promedio de los primeros 7 días" />
          <KpiTile label="Variación" value={formatPct(m.variation_pct)} valueColor={variationColor(m.variation_pct)} sub="vs. baseline diario" />
          <KpiTile label="Estado" value={<StatusBadge status={m.status} />} sub={m.location} />
        </div>

        <Panel
          title={
            <span className="flex items-center gap-3">
              Consumo
              {noBaseline && <span className="normal-case tracking-normal text-amber">Sin baseline: datos insuficientes</span>}
            </span>
          }
          actions={<Segmented options={GRANULARITY} value={granularity} onChange={setGranularity} ariaLabel="Granularidad" />}
          bodyClassName="p-4"
        >
          {granularity === 'hour' ? (
            hourly.isPending ? (
              <Skeleton className="h-[320px]" />
            ) : hourly.isError ? (
              <ErrorState error={hourly.error} text="No se pudieron cargar las lecturas" onRetry={() => void hourly.refetch()} />
            ) : (
              <HourlyConsumptionChart items={readings} baselineProfile={hourly.data.baseline_profile} events={m.events} meterId={m.meter_id} />
            )
          ) : daily.isPending ? (
            <Skeleton className="h-[320px]" />
          ) : daily.isError ? (
            <ErrorState error={daily.error} text="No se pudieron cargar las lecturas" onRetry={() => void daily.refetch()} />
          ) : (
            <DailyBarsChart data={daily.data.items} baselineDailyKwh={m.baseline_daily_kwh} events={m.events} ariaLabel={`Consumo diario de ${m.meter_id} con línea de baseline`} />
          )}
        </Panel>

        <div className="grid grid-cols-3 gap-4">
          {(
            [
              { key: 'voltage_v', title: 'Voltaje (V)', limits: [209, 231] },
              { key: 'current_a', title: 'Corriente (A)', limits: [] },
              { key: 'power_factor', title: 'Factor de potencia', limits: [] },
            ] as const
          ).map((c) => (
            <Panel key={c.key} title={c.title} titleAs="h3" bodyClassName="p-3">
              {hourly.isPending ? (
                <Skeleton className="h-[180px]" />
              ) : hourly.isError ? (
                <ErrorState error={hourly.error} text="No se pudieron cargar las lecturas" className="py-4" />
              ) : (
                <ElectricalChart items={readings} metric={c.key} baselineAvg={m.electrical[c.key].baseline_avg} limits={[...c.limits]} meterId={m.meter_id} />
              )}
            </Panel>
          ))}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Panel title="Anomalía vigente" bodyClassName="p-4">
            {m.anomaly ? (
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3 text-sm">
                  <span className="font-semibold text-ink">{typeLabel(m.anomaly.type)}</span>
                  <SeverityBadge severity={m.anomaly.severity} />
                  <span className="tnum text-ink-2">Confianza {formatConfidence(m.anomaly.confidence)}</span>
                </div>
                <Link to={`/anomalies/${m.anomaly.id}`} className="btn-secondary h-9">
                  Ver investigación <ArrowRight size={14} strokeWidth={1.5} aria-hidden="true" />
                </Link>
              </div>
            ) : (
              <p className="text-sm text-ink-2">Sin anomalía vigente</p>
            )}
          </Panel>
          <Panel title="Eventos del medidor" bodyClassName="p-4">
            {m.events.length === 0 ? (
              <p className="text-sm text-ink-2">Sin eventos registrados</p>
            ) : (
              <ul className="flex flex-col gap-2">
                {m.events.map((e) => (
                  <li key={e.id} className="text-sm">
                    <span className="tnum text-ink-2">{formatDateTime(e.timestamp)}</span>
                    <span className="mx-2 text-ink-3">·</span>
                    <span className="font-medium text-amber">{e.type}</span>
                    <div className="text-ink-2">{e.description}</div>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
        </div>
      </div>
    </AppLayout>
  );
}
