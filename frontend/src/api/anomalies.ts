import { request } from './client';
import type { AnomalyDetail, AnomalyListQuery, AnomalyListResponse, AnomalyStatus } from '@/types/api';

export function listAnomalies(query: AnomalyListQuery = {}): Promise<AnomalyListResponse> {
  return request<AnomalyListResponse>('/anomalies', { query: { ...query } });
}

export function getAnomaly(id: string): Promise<AnomalyDetail> {
  return request<AnomalyDetail>(`/anomalies/${encodeURIComponent(id)}`);
}

export function updateAnomalyStatus(id: string, status: AnomalyStatus): Promise<AnomalyDetail> {
  return request<AnomalyDetail>(`/anomalies/${encodeURIComponent(id)}/status`, { method: 'PATCH', body: { status } });
}
