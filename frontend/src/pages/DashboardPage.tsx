// RF-F-04 — Dashboard: 6 KPI, estado de la red (consola), consumo diario total, top anomalías.
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { ArrowRight } from 'lucide-react';
import { getDashboardSummary } from '@/api/dashboard';
import { listMeters } from '@/api/meters';
import { AppLayout } from '@/app/layout/AppLayout';
import { SeverityBadge } from '@/components/Badge';
import { KpiTile } from '@/components/KpiTile';
import { NetworkStrip } from '@/components/NetworkStrip';
import { Panel } from '@/components/Panel';
import { EmptyState, ErrorState, Skeleton } from '@/components/States';
import { DailyConsumptionChart } from '@/components/charts/DailyConsumptionChart';
import { useAnalysis } from '@/hooks/useAnalysis';
import { STROKE, palette } from '@/lib/colors';
import { formatDateTime, formatInt, formatKwh, formatPctInt } from '@/lib/format';
import { analysisStatusLabel, confidenceLabel, typeLabel } from '@/lib/labels';
import type { DashboardSummary } from '@/types/api';

function lastAnalysisTile(summary: DashboardSummary) {
  const last = summary.last_analysis;
  if (!last) return { value: '—', sub: 'Sin análisis' };
  const status = last.status === 'QUEUED' || last.status === 'RUNNING' ? 'En curso' : analysisStatusLabel(last.status);
  return { value: last.finished_at ? formatDateTime(last.finished_at) : '—', sub: status };
}

export function DashboardPage() {
  const summary = useQuery({ queryKey: ['dashboard'], queryFn: getDashboardSummary });
  const meters = useQuery({ queryKey: ['meters', { sort: 'severity', order: 'desc' }], queryFn: () => listMeters({ sort: 'severity', order: 'desc' }) });
  const { lastCompletedAt } = useAnalysis();

  return (
    <AppLayout title="Dashboard">
      {summary.isPending ? (
        <div className="flex flex-col gap-4" aria-busy="true">
          <div className="grid grid-cols-6 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-[104px]" />
            ))}
          </div>
          <Skeleton className="h-[92px]" />
          <div className="grid grid-cols-3 gap-4">
            <Skeleton className="col-span-2 h-[380px]" />
            <Skeleton className="h-[380px]" />
          </div>
        </div>
      ) : summary.isError ? (
        <ErrorState error={summary.error} text="No se pudo cargar el dashboard" onRetry={() => void summary.refetch()} />
      ) : (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-6 gap-4">
            <KpiTile label="Medidores" value={formatInt(summary.data.meters_total)} />
            <KpiTile label="Consumo" value={formatKwh(summary.data.total_consumption_kwh)} sub="kWh en el periodo" />
            <KpiTile label="Anomalías IA" value={formatInt(summary.data.anomalies_total)} />
            <KpiTile label="Alta prioridad" value={formatInt(summary.data.high_priority_total)} valueColor={summary.data.high_priority_total > 0 ? palette.red : undefined} />
            <KpiTile label="Confianza IA" value={formatPctInt(summary.data.ai_confidence_avg)} />
            <KpiTile label="Último análisis" compact {...lastAnalysisTile(summary.data)} />
          </div>

          <Panel
            title="Estado de la red"
            actions={
              <span className="text-xs text-ink-2">
                {summary.data.meters_by_status.CRITICAL} críticos · {summary.data.meters_by_status.ALERT} en alerta · {summary.data.meters_by_status.OK} normales
              </span>
            }
            bodyClassName="p-3"
          >
            {meters.isError ? (
              <ErrorState error={meters.error} text="No se pudieron cargar los medidores" onRetry={() => void meters.refetch()} className="py-4" />
            ) : (
              <NetworkStrip meters={meters.data?.items} isLoading={meters.isPending} igniteKey={lastCompletedAt} />
            )}
          </Panel>

          <div className="grid grid-cols-3 gap-4">
            <Panel title="Consumo diario total (kWh)" className="col-span-2" bodyClassName="p-4">
              <DailyConsumptionChart data={summary.data.daily_consumption} />
            </Panel>
            <Panel
              title="Top anomalías"
              actions={
                summary.data.anomalies_total > 0 && (
                  <Link to="/anomalies" className="inline-flex items-center gap-1 text-xs font-semibold text-mint hover:text-mint-bright">
                    Ver todas las anomalías <ArrowRight size={12} strokeWidth={STROKE} aria-hidden="true" />
                  </Link>
                )
              }
              bodyClassName="flex flex-col"
            >
              {summary.data.top_anomalies.length === 0 ? (
                <EmptyState text="Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías." className="flex-1" />
              ) : (
                <table className="table-base w-full table-fixed [&_td]:px-2 [&_th]:px-2 [&_th]:text-[11px] [&_th]:tracking-[0.03em] [&_td:first-child]:pl-4 [&_th:first-child]:pl-4">
                  <colgroup>
                    <col className="w-8" />
                    <col className="w-[64px]" />
                    <col />
                    <col className="w-[78px]" />
                    <col className="w-[82px]" />
                  </colgroup>
                  <thead>
                    <tr>
                      <th scope="col">#</th>
                      <th scope="col">Medidor</th>
                      <th scope="col">Tipo</th>
                      <th scope="col">Severidad</th>
                      <th scope="col">Confianza</th>
                    </tr>
                  </thead>
                  <tbody>
                    {summary.data.top_anomalies.map((a, i) => (
                      <tr key={a.id} className="hover:bg-hover">
                        <td className="tnum text-ink-2">{i + 1}</td>
                        <td>
                          <Link to={`/anomalies/${a.id}`} className="font-semibold text-ink hover:text-mint">
                            {a.meter_id}
                          </Link>
                        </td>
                        <td className="leading-tight text-ink-2">{typeLabel(a.type)}</td>
                        <td>
                          <SeverityBadge severity={a.severity} />
                        </td>
                        <td className="text-ink-2">{confidenceLabel(a.confidence)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </Panel>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
