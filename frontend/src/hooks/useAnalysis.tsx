// Contexto de análisis: arranque (POST /ai/analyze), polling, modal, 409, reanudación tras recarga (CB-F-08).
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { startAnalysis } from '@/api/analysis';
import { isApiError } from '@/api/client';
import { storage } from '@/lib/storage';
import { useAnalysisPolling } from './useAnalysisPolling';
import { useToast } from '@/components/Toast';
import type { AnalysisStatus } from '@/types/api';

interface AnalysisContextValue {
  analysisId: string | null;
  analysis: AnalysisStatus | null;
  isPolling: boolean;
  isStarting: boolean;
  isModalOpen: boolean;
  startedAt: number | null;
  /** Ids de medidores cuyo estado cambió en el último análisis completado (para el encendido secuencial). */
  lastCompletedAt: number | null;
  run: () => Promise<void>;
  openModal: () => void;
  closeModal: () => void;
}

const AnalysisContext = createContext<AnalysisContextValue | null>(null);

export function AnalysisProvider({ children }: { children: ReactNode }) {
  const [analysisId, setAnalysisId] = useState<string | null>(() => storage.getAnalysisId());
  const [isStarting, setIsStarting] = useState(false);
  const [isModalOpen, setModalOpen] = useState(false);
  const [startedAt, setStartedAt] = useState<number | null>(null);
  const [lastCompletedAt, setLastCompletedAt] = useState<number | null>(null);
  const queryClient = useQueryClient();
  const toast = useToast();
  const modalOpenRef = useRef(isModalOpen);
  modalOpenRef.current = isModalOpen;

  const onComplete = useCallback(() => {
    setLastCompletedAt(Date.now());
  }, []);

  const onFailed = useCallback(
    (a: AnalysisStatus) => {
      if (!modalOpenRef.current) toast.show(`El análisis falló: ${a.error_message ?? 'error desconocido'}`);
    },
    [toast],
  );

  const onNotFound = useCallback(() => {
    setAnalysisId(null);
    setModalOpen(false);
    toast.show('El análisis ya no existe');
  }, [toast]);

  const { analysis, isPolling } = useAnalysisPolling(analysisId, { onComplete, onFailed, onNotFound });

  // Reanudación tras recarga: si hay un id guardado se retoma el polling sin abrir el modal.
  useEffect(() => {
    if (analysisId && !startedAt) setStartedAt(Date.now());
  }, [analysisId, startedAt]);

  const run = useCallback(async () => {
    if (isStarting || isPolling) return;
    setIsStarting(true);
    try {
      const res = await startAnalysis();
      storage.setAnalysisId(res.analysis_id);
      setAnalysisId(res.analysis_id);
      setStartedAt(Date.now());
      setModalOpen(true);
    } catch (e) {
      if (isApiError(e) && e.status === 409) {
        toast.show('Ya hay un análisis en curso');
        const details = e.details as { analysis_id?: string } | null;
        if (details?.analysis_id) {
          storage.setAnalysisId(details.analysis_id);
          setAnalysisId(details.analysis_id);
          setStartedAt(Date.now());
          setModalOpen(true);
        } else {
          queryClient.invalidateQueries({ queryKey: ['dashboard'] });
        }
      } else if (isApiError(e) && e.isNetwork) {
        toast.show('No se pudo conectar con el servidor');
      } else {
        toast.show('No se pudo iniciar el análisis');
      }
    } finally {
      setIsStarting(false);
    }
  }, [isStarting, isPolling, toast, queryClient]);

  const value = useMemo<AnalysisContextValue>(
    () => ({
      analysisId,
      analysis,
      isPolling,
      isStarting,
      isModalOpen,
      startedAt,
      lastCompletedAt,
      run,
      openModal: () => setModalOpen(true),
      closeModal: () => setModalOpen(false),
    }),
    [analysisId, analysis, isPolling, isStarting, isModalOpen, startedAt, lastCompletedAt, run],
  );

  return <AnalysisContext.Provider value={value}>{children}</AnalysisContext.Provider>;
}

export function useAnalysis(): AnalysisContextValue {
  const ctx = useContext(AnalysisContext);
  if (!ctx) throw new Error('useAnalysis debe usarse dentro de AnalysisProvider');
  return ctx;
}
