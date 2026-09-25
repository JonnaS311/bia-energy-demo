// Utilidad de render para tests de páginas: router en memoria + proveedores + sesión demo.
import { QueryClient } from '@tanstack/react-query';
import { render } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { RootProviders, RouterProviders } from '@/app/providers';
import { storage } from '@/lib/storage';
import { DEMO_TOKEN } from '@/mocks/handlers';

export function loginAsDemo(): void {
  storage.setToken(DEMO_TOKEN);
  storage.setUser({ email: 'analista@energy.local', name: 'Analista Demo' });
}

export function testQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 0, gcTime: 0 } } });
}

export function renderRoute(path: string, routes: { path: string; element: ReactNode }[]) {
  const client = testQueryClient();
  const utils = render(
    <RootProviders client={client}>
      <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <RouterProviders>
          <Routes>
            {routes.map((r) => (
              <Route key={r.path} path={r.path} element={r.element} />
            ))}
          </Routes>
        </RouterProviders>
      </MemoryRouter>
    </RootProviders>,
  );
  return { ...utils, client };
}
