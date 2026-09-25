// RF-F-08 — Investigación: las 7 secciones literales del PDF.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Bot, FileText } from 'lucide-react';
import { getAnomaly, updateAnomalyStatus } from '@/api/anomalies';
import { isApiError } from '@/api/client';
import { getReadings } from '@/api/meters';
import { getDashboardSummary } from '@/api/dashboard';
import { AppLayout } from '@/app/layout/AppLayout';
import { AnomalyStatusBadge, Badge, SeverityBadge, TypeBadge } from '@/components/Badge';
import { ConfidenceBar } from '@/components/ConfidenceBar';
import { EmptyState, ErrorState, Skeleton } from '@/components/States';
import { useToast } from '@/components/Toast';
import { DailyBarsChart } from '@/components/charts/DailyBarsChart';
import { palette, variationColor } from '@/lib/colors';
import { formatConfidence, formatDate, formatDateTime, formatDayShort, formatNumber, formatPct, formatWithUnit } from '@/lib/format';
import { anomalyStatusLabel, confidenceLabel, metricLabel } from '@/lib/labels';
import type { AnomalyDetail, AnomalyStatus, ComparisonEntry } from '@/types/api';

const SECTIONS = ['Qué encontró la IA', 'Variables que cambiaron', 'Comparación contra baseline', 'Eventos relacionados', 'Severidad y confianza', 'Acción recomendada', 'Evidencia'] as const;

function Section({ title, children }: { title: (typeof SECTIONS)[number]; children: React.ReactNode }) {
  return (
    <section className="panel px-5 py-4">
      <h2 className="mb-3 text-md font-semibold text-ink">{title}</h2>
      {children}
    </section>
  );
}

const VARIABLES: { key: keyof AnomalyDetail['comparison']; title: string; unit: string; decimals: number }[] = [
  { key: 'consumption_kwh', title: 'Consumo', unit: ' kWh', decimals: 1 },
  { key: 'voltage_v', title: 'Voltaje', unit: ' V', decimals: 1 },
  { key: 'current_a', title: 'Corriente', unit: ' A', decimals: 1 },
  { key: 'power_factor', title: 'Factor de potencia', unit: '', decimals: 3 },
];

function VariableCard({ title, entry, unit, decimals }: { title: string; entry: ComparisonEntry; unit: string; decimals: number }) {
  return (
    <div className="px-4 py-1 first:pl-0 last:pr-0">
      <div className="text-xs text-ink-2">{title}</div>
      <div className="tnum mt-1 text-lg font-semibold text-ink">
        {formatNumber(entry.observed, decimals)}
        {unit}
      </div>
      <div className="tnum mt-0.5 text-sm font-medium" style={{ color: variationColor(entry.delta_pct) }}>
        {formatPct(entry.delta_pct)}
      </div>
      <div className="tnum mt-1 text-xs text-ink-2">
        base {formatNumber(entry.baseline, decimals)}
        {unit}
      </div>
    </div>
  );
}

const ACTIONS: { status: AnomalyStatus; label: string }[] = [
  { status: 'INVESTIGATING', label: 'Marcar en investigación' },
  { status: 'RESOLVED', label: 'Marcar resuelta' },
  { status: 'DISMISSED', label: 'Descartar' },
];

export function AnomalyDetailPage() {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const toast = useToast();
  const anomaly = useQuery({ queryKey: ['anomaly', id], queryFn: () => getAnomaly(id), enabled: Boolean(id) });
  const meterId = anomaly.data?.meter_id;
  const daily = useQuery({ queryKey: ['readings', meterId, 'day'], queryFn: () => getReadings(meterId as string, 'day'), enabled: Boolean(meterId), staleTime: 5 * 60_000 });
  const dashboard = useQuery({ queryKey: ['dashboard'], queryFn: getDashboardSummary });

  const mutation = useMutation({
    mutationFn: (status: AnomalyStatus) => updateAnomalyStatus(id, status),
    onSuccess: (data) => {
      queryClient.setQueryData(['anomaly', id], data);
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      toast.show(`Estado actualizado a ${anomalyStatusLabel(data.status)}`);
    },
    onError: (_e, status) => {
      toast.show('No se pudo actualizar el estado', { label: 'Reintentar', onClick: () => mutation.mutate(status) });
    },
  });

  const title = `Investigación · ${meterId ?? ''}`;
  const back = (
    <Link to="/anomalies" className="mb-4 inline-flex items-center gap-1.5 text-sm text-ink-2 hover:text-ink">
      <ArrowLeft size={14} strokeWidth={1.5} aria-hidden="true" /> Anomalías IA
    </Link>
  );

  if (anomaly.isError) {
    const notFound = isApiError(anomaly.error) && anomaly.error.status === 404;
    return (
      <AppLayout title="Investigación">
        {back}
        {notFound ? <EmptyState text="La anomalía no existe" actionLabel="Volver a anomalías" onAction={() => window.history.back()} /> : <ErrorState error={anomaly.error} text="No se pudo cargar la anomalía" onRetry={() => void anomaly.refetch()} />}
      </AppLayout>
    );
  }

  if (anomaly.isPending) {
    return (
      <AppLayout title="Investigación">
        {back}
        <div className="flex flex-col gap-4" aria-busy="true">
          <Skeleton className="h-10" />
          {SECTIONS.map((s) => (
            <Skeleton key={s} className="h-[120px]" />
          ))}
        </div>
      </AppLayout>
    );
  }

  const a = anomaly.data;
  const superseded = a.superseded || (dashboard.data?.last_analysis ? dashboard.data.last_analysis.id !== a.analysis_id : false);
  const actionsDisabled = superseded || mutation.isPending;

  return (
    <AppLayout title={title}>
      {back}
      {superseded && (
        <div role="status" className="mb-4 flex items-center justify-between rounded-md border border-amber/40 bg-amber/10 px-4 py-2.5 text-sm text-amber">
          <span>Esta anomalía pertenece a un análisis anterior</span>
          <Link to="/anomalies" className="font-semibold underline underline-offset-4 hover:text-ink">
            Ver anomalías vigentes
          </Link>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-3 text-sm text-ink-2">
        <Link to={`/meters/${a.meter_id}`} className="text-lg font-semibold text-ink hover:text-mint">
          {a.meter_id}
        </Link>
        <span className="text-ink-3">·</span>
        <TypeBadge type={a.type} />
        <SeverityBadge severity={a.severity} />
        <span className="text-ink-3">·</span>
        <span className="tnum">Confianza {formatConfidence(a.confidence)}</span>
        <span className="text-ink-3">·</span>
        <span className="tnum">
          Ventana {formatDayShort(a.window_start)} – {formatDate(a.window_end)}
        </span>
        <span className="text-ink-3">·</span>
        <Badge color={palette.ink2} icon={a.explanation_source === 'llm' ? Bot : FileText} label={a.explanation_source === 'llm' ? 'Explicación generada por IA' : 'Explicación por plantilla'} />
      </div>

      <div className="flex flex-col gap-4">
        <Section title="Qué encontró la IA">
          <p className="max-w-[75ch] text-sm leading-relaxed text-ink">{a.reason}</p>
          {a.explanation_points.length > 0 && (
            <ul className="mt-3 flex max-w-[75ch] list-disc flex-col gap-1.5 pl-5 text-sm text-ink-2">
              {a.explanation_points.map((p, i) => (
                <li key={i}>{p}</li>
              ))}
            </ul>
          )}
        </Section>

        <Section title="Variables que cambiaron">
          <div className="grid grid-cols-4 divide-x divide-line">
            {VARIABLES.map((v) => (
              <VariableCard key={v.key} title={v.title} entry={a.comparison[v.key]} unit={v.unit} decimals={v.decimals} />
            ))}
          </div>
        </Section>

        <Section title="Comparación contra baseline">
          {daily.isPending ? (
            <Skeleton className="h-[320px]" />
          ) : daily.isError ? (
            <ErrorState error={daily.error} text="No se pudieron cargar las lecturas" onRetry={() => void daily.refetch()} />
          ) : (
            <DailyBarsChart data={daily.data.items} baselineDailyKwh={a.comparison.consumption_kwh.baseline} window={{ start: a.window_start, end: a.window_end }} ariaLabel={`Consumo diario de ${a.meter_id} con baseline y ventana de la anomalía`} />
          )}
        </Section>

        <Section title="Eventos relacionados">
          {a.related_event ? (
            <div className="rounded-md border border-line bg-raised px-4 py-3 text-sm">
              <span className="tnum text-ink-2">{formatDateTime(a.related_event.timestamp)}</span>
              <span className="mx-2 text-ink-3">·</span>
              <span className="font-medium text-amber">{a.related_event.type}</span>
              <div className="mt-1 text-ink">{a.related_event.description}</div>
            </div>
          ) : (
            <p className="text-sm text-ink-2">Sin eventos relacionados</p>
          )}
        </Section>

        <Section title="Severidad y confianza">
          <div className="flex flex-wrap items-center gap-x-8 gap-y-2 text-sm">
            <span className="flex items-center gap-2">
              <span className="text-ink-2">Severidad:</span> <SeverityBadge severity={a.severity} />
            </span>
            <span className="flex items-center gap-2">
              <span className="text-ink-2">Confianza:</span>
              <span className="tnum text-ink">{formatConfidence(a.confidence)}</span>
              <span className="text-ink-2">({confidenceLabel(a.confidence)})</span>
              <ConfidenceBar value={a.confidence} width={120} />
            </span>
            <span className="flex items-center gap-2">
              <span className="text-ink-2">Estado:</span> <AnomalyStatusBadge status={a.status} />
              <span className="sr-only">Estado: {anomalyStatusLabel(a.status)}</span>
            </span>
          </div>
        </Section>

        <Section title="Acción recomendada">
          <p className="text-md font-medium text-ink">{a.recommended_action}</p>
          <div className="mt-4 flex gap-2">
            {ACTIONS.map((act) => (
              <button key={act.status} type="button" className={act.status === 'INVESTIGATING' ? 'btn-primary h-9' : 'btn-secondary h-9'} disabled={actionsDisabled || a.status === act.status} onClick={() => mutation.mutate(act.status)}>
                {act.label}
              </button>
            ))}
          </div>
        </Section>

        <Section title="Evidencia">
          <table className="table-base">
            <thead>
              <tr>
                <th scope="col" className="w-10">
                  #
                </th>
                <th scope="col">Métrica</th>
                <th scope="col" className="text-right">
                  Observado
                </th>
                <th scope="col" className="text-right">
                  Baseline
                </th>
                <th scope="col" className="text-right">
                  Delta
                </th>
                <th scope="col">Ventana</th>
                <th scope="col">Detalle</th>
              </tr>
            </thead>
            <tbody>
              {[...a.evidence]
                .sort((x, y) => x.position - y.position)
                .map((e) => (
                  <tr key={e.position}>
                    <td className="tnum text-ink-2">{e.position}</td>
                    <td className="font-medium text-ink">{metricLabel(e.metric)}</td>
                    <td className="tnum whitespace-nowrap text-right text-ink">{formatWithUnit(e.observed, e.unit)}</td>
                    <td className="tnum whitespace-nowrap text-right text-ink-2">{formatWithUnit(e.baseline, e.unit)}</td>
                    <td className="tnum whitespace-nowrap text-right font-medium" style={{ color: e.delta_pct === null ? palette.ink3 : variationColor(e.delta_pct) }}>
                      {formatPct(e.delta_pct)}
                    </td>
                    <td className="tnum whitespace-nowrap text-ink-2">{e.window}</td>
                    <td className="text-ink-2">{e.detail}</td>
                  </tr>
                ))}
            </tbody>
          </table>
        </Section>
      </div>
    </AppLayout>
  );
}
