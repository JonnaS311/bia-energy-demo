// RF-F-07 — Anomalías IA: tabla priorizada en el orden del backend, filtros por tipo/severidad/estado.
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { listAnomalies } from '@/api/anomalies';
import { AppLayout } from '@/app/layout/AppLayout';
import { AnomalyStatusBadge, SeverityBadge, TypeBadge } from '@/components/Badge';
import { ConfidenceBar } from '@/components/ConfidenceBar';
import { EmptyState, ErrorState, SkeletonRows } from '@/components/States';
import { ANOMALY_STATUSES, ANOMALY_TYPES, SEVERITIES, anomalyStatusLabel, severityLabel, typeLabel } from '@/lib/labels';
import type { AnomalyStatus, AnomalyType, Severity } from '@/types/api';

function Select<T extends string>({ id, label, value, options, allLabel, onChange }: { id: string; label: string; value: T | ''; options: { value: T; label: string }[]; allLabel: string; onChange: (v: T | '') => void }) {
  return (
    <label className="flex items-center gap-2 text-xs text-ink-2" htmlFor={id}>
      {label}
      <select id={id} className="field h-9 w-auto min-w-[150px] pr-8" value={value} onChange={(e) => onChange(e.target.value as T | '')}>
        <option value="">{allLabel}</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}

export function AnomaliesPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const type = (params.get('type') as AnomalyType | null) ?? '';
  const severity = (params.get('severity') as Severity | null) ?? '';
  const status = (params.get('status') as AnomalyStatus | null) ?? '';
  const hasFilters = Boolean(type || severity || status);

  const query = useQuery({
    queryKey: ['anomalies', { type: type || undefined, severity: severity || undefined, status: status || undefined }],
    queryFn: () => listAnomalies({ type: type || undefined, severity: severity || undefined, status: status || undefined }),
    placeholderData: (prev) => prev,
  });

  function setParam(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next);
  }

  return (
    <AppLayout title="Anomalías IA">
      <div className="mb-4 flex items-center gap-4">
        <Select id="f-type" label="Tipo" value={type} allLabel="Todos" options={ANOMALY_TYPES.map((t) => ({ value: t, label: typeLabel(t) }))} onChange={(v) => setParam('type', v)} />
        <Select id="f-severity" label="Severidad" value={severity} allLabel="Todas" options={SEVERITIES.map((s) => ({ value: s, label: severityLabel(s) }))} onChange={(v) => setParam('severity', v)} />
        <Select id="f-status" label="Estado" value={status} allLabel="Todos" options={ANOMALY_STATUSES.map((s) => ({ value: s, label: anomalyStatusLabel(s) }))} onChange={(v) => setParam('status', v)} />
      </div>

      <div className="panel overflow-hidden">
        {query.isPending ? (
          <div className="p-4">
            <SkeletonRows rows={4} />
          </div>
        ) : query.isError ? (
          <ErrorState error={query.error} text="No se pudieron cargar las anomalías" onRetry={() => void query.refetch()} />
        ) : query.data.items.length === 0 ? (
          hasFilters ? (
            <EmptyState text="Ninguna anomalía coincide con los filtros" actionLabel="Limpiar filtros" onAction={() => setParams(new URLSearchParams())} />
          ) : (
            <EmptyState text="Aún no hay análisis. Pulsa Run AI Analysis para detectar anomalías." />
          )
        ) : (
          <table className="table-base" style={{ opacity: query.isFetching ? 0.7 : 1, transition: 'opacity 150ms' }}>
            <thead>
              <tr>
                <th scope="col" className="w-10">
                  #
                </th>
                <th scope="col">Medidor</th>
                <th scope="col">Tipo</th>
                <th scope="col">Severidad</th>
                <th scope="col">Confianza</th>
                <th scope="col">Acción recomendada</th>
                <th scope="col">Estado</th>
              </tr>
            </thead>
            <tbody>
              {query.data.items.map((a) => (
                <tr key={a.id} className="cursor-pointer transition-colors duration-150 hover:bg-hover" onClick={() => navigate(`/anomalies/${a.id}`)}>
                  <td className="tnum text-ink-2">{a.priority_rank}</td>
                  <td className="font-semibold text-ink">{a.meter_id}</td>
                  <td>
                    <TypeBadge type={a.type} />
                  </td>
                  <td>
                    <SeverityBadge severity={a.severity} />
                  </td>
                  <td>
                    <ConfidenceBar value={a.confidence} />
                  </td>
                  <td className="max-w-[320px]">
                    <span className="block truncate text-ink-2" title={a.recommended_action}>
                      {a.recommended_action}
                    </span>
                  </td>
                  <td>
                    <AnomalyStatusBadge status={a.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </AppLayout>
  );
}
