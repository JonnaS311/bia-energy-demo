import { request } from './client';
import type { LoginResponse } from '@/types/api';

export function login(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/login', { method: 'POST', body: { email, password }, auth: false });
}
