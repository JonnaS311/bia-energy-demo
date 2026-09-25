import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { cleanup } from '@testing-library/react';
import { server } from '@/mocks/server';
import { resetMockState } from '@/mocks/handlers';

// jsdom no implementa ResizeObserver (Recharts ResponsiveContainer lo necesita).
class RO {
  observe() {}
  unobserve() {}
  disconnect() {}
}
(globalThis as unknown as { ResizeObserver: typeof RO }).ResizeObserver = RO;

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
  resetMockState();
  localStorage.clear();
  sessionStorage.clear();
});
afterAll(() => server.close());
