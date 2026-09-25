// RF-F-09 — polling: 1 petición por segundo, se detiene en estado final, sessionStorage.
import { QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { API_URL } from '@/api/client';
import { storage } from '@/lib/storage';
import { server } from '@/mocks/server';
import { loginAsDemo, testQueryClient } from '@/test/render';
import type { AnalysisStatus } from '@/types/api';
import { useAnalysisPolling } from './useAnalysisPolling';

function baseAnalysis(status: AnalysisStatus['status']): AnalysisStatus {
  return { id: 'a1', status, steps: [], summary: status === 'COMPLETED' ? { anomalies_detected: 4, high_priority: 2, meters_analyzed: 12, explanation_source: 'template' } : null, error_message: null, started_at: null, finished_at: null };
}

describe('useAnalysisPolling', () => {
  it('consulta cada segundo hasta COMPLETED, invoca onComplete y borra sessionStorage', async () => {
    loginAsDemo();
    storage.setAnalysisId('a1');
    let calls = 0;
    server.use(
      http.get(`${API_URL}/ai/analysis/a1`, () => {
        calls += 1;
        return HttpResponse.json(baseAnalysis(calls >= 3 ? 'COMPLETED' : 'RUNNING'));
      }),
    );
    const client = testQueryClient();
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAnalysisPolling('a1', { onComplete }), { wrapper });

    await waitFor(() => expect(result.current.isPolling).toBe(true));
    await waitFor(() => expect(onComplete).toHaveBeenCalledTimes(1), { timeout: 5000 });
    expect(result.current.isPolling).toBe(false);
    expect(storage.getAnalysisId()).toBeNull();
    const callsAtCompletion = calls;
    await new Promise((r) => setTimeout(r, 1500));
    expect(calls).toBe(callsAtCompletion);
  });

  it('404 detiene el polling y llama onNotFound', async () => {
    loginAsDemo();
    storage.setAnalysisId('missing');
    const client = testQueryClient();
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    const onNotFound = vi.fn();
    const { result } = renderHook(() => useAnalysisPolling('missing', { onNotFound }), { wrapper });
    await waitFor(() => expect(onNotFound).toHaveBeenCalled());
    expect(result.current.isPolling).toBe(false);
    expect(storage.getAnalysisId()).toBeNull();
  });
});
