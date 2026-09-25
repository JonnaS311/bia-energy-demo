// RF-F-09 — modal "Análisis de IA" con stepper de 7 pasos, contador vivo y pie por estado.
import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Check, Circle, LoaderCircle, X, XCircle } from 'lucide-react';
import { useAnalysis } from '@/hooks/useAnalysis';
import { STROKE } from '@/lib/colors';
import type { AnalysisStep, AnalysisStatus } from '@/types/api';

const WAIT_LIMIT_S = 180;

function StepIcon({ status }: { status: AnalysisStep['status'] }) {
  switch (status) {
    case 'COMPLETED':
      return <Check size={16} strokeWidth={STROKE} className="text-mint" aria-label="Completado" />;
    case 'RUNNING':
      return <LoaderCircle size={16} strokeWidth={STROKE} className="spin-slow text-blue" aria-label="En curso" />;
    case 'FAILED':
      return <XCircle size={16} strokeWidth={STROKE} className="text-red" aria-label="Falló" />;
    default:
      return <Circle size={16} strokeWidth={1.5} className="text-ink-3" aria-label="Pendiente" />;
  }
}

export function AnalysisStepper({ steps }: { steps: AnalysisStep[] }) {
  return (
    <ol className="flex flex-col gap-2" aria-label="Etapas del análisis">
      {steps.map((s) => (
        <li key={s.key} className="flex items-center gap-3 text-sm">
          <span className="flex h-5 w-5 items-center justify-center">
            <StepIcon status={s.status} />
          </span>
          <span className={`w-32 ${s.status === 'PENDING' ? 'text-ink-2' : 'text-ink'}`}>{s.label}</span>
          <span className="tnum flex-1 text-xs text-ink-2">{s.detail ?? (s.status === 'RUNNING' ? '(en curso)' : '')}</span>
        </li>
      ))}
    </ol>
  );
}

function useElapsed(startedAt: number | null, running: boolean): number {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!running) return;
    const t = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(t);
  }, [running]);
  if (!startedAt) return 0;
  return Math.max(0, Math.floor((now - startedAt) / 1000));
}

export function AnalysisModal() {
  const { analysis, isModalOpen, closeModal, startedAt, isPolling, run, analysisId } = useAnalysis();
  const navigate = useNavigate();
  const [waitExtended, setWaitExtended] = useState(0);
  const elapsed = useElapsed(startedAt, isPolling);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isModalOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && closeModal();
    window.addEventListener('keydown', onKey);
    dialogRef.current?.focus();
    return () => window.removeEventListener('keydown', onKey);
  }, [isModalOpen, closeModal]);

  useEffect(() => {
    if (!isPolling) setWaitExtended(0);
  }, [isPolling, analysisId]);

  if (!isModalOpen) return null;

  const status: AnalysisStatus['status'] | 'STARTING' = analysis?.status ?? 'STARTING';
  const tooLong = isPolling && elapsed >= WAIT_LIMIT_S + waitExtended;

  let footer: React.ReactNode;
  if (status === 'COMPLETED' && analysis?.summary) {
    footer = (
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-medium text-ink">
          {analysis.summary.anomalies_detected} anomalías detectadas · {analysis.summary.high_priority} requieren atención prioritaria
        </p>
        <div className="flex shrink-0 gap-2">
          <button type="button" className="btn-ghost h-9 whitespace-nowrap" onClick={closeModal}>
            Cerrar
          </button>
          <button
            type="button"
            className="btn-primary h-9 whitespace-nowrap"
            onClick={() => {
              closeModal();
              navigate('/anomalies');
            }}
          >
            Ver anomalías
          </button>
        </div>
      </div>
    );
  } else if (status === 'FAILED') {
    footer = (
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-red">El análisis falló: {analysis?.error_message ?? 'error desconocido'}</p>
        <button type="button" className="btn-secondary h-9" onClick={() => void run()}>
          Reintentar
        </button>
      </div>
    );
  } else if (tooLong) {
    footer = (
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-amber">El análisis está tardando más de lo esperado</p>
        <div className="flex gap-2">
          <button type="button" className="btn-ghost h-9" onClick={closeModal}>
            Cerrar
          </button>
          <button type="button" className="btn-secondary h-9" onClick={() => setWaitExtended((w) => w + WAIT_LIMIT_S)}>
            Seguir esperando
          </button>
        </div>
      </div>
    );
  } else {
    footer = (
      <p className="tnum text-sm text-ink-2" aria-live="polite">
        Analizando… {elapsed} s
      </p>
    );
  }

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40" onClick={closeModal}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="analysis-modal-title"
        tabIndex={-1}
        className="w-[560px] rounded-lg border border-line-strong bg-raised shadow-panel outline-none"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-center justify-between border-b border-line px-5 py-3">
          <h2 id="analysis-modal-title" className="text-md font-semibold text-ink">
            Análisis de IA
          </h2>
          <button type="button" aria-label="Cerrar" className="text-ink-2 hover:text-ink" onClick={closeModal}>
            <X size={18} strokeWidth={1.5} />
          </button>
        </header>
        <div className="px-5 py-4">
          {analysis ? <AnalysisStepper steps={analysis.steps} /> : <p className="text-sm text-ink-2">Iniciando análisis…</p>}
        </div>
        <footer className="border-t border-line px-5 py-3">{footer}</footer>
      </div>
    </div>
  );
}
