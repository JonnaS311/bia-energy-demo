import { request } from './client';
import type { DashboardSummary } from '@/types/api';

export function getDashboardSummary(): Promise<DashboardSummary> {
  return request<DashboardSummary>('/dashboard/summary');
}
