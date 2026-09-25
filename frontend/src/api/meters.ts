import { request } from './client';
import type {
  DailyReadingsResponse,
  Granularity,
  HourlyReadingsResponse,
  MeterDetail,
  MeterListQuery,
  MeterListResponse,
} from '@/types/api';

export function listMeters(query: MeterListQuery = {}): Promise<MeterListResponse> {
  return request<MeterListResponse>('/meters', { query: { ...query } });
}

export function getMeter(meterId: string): Promise<MeterDetail> {
  return request<MeterDetail>(`/meters/${encodeURIComponent(meterId)}`);
}

export function getReadings(meterId: string, granularity: 'hour'): Promise<HourlyReadingsResponse>;
export function getReadings(meterId: string, granularity: 'day'): Promise<DailyReadingsResponse>;
export function getReadings(meterId: string, granularity: Granularity) {
  return request(`/meters/${encodeURIComponent(meterId)}/readings`, { query: { granularity } });
}
