// RF-F-01 — cliente HTTP.
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';
import { server } from '@/mocks/server';
import { DEMO_TOKEN } from '@/mocks/handlers';
import { storage } from '@/lib/storage';
import { API_URL, ApiError, request, setUnauthorizedHandler } from './client';

describe('api client', () => {
  it('envía Authorization: Bearer cuando hay token', async () => {
    storage.setToken(DEMO_TOKEN);
    let auth: string | null = null;
    server.use(
      http.get(`${API_URL}/meters`, ({ request: req }) => {
        auth = req.headers.get('authorization');
        return HttpResponse.json({ items: [], total: 0 });
      }),
    );
    await request('/meters');
    expect(auth).toBe(`Bearer ${DEMO_TOKEN}`);
  });

  it('convierte un error del backend en ApiError', async () => {
    storage.setToken(DEMO_TOKEN);
    await expect(request('/meters/M-999')).rejects.toMatchObject({ status: 404, code: 'METER_NOT_FOUND', message: 'No existe el medidor M-999' });
  });

  it('ante 401 borra el token y llama al handler una sola vez', async () => {
    storage.setToken('token-invalido');
    const handler = vi.fn();
    setUnauthorizedHandler(handler);
    await Promise.allSettled([request('/meters'), request('/dashboard/summary')]);
    expect(storage.getToken()).toBeNull();
    expect(handler).toHaveBeenCalledTimes(1);
    setUnauthorizedHandler(null);
  });

  it('error de red produce ApiError con isNetwork', async () => {
    server.use(http.get(`${API_URL}/health`, () => HttpResponse.error()));
    const err = await request('/health').catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isNetwork).toBe(true);
    expect((err as ApiError).message).toBe('No se pudo conectar con el servidor');
  });
});
