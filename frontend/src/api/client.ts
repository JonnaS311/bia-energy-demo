// RF-F-01 — cliente HTTP: base URL, Authorization, manejo global de 401, ApiError.
import type { ApiErrorBody } from '@/types/api';
import { storage } from '@/lib/storage';

export const API_URL: string = (import.meta.env.VITE_API_URL as string | undefined) || 'http://localhost:8080';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: unknown;
  constructor(status: number, code: string, message: string, details: unknown = null) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
  /** Error de red: sin respuesta HTTP. */
  get isNetwork(): boolean {
    return this.status === 0;
  }
}

export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError;
}

type UnauthorizedHandler = () => void;
let onUnauthorized: UnauthorizedHandler | null = null;
let redirecting = false;

/** El AuthProvider registra aquí la redirección a /login (una sola vez aunque fallen varias peticiones). */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  onUnauthorized = handler;
  redirecting = false;
}

function handleUnauthorized(): void {
  if (redirecting) return;
  redirecting = true;
  storage.clearSession();
  onUnauthorized?.();
  // Permite una nueva redirección tras un nuevo login.
  setTimeout(() => {
    redirecting = false;
  }, 1000);
}

export type Query = Record<string, string | number | boolean | undefined | null>;

export function buildQuery(query?: Query): string {
  if (!query) return '';
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v === undefined || v === null || v === '') continue;
    params.set(k, String(v));
  }
  const s = params.toString();
  return s ? `?${s}` : '';
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH';
  body?: unknown;
  query?: Query;
  auth?: boolean;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, auth = true } = options;
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const token = auth ? storage.getToken() : null;
  if (token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}${buildQuery(query)}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      cache: 'no-store',
    });
  } catch (e) {
    throw new ApiError(0, 'NETWORK_ERROR', 'No se pudo conectar con el servidor', e);
  }

  if (res.status === 204) return undefined as T;

  let payload: unknown = null;
  const text = await res.text();
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = null;
    }
  }

  if (!res.ok) {
    const errBody = (payload ?? {}) as Partial<ApiErrorBody>;
    const code = errBody.error?.code ?? (res.status === 401 ? 'UNAUTHORIZED' : 'INTERNAL_ERROR');
    const message = errBody.error?.message ?? `Error ${res.status}`;
    const details = errBody.error?.details ?? null;
    if (res.status === 401 && auth) handleUnauthorized();
    throw new ApiError(res.status, code, message, details);
  }

  return payload as T;
}
