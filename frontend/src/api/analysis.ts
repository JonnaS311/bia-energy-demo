import { request } from './client';
import type { AnalysisStatus, AnalyzeResponse } from '@/types/api';

export function startAnalysis(): Promise<AnalyzeResponse> {
  return request<AnalyzeResponse>('/ai/analyze', { method: 'POST' });
}

export function getAnalysis(id: string): Promise<AnalysisStatus> {
  return request<AnalysisStatus>(`/ai/analysis/${encodeURIComponent(id)}`);
}
