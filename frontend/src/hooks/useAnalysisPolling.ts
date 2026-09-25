// RF-F-09 — polling de GET /ai/analysis/:id cada 1.000 ms mientras el estado no sea final.
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { getAnalysis } from '@/api/analysis';
import { isApiError } from '@/api/client';
import { storage } from '@/lib/storage';
import type { AnalysisStatus } from '@/types/api';

export const POLL_INTERVAL_MS = 1000;
export const INVALIDATE_ON_COMPLETE = [['dashboard'], ['meters'], ['meter'], ['anomalies'], ['anomaly']] as const;

export function isFinal(status: AnalysisStatus['status'] | undefined): boolean {
  return status === 'COMPLETED' || status === 'FAILED';
}

interface Options {
  onComplete?: (analysis: AnalysisStatus) => void;
  onFailed?: (analysis: AnalysisStatus) => void;
  onNotFound?: () => void;
}

export function useAnalysisPolling(analysisId: string | null, options: Options = {}) {
  const queryClient = useQueryClient();
  const handled = useRef<string | null>(null);
  const { onComplete, onFailed, onNotFound } = options;

  const query = useQuery({
    queryKey: ['analysis', analysisId],
    queryFn: () => getAnalysis(analysisId as string),
    enabled: Boolean(analysisId),
    staleTime: 0,
    gcTime: 0,
    retry: (count, error) => !(isApiError(error) && error.status === 404) && count < 2,
    refetchInterval: (q) => (isFinal(q.state.data?.status) ? false : POLL_INTERVAL_MS),
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: false,
  });

  const data = query.data;
  const error = query.error;

  useEffect(() => {
    if (!analysisId || !data || !isFinal(data.status)) return;
    if (handled.current === analysisId) return;
    handled.current = analysisId;
    storage.clearAnalysisId();
    if (data.status === 'COMPLETED') {
      INVALIDATE_ON_COMPLETE.forEach((key) => queryClient.invalidateQueries({ queryKey: [...key] }));
      onComplete?.(data);
    } else {
      onFailed?.(data);
    }
  }, [analysisId, data, queryClient, onComplete, onFailed]);

  useEffect(() => {
    if (!analysisId || !error) return;
    if (isApiError(error) && error.status === 404) {
      storage.clearAnalysisId();
      onNotFound?.();
    }
  }, [analysisId, error, onNotFound]);

  return {
    analysis: data ?? null,
    isPolling: Boolean(analysisId) && !isFinal(data?.status) && !(isApiError(error) && error.status === 404),
    error,
  };
}
