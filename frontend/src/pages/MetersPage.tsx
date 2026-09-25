// RF-F-05 — Medidores: filtro segmentado, búsqueda con debounce, orden por cabecera, filas clicables.
import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ChevronDown, ChevronUp, Search } from 'lucide-react';
import { listMeters } from '@/api/meters';
import { AppLayout } from '@/app/layout/AppLayout';
import { SeverityBadge, StatusBadge } from '@/components/Badge';
import { Segmented } from '@/components/Segmented';
import { EmptyState, ErrorState, SkeletonRows } from '@/components/States';
import { useDebounce } from '@/hooks/useDebounce';
import { variationColor } from '@/lib/colors';
import { formatKwh, formatPct } from '@/lib/format';
import type { MeterListItem, MeterSort, MeterStatus, SortOrder } from '@/types/api';

type Filter = 'ALL' | MeterStatus;
type ClientSort = MeterSort | 'meter';

const FILTERS: { value: Filter; label: string }[] = [
  { value: 'ALL', label: 'Todos' },
  { value: 'OK', label: 'Normales' },
  { value: 'ALERT', label: 'Alertas' },
  { value: 'CRITICAL', label: 'Críticas' },
];

const DEFAULT_SORT: MeterSort = 'severity';
const DEFAULT_ORDER: SortOrder = 'desc';

function SortIndicator({ active, order }: { active: boolean; order: SortOrder }) {
  if (!active) return null;
  return order === 'desc' ? <ChevronDown size={14} strokeWidth={1.5} aria-label="▼" /> : <ChevronUp size={14} strokeWidth={1.5} aria-label="▲" />;
}

export function MetersPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const status = (params.get('status') as MeterStatus | null) ?? null;
  const filter: Filter = status ?? 'ALL';
  const sortParam = (params.get('sort') as ClientSort | null) ?? DEFAULT_SORT;
  const order = (params.get('order') as SortOrder | null) ?? DEFAULT_ORDER;
  const [search, setSearch] = useState(params.get('q') ?? '');
  const q = useDebounce(search.trim(), 300);

  useEffect(() => {
    const current = params.get('q') ?? '';
    if (current !== q) {
      const next = new URLSearchParams(params);
      if (q) next.set('q', q);
      else next.delete('q');
      setParams(next, { replace: true });
    }
  }, [q, params, setParams]);

  const serverSort: MeterSort = sortParam === 'meter' ? DEFAULT_SORT : sortParam;
  const query = useQuery({
    queryKey: ['meters', { status: status ?? undefined, q: q || undefined, sort: serverSort, order }],
    queryFn: () => listMeters({ status: status ?? undefined, q: q || undefined, sort: serverSort, order }),
    placeholderData: (prev) => prev,
  });

  const items = useMemo<MeterListItem[]>(() => {
    const list = query.data?.items ?? [];
    if (sortParam !== 'meter') return list;
    return [...list].sort((a, b) => (order === 'asc' ? a.meter_id.localeCompare(b.meter_id) : b.meter_id.localeCompare(a.meter_id)));
  }, [query.data, sortParam, order]);

  function update(mutate: (p: URLSearchParams) => void) {
    const next = new URLSearchParams(params);
    mutate(next);
    setParams(next);
  }

  function setFilter(f: Filter) {
    update((p) => (f === 'ALL' ? p.delete('status') : p.set('status', f)));
  }

  function toggleSort(col: ClientSort) {
    update((p) => {
      if (sortParam !== col) {
        p.set('sort', col);
        p.set('order', 'desc');
      } else if (order === 'desc') {
        p.set('order', 'asc');
      } else {
        p.delete('sort');
        p.delete('order');
      }
    });
  }

  function clearFilters() {
    setSearch('');
    update((p) => {
      p.delete('status');
      p.delete('q');
    });
  }

  const hasFilters = Boolean(status || q);
  const columns: { key: ClientSort; label: string; className?: string }[] = [
    { key: 'meter', label: 'Medidor' },
    { key: 'consumption', label: 'Consumo (kWh)', className: 'text-right' },
    { key: 'variation', label: 'Variación', className: 'text-right' },
    { key: 'severity', label: 'Estado' },
  ];

  return (
    <AppLayout title="Medidores">
      <div className="mb-4 flex items-center justify-between gap-4">
        <Segmented options={FILTERS} value={filter} onChange={setFilter} ariaLabel="Filtrar por estado" />
        <label className="relative w-[280px]">
          <span className="sr-only">Buscar por ID de medidor</span>
          <Search size={16} strokeWidth={1.5} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-3" aria-hidden="true" />
          <input type="search" className="field h-9 pl-9" placeholder="Buscar por ID de medidor" value={search} onChange={(e) => setSearch(e.target.value)} />
        </label>
      </div>

      <div className="panel overflow-hidden">
        {query.isPending ? (
          <div className="p-4">
            <SkeletonRows rows={12} />
          </div>
        ) : query.isError ? (
          <ErrorState error={query.error} text="No se pudieron cargar los medidores" onRetry={() => void query.refetch()} />
        ) : (
          <table className="table-base" style={{ opacity: query.isFetching ? 0.7 : 1, transition: 'opacity 150ms' }}>
            <thead>
              <tr>
                {columns.map((c) => (
                  <th key={c.key} scope="col" className={c.className}>
                    <button type="button" onClick={() => toggleSort(c.key)} className={`inline-flex items-center gap-1 uppercase tracking-[0.06em] hover:text-ink ${c.className === 'text-right' ? 'flex-row-reverse' : ''}`} aria-sort={sortParam === c.key ? (order === 'asc' ? 'ascending' : 'descending') : 'none'}>
                      {c.label}
                      <SortIndicator active={sortParam === c.key} order={order} />
                    </button>
                  </th>
                ))}
                <th scope="col">Anomalía</th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5}>
                    <EmptyState text="Ningún medidor coincide con los filtros" actionLabel={hasFilters ? 'Limpiar filtros' : undefined} onAction={hasFilters ? clearFilters : undefined} />
                  </td>
                </tr>
              ) : (
                items.map((m) => (
                  <tr key={m.meter_id} className="cursor-pointer transition-colors duration-150 hover:bg-hover" onClick={() => navigate(`/meters/${m.meter_id}`)}>
                    <td>
                      <div className="font-semibold text-ink">{m.meter_id}</div>
                      <div className="text-xs text-ink-2">{m.name}</div>
                    </td>
                    <td className="tnum text-right text-ink">{formatKwh(m.current_consumption_kwh)}</td>
                    <td className="tnum text-right font-medium" style={{ color: variationColor(m.variation_pct) }}>
                      {formatPct(m.variation_pct)}
                    </td>
                    <td>
                      <StatusBadge status={m.status} />
                    </td>
                    <td>{m.anomaly ? <SeverityBadge severity={m.anomaly.severity} /> : <span className="text-ink-3">—</span>}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
      {query.data && <p className="tnum mt-3 text-xs text-ink-2">{query.data.total} medidores</p>}
    </AppLayout>
  );
}
