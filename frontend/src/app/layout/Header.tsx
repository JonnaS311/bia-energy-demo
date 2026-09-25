import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Sparkles } from 'lucide-react';
import { getDashboardSummary } from '@/api/dashboard';
import { isApiError } from '@/api/client';
import { useAnalysis } from '@/hooks/useAnalysis';
import { STROKE, palette } from '@/lib/colors';
import { formatRelative } from '@/lib/format';
import type { AnalysisStatus, DashboardSummary } from '@/types/api';

export function chipState(
  summary: DashboardSummary | undefined,
  summaryError: unknown,
  polling: AnalysisStatus | null,
  isPolling: boolean,
  now: Date,
): { text: string; color: string } {
  if (isPolling) {
    if (!polling || polling.status === 'QUEUED') return { text: 'En curso · En cola', color: palette.blue };
    const idx = polling.steps.findIndex((s) => s.status === 'RUNNING');
    if (polling.status === 'RUNNING' && idx >= 0) return { text: `En curso · ${polling.steps[idx].label} ${idx + 1}/7`, color: palette.blue };
    return { text: 'En curso', color: palette.blue };
  }
  if (summaryError && isApiError(summaryError) && summaryError.isNetwork) return { text: 'Sin conexión', color: palette.red };
  const last = summary?.last_analysis ?? null;
  if (!last) return { text: 'Sin análisis', color: palette.ink2 };
  if (last.status === 'COMPLETED') return { text: `Completado ${last.finished_at ? formatRelative(last.finished_at, now) : ''}`.trim(), color: palette.mint };
  if (last.status === 'FAILED') return { text: 'Falló', color: palette.red };
  return { text: 'En curso', color: palette.blue };
}

export function AnalysisChip() {
  const { analysis, isPolling } = useAnalysis();
  const { data, error } = useQuery({ queryKey: ['dashboard'], queryFn: getDashboardSummary, staleTime: 30_000 });
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 30_000);
    return () => clearInterval(t);
  }, []);
  const state = chipState(data, error, analysis, isPolling, now);
  return (
    <span className="inline-flex h-8 items-center gap-2 rounded-full border border-line bg-panel px-3 text-xs font-medium text-ink-2" aria-live="polite">
      <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: state.color, boxShadow: isPolling ? `0 0 8px ${state.color}` : undefined }} aria-hidden="true" />
      {state.text}
    </span>
  );
}

export function Header({ title }: { title: string }) {
  const { run, isPolling, isStarting, openModal, analysisId } = useAnalysis();
  const busy = isPolling || isStarting;
  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-line bg-canvas/95 px-8 backdrop-blur-sm">
      <h1 className="text-lg font-semibold tracking-tight text-ink">{title}</h1>
      <div className="flex items-center gap-3">
        <button type="button" className="rounded-full" onClick={analysisId ? openModal : undefined} aria-label="Estado del último análisis">
          <AnalysisChip />
        </button>
        <button type="button" className="btn-primary" onClick={() => void run()} disabled={busy} aria-disabled={busy}>
          <Sparkles size={16} strokeWidth={STROKE} aria-hidden="true" />
          {busy ? 'Analizando…' : 'Run AI Analysis'}
        </button>
      </div>
    </header>
  );
}
